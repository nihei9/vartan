package grammar

import "fmt"

type slr1Automaton struct {
	*lr0Automaton
}

func genSLR1Automaton(lr0 *lr0Automaton, prods *productionSet, follow *followSet) (*slr1Automaton, error) {
	for _, state := range lr0.states {
		for prodID := range state.reducible {
			prod, ok := prods.findByID(prodID)
			if !ok {
				return nil, fmt.Errorf("reducible production not found: %v", prodID)
			}

			flw, err := follow.find(prod.lhs)
			if err != nil {
				return nil, err
			}

			var reducibleItem *lrItem
			for _, item := range state.items {
				if item.prod != prodID {
					continue
				}

				reducibleItem = item
				break
			}
			if reducibleItem == nil {
				for _, item := range state.emptyProdItems {
					if item.prod != prodID {
						continue
					}

					reducibleItem = item
					break
				}
				if reducibleItem == nil {
					return nil, fmt.Errorf("reducible item not found; state: %v, production: %v", state.num, prodID)
				}
			}

			if reducibleItem.lookAhead.symbols == nil {
				reducibleItem.lookAhead.symbols = map[symbol]struct{}{}
			}

			for sym := range flw.symbols {
				reducibleItem.lookAhead.symbols[sym] = struct{}{}
			}
			if flw.eof {
				reducibleItem.lookAhead.symbols[symbolEOF] = struct{}{}
			}
		}
	}

	return &slr1Automaton{
		lr0Automaton: lr0,
	}, nil
}
