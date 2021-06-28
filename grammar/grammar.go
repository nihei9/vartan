package grammar

import (
	"fmt"

	mlcompiler "github.com/nihei9/maleeni/compiler"
	mlspec "github.com/nihei9/maleeni/spec"
	"github.com/nihei9/vartan/spec"
)

type astActionEntry struct {
	position  int
	expansion bool
}

type Grammar struct {
	lexSpec              *mlspec.LexSpec
	skipLexKinds         []mlspec.LexKind
	productionSet        *productionSet
	augmentedStartSymbol symbol
	symbolTable          *symbolTable
	astActions           map[productionID][]*astActionEntry
}

func NewGrammar(root *spec.RootNode) (*Grammar, error) {
	symTab := newSymbolTable()
	anonPat2Sym := map[string]symbol{}
	var lexSpec *mlspec.LexSpec
	var skip []mlspec.LexKind
	{
		entries := []*mlspec.LexEntry{}
		anonPats := []string{}
		for _, prod := range root.Productions {
			if isLexicalProduction(prod) {
				_, err := symTab.registerTerminalSymbol(prod.LHS)
				if err != nil {
					return nil, err
				}

				var modes []mlspec.LexModeName
				if prod.Modifier != nil {
					mod := prod.Modifier
					switch mod.Name {
					case "mode":
						if mod.Parameter == "" {
							return nil, fmt.Errorf("modifier 'mode' needs a parameter")
						}
						modes = []mlspec.LexModeName{
							mlspec.LexModeName(mod.Parameter),
						}
					default:
						return nil, fmt.Errorf("invalid modifier name '%v'", mod.Name)
					}
				}

				alt := prod.RHS[0]
				var push mlspec.LexModeName
				var pop bool
				if alt.Action != nil {
					act := alt.Action
					switch act.Name {
					case "skip":
						param := act.Parameter
						if param != nil {
							return nil, fmt.Errorf("action 'skip' needs no parameter")
						}
						skip = append(skip, mlspec.LexKind(prod.LHS))
					case "push":
						param := act.Parameter
						if param == nil || param.ID == "" {
							return nil, fmt.Errorf("action 'push' needs an ID parameter")
						}
						push = mlspec.LexModeName(param.ID)
					case "pop":
						param := act.Parameter
						if param != nil {
							return nil, fmt.Errorf("action 'pop' needs no parameter")
						}
						pop = true
					default:
						return nil, fmt.Errorf("invalid action name '%v'", act.Name)
					}
				}

				entries = append(entries, &mlspec.LexEntry{
					Modes:   modes,
					Kind:    mlspec.LexKind(prod.LHS),
					Pattern: mlspec.LexPattern(alt.Elements[0].Pattern),
					Push:    push,
					Pop:     pop,
				})

				continue
			}

			for _, alt := range prod.RHS {
				for _, elem := range alt.Elements {
					if elem.Pattern == "" {
						continue
					}
					exist := false
					for _, p := range anonPats {
						if p == elem.Pattern {
							exist = true
							break
						}
					}
					if exist {
						continue
					}
					anonPats = append(anonPats, elem.Pattern)
				}
			}
		}

		var anonEntries []*mlspec.LexEntry
		for i, p := range anonPats {
			kind := fmt.Sprintf("__%v__", i+1)

			sym, err := symTab.registerTerminalSymbol(kind)
			if err != nil {
				return nil, err
			}
			anonPat2Sym[p] = sym

			anonEntries = append(anonEntries, &mlspec.LexEntry{
				Kind:    mlspec.LexKind(kind),
				Pattern: mlspec.LexPattern(p),
			})
		}
		// Anonymous patterns take precedence over explicitly defined lexical specifications.
		entries = append(anonEntries, entries...)

		for _, fragment := range root.Fragments {
			entries = append(entries, &mlspec.LexEntry{
				Fragment: true,
				Kind:     mlspec.LexKind(fragment.LHS),
				Pattern:  mlspec.LexPattern(fragment.RHS),
			})
		}

		lexSpec = &mlspec.LexSpec{
			Entries: entries,
		}
	}

	prods := newProductionSet()
	var augStartSym symbol
	astActs := map[productionID][]*astActionEntry{}
	{
		startProd := root.Productions[0]
		augStartText := fmt.Sprintf("%s'", startProd.LHS)
		var err error
		augStartSym, err = symTab.registerStartSymbol(augStartText)
		if err != nil {
			return nil, err
		}
		startSym, err := symTab.registerNonTerminalSymbol(startProd.LHS)
		if err != nil {
			return nil, err
		}
		p, err := newProduction(augStartSym, []symbol{
			startSym,
		})
		if err != nil {
			return nil, err
		}
		prods.append(p)

		for _, prod := range root.Productions {
			if isLexicalProduction(prod) {
				continue
			}
			_, err := symTab.registerNonTerminalSymbol(prod.LHS)
			if err != nil {
				return nil, err
			}
		}

		for _, prod := range root.Productions {
			if isLexicalProduction(prod) {
				continue
			}
			lhsSym, ok := symTab.toSymbol(prod.LHS)
			if !ok {
				return nil, fmt.Errorf("symbol '%v' is undefined", prod.LHS)
			}
			for _, alt := range prod.RHS {
				altSyms := make([]symbol, len(alt.Elements))
				for i, elem := range alt.Elements {
					var sym symbol
					if elem.Pattern != "" {
						var ok bool
						sym, ok = anonPat2Sym[elem.Pattern]
						if !ok {
							return nil, fmt.Errorf("pattern '%v' is undefined", elem.Pattern)
						}
					} else {
						var ok bool
						sym, ok = symTab.toSymbol(elem.ID)
						if !ok {
							return nil, fmt.Errorf("symbol '%v' is undefined", elem.ID)
						}
					}
					altSyms[i] = sym
				}
				p, err := newProduction(lhsSym, altSyms)
				if err != nil {
					return nil, err
				}
				prods.append(p)

				if alt.Action != nil {
					act := alt.Action
					switch act.Name {
					case "ast":
						param := act.Parameter
						if param == nil || param.Tree == nil {
							return nil, fmt.Errorf("action 'push' needs a tree parameter")
						}
						lhsText, ok := symTab.toText(p.lhs)
						if !ok || param.Tree.Name != lhsText {
							return nil, fmt.Errorf("a name of a tree structure must be the same ID as an LHS of a production; LHS: %v", lhsText)
						}
						astAct := make([]*astActionEntry, len(param.Tree.Children))
						for i, c := range param.Tree.Children {
							if c.Position > len(alt.Elements) {
								return nil, fmt.Errorf("a position must be less than or equal to the length of an alternative; alternative length: %v", len(alt.Elements))
							}

							if c.Expansion {
								offset := c.Position - 1
								elem := alt.Elements[offset]
								if elem.Pattern != "" {
									return nil, fmt.Errorf("the expansion symbol cannot be applied to a pattern ($%v: %v)", c.Position, elem.Pattern)
								}
								elemSym, ok := symTab.toSymbol(elem.ID)
								if !ok {
									// If the symbol was not found, it's a bug.
									return nil, fmt.Errorf("a symbol corresponding to a position ($%v: %v) was not found", c.Position, elem.ID)
								}
								if elemSym.isTerminal() {
									return nil, fmt.Errorf("the expansion symbol cannot be applied to a terminal symbol ($%v: %v)", c.Position, elem.ID)
								}
							}

							astAct[i] = &astActionEntry{
								position:  c.Position,
								expansion: c.Expansion,
							}
						}
						astActs[p.id] = astAct
					default:
						return nil, fmt.Errorf("invalid action name '%v'", act.Name)
					}
				}
			}
		}
	}

	return &Grammar{
		lexSpec:              lexSpec,
		skipLexKinds:         skip,
		productionSet:        prods,
		augmentedStartSymbol: augStartSym,
		symbolTable:          symTab,
		astActions:           astActs,
	}, nil
}

func isLexicalProduction(prod *spec.ProductionNode) bool {
	if len(prod.RHS) == 1 && len(prod.RHS[0].Elements) == 1 && prod.RHS[0].Elements[0].Pattern != "" {
		return true
	}
	return false
}

func Compile(gram *Grammar) (*spec.CompiledGrammar, error) {
	lexSpec, err := mlcompiler.Compile(gram.lexSpec, mlcompiler.CompressionLevel(mlcompiler.CompressionLevelMax))
	if err != nil {
		return nil, err
	}

	kind2Term := make([][]int, len(lexSpec.Modes))
	skip := make([][]int, len(lexSpec.Modes))
	for modeNum, spec := range lexSpec.Specs {
		if modeNum == 0 {
			kind2Term[0] = nil
			skip[0] = nil
			continue
		}

		k2tRec := make([]int, len(spec.Kinds))
		skipRec := make([]int, len(spec.Kinds))
		for n, k := range spec.Kinds {
			if n == 0 {
				k2tRec[0] = symbolNil.num().Int()
				continue
			}

			sym, ok := gram.symbolTable.toSymbol(k.String())
			if !ok {
				return nil, fmt.Errorf("terminal symbol '%v' (in '%v' mode) is not found in a symbol table", k, lexSpec.Modes[modeNum])
			}
			k2tRec[n] = sym.num().Int()

			for _, sk := range gram.skipLexKinds {
				if k != sk {
					continue
				}
				skipRec[n] = 1
			}
		}
		kind2Term[modeNum] = k2tRec
		skip[modeNum] = skipRec
	}

	terms, err := gram.symbolTable.getTerminalTexts()
	if err != nil {
		return nil, err
	}

	nonTerms, err := gram.symbolTable.getNonTerminalTexts()
	if err != nil {
		return nil, err
	}

	firstSet, err := genFirstSet(gram.productionSet)
	if err != nil {
		return nil, err
	}

	followSet, err := genFollowSet(gram.productionSet, firstSet)
	if err != nil {
		return nil, err
	}

	lr0, err := genLR0Automaton(gram.productionSet, gram.augmentedStartSymbol)
	if err != nil {
		return nil, err
	}

	tab, err := genSLRParsingTable(lr0, gram.productionSet, followSet, len(terms), len(nonTerms))
	if err != nil {
		return nil, err
	}

	action := make([]int, len(tab.actionTable))
	for i, e := range tab.actionTable {
		action[i] = int(e)
	}
	goTo := make([]int, len(tab.goToTable))
	for i, e := range tab.goToTable {
		goTo[i] = int(e)
	}

	lhsSyms := make([]int, len(gram.productionSet.getAllProductions())+1)
	altSymCounts := make([]int, len(gram.productionSet.getAllProductions())+1)
	astActEnties := make([][]int, len(gram.productionSet.getAllProductions())+1)
	for _, p := range gram.productionSet.getAllProductions() {
		lhsSyms[p.num] = p.lhs.num().Int()
		altSymCounts[p.num] = p.rhsLen

		astAct, ok := gram.astActions[p.id]
		if !ok {
			continue
		}
		astActEntry := make([]int, len(astAct))
		for i, e := range astAct {
			if e.expansion {
				astActEntry[i] = e.position * -1
			} else {
				astActEntry[i] = e.position
			}
		}
		astActEnties[p.num] = astActEntry
	}

	return &spec.CompiledGrammar{
		LexicalSpecification: &spec.LexicalSpecification{
			Lexer: "maleeni",
			Maleeni: &spec.Maleeni{
				Spec:           lexSpec,
				KindToTerminal: kind2Term,
				Skip:           skip,
			},
		},
		ParsingTable: &spec.ParsingTable{
			Action:                  action,
			GoTo:                    goTo,
			StateCount:              tab.stateCount,
			InitialState:            tab.InitialState.Int(),
			StartProduction:         productionNumStart.Int(),
			LHSSymbols:              lhsSyms,
			AlternativeSymbolCounts: altSymCounts,
			Terminals:               terms,
			TerminalCount:           tab.terminalCount,
			NonTerminals:            nonTerms,
			NonTerminalCount:        tab.nonTerminalCount,
			EOFSymbol:               symbolEOF.num().Int(),
		},
		ASTAction: &spec.ASTAction{
			Entries: astActEnties,
		},
	}, nil
}
