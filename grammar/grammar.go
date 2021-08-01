package grammar

import (
	"fmt"
	"os"
	"strings"

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
	skipLexKinds         []mlspec.LexKindName
	sym2AnonPat          map[symbol]string
	productionSet        *productionSet
	augmentedStartSymbol symbol
	symbolTable          *symbolTable
	astActions           map[productionID][]*astActionEntry
}

type GrammarBuilder struct {
	AST *spec.RootNode

	errs verr.SpecErrors
}

func (b *GrammarBuilder) Build() (*Grammar, error) {
	symTabAndLexSpec, err := b.genSymbolTableAndLexSpec(b.AST)
	if err != nil {
		return nil, err
	}

	prodsAndActs, err := b.genProductionsAndActions(b.AST, symTabAndLexSpec)
	if err != nil {
		return nil, err
	}

	syms, err := findUsedAndUnusedSymbols(b.AST)
	if err != nil {
		return nil, err
	}

	// When a terminal symbol that cannot be reached from the start symbol has the skip directive,
	// the compiler treats its terminal as a used symbol, not unused.
	for _, sym := range symTabAndLexSpec.skipSyms {
		if _, ok := syms.unusedTerminals[sym]; !ok {
			prod := syms.usedTerminals[sym]

			b.errs = append(b.errs, &verr.SpecError{
				Cause:  semErrTermCannotBeSkipped,
				Detail: sym,
				Row:    prod.Pos.Row,
			})
			continue
		}

		delete(syms.unusedTerminals, sym)
	}

	for sym, prod := range syms.unusedProductions {
		b.errs = append(b.errs, &verr.SpecError{
			Cause:  semErrUnusedProduction,
			Detail: sym,
			Row:    prod.Pos.Row,
		})
	}

	for sym, prod := range syms.unusedTerminals {
		b.errs = append(b.errs, &verr.SpecError{
			Cause:  semErrUnusedTerminal,
			Detail: sym,
			Row:    prod.Pos.Row,
		})
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

type usedAndUnusedSymbols struct {
	unusedProductions map[string]*spec.ProductionNode
	unusedTerminals   map[string]*spec.ProductionNode
	usedTerminals     map[string]*spec.ProductionNode
}

func findUsedAndUnusedSymbols(root *spec.RootNode) (*usedAndUnusedSymbols, error) {
	prods := map[string]*spec.ProductionNode{}
	lexProds := map[string]*spec.ProductionNode{}
	mark := map[string]bool{}
	{
		for _, p := range root.Productions {
			prods[p.LHS] = p
			mark[p.LHS] = false
			for _, alt := range p.RHS {
				for _, e := range alt.Elements {
					if e.ID == "" {
						continue
					}
					mark[e.ID] = false
				}
			}
		}

		for _, p := range root.LexProductions {
			lexProds[p.LHS] = p
			mark[p.LHS] = false
		}

		start := root.Productions[0]
		mark[start.LHS] = true
		markUsedSymbols(mark, map[string]bool{}, prods, start)
	}

	// usedProds := make(map[string]*spec.ProductionNode, len(prods))
	usedTerms := make(map[string]*spec.ProductionNode, len(lexProds))
	unusedProds := map[string]*spec.ProductionNode{}
	unusedTerms := map[string]*spec.ProductionNode{}
	for sym, used := range mark {
		if p, ok := prods[sym]; ok {
			if used {
				continue
			}
			unusedProds[sym] = p
			continue
		}
		if p, ok := lexProds[sym]; ok {
			if used {
				usedTerms[sym] = p
			} else {
				unusedTerms[sym] = p
			}
			continue
		}
		return nil, fmt.Errorf("a definition of unused production was not found: %v", sym)
	}

	return &usedAndUnusedSymbols{
		usedTerminals:     usedTerms,
		unusedProductions: unusedProds,
		unusedTerminals:   unusedTerms,
	}, nil
}

func markUsedSymbols(mark map[string]bool, marked map[string]bool, prods map[string]*spec.ProductionNode, prod *spec.ProductionNode) {
	if marked[prod.LHS] {
		return
	}

	for _, alt := range prod.RHS {
		for _, e := range alt.Elements {
			if e.ID == "" {
				continue
			}

			mark[e.ID] = true

			p, ok := prods[e.ID]
			if !ok {
				continue
			}

			// Remove a production to avoid inifinite recursion.
			marked[prod.LHS] = true

			markUsedSymbols(mark, marked, prods, p)
		}
	}
}

type symbolTableAndLexSpec struct {
	symTab      *symbolTable
	anonPat2Sym map[string]symbol
	sym2AnonPat map[symbol]string
	lexSpec     *mlspec.LexSpec
	skip        []mlspec.LexKindName
	skipSyms    []string
}

func (b *GrammarBuilder) genSymbolTableAndLexSpec(root *spec.RootNode) (*symbolTableAndLexSpec, error) {
	// Anonymous patterns take precedence over explicitly defined lexical specifications (named patterns).
	// Thus anonymous patterns must be registered to `symTab` and `entries` before named patterns.
	symTab := newSymbolTable()
	entries := []*mlspec.LexEntry{}

	anonPat2Sym := map[string]symbol{}
	sym2AnonPat := map[symbol]string{}
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

			entries = append(entries, &mlspec.LexEntry{
				Kind:    mlspec.LexKindName(kind),
				Pattern: mlspec.LexPattern(p),
			})
		}
	}

	skipKinds := []mlspec.LexKindName{}
	skipSyms := []string{}
	for _, prod := range root.LexProductions {
		if _, exist := symTab.toSymbol(prod.LHS); exist {
			b.errs = append(b.errs, &verr.SpecError{
				Cause:  semErrDuplicateTerminal,
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
			skipKinds = append(skipKinds, mlspec.LexKindName(prod.LHS))
			skipSyms = append(skipSyms, prod.LHS)
		}
		entries = append(entries, entry)
	}

	checkedFragments := map[string]struct{}{}
	for _, fragment := range root.Fragments {
		if _, exist := checkedFragments[fragment.LHS]; exist {
			b.errs = append(b.errs, &verr.SpecError{
				Cause:  semErrDuplicateTerminal,
				Detail: fragment.LHS,
				Row:    fragment.Pos.Row,
			})
			continue
		}
		checkedFragments[fragment.LHS] = struct{}{}

		entries = append(entries, &mlspec.LexEntry{
			Fragment: true,
			Kind:     mlspec.LexKindName(fragment.LHS),
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
		skip:     skipKinds,
		skipSyms: skipSyms,
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
		Kind:    mlspec.LexKindName(prod.LHS),
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
		sym, err := symTab.registerNonTerminalSymbol(prod.LHS)
		if err != nil {
			return nil, err
		}
		if sym.isTerminal() {
			b.errs = append(b.errs, &verr.SpecError{
				Cause:  semErrDuplicateName,
				Detail: prod.LHS,
				Row:    prod.Pos.Row,
			})
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
			if _, exist := prods.findByID(p.id); exist {
				// Report the line number of a duplicate alternative.
				// When the alternative is empty, we report the position of its LHS.
				var row int
				if len(alt.Elements) > 0 {
					row = alt.Elements[0].Pos.Row
				} else {
					row = prod.Pos.Row
				}

				var detail string
				{
					var b strings.Builder
					fmt.Fprintf(&b, "%v →", prod.LHS)
					for _, elem := range alt.Elements {
						switch {
						case elem.ID != "":
							fmt.Fprintf(&b, " %v", elem.ID)
						case elem.Pattern != "":
							fmt.Fprintf(&b, ` "%v"`, elem.Pattern)
						}
					}
					if len(alt.Elements) == 0 {
						fmt.Fprintf(&b, " ε")
					}

					detail = b.String()
				}

				b.errs = append(b.errs, &verr.SpecError{
					Cause:  semErrDuplicateProduction,
					Detail: detail,
					Row:    row,
				})
				continue LOOP_RHS
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

type compileConfig struct {
	descriptionFileName string
}

type compileOption func(config *compileConfig)

func EnableDescription(fileName string) compileOption {
	return func(config *compileConfig) {
		config.descriptionFileName = fileName
	}
}

func Compile(gram *Grammar, opts ...compileOption) (*spec.CompiledGrammar, error) {
	config := &compileConfig{}
	for _, opt := range opts {
		opt(config)
	}

	lexSpec, err := mlcompiler.Compile(gram.lexSpec, mlcompiler.CompressionLevel(mlcompiler.CompressionLevelMax))
	if err != nil {
		return nil, err
	}

	kind2Term := make([]int, len(lexSpec.KindNames))
	term2Kind := make([]int, gram.symbolTable.termNum.Int())
	skip := make([]int, len(lexSpec.KindNames))
	for i, k := range lexSpec.KindNames {
		if k == mlspec.LexKindNameNil {
			kind2Term[mlspec.LexKindIDNil] = symbolNil.num().Int()
			term2Kind[symbolNil.num()] = mlspec.LexKindIDNil.Int()
			continue
		}

		sym, ok := gram.symbolTable.toSymbol(k.String())
		if !ok {
			return nil, fmt.Errorf("terminal symbol '%v' was not found in a symbol table", k)
		}
		kind2Term[i] = sym.num().Int()
		term2Kind[sym.num()] = i

		for _, sk := range gram.skipLexKinds {
			if k != sk {
				continue
			}
			skip[i] = 1
			break
		}
	}

	terms, err := gram.symbolTable.terminalTexts()
	if err != nil {
		return nil, err
	}

	nonTerms, err := gram.symbolTable.nonTerminalTexts()
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
	if config.descriptionFileName != "" {
		f, err := os.OpenFile(config.descriptionFileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		slr.writeDescription(f)
	}
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
				TerminalToKind: term2Kind,
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
			ExpectedTerminals:       tab.expectedTerminals,
		},
		ASTAction: &spec.ASTAction{
			Entries: astActEnties,
		},
	}, nil
}
