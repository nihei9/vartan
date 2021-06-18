package grammar

import (
	"fmt"

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

				entries = append(entries, &mlspec.LexEntry{
					Kind:    mlspec.LexKind(prod.LHS),
					Pattern: mlspec.LexPattern(prod.RHS[0].Elements[0].Pattern),
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
