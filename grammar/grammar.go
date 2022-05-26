package grammar

import (
	"fmt"
	"io"
	"strings"

	mlcompiler "github.com/nihei9/maleeni/compiler"
	mlspec "github.com/nihei9/maleeni/spec"
	verr "github.com/nihei9/vartan/error"
	spec "github.com/nihei9/vartan/spec/grammar"
)

type astActionEntry struct {
	position  int
	expansion bool
}

type assocType string

const (
	assocTypeNil   = assocType("")
	assocTypeLeft  = assocType("left")
	assocTypeRight = assocType("right")
)

const (
	precNil = 0
	precMin = 1
)

// precAndAssoc represents precedence and associativities of terminal symbols and productions.
// We use the priority of the production to resolve shift/reduce conflicts.
type precAndAssoc struct {
	// termPrec and termAssoc represent the precedence of the terminal symbols.
	termPrec  map[symbolNum]int
	termAssoc map[symbolNum]assocType

	// prodPrec and prodAssoc represent the precedence and the associativities of the production.
	// These values are inherited from the right-most terminal symbols in the RHS of the productions.
	prodPrec  map[productionNum]int
	prodAssoc map[productionNum]assocType
}

func (pa *precAndAssoc) terminalPrecedence(sym symbolNum) int {
	prec, ok := pa.termPrec[sym]
	if !ok {
		return precNil
	}

	return prec
}

func (pa *precAndAssoc) terminalAssociativity(sym symbolNum) assocType {
	assoc, ok := pa.termAssoc[sym]
	if !ok {
		return assocTypeNil
	}

	return assoc
}

func (pa *precAndAssoc) productionPredence(prod productionNum) int {
	prec, ok := pa.prodPrec[prod]
	if !ok {
		return precNil
	}

	return prec
}

func (pa *precAndAssoc) productionAssociativity(prod productionNum) assocType {
	assoc, ok := pa.prodAssoc[prod]
	if !ok {
		return assocTypeNil
	}

	return assoc
}

const reservedSymbolNameError = "error"

type Grammar struct {
	name                 string
	lexSpec              *mlspec.LexSpec
	skipLexKinds         []mlspec.LexKindName
	kindAliases          map[symbol]string
	sym2AnonPat          map[symbol]string
	productionSet        *productionSet
	augmentedStartSymbol symbol
	errorSymbol          symbol
	symbolTable          *symbolTable
	astActions           map[productionID][]*astActionEntry
	precAndAssoc         *precAndAssoc

	// recoverProductions is a set of productions having the recover directive.
	recoverProductions map[productionID]struct{}
}

type GrammarBuilder struct {
	AST *spec.RootNode

	errs verr.SpecErrors
}

func (b *GrammarBuilder) Build() (*Grammar, error) {
	var specName string
	{
		errOccurred := false
		for _, dir := range b.AST.Directives {
			if dir.Name != "name" {
				continue
			}

			if len(dir.Parameters) != 1 || dir.Parameters[0].ID == "" {
				b.errs = append(b.errs, &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: "'name' takes just one ID parameter",
					Row:    dir.Pos.Row,
					Col:    dir.Pos.Col,
				})

				errOccurred = true
				break
			}

			specName = dir.Parameters[0].ID
			break
		}

		if specName == "" && !errOccurred {
			b.errs = append(b.errs, &verr.SpecError{
				Cause: semErrNoGrammarName,
			})
		}
	}

	b.checkSpellingInconsistenciesOfUserDefinedIDs(b.AST)
	if len(b.errs) > 0 {
		return nil, b.errs
	}

	symTabAndLexSpec, err := b.genSymbolTableAndLexSpec(b.AST)
	if err != nil {
		return nil, err
	}

	prodsAndActs, err := b.genProductionsAndActions(b.AST, symTabAndLexSpec)
	if err != nil {
		return nil, err
	}
	if prodsAndActs == nil && len(b.errs) > 0 {
		return nil, b.errs
	}

	pa, err := b.genPrecAndAssoc(symTabAndLexSpec.symTab, symTabAndLexSpec.errSym, prodsAndActs)
	if err != nil {
		return nil, err
	}
	if pa == nil && len(b.errs) > 0 {
		return nil, b.errs
	}

	syms := findUsedAndUnusedSymbols(b.AST)
	if syms == nil && len(b.errs) > 0 {
		return nil, b.errs
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
				Col:    prod.Pos.Col,
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
			Col:    prod.Pos.Col,
		})
	}

	for sym, prod := range syms.unusedTerminals {
		b.errs = append(b.errs, &verr.SpecError{
			Cause:  semErrUnusedTerminal,
			Detail: sym,
			Row:    prod.Pos.Row,
			Col:    prod.Pos.Col,
		})
	}

	if len(b.errs) > 0 {
		return nil, b.errs
	}

	symTabAndLexSpec.lexSpec.Name = specName

	return &Grammar{
		name:                 specName,
		lexSpec:              symTabAndLexSpec.lexSpec,
		skipLexKinds:         symTabAndLexSpec.skip,
		kindAliases:          symTabAndLexSpec.aliases,
		sym2AnonPat:          symTabAndLexSpec.sym2AnonPat,
		productionSet:        prodsAndActs.prods,
		augmentedStartSymbol: prodsAndActs.augStartSym,
		errorSymbol:          symTabAndLexSpec.errSym,
		symbolTable:          symTabAndLexSpec.symTab,
		astActions:           prodsAndActs.astActs,
		recoverProductions:   prodsAndActs.recoverProds,
		precAndAssoc:         pa,
	}, nil
}

type usedAndUnusedSymbols struct {
	unusedProductions map[string]*spec.ProductionNode
	unusedTerminals   map[string]*spec.ProductionNode
	usedTerminals     map[string]*spec.ProductionNode
}

func findUsedAndUnusedSymbols(root *spec.RootNode) *usedAndUnusedSymbols {
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

		// We don't have to check the error symbol because the error symbol doesn't have a production.
		delete(mark, reservedSymbolNameError)
	}

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

		// May be reached here when a fragment name appears on the right-hand side of a production rule. However, an error
		// to the effect that a production rule cannot contain a fragment will be detected in a subsequent process. So we can
		// ignore it here.
	}

	return &usedAndUnusedSymbols{
		usedTerminals:     usedTerms,
		unusedProductions: unusedProds,
		unusedTerminals:   unusedTerms,
	}
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

func (b *GrammarBuilder) checkSpellingInconsistenciesOfUserDefinedIDs(root *spec.RootNode) {
	var ids []string
	{
		for _, prod := range root.Productions {
			ids = append(ids, prod.LHS)
			for _, alt := range prod.RHS {
				for _, elem := range alt.Elements {
					if elem.Label != nil {
						ids = append(ids, elem.Label.Name)
					}
				}
			}
		}
		for _, prod := range root.LexProductions {
			ids = append(ids, prod.LHS)
		}
		for _, dir := range root.Directives {
			dirIDs := collectUserDefinedIDsFromDirective(dir)
			if len(dirIDs) > 0 {
				ids = append(ids, dirIDs...)
			}
		}
	}

	duplicated := mlspec.FindSpellingInconsistencies(ids)
	if len(duplicated) == 0 {
		return
	}

	for _, dup := range duplicated {
		var s string
		{
			var b strings.Builder
			fmt.Fprintf(&b, "%+v", dup[0])
			for _, id := range dup[1:] {
				fmt.Fprintf(&b, ", %+v", id)
			}
			s = b.String()
		}

		b.errs = append(b.errs, &verr.SpecError{
			Cause:  semErrSpellingInconsistency,
			Detail: s,
		})
	}
}

func collectUserDefinedIDsFromDirective(dir *spec.DirectiveNode) []string {
	var ids []string
	for _, param := range dir.Parameters {
		if param.Group != nil {
			for _, d := range param.Group {
				dIDs := collectUserDefinedIDsFromDirective(d)
				if len(dIDs) > 0 {
					ids = append(ids, dIDs...)
				}
			}
		}
		if param.OrderedSymbol != "" {
			ids = append(ids, param.OrderedSymbol)
		}
	}
	return ids
}

type symbolTableAndLexSpec struct {
	symTab      *symbolTable
	anonPat2Sym map[string]symbol
	sym2AnonPat map[symbol]string
	lexSpec     *mlspec.LexSpec
	errSym      symbol
	skip        []mlspec.LexKindName
	skipSyms    []string
	aliases     map[symbol]string
}

func (b *GrammarBuilder) genSymbolTableAndLexSpec(root *spec.RootNode) (*symbolTableAndLexSpec, error) {
	// Anonymous patterns take precedence over explicitly defined lexical specifications (named patterns).
	// Thus anonymous patterns must be registered to `symTab` and `entries` before named patterns.
	symTab := newSymbolTable()
	entries := []*mlspec.LexEntry{}

	// We need to register the reserved symbol before registering others.
	var errSym symbol
	{
		sym, err := symTab.registerTerminalSymbol(reservedSymbolNameError)
		if err != nil {
			return nil, err
		}
		errSym = sym
	}

	anonPat2Sym := map[string]symbol{}
	sym2AnonPat := map[symbol]string{}
	aliases := map[symbol]string{}
	{
		knownPats := map[string]struct{}{}
		anonPats := []string{}
		literalPats := map[string]struct{}{}
		for _, prod := range root.Productions {
			for _, alt := range prod.RHS {
				for _, elem := range alt.Elements {
					if elem.Pattern == "" {
						continue
					}

					var pattern string
					if elem.Literally {
						pattern = mlspec.EscapePattern(elem.Pattern)
					} else {
						pattern = elem.Pattern
					}

					if _, ok := knownPats[pattern]; ok {
						continue
					}

					knownPats[pattern] = struct{}{}
					anonPats = append(anonPats, pattern)
					if elem.Literally {
						literalPats[pattern] = struct{}{}
					}
				}
			}
		}

		for i, p := range anonPats {
			kind := fmt.Sprintf("x_%v", i+1)

			sym, err := symTab.registerTerminalSymbol(kind)
			if err != nil {
				return nil, err
			}

			anonPat2Sym[p] = sym
			sym2AnonPat[sym] = p

			if _, ok := literalPats[p]; ok {
				aliases[sym] = p
			}

			entries = append(entries, &mlspec.LexEntry{
				Kind:    mlspec.LexKindName(kind),
				Pattern: mlspec.LexPattern(p),
			})
		}
	}

	skipKinds := []mlspec.LexKindName{}
	skipSyms := []string{}
	for _, prod := range root.LexProductions {
		if sym, exist := symTab.toSymbol(prod.LHS); exist {
			if sym == errSym {
				b.errs = append(b.errs, &verr.SpecError{
					Cause: semErrErrSymIsReserved,
					Row:   prod.Pos.Row,
					Col:   prod.Pos.Col,
				})
			} else {
				b.errs = append(b.errs, &verr.SpecError{
					Cause:  semErrDuplicateTerminal,
					Detail: prod.LHS,
					Row:    prod.Pos.Row,
					Col:    prod.Pos.Col,
				})
			}

			continue
		}

		lhsSym, err := symTab.registerTerminalSymbol(prod.LHS)
		if err != nil {
			return nil, err
		}

		entry, skip, alias, specErr, err := genLexEntry(prod)
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
		if alias != "" {
			aliases[lhsSym] = alias
		}
		entries = append(entries, entry)
	}

	checkedFragments := map[string]struct{}{}
	for _, fragment := range root.Fragments {
		if _, exist := checkedFragments[fragment.LHS]; exist {
			b.errs = append(b.errs, &verr.SpecError{
				Cause:  semErrDuplicateFragment,
				Detail: fragment.LHS,
				Row:    fragment.Pos.Row,
				Col:    fragment.Pos.Col,
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
		errSym:   errSym,
		skip:     skipKinds,
		skipSyms: skipSyms,
		aliases:  aliases,
	}, nil
}

func genLexEntry(prod *spec.ProductionNode) (*mlspec.LexEntry, bool, string, *verr.SpecError, error) {
	alt := prod.RHS[0]
	elem := alt.Elements[0]

	var pattern string
	var alias string
	if elem.Literally {
		pattern = mlspec.EscapePattern(elem.Pattern)
		alias = elem.Pattern
	} else {
		pattern = elem.Pattern
	}

	var modes []mlspec.LexModeName
	var skip bool
	var push mlspec.LexModeName
	var pop bool
	dirConsumed := map[string]struct{}{}
	for _, dir := range prod.Directives {
		if _, consumed := dirConsumed[dir.Name]; consumed {
			return nil, false, "", &verr.SpecError{
				Cause:  semErrDuplicateDir,
				Detail: dir.Name,
				Row:    dir.Pos.Row,
				Col:    dir.Pos.Col,
			}, nil
		}
		dirConsumed[dir.Name] = struct{}{}

		switch dir.Name {
		case "mode":
			if len(dir.Parameters) == 0 {
				return nil, false, "", &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: "'mode' directive needs an ID parameter",
					Row:    dir.Pos.Row,
					Col:    dir.Pos.Col,
				}, nil
			}
			for _, param := range dir.Parameters {
				if param.ID == "" {
					return nil, false, "", &verr.SpecError{
						Cause:  semErrDirInvalidParam,
						Detail: "'mode' directive needs an ID parameter",
						Row:    param.Pos.Row,
						Col:    param.Pos.Col,
					}, nil
				}
				modes = append(modes, mlspec.LexModeName(param.ID))
			}
		case "skip":
			if len(dir.Parameters) > 0 {
				return nil, false, "", &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: "'skip' directive needs no parameter",
					Row:    dir.Pos.Row,
					Col:    dir.Pos.Col,
				}, nil
			}
			skip = true
		case "push":
			if len(dir.Parameters) != 1 || dir.Parameters[0].ID == "" {
				return nil, false, "", &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: "'push' directive needs an ID parameter",
					Row:    dir.Pos.Row,
					Col:    dir.Pos.Col,
				}, nil
			}
			push = mlspec.LexModeName(dir.Parameters[0].ID)
		case "pop":
			if len(dir.Parameters) > 0 {
				return nil, false, "", &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: "'pop' directive needs no parameter",
					Row:    dir.Pos.Row,
					Col:    dir.Pos.Col,
				}, nil
			}
			pop = true
		case "alias":
			if len(dir.Parameters) != 1 || dir.Parameters[0].String == "" {
				return nil, false, "", &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: "'alias' directive needs a string parameter",
					Row:    dir.Pos.Row,
					Col:    dir.Pos.Col,
				}, nil
			}
			alias = dir.Parameters[0].String
		default:
			return nil, false, "", &verr.SpecError{
				Cause:  semErrDirInvalidName,
				Detail: dir.Name,
				Row:    dir.Pos.Row,
				Col:    dir.Pos.Col,
			}, nil
		}
	}

	if len(alt.Directives) > 0 {
		return nil, false, "", &verr.SpecError{
			Cause:  semErrInvalidAltDir,
			Detail: "a lexical production cannot have alternative directives",
			Row:    alt.Directives[0].Pos.Row,
			Col:    alt.Directives[0].Pos.Col,
		}, nil
	}

	return &mlspec.LexEntry{
		Modes:   modes,
		Kind:    mlspec.LexKindName(prod.LHS),
		Pattern: mlspec.LexPattern(pattern),
		Push:    push,
		Pop:     pop,
	}, skip, alias, nil, nil
}

type productionsAndActions struct {
	prods           *productionSet
	augStartSym     symbol
	astActs         map[productionID][]*astActionEntry
	prodPrecsTerm   map[productionID]symbol
	prodPrecsOrdSym map[productionID]string
	prodPrecPoss    map[productionID]*spec.Position
	recoverProds    map[productionID]struct{}
}

func (b *GrammarBuilder) genProductionsAndActions(root *spec.RootNode, symTabAndLexSpec *symbolTableAndLexSpec) (*productionsAndActions, error) {
	symTab := symTabAndLexSpec.symTab
	anonPat2Sym := symTabAndLexSpec.anonPat2Sym
	errSym := symTabAndLexSpec.errSym

	if len(root.Productions) == 0 {
		b.errs = append(b.errs, &verr.SpecError{
			Cause: semErrNoProduction,
		})
		return nil, nil
	}

	prods := newProductionSet()
	var augStartSym symbol
	astActs := map[productionID][]*astActionEntry{}
	prodPrecsTerm := map[productionID]symbol{}
	prodPrecsOrdSym := map[productionID]string{}
	prodPrecPoss := map[productionID]*spec.Position{}
	recoverProds := map[productionID]struct{}{}

	startProd := root.Productions[0]
	augStartText := fmt.Sprintf("%s'", startProd.LHS)
	var err error
	augStartSym, err = symTab.registerStartSymbol(augStartText)
	if err != nil {
		return nil, err
	}
	if augStartSym == errSym {
		b.errs = append(b.errs, &verr.SpecError{
			Cause: semErrErrSymIsReserved,
			Row:   startProd.Pos.Row,
			Col:   startProd.Pos.Col,
		})
	}

	startSym, err := symTab.registerNonTerminalSymbol(startProd.LHS)
	if err != nil {
		return nil, err
	}
	if startSym == errSym {
		b.errs = append(b.errs, &verr.SpecError{
			Cause: semErrErrSymIsReserved,
			Row:   startProd.Pos.Row,
			Col:   startProd.Pos.Col,
		})
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
				Col:    prod.Pos.Col,
			})
		}
		if sym == errSym {
			b.errs = append(b.errs, &verr.SpecError{
				Cause: semErrErrSymIsReserved,
				Row:   prod.Pos.Row,
				Col:   prod.Pos.Col,
			})
		}
	}

	for _, prod := range root.Productions {
		lhsSym, ok := symTab.toSymbol(prod.LHS)
		if !ok {
			// All symbols are assumed to be pre-detected, so it's a bug if we cannot find them here.
			return nil, fmt.Errorf("symbol '%v' is undefined", prod.LHS)
		}

		if len(prod.Directives) > 0 {
			b.errs = append(b.errs, &verr.SpecError{
				Cause:  semErrInvalidProdDir,
				Detail: "a production cannot have production directives",
				Row:    prod.Directives[0].Pos.Row,
				Col:    prod.Directives[0].Pos.Col,
			})
			continue
		}

	LOOP_RHS:
		for _, alt := range prod.RHS {
			altSyms := make([]symbol, len(alt.Elements))
			offsets := map[string]int{}
			ambiguousIDOffsets := map[string]struct{}{}
			for i, elem := range alt.Elements {
				var sym symbol
				if elem.Pattern != "" {
					var pattern string
					if elem.Literally {
						pattern = mlspec.EscapePattern(elem.Pattern)
					} else {
						pattern = elem.Pattern
					}

					var ok bool
					sym, ok = anonPat2Sym[pattern]
					if !ok {
						// All patterns are assumed to be pre-detected, so it's a bug if we cannot find them here.
						return nil, fmt.Errorf("pattern '%v' is undefined", pattern)
					}
				} else {
					var ok bool
					sym, ok = symTab.toSymbol(elem.ID)
					if !ok {
						b.errs = append(b.errs, &verr.SpecError{
							Cause:  semErrUndefinedSym,
							Detail: elem.ID,
							Row:    elem.Pos.Row,
							Col:    elem.Pos.Col,
						})
						continue LOOP_RHS
					}
				}
				altSyms[i] = sym

				if elem.Label != nil {
					if _, added := offsets[elem.Label.Name]; added {
						b.errs = append(b.errs, &verr.SpecError{
							Cause:  semErrDuplicateLabel,
							Detail: elem.Label.Name,
							Row:    elem.Label.Pos.Row,
							Col:    elem.Label.Pos.Col,
						})
						continue LOOP_RHS
					}
					if _, found := symTab.toSymbol(elem.Label.Name); found {
						b.errs = append(b.errs, &verr.SpecError{
							Cause:  semErrInvalidLabel,
							Detail: elem.Label.Name,
							Row:    elem.Label.Pos.Row,
							Col:    elem.Label.Pos.Col,
						})
						continue LOOP_RHS
					}
					offsets[elem.Label.Name] = i
				}
				// A symbol having a label can be specified by both the label and the symbol name.
				// So record the symbol's position, whether or not it has a label.
				if elem.ID != "" {
					if _, exist := offsets[elem.ID]; exist {
						// When the same symbol appears multiple times in an alternative, the symbol is ambiguous. When we need
						// to specify the symbol in a directive, we cannot use the name of the ambiguous symbol. Instead, specify
						// a label to resolve the ambiguity.
						delete(offsets, elem.ID)
						ambiguousIDOffsets[elem.ID] = struct{}{}
					} else {
						offsets[elem.ID] = i
					}
				}
			}

			p, err := newProduction(lhsSym, altSyms)
			if err != nil {
				return nil, err
			}
			if _, exist := prods.findByID(p.id); exist {
				// Report the line number of a duplicate alternative.
				// When the alternative is empty, we report the position of its LHS.
				var row int
				var col int
				if len(alt.Elements) > 0 {
					row = alt.Elements[0].Pos.Row
					col = alt.Elements[0].Pos.Col
				} else {
					row = prod.Pos.Row
					col = prod.Pos.Col
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
					Col:    col,
				})
				continue LOOP_RHS
			}
			prods.append(p)

			dirConsumed := map[string]struct{}{}
			for _, dir := range alt.Directives {
				if _, consumed := dirConsumed[dir.Name]; consumed {
					b.errs = append(b.errs, &verr.SpecError{
						Cause:  semErrDuplicateDir,
						Detail: dir.Name,
						Row:    dir.Pos.Row,
						Col:    dir.Pos.Col,
					})
				}
				dirConsumed[dir.Name] = struct{}{}

				switch dir.Name {
				case "ast":
					if len(dir.Parameters) == 0 {
						b.errs = append(b.errs, &verr.SpecError{
							Cause:  semErrDirInvalidParam,
							Detail: "'ast' directive needs at least one parameter",
							Row:    dir.Pos.Row,
							Col:    dir.Pos.Col,
						})
						continue LOOP_RHS
					}
					astAct := make([]*astActionEntry, len(dir.Parameters))
					consumedOffsets := map[int]struct{}{}
					for i, param := range dir.Parameters {
						if param.ID == "" {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrDirInvalidParam,
								Detail: "'ast' directive can take only ID parameters",
								Row:    dir.Pos.Row,
								Col:    dir.Pos.Col,
							})
							continue LOOP_RHS
						}

						if _, ambiguous := ambiguousIDOffsets[param.ID]; ambiguous {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrAmbiguousElem,
								Detail: fmt.Sprintf("'%v' is ambiguous", param.ID),
								Row:    param.Pos.Row,
								Col:    param.Pos.Col,
							})
							continue LOOP_RHS
						}

						offset, ok := offsets[param.ID]
						if !ok {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrDirInvalidParam,
								Detail: fmt.Sprintf("a symbol was not found in an alternative: %v", param.ID),
								Row:    param.Pos.Row,
								Col:    param.Pos.Col,
							})
							continue LOOP_RHS
						}
						if _, consumed := consumedOffsets[offset]; consumed {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrDuplicateElem,
								Detail: param.ID,
								Row:    param.Pos.Row,
								Col:    param.Pos.Col,
							})
							continue LOOP_RHS
						}
						consumedOffsets[offset] = struct{}{}

						if param.Expansion {
							elem := alt.Elements[offset]
							if elem.Pattern != "" {
								b.errs = append(b.errs, &verr.SpecError{
									Cause:  semErrDirInvalidParam,
									Detail: fmt.Sprintf("the expansion symbol cannot be applied to a pattern (%v: \"%v\")", param.ID, elem.Pattern),
									Row:    param.Pos.Row,
									Col:    param.Pos.Col,
								})
								continue LOOP_RHS
							}
							elemSym, ok := symTab.toSymbol(elem.ID)
							if !ok {
								// If the symbol was not found, it's a bug.
								return nil, fmt.Errorf("a symbol corresponding to an ID (%v) was not found", elem.ID)
							}
							if elemSym.isTerminal() {
								b.errs = append(b.errs, &verr.SpecError{
									Cause:  semErrDirInvalidParam,
									Detail: fmt.Sprintf("the expansion symbol cannot be applied to a terminal symbol (%v: %v)", param.ID, elem.ID),
									Row:    param.Pos.Row,
									Col:    param.Pos.Col,
								})
								continue LOOP_RHS
							}
						}

						astAct[i] = &astActionEntry{
							position:  offset + 1,
							expansion: param.Expansion,
						}
					}
					astActs[p.id] = astAct
				case "prec":
					if len(dir.Parameters) != 1 || (dir.Parameters[0].ID == "" && dir.Parameters[0].OrderedSymbol == "") {
						b.errs = append(b.errs, &verr.SpecError{
							Cause:  semErrDirInvalidParam,
							Detail: "'prec' directive needs just one ID parameter or ordered symbol",
							Row:    dir.Pos.Row,
							Col:    dir.Pos.Col,
						})
						continue LOOP_RHS
					}
					param := dir.Parameters[0]
					switch {
					case param.ID != "":
						sym, ok := symTab.toSymbol(param.ID)
						if !ok {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrDirInvalidParam,
								Detail: fmt.Sprintf("unknown terminal symbol: %v", param.ID),
								Row:    param.Pos.Row,
								Col:    param.Pos.Col,
							})
							continue LOOP_RHS
						}
						if sym == errSym {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrDirInvalidParam,
								Detail: fmt.Sprintf("'%v' directive cannot be applied to an error symbol", dir.Name),
								Row:    param.Pos.Row,
								Col:    param.Pos.Col,
							})
						}
						if !sym.isTerminal() {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrDirInvalidParam,
								Detail: fmt.Sprintf("the symbol must be a terminal: %v", param.ID),
								Row:    param.Pos.Row,
								Col:    param.Pos.Col,
							})
							continue LOOP_RHS
						}
						prodPrecsTerm[p.id] = sym
						prodPrecPoss[p.id] = &param.Pos
					case param.OrderedSymbol != "":
						prodPrecsOrdSym[p.id] = param.OrderedSymbol
						prodPrecPoss[p.id] = &param.Pos
					}
				case "recover":
					if len(dir.Parameters) > 0 {
						b.errs = append(b.errs, &verr.SpecError{
							Cause:  semErrDirInvalidParam,
							Detail: "'recover' directive needs no parameter",
							Row:    dir.Pos.Row,
							Col:    dir.Pos.Col,
						})
						continue LOOP_RHS
					}
					recoverProds[p.id] = struct{}{}
				default:
					b.errs = append(b.errs, &verr.SpecError{
						Cause:  semErrDirInvalidName,
						Detail: fmt.Sprintf("invalid directive name '%v'", dir.Name),
						Row:    dir.Pos.Row,
						Col:    dir.Pos.Col,
					})
					continue LOOP_RHS
				}
			}
		}
	}

	return &productionsAndActions{
		prods:           prods,
		augStartSym:     augStartSym,
		astActs:         astActs,
		prodPrecsTerm:   prodPrecsTerm,
		prodPrecsOrdSym: prodPrecsOrdSym,
		prodPrecPoss:    prodPrecPoss,
		recoverProds:    recoverProds,
	}, nil
}

func (b *GrammarBuilder) genPrecAndAssoc(symTab *symbolTable, errSym symbol, prodsAndActs *productionsAndActions) (*precAndAssoc, error) {
	termPrec := map[symbolNum]int{}
	termAssoc := map[symbolNum]assocType{}
	ordSymPrec := map[string]int{}
	{
		var precGroup []*spec.DirectiveNode
		for _, dir := range b.AST.Directives {
			if dir.Name == "prec" {
				if dir.Parameters == nil || len(dir.Parameters) != 1 || dir.Parameters[0].Group == nil {
					b.errs = append(b.errs, &verr.SpecError{
						Cause:  semErrDirInvalidParam,
						Detail: "'prec' needs just one directive group",
						Row:    dir.Pos.Row,
						Col:    dir.Pos.Col,
					})
					continue
				}
				precGroup = dir.Parameters[0].Group
				continue
			}

			if dir.Name != "name" && dir.Name != "prec" {
				b.errs = append(b.errs, &verr.SpecError{
					Cause:  semErrDirInvalidName,
					Detail: dir.Name,
					Row:    dir.Pos.Row,
					Col:    dir.Pos.Col,
				})
				continue
			}
		}

		precN := precMin
		for _, dir := range precGroup {
			var assocTy assocType
			switch dir.Name {
			case "left":
				assocTy = assocTypeLeft
			case "right":
				assocTy = assocTypeRight
			case "assign":
				assocTy = assocTypeNil
			default:
				b.errs = append(b.errs, &verr.SpecError{
					Cause:  semErrDirInvalidName,
					Detail: dir.Name,
					Row:    dir.Pos.Row,
					Col:    dir.Pos.Col,
				})
				return nil, nil
			}

			if len(dir.Parameters) == 0 {
				b.errs = append(b.errs, &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: "associativity needs at least one symbol",
					Row:    dir.Pos.Row,
					Col:    dir.Pos.Col,
				})
				return nil, nil
			}
		ASSOC_PARAM_LOOP:
			for _, p := range dir.Parameters {
				switch {
				case p.ID != "":
					sym, ok := symTab.toSymbol(p.ID)
					if !ok {
						b.errs = append(b.errs, &verr.SpecError{
							Cause:  semErrDirInvalidParam,
							Detail: fmt.Sprintf("'%v' is undefined", p.ID),
							Row:    p.Pos.Row,
							Col:    p.Pos.Col,
						})
						return nil, nil
					}
					if sym == errSym {
						b.errs = append(b.errs, &verr.SpecError{
							Cause:  semErrDirInvalidParam,
							Detail: fmt.Sprintf("'%v' directive cannot be applied to an error symbol", dir.Name),
							Row:    p.Pos.Row,
							Col:    p.Pos.Col,
						})
						return nil, nil
					}
					if !sym.isTerminal() {
						b.errs = append(b.errs, &verr.SpecError{
							Cause:  semErrDirInvalidParam,
							Detail: fmt.Sprintf("associativity can take only terminal symbol ('%v' is a non-terminal)", p.ID),
							Row:    p.Pos.Row,
							Col:    p.Pos.Col,
						})
						return nil, nil
					}
					if prec, alreadySet := termPrec[sym.num()]; alreadySet {
						if prec == precN {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrDuplicateAssoc,
								Detail: fmt.Sprintf("'%v' already has the same associativity and precedence", p.ID),
								Row:    p.Pos.Row,
								Col:    p.Pos.Col,
							})
						} else if assoc := termAssoc[sym.num()]; assoc == assocTy {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrDuplicateAssoc,
								Detail: fmt.Sprintf("'%v' already has different precedence", p.ID),
								Row:    p.Pos.Row,
								Col:    p.Pos.Col,
							})
						} else {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrDuplicateAssoc,
								Detail: fmt.Sprintf("'%v' already has different associativity and precedence", p.ID),
								Row:    p.Pos.Row,
								Col:    p.Pos.Col,
							})
						}
						break ASSOC_PARAM_LOOP
					}

					termPrec[sym.num()] = precN
					termAssoc[sym.num()] = assocTy
				case p.OrderedSymbol != "":
					if prec, alreadySet := ordSymPrec[p.OrderedSymbol]; alreadySet {
						if prec == precN {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrDuplicateAssoc,
								Detail: fmt.Sprintf("'$%v' already has the same precedence", p.OrderedSymbol),
								Row:    p.Pos.Row,
								Col:    p.Pos.Col,
							})
						} else {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrDuplicateAssoc,
								Detail: fmt.Sprintf("'$%v' already has different precedence", p.OrderedSymbol),
								Row:    p.Pos.Row,
								Col:    p.Pos.Col,
							})
						}
						break ASSOC_PARAM_LOOP
					}

					ordSymPrec[p.OrderedSymbol] = precN
				default:
					b.errs = append(b.errs, &verr.SpecError{
						Cause:  semErrDirInvalidParam,
						Detail: "a parameter must be an ID or an ordered symbol",
						Row:    p.Pos.Row,
						Col:    p.Pos.Col,
					})
					return nil, nil
				}
			}

			precN++
		}
	}
	if len(b.errs) > 0 {
		return nil, nil
	}

	prodPrec := map[productionNum]int{}
	prodAssoc := map[productionNum]assocType{}
	for _, prod := range prodsAndActs.prods.getAllProductions() {
		// A #prec directive changes only precedence, not associativity.
		if term, ok := prodsAndActs.prodPrecsTerm[prod.id]; ok {
			if prec, ok := termPrec[term.num()]; ok {
				prodPrec[prod.num] = prec
				prodAssoc[prod.num] = assocTypeNil
			} else {
				text, _ := symTab.toText(term)
				b.errs = append(b.errs, &verr.SpecError{
					Cause:  semErrUndefinedPrec,
					Detail: text,
					Row:    prodsAndActs.prodPrecPoss[prod.id].Row,
					Col:    prodsAndActs.prodPrecPoss[prod.id].Col,
				})
			}
		} else if ordSym, ok := prodsAndActs.prodPrecsOrdSym[prod.id]; ok {
			if prec, ok := ordSymPrec[ordSym]; ok {
				prodPrec[prod.num] = prec
				prodAssoc[prod.num] = assocTypeNil
			} else {
				b.errs = append(b.errs, &verr.SpecError{
					Cause:  semErrUndefinedOrdSym,
					Detail: fmt.Sprintf("$%v", ordSym),
					Row:    prodsAndActs.prodPrecPoss[prod.id].Row,
					Col:    prodsAndActs.prodPrecPoss[prod.id].Col,
				})
			}
		} else {
			// A production inherits precedence and associativity from the right-most terminal symbol.
			mostrightTerm := symbolNil
			for _, sym := range prod.rhs {
				if !sym.isTerminal() {
					continue
				}
				mostrightTerm = sym
			}
			if !mostrightTerm.isNil() {
				prodPrec[prod.num] = termPrec[mostrightTerm.num()]
				prodAssoc[prod.num] = termAssoc[mostrightTerm.num()]
			}
		}
	}
	if len(b.errs) > 0 {
		return nil, nil
	}

	return &precAndAssoc{
		termPrec:  termPrec,
		termAssoc: termAssoc,
		prodPrec:  prodPrec,
		prodAssoc: prodAssoc,
	}, nil
}

type compileConfig struct {
	isReportingEnabled bool
}

type CompileOption func(config *compileConfig)

func EnableReporting() CompileOption {
	return func(config *compileConfig) {
		config.isReportingEnabled = true
	}
}

func Compile(gram *Grammar, opts ...CompileOption) (*spec.CompiledGrammar, *spec.Report, error) {
	config := &compileConfig{}
	for _, opt := range opts {
		opt(config)
	}

	lexSpec, err, cErrs := mlcompiler.Compile(gram.lexSpec, mlcompiler.CompressionLevel(mlcompiler.CompressionLevelMax))
	if err != nil {
		if len(cErrs) > 0 {
			var b strings.Builder
			writeCompileError(&b, cErrs[0])
			for _, cerr := range cErrs[1:] {
				fmt.Fprintf(&b, "\n")
				writeCompileError(&b, cerr)
			}
			return nil, nil, fmt.Errorf(b.String())
		}
		return nil, nil, err
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
			return nil, nil, fmt.Errorf("terminal symbol '%v' was not found in a symbol table", k)
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
		return nil, nil, err
	}

	kindAliases := make([]string, gram.symbolTable.termNum.Int())
	for _, sym := range gram.symbolTable.terminalSymbols() {
		kindAliases[sym.num().Int()] = gram.kindAliases[sym]
	}

	nonTerms, err := gram.symbolTable.nonTerminalTexts()
	if err != nil {
		return nil, nil, err
	}

	firstSet, err := genFirstSet(gram.productionSet)
	if err != nil {
		return nil, nil, err
	}

	lr0, err := genLR0Automaton(gram.productionSet, gram.augmentedStartSymbol, gram.errorSymbol)
	if err != nil {
		return nil, nil, err
	}

	var tab *ParsingTable
	var report *spec.Report
	{
		lalr1, err := genLALR1Automaton(lr0, gram.productionSet, firstSet)
		if err != nil {
			return nil, nil, err
		}

		b := &lrTableBuilder{
			automaton:    lalr1.lr0Automaton,
			prods:        gram.productionSet,
			termCount:    len(terms),
			nonTermCount: len(nonTerms),
			symTab:       gram.symbolTable,
			sym2AnonPat:  gram.sym2AnonPat,
			precAndAssoc: gram.precAndAssoc,
		}
		tab, err = b.build()
		if err != nil {
			return nil, nil, err
		}

		if config.isReportingEnabled {
			report, err = b.genReport(tab, gram)
			if err != nil {
				return nil, nil, err
			}
		}
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
	recoverProds := make([]int, len(gram.productionSet.getAllProductions())+1)
	astActEnties := make([][]int, len(gram.productionSet.getAllProductions())+1)
	for _, p := range gram.productionSet.getAllProductions() {
		lhsSyms[p.num] = p.lhs.num().Int()
		altSymCounts[p.num] = p.rhsLen

		if _, ok := gram.recoverProductions[p.id]; ok {
			recoverProds[p.num] = 1
		}

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
		Name: gram.name,
		LexicalSpecification: &spec.LexicalSpecification{
			Lexer: "maleeni",
			Maleeni: &spec.Maleeni{
				Spec:           lexSpec,
				KindToTerminal: kind2Term,
				TerminalToKind: term2Kind,
				Skip:           skip,
				KindAliases:    kindAliases,
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
			ErrorSymbol:             gram.errorSymbol.num().Int(),
			ErrorTrapperStates:      tab.errorTrapperStates,
			RecoverProductions:      recoverProds,
		},
		ASTAction: &spec.ASTAction{
			Entries: astActEnties,
		},
	}, report, nil
}

func writeCompileError(w io.Writer, cErr *mlcompiler.CompileError) {
	if cErr.Fragment {
		fmt.Fprintf(w, "fragment ")
	}
	fmt.Fprintf(w, "%v: %v", cErr.Kind, cErr.Cause)
	if cErr.Detail != "" {
		fmt.Fprintf(w, ": %v", cErr.Detail)
	}
}
