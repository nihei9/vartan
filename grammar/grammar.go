package grammar

import (
	"fmt"
	"io"
	"strings"

	verr "github.com/nihei9/vartan/error"
	"github.com/nihei9/vartan/grammar/lexical"
	"github.com/nihei9/vartan/grammar/symbol"
	spec "github.com/nihei9/vartan/spec/grammar"
	"github.com/nihei9/vartan/spec/grammar/parser"
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
	termPrec  map[symbol.SymbolNum]int
	termAssoc map[symbol.SymbolNum]assocType

	// prodPrec and prodAssoc represent the precedence and the associativities of the production.
	// These values are inherited from the right-most terminal symbols in the RHS of the productions.
	prodPrec  map[productionNum]int
	prodAssoc map[productionNum]assocType
}

func (pa *precAndAssoc) terminalPrecedence(sym symbol.SymbolNum) int {
	prec, ok := pa.termPrec[sym]
	if !ok {
		return precNil
	}

	return prec
}

func (pa *precAndAssoc) terminalAssociativity(sym symbol.SymbolNum) assocType {
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
	lexSpec              *lexical.LexSpec
	skipSymbols          []symbol.Symbol
	productionSet        *productionSet
	augmentedStartSymbol symbol.Symbol
	errorSymbol          symbol.Symbol
	symbolTable          *symbol.SymbolTableReader
	astActions           map[productionID][]*astActionEntry
	precAndAssoc         *precAndAssoc

	// recoverProductions is a set of productions having the recover directive.
	recoverProductions map[productionID]struct{}
}

type buildConfig struct {
	isReportingEnabled bool
}

type BuildOption func(config *buildConfig)

func EnableReporting() BuildOption {
	return func(config *buildConfig) {
		config.isReportingEnabled = true
	}
}

type GrammarBuilder struct {
	AST *parser.RootNode

	errs verr.SpecErrors
}

func (b *GrammarBuilder) Build(opts ...BuildOption) (*spec.CompiledGrammar, *spec.Report, error) {
	gram, err := b.build()
	if err != nil {
		return nil, nil, err
	}

	return compile(gram, opts...)
}

func (b *GrammarBuilder) build() (*Grammar, error) {
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

	symTab, ss, err := b.genSymbolTable(b.AST)
	if err != nil {
		return nil, err
	}

	lexSpec, skip, err := b.genLexSpecAndSkipSymbols(symTab.Reader(), b.AST)
	if err != nil {
		return nil, err
	}

	prodsAndActs, err := b.genProductionsAndActions(b.AST, symTab.Reader(), ss.errSym, ss.augStartSym, ss.startSym)
	if err != nil {
		return nil, err
	}
	if prodsAndActs == nil && len(b.errs) > 0 {
		return nil, b.errs
	}

	pa, err := b.genPrecAndAssoc(symTab.Reader(), ss.errSym, prodsAndActs)
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
	{
		r := symTab.Reader()
		for _, sym := range skip {
			s, _ := r.ToText(sym)
			if _, ok := syms.unusedTerminals[s]; !ok {
				prod := syms.usedTerminals[s]
				b.errs = append(b.errs, &verr.SpecError{
					Cause:  semErrTermCannotBeSkipped,
					Detail: s,
					Row:    prod.Pos.Row,
					Col:    prod.Pos.Col,
				})
				continue
			}

			delete(syms.unusedTerminals, s)
		}
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

	return &Grammar{
		name:                 specName,
		lexSpec:              lexSpec,
		skipSymbols:          skip,
		productionSet:        prodsAndActs.prods,
		augmentedStartSymbol: prodsAndActs.augStartSym,
		errorSymbol:          ss.errSym,
		symbolTable:          symTab.Reader(),
		astActions:           prodsAndActs.astActs,
		recoverProductions:   prodsAndActs.recoverProds,
		precAndAssoc:         pa,
	}, nil
}

type usedAndUnusedSymbols struct {
	unusedProductions map[string]*parser.ProductionNode
	unusedTerminals   map[string]*parser.ProductionNode
	usedTerminals     map[string]*parser.ProductionNode
}

func findUsedAndUnusedSymbols(root *parser.RootNode) *usedAndUnusedSymbols {
	prods := map[string]*parser.ProductionNode{}
	lexProds := map[string]*parser.ProductionNode{}
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

	usedTerms := make(map[string]*parser.ProductionNode, len(lexProds))
	unusedProds := map[string]*parser.ProductionNode{}
	unusedTerms := map[string]*parser.ProductionNode{}
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

func markUsedSymbols(mark map[string]bool, marked map[string]bool, prods map[string]*parser.ProductionNode, prod *parser.ProductionNode) {
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

func (b *GrammarBuilder) checkSpellingInconsistenciesOfUserDefinedIDs(root *parser.RootNode) {
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

	duplicated := lexical.FindSpellingInconsistencies(ids)
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

func collectUserDefinedIDsFromDirective(dir *parser.DirectiveNode) []string {
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

type symbols struct {
	errSym      symbol.Symbol
	augStartSym symbol.Symbol
	startSym    symbol.Symbol
}

func (b *GrammarBuilder) genSymbolTable(root *parser.RootNode) (*symbol.SymbolTable, *symbols, error) {
	symTab := symbol.NewSymbolTable()
	w := symTab.Writer()
	r := symTab.Reader()

	// We need to register the reserved symbol before registering others.
	var errSym symbol.Symbol
	{
		sym, err := w.RegisterTerminalSymbol(reservedSymbolNameError)
		if err != nil {
			return nil, nil, err
		}
		errSym = sym
	}

	for _, prod := range root.LexProductions {
		if sym, exist := r.ToSymbol(prod.LHS); exist {
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

		_, err := w.RegisterTerminalSymbol(prod.LHS)
		if err != nil {
			return nil, nil, err
		}
	}

	startProd := root.Productions[0]
	augStartText := fmt.Sprintf("%s'", startProd.LHS)
	var err error
	augStartSym, err := w.RegisterStartSymbol(augStartText)
	if err != nil {
		return nil, nil, err
	}
	if augStartSym == errSym {
		b.errs = append(b.errs, &verr.SpecError{
			Cause: semErrErrSymIsReserved,
			Row:   startProd.Pos.Row,
			Col:   startProd.Pos.Col,
		})
	}

	startSym, err := w.RegisterNonTerminalSymbol(startProd.LHS)
	if err != nil {
		return nil, nil, err
	}
	if startSym == errSym {
		b.errs = append(b.errs, &verr.SpecError{
			Cause: semErrErrSymIsReserved,
			Row:   startProd.Pos.Row,
			Col:   startProd.Pos.Col,
		})
	}

	for _, prod := range root.Productions {
		sym, err := w.RegisterNonTerminalSymbol(prod.LHS)
		if err != nil {
			return nil, nil, err
		}
		if sym.IsTerminal() {
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

	return symTab, &symbols{
		errSym:      errSym,
		augStartSym: augStartSym,
		startSym:    startSym,
	}, nil
}

func (b *GrammarBuilder) genLexSpecAndSkipSymbols(symTab *symbol.SymbolTableReader, root *parser.RootNode) (*lexical.LexSpec, []symbol.Symbol, error) {
	entries := []*lexical.LexEntry{}
	skipSyms := []symbol.Symbol{}
	for _, prod := range root.LexProductions {
		entry, skip, specErr, err := genLexEntry(prod)
		if err != nil {
			return nil, nil, err
		}
		if specErr != nil {
			b.errs = append(b.errs, specErr)
			continue
		}
		if skip {
			sym, _ := symTab.ToSymbol(prod.LHS)
			skipSyms = append(skipSyms, sym)
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

		entries = append(entries, &lexical.LexEntry{
			Fragment: true,
			Kind:     spec.LexKindName(fragment.LHS),
			Pattern:  fragment.RHS,
		})
	}

	return &lexical.LexSpec{
		Entries: entries,
	}, skipSyms, nil
}

func genLexEntry(prod *parser.ProductionNode) (*lexical.LexEntry, bool, *verr.SpecError, error) {
	alt := prod.RHS[0]
	elem := alt.Elements[0]

	var pattern string
	if elem.Literally {
		pattern = spec.EscapePattern(elem.Pattern)
	} else {
		pattern = elem.Pattern
	}

	var modes []spec.LexModeName
	var skip bool
	var push spec.LexModeName
	var pop bool
	dirConsumed := map[string]struct{}{}
	for _, dir := range prod.Directives {
		if _, consumed := dirConsumed[dir.Name]; consumed {
			return nil, false, &verr.SpecError{
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
				return nil, false, &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: "'mode' directive needs an ID parameter",
					Row:    dir.Pos.Row,
					Col:    dir.Pos.Col,
				}, nil
			}
			for _, param := range dir.Parameters {
				if param.ID == "" {
					return nil, false, &verr.SpecError{
						Cause:  semErrDirInvalidParam,
						Detail: "'mode' directive needs an ID parameter",
						Row:    param.Pos.Row,
						Col:    param.Pos.Col,
					}, nil
				}
				modes = append(modes, spec.LexModeName(param.ID))
			}
		case "skip":
			if len(dir.Parameters) > 0 {
				return nil, false, &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: "'skip' directive needs no parameter",
					Row:    dir.Pos.Row,
					Col:    dir.Pos.Col,
				}, nil
			}
			skip = true
		case "push":
			if len(dir.Parameters) != 1 || dir.Parameters[0].ID == "" {
				return nil, false, &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: "'push' directive needs an ID parameter",
					Row:    dir.Pos.Row,
					Col:    dir.Pos.Col,
				}, nil
			}
			push = spec.LexModeName(dir.Parameters[0].ID)
		case "pop":
			if len(dir.Parameters) > 0 {
				return nil, false, &verr.SpecError{
					Cause:  semErrDirInvalidParam,
					Detail: "'pop' directive needs no parameter",
					Row:    dir.Pos.Row,
					Col:    dir.Pos.Col,
				}, nil
			}
			pop = true
		default:
			return nil, false, &verr.SpecError{
				Cause:  semErrDirInvalidName,
				Detail: dir.Name,
				Row:    dir.Pos.Row,
				Col:    dir.Pos.Col,
			}, nil
		}
	}

	if len(alt.Directives) > 0 {
		return nil, false, &verr.SpecError{
			Cause:  semErrInvalidAltDir,
			Detail: "a lexical production cannot have alternative directives",
			Row:    alt.Directives[0].Pos.Row,
			Col:    alt.Directives[0].Pos.Col,
		}, nil
	}

	return &lexical.LexEntry{
		Modes:   modes,
		Kind:    spec.LexKindName(prod.LHS),
		Pattern: pattern,
		Push:    push,
		Pop:     pop,
	}, skip, nil, nil
}

type productionsAndActions struct {
	prods           *productionSet
	augStartSym     symbol.Symbol
	astActs         map[productionID][]*astActionEntry
	prodPrecsTerm   map[productionID]symbol.Symbol
	prodPrecsOrdSym map[productionID]string
	prodPrecPoss    map[productionID]*parser.Position
	recoverProds    map[productionID]struct{}
}

func (b *GrammarBuilder) genProductionsAndActions(root *parser.RootNode, symTab *symbol.SymbolTableReader, errSym symbol.Symbol, augStartSym symbol.Symbol, startSym symbol.Symbol) (*productionsAndActions, error) {
	if len(root.Productions) == 0 {
		b.errs = append(b.errs, &verr.SpecError{
			Cause: semErrNoProduction,
		})
		return nil, nil
	}

	prods := newProductionSet()
	astActs := map[productionID][]*astActionEntry{}
	prodPrecsTerm := map[productionID]symbol.Symbol{}
	prodPrecsOrdSym := map[productionID]string{}
	prodPrecPoss := map[productionID]*parser.Position{}
	recoverProds := map[productionID]struct{}{}

	p, err := newProduction(augStartSym, []symbol.Symbol{
		startSym,
	})
	if err != nil {
		return nil, err
	}

	prods.append(p)

	for _, prod := range root.Productions {
		lhsSym, ok := symTab.ToSymbol(prod.LHS)
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
			altSyms := make([]symbol.Symbol, len(alt.Elements))
			offsets := map[string]int{}
			ambiguousIDOffsets := map[string]struct{}{}
			for i, elem := range alt.Elements {
				sym, ok := symTab.ToSymbol(elem.ID)
				if !ok {
					b.errs = append(b.errs, &verr.SpecError{
						Cause:  semErrUndefinedSym,
						Detail: elem.ID,
						Row:    elem.Pos.Row,
						Col:    elem.Pos.Col,
					})
					continue LOOP_RHS
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
					if _, found := symTab.ToSymbol(elem.Label.Name); found {
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
								// Currently, it is a bug to reach here because it is
								// forbidden to have anything other than ID appear in
								// production rules.
								b.errs = append(b.errs, &verr.SpecError{
									Cause:  semErrDirInvalidParam,
									Detail: fmt.Sprintf("the expansion symbol cannot be applied to a pattern (%v: \"%v\")", param.ID, elem.Pattern),
									Row:    param.Pos.Row,
									Col:    param.Pos.Col,
								})
								continue LOOP_RHS
							}
							elemSym, ok := symTab.ToSymbol(elem.ID)
							if !ok {
								// If the symbol was not found, it's a bug.
								return nil, fmt.Errorf("a symbol corresponding to an ID (%v) was not found", elem.ID)
							}
							if elemSym.IsTerminal() {
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
						sym, ok := symTab.ToSymbol(param.ID)
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
						if !sym.IsTerminal() {
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

func (b *GrammarBuilder) genPrecAndAssoc(symTab *symbol.SymbolTableReader, errSym symbol.Symbol, prodsAndActs *productionsAndActions) (*precAndAssoc, error) {
	termPrec := map[symbol.SymbolNum]int{}
	termAssoc := map[symbol.SymbolNum]assocType{}
	ordSymPrec := map[string]int{}
	{
		var precGroup []*parser.DirectiveNode
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
					sym, ok := symTab.ToSymbol(p.ID)
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
					if !sym.IsTerminal() {
						b.errs = append(b.errs, &verr.SpecError{
							Cause:  semErrDirInvalidParam,
							Detail: fmt.Sprintf("associativity can take only terminal symbol ('%v' is a non-terminal)", p.ID),
							Row:    p.Pos.Row,
							Col:    p.Pos.Col,
						})
						return nil, nil
					}
					if prec, alreadySet := termPrec[sym.Num()]; alreadySet {
						if prec == precN {
							b.errs = append(b.errs, &verr.SpecError{
								Cause:  semErrDuplicateAssoc,
								Detail: fmt.Sprintf("'%v' already has the same associativity and precedence", p.ID),
								Row:    p.Pos.Row,
								Col:    p.Pos.Col,
							})
						} else if assoc := termAssoc[sym.Num()]; assoc == assocTy {
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

					termPrec[sym.Num()] = precN
					termAssoc[sym.Num()] = assocTy
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
			if prec, ok := termPrec[term.Num()]; ok {
				prodPrec[prod.num] = prec
				prodAssoc[prod.num] = assocTypeNil
			} else {
				text, _ := symTab.ToText(term)
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
			mostrightTerm := symbol.SymbolNil
			for _, sym := range prod.rhs {
				if !sym.IsTerminal() {
					continue
				}
				mostrightTerm = sym
			}
			if !mostrightTerm.IsNil() {
				prodPrec[prod.num] = termPrec[mostrightTerm.Num()]
				prodAssoc[prod.num] = termAssoc[mostrightTerm.Num()]
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

func compile(gram *Grammar, opts ...BuildOption) (*spec.CompiledGrammar, *spec.Report, error) {
	config := &buildConfig{}
	for _, opt := range opts {
		opt(config)
	}

	lexSpec, err, cErrs := lexical.Compile(gram.lexSpec, lexical.CompressionLevelMax)
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
	for i, k := range lexSpec.KindNames {
		if k == spec.LexKindNameNil {
			kind2Term[spec.LexKindIDNil] = symbol.SymbolNil.Num().Int()
			continue
		}

		sym, ok := gram.symbolTable.ToSymbol(k.String())
		if !ok {
			return nil, nil, fmt.Errorf("terminal symbol '%v' was not found in a symbol table", k)
		}
		kind2Term[i] = sym.Num().Int()
	}

	termTexts, err := gram.symbolTable.TerminalTexts()
	if err != nil {
		return nil, nil, err
	}

	var termSkip []int
	{
		r := gram.symbolTable.Reader()
		// I want to use gram.symbolTable.terminalSymbols() here instead of gram.symbolTable.terminalTexts(),
		// but gram.symbolTable.terminalSymbols() is different in length from terminalTexts
		// because it does not contain a predefined symbol, like EOF.
		// Therefore, we use terminalTexts, although it takes more time to lookup for symbols.
		termSkip = make([]int, len(termTexts))
		for _, t := range termTexts {
			s, _ := r.ToSymbol(t)
			for _, sk := range gram.skipSymbols {
				if s != sk {
					continue
				}
				termSkip[s.Num()] = 1
				break
			}
		}
	}

	nonTerms, err := gram.symbolTable.NonTerminalTexts()
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
			termCount:    len(termTexts),
			nonTermCount: len(nonTerms),
			symTab:       gram.symbolTable,
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
		lhsSyms[p.num] = p.lhs.Num().Int()
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
		Name:    gram.name,
		Lexical: lexSpec,
		Syntactic: &spec.SyntacticSpec{
			Action:                  action,
			GoTo:                    goTo,
			StateCount:              tab.stateCount,
			InitialState:            tab.InitialState.Int(),
			StartProduction:         productionNumStart.Int(),
			LHSSymbols:              lhsSyms,
			AlternativeSymbolCounts: altSymCounts,
			Terminals:               termTexts,
			TerminalCount:           tab.terminalCount,
			TerminalSkip:            termSkip,
			KindToTerminal:          kind2Term,
			NonTerminals:            nonTerms,
			NonTerminalCount:        tab.nonTerminalCount,
			EOFSymbol:               symbol.SymbolEOF.Num().Int(),
			ErrorSymbol:             gram.errorSymbol.Num().Int(),
			ErrorTrapperStates:      tab.errorTrapperStates,
			RecoverProductions:      recoverProds,
		},
		ASTAction: &spec.ASTAction{
			Entries: astActEnties,
		},
	}, report, nil
}

func writeCompileError(w io.Writer, cErr *lexical.CompileError) {
	if cErr.Fragment {
		fmt.Fprintf(w, "fragment ")
	}
	fmt.Fprintf(w, "%v: %v", cErr.Kind, cErr.Cause)
	if cErr.Detail != "" {
		fmt.Fprintf(w, ": %v", cErr.Detail)
	}
}
