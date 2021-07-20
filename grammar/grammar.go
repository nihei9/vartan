package grammar

import (
	"fmt"

	mlcompiler "github.com/nihei9/maleeni/compiler"
	mlspec "github.com/nihei9/maleeni/spec"
	verr "github.com/nihei9/vartan/error"
	"github.com/nihei9/vartan/spec"
)

type astActionEntry struct {
	position  int
	expansion bool
}

type Grammar struct {
	lexSpec              *mlspec.LexSpec
	skipLexKinds         []mlspec.LexKind
	sym2AnonPat          map[symbol]string
	productionSet        *productionSet
	augmentedStartSymbol symbol
	symbolTable          *symbolTable
	astActions           map[productionID][]*astActionEntry
}

type GrammarBuilder struct {
	errs verr.SpecErrors
}

func (b *GrammarBuilder) Build(root *spec.RootNode) (*Grammar, error) {
	b.errs = nil

	symTabAndLexSpec, err := b.genSymbolTableAndLexSpec(root)
	if err != nil {
		return nil, err
	}

	prodsAndActs, err := b.genProductionsAndActions(root, symTabAndLexSpec)
	if err != nil {
		return nil, err
	}

	if len(b.errs) > 0 {
		return nil, b.errs
	}

	return &Grammar{
		lexSpec:              symTabAndLexSpec.lexSpec,
		skipLexKinds:         symTabAndLexSpec.skip,
		sym2AnonPat:          symTabAndLexSpec.sym2AnonPat,
		productionSet:        prodsAndActs.prods,
		augmentedStartSymbol: prodsAndActs.augStartSym,
		symbolTable:          symTabAndLexSpec.symTab,
		astActions:           prodsAndActs.astActs,
	}, nil
}

type symbolTableAndLexSpec struct {
	symTab      *symbolTable
	anonPat2Sym map[string]symbol
	sym2AnonPat map[symbol]string
	lexSpec     *mlspec.LexSpec
	skip        []mlspec.LexKind
}

func (b *GrammarBuilder) genSymbolTableAndLexSpec(root *spec.RootNode) (*symbolTableAndLexSpec, error) {
	symTab := newSymbolTable()
	skipKinds := []mlspec.LexKind{}
	entries := []*mlspec.LexEntry{}
	for _, prod := range root.LexProductions {
		if _, exist := symTab.toSymbol(prod.LHS); exist {
			b.errs = append(b.errs, &verr.SpecError{
				Cause:  semErrDuplicateSym,
				Detail: prod.LHS,
				Row:    prod.Pos.Row,
			})
			continue
		}

		_, err := symTab.registerTerminalSymbol(prod.LHS)
		if err != nil {
			return nil, err
		}

		entry, skip, specErr, err := genLexEntry(prod)
		if err != nil {
			return nil, err
		}
		if specErr != nil {
			b.errs = append(b.errs, specErr)
			continue
		}
		if skip {
			skipKinds = append(skipKinds, mlspec.LexKind(prod.LHS))
		}
		entries = append(entries, entry)
	}

	anonPat2Sym := map[string]symbol{}
	sym2AnonPat := map[symbol]string{}
	var anonEntries []*mlspec.LexEntry
	{
		anonPats := []string{}
		for _, prod := range root.Productions {
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
			sym2AnonPat[sym] = p

			anonEntries = append(anonEntries, &mlspec.LexEntry{
				Kind:    mlspec.LexKind(kind),
				Pattern: mlspec.LexPattern(p),
			})
		}
	}

	// Anonymous patterns take precedence over explicitly defined lexical specifications.
	entries = append(anonEntries, entries...)

	checkedFragments := map[string]struct{}{}
	for _, fragment := range root.Fragments {
		if _, exist := checkedFragments[fragment.LHS]; exist {
			b.errs = append(b.errs, &verr.SpecError{
				Cause:  semErrDuplicateSym,
				Detail: fragment.LHS,
				Row:    fragment.Pos.Row,
			})
			continue
		}
		checkedFragments[fragment.LHS] = struct{}{}

		entries = append(entries, &mlspec.LexEntry{
			Fragment: true,
			Kind:     mlspec.LexKind(fragment.LHS),
			Pattern:  mlspec.LexPattern(fragment.RHS),
		})
	}

	return &symbolTableAndLexSpec{
		symTab:      symTab,
		anonPat2Sym: anonPat2Sym,
		sym2AnonPat: sym2AnonPat,
		lexSpec: &mlspec.LexSpec{
			Entries: entries,
		},
		skip: skipKinds,
	}, nil
}

func genLexEntry(prod *spec.ProductionNode) (*mlspec.LexEntry, bool, *verr.SpecError, error) {
	var modes []mlspec.LexModeName
	if prod.Directive != nil {
		dir := prod.Directive
		switch dir.Name {
		case "mode":
			if len(dir.Parameters) == 0 {
				return nil, false, &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: fmt.Sprintf("'mode' directive needs an ID parameter"),
					Row:    dir.Pos.Row,
				}, nil
			}
			for _, param := range dir.Parameters {
				if param.ID == "" {
					return nil, false, &verr.SpecError{
						Cause:  semErrDirInvalidParam,
						Detail: fmt.Sprintf("'mode' directive needs an ID parameter"),
						Row:    param.Pos.Row,
					}, nil
				}
				modes = append(modes, mlspec.LexModeName(param.ID))
			}
		default:
			return nil, false, &verr.SpecError{
				Cause:  semErrDirInvalidName,
				Detail: dir.Name,
				Row:    dir.Pos.Row,
			}, nil
		}
	}

	alt := prod.RHS[0]
	var skip bool
	var push mlspec.LexModeName
	var pop bool
	if alt.Directive != nil {
		dir := alt.Directive
		switch dir.Name {
		case "skip":
			if len(dir.Parameters) > 0 {
				return nil, false, &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: fmt.Sprintf("'skip' directive needs no parameter"),
					Row:    dir.Pos.Row,
				}, nil
			}
			skip = true
		case "push":
			if len(dir.Parameters) != 1 || dir.Parameters[0].ID == "" {
				return nil, false, &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: fmt.Sprintf("'push' directive needs an ID parameter"),
					Row:    dir.Pos.Row,
				}, nil
			}
			push = mlspec.LexModeName(dir.Parameters[0].ID)
		case "pop":
			if len(dir.Parameters) > 0 {
				return nil, false, &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: fmt.Sprintf("'pop' directive needs no parameter"),
					Row:    dir.Pos.Row,
				}, nil
			}
			pop = true
		default:
			return nil, false, &verr.SpecError{
				Cause:  semErrDirInvalidName,
				Detail: dir.Name,
				Row:    dir.Pos.Row,
			}, nil
		}
	}

	return &mlspec.LexEntry{
		Modes:   modes,
		Kind:    mlspec.LexKind(prod.LHS),
		Pattern: mlspec.LexPattern(alt.Elements[0].Pattern),
		Push:    push,
		Pop:     pop,
	}, skip, nil, nil
}

type productionsAndActions struct {
	prods       *productionSet
	augStartSym symbol
	astActs     map[productionID][]*astActionEntry
}

func (b *GrammarBuilder) genProductionsAndActions(root *spec.RootNode, symTabAndLexSpec *symbolTableAndLexSpec) (*productionsAndActions, error) {
	symTab := symTabAndLexSpec.symTab
	anonPat2Sym := symTabAndLexSpec.anonPat2Sym

	if len(root.Productions) == 0 {
		b.errs = append(b.errs, &verr.SpecError{
			Cause: semErrNoProduction,
		})
		return nil, nil
	}

	prods := newProductionSet()
	var augStartSym symbol
	astActs := map[productionID][]*astActionEntry{}

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
		_, err := symTab.registerNonTerminalSymbol(prod.LHS)
		if err != nil {
			return nil, err
		}
	}

	for _, prod := range root.Productions {
		lhsSym, ok := symTab.toSymbol(prod.LHS)
		if !ok {
			// All symbols are assumed to be pre-detected, so it's a bug if we cannot find them here.
			return nil, fmt.Errorf("symbol '%v' is undefined", prod.LHS)
		}

	LOOP_RHS:
		for _, alt := range prod.RHS {
			altSyms := make([]symbol, len(alt.Elements))
			for i, elem := range alt.Elements {
				var sym symbol
				if elem.Pattern != "" {
					var ok bool
					sym, ok = anonPat2Sym[elem.Pattern]
					if !ok {
						// All patterns are assumed to be pre-detected, so it's a bug if we cannot find them here.
						return nil, fmt.Errorf("pattern '%v' is undefined", elem.Pattern)
					}
				} else {
					var ok bool
					sym, ok = symTab.toSymbol(elem.ID)
					if !ok {
						b.errs = append(b.errs, &verr.SpecError{
							Cause:  semErrUndefinedSym,
							Detail: elem.ID,
							Row:    elem.Pos.Row,
						})
						continue LOOP_RHS
					}
				}
				altSyms[i] = sym
			}

			p, err := newProduction(lhsSym, altSyms)
			if err != nil {
				return nil, err
			}
			prods.append(p)

			if alt.Directive != nil {
				dir := alt.Directive
				switch dir.Name {
				case "ast":
					if len(dir.Parameters) != 1 || dir.Parameters[0].Tree == nil {
						b.errs = append(b.errs, &verr.SpecError{
							Cause:  semErrDirInvalidParam,
							Detail: fmt.Sprintf("'ast' directive needs a tree parameter"),
							Row:    dir.Pos.Row,
						})
						continue LOOP_RHS
					}
					param := dir.Parameters[0]
					lhsText, ok := symTab.toText(p.lhs)
					if !ok || param.Tree.Name != lhsText {
						b.errs = append(b.errs, &verr.SpecError{
							Cause:  semErrDirInvalidParam,
							Detail: fmt.Sprintf("a name of a tree structure must be the same ID as an LHS of a production; LHS: %v", lhsText),
							Row:    param.Pos.Row,
						})
						continue LOOP_RHS
					}
					astAct := make([]*astActionEntry, len(param.Tree.Children))
					for i, c := range param.Tree.Children {
						if c.Position > len(alt.Elements) {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrDirInvalidParam,
								Detail: fmt.Sprintf("a position must be less than or equal to the length of an alternativ (%v)", len(alt.Elements)),
								Row:    c.Pos.Row,
							})
							continue LOOP_RHS
						}

						if c.Expansion {
							offset := c.Position - 1
							elem := alt.Elements[offset]
							if elem.Pattern != "" {
								b.errs = append(b.errs, &verr.SpecError{
									Cause:  semErrDirInvalidParam,
									Detail: fmt.Sprintf("the expansion symbol cannot be applied to a pattern ($%v: %v)", c.Position, elem.Pattern),
									Row:    c.Pos.Row,
								})
								continue LOOP_RHS
							}
							elemSym, ok := symTab.toSymbol(elem.ID)
							if !ok {
								// If the symbol was not found, it's a bug.
								return nil, fmt.Errorf("a symbol corresponding to a position ($%v: %v) was not found", c.Position, elem.ID)
							}
							if elemSym.isTerminal() {
								b.errs = append(b.errs, &verr.SpecError{
									Cause:  semErrDirInvalidParam,
									Detail: fmt.Sprintf("the expansion symbol cannot be applied to a terminal symbol ($%v: %v)", c.Position, elem.ID),
									Row:    c.Pos.Row,
								})
								continue LOOP_RHS
							}
						}

						astAct[i] = &astActionEntry{
							position:  c.Position,
							expansion: c.Expansion,
						}
					}
					astActs[p.id] = astAct
				default:
					b.errs = append(b.errs, &verr.SpecError{
						Cause:  semErrDirInvalidName,
						Detail: fmt.Sprintf("invalid directive name '%v'", dir.Name),
						Row:    dir.Pos.Row,
					})
					continue LOOP_RHS
				}
			}
		}
	}

	return &productionsAndActions{
		prods:       prods,
		augStartSym: augStartSym,
		astActs:     astActs,
	}, nil
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

	slr := &slrTableBuilder{
		automaton:    lr0,
		prods:        gram.productionSet,
		follow:       followSet,
		termCount:    len(terms),
		nonTermCount: len(nonTerms),
		symTab:       gram.symbolTable,
		sym2AnonPat:  gram.sym2AnonPat,
	}
	tab, err := slr.build()
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
