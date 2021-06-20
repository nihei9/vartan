package grammar

import (
	"fmt"

	mlcompiler "github.com/nihei9/maleeni/compiler"
	mlspec "github.com/nihei9/maleeni/spec"
	"github.com/nihei9/vartan/spec"
)

type Grammar struct {
	lexSpec              *mlspec.LexSpec
	productionSet        *productionSet
	augmentedStartSymbol symbol
	symbolTable          *symbolTable
}

func NewGrammar(root *spec.RootNode) (*Grammar, error) {
	symTab := newSymbolTable()
	anonPat2Sym := map[string]symbol{}
	var lexSpec *mlspec.LexSpec
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
					case "push":
						if act.Parameter == "" {
							return nil, fmt.Errorf("action 'push' needs a parameter")
						}
						push = mlspec.LexModeName(act.Parameter)
					case "pop":
						if act.Parameter != "" {
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
		for i, p := range anonPats {
			kind := fmt.Sprintf("__%v__", i+1)

			sym, err := symTab.registerTerminalSymbol(kind)
			if err != nil {
				return nil, err
			}
			anonPat2Sym[p] = sym

			entries = append(entries, &mlspec.LexEntry{
				Kind:    mlspec.LexKind(kind),
				Pattern: mlspec.LexPattern(p),
			})
		}

		lexSpec = &mlspec.LexSpec{
			Entries: entries,
		}
	}

	prods := newProductionSet()
	var augStartSym symbol
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
			}
		}
	}

	return &Grammar{
		lexSpec:              lexSpec,
		productionSet:        prods,
		augmentedStartSymbol: augStartSym,
		symbolTable:          symTab,
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
	for modeNum, spec := range lexSpec.Specs {
		if modeNum == 0 {
			kind2Term[0] = nil
			continue
		}
		rec := make([]int, len(spec.Kinds))
		for n, k := range spec.Kinds {
			if n == 0 {
				rec[0] = symbolNil.num().Int()
				continue
			}
			sym, ok := gram.symbolTable.toSymbol(k.String())
			if !ok {
				return nil, fmt.Errorf("terminal symbol '%v' (in '%v' mode) is not found in a symbol table", k, lexSpec.Modes[modeNum])
			}
			rec[n] = sym.num().Int()
		}
		kind2Term[modeNum] = rec
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
	for _, p := range gram.productionSet.getAllProductions() {
		lhsSyms[p.num] = p.lhs.num().Int()
		altSymCounts[p.num] = p.rhsLen
	}

	return &spec.CompiledGrammar{
		LexicalSpecification: &spec.LexicalSpecification{
			Lexer: "maleeni",
			Maleeni: &spec.Maleeni{
				Spec:           lexSpec,
				KindToTerminal: kind2Term,
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
	}, nil
}
