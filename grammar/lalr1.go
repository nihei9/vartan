package grammar

import (
	"fmt"

	"github.com/nihei9/vartan/grammar/symbol"
)

type stateAndLRItem struct {
	kernelID kernelID
	itemID   lrItemID
}

type propagation struct {
	src  *stateAndLRItem
	dest []*stateAndLRItem
}

type lalr1Automaton struct {
	*lr0Automaton
}

func genLALR1Automaton(lr0 *lr0Automaton, prods *productionSet, first *firstSet) (*lalr1Automaton, error) {
	// Set the look-ahead symbol <EOF> to the initial item: [S' → ・S, $]
	iniState := lr0.states[lr0.initialState]
	iniState.items[0].lookAhead.symbols = map[symbol.Symbol]struct{}{
		symbol.SymbolEOF: {},
	}

	var props []*propagation
	for _, state := range lr0.states {
		for _, kItem := range state.items {
			items, err := genLALR1Closure(kItem, prods, first)
			if err != nil {
				return nil, err
			}

			kItem.lookAhead.propagation = true

			var propDests []*stateAndLRItem
			for _, item := range items {
				if item.reducible {
					p, ok := prods.findByID(item.prod)
					if !ok {
						return nil, fmt.Errorf("production not found: %v", item.prod)
					}

					if p.isEmpty() {
						var reducibleItem *lrItem
						for _, it := range state.emptyProdItems {
							if it.id != item.id {
								continue
							}

							reducibleItem = it
							break
						}
						if reducibleItem == nil {
							return nil, fmt.Errorf("reducible item not found: %v", item.id)
						}
						if reducibleItem.lookAhead.symbols == nil {
							reducibleItem.lookAhead.symbols = map[symbol.Symbol]struct{}{}
						}
						for a := range item.lookAhead.symbols {
							reducibleItem.lookAhead.symbols[a] = struct{}{}
						}

						propDests = append(propDests, &stateAndLRItem{
							kernelID: state.id,
							itemID:   item.id,
						})
					}

					continue
				}

				nextKID := state.next[item.dottedSymbol]
				var nextItemID lrItemID
				{
					p, ok := prods.findByID(item.prod)
					if !ok {
						return nil, fmt.Errorf("production not found: %v", item.prod)
					}
					it, err := newLR0Item(p, item.dot+1)
					if err != nil {
						return nil, fmt.Errorf("failed to generate an item ID: %v", err)
					}
					nextItemID = it.id
				}

				if item.lookAhead.propagation {
					propDests = append(propDests, &stateAndLRItem{
						kernelID: nextKID,
						itemID:   nextItemID,
					})
				} else {
					nextState := lr0.states[nextKID]
					var nextItem *lrItem
					for _, it := range nextState.items {
						if it.id != nextItemID {
							continue
						}
						nextItem = it
						break
					}
					if nextItem == nil {
						return nil, fmt.Errorf("item not found: %v", nextItemID)
					}

					if nextItem.lookAhead.symbols == nil {
						nextItem.lookAhead.symbols = map[symbol.Symbol]struct{}{}
					}

					for a := range item.lookAhead.symbols {
						nextItem.lookAhead.symbols[a] = struct{}{}
					}
				}
			}
			if len(propDests) == 0 {
				continue
			}

			props = append(props, &propagation{
				src: &stateAndLRItem{
					kernelID: state.id,
					itemID:   kItem.id,
				},
				dest: propDests,
			})
		}
	}

	err := propagateLookAhead(lr0, props)
	if err != nil {
		return nil, fmt.Errorf("failed to propagate look-ahead symbols: %v", err)
	}

	return &lalr1Automaton{
		lr0Automaton: lr0,
	}, nil
}

func genLALR1Closure(srcItem *lrItem, prods *productionSet, first *firstSet) ([]*lrItem, error) {
	items := []*lrItem{}
	knownItems := map[lrItemID]map[symbol.Symbol]struct{}{}
	knownItemsProp := map[lrItemID]struct{}{}
	uncheckedItems := []*lrItem{}
	items = append(items, srcItem)
	uncheckedItems = append(uncheckedItems, srcItem)
	for len(uncheckedItems) > 0 {
		nextUncheckedItems := []*lrItem{}
		for _, item := range uncheckedItems {
			if item.dottedSymbol.IsTerminal() {
				continue
			}

			p, ok := prods.findByID(item.prod)
			if !ok {
				return nil, fmt.Errorf("production not found: %v", item.prod)
			}

			var fstSyms []symbol.Symbol
			var isFstNullable bool
			{
				fst, err := first.find(p, item.dot+1)
				if err != nil {
					return nil, err
				}

				fstSyms = make([]symbol.Symbol, len(fst.symbols))
				i := 0
				for s := range fst.symbols {
					fstSyms[i] = s
					i++
				}
				if fst.empty {
					isFstNullable = true
				}
			}

			ps, _ := prods.findByLHS(item.dottedSymbol)
			for _, prod := range ps {
				var lookAhead []symbol.Symbol
				{
					var lookAheadCount int
					if isFstNullable {
						lookAheadCount = len(fstSyms) + len(item.lookAhead.symbols)
					} else {
						lookAheadCount = len(fstSyms)
					}

					lookAhead = make([]symbol.Symbol, lookAheadCount)
					i := 0
					for _, s := range fstSyms {
						lookAhead[i] = s
						i++
					}
					if isFstNullable {
						for a := range item.lookAhead.symbols {
							lookAhead[i] = a
							i++
						}
					}
				}

				for _, a := range lookAhead {
					newItem, err := newLR0Item(prod, 0)
					if err != nil {
						return nil, err
					}
					if items, exist := knownItems[newItem.id]; exist {
						if _, exist := items[a]; exist {
							continue
						}
					}

					newItem.lookAhead.symbols = map[symbol.Symbol]struct{}{
						a: {},
					}

					items = append(items, newItem)
					if knownItems[newItem.id] == nil {
						knownItems[newItem.id] = map[symbol.Symbol]struct{}{}
					}
					knownItems[newItem.id][a] = struct{}{}
					nextUncheckedItems = append(nextUncheckedItems, newItem)
				}

				if isFstNullable {
					newItem, err := newLR0Item(prod, 0)
					if err != nil {
						return nil, err
					}
					if _, exist := knownItemsProp[newItem.id]; exist {
						continue
					}

					newItem.lookAhead.propagation = true

					items = append(items, newItem)
					knownItemsProp[newItem.id] = struct{}{}
					nextUncheckedItems = append(nextUncheckedItems, newItem)
				}
			}
		}
		uncheckedItems = nextUncheckedItems
	}

	return items, nil
}

func propagateLookAhead(lr0 *lr0Automaton, props []*propagation) error {
	for {
		changed := false
		for _, prop := range props {
			srcState, ok := lr0.states[prop.src.kernelID]
			if !ok {
				return fmt.Errorf("source state not found: %v", prop.src.kernelID)
			}
			var srcItem *lrItem
			for _, item := range srcState.items {
				if item.id != prop.src.itemID {
					continue
				}
				srcItem = item
				break
			}
			if srcItem == nil {
				return fmt.Errorf("source item not found: %v", prop.src.itemID)
			}

			for _, dest := range prop.dest {
				destState, ok := lr0.states[dest.kernelID]
				if !ok {
					return fmt.Errorf("destination state not found: %v", dest.kernelID)
				}
				var destItem *lrItem
				for _, item := range destState.items {
					if item.id != dest.itemID {
						continue
					}
					destItem = item
					break
				}
				if destItem == nil {
					for _, item := range destState.emptyProdItems {
						if item.id != dest.itemID {
							continue
						}
						destItem = item
						break
					}
					if destItem == nil {
						return fmt.Errorf("destination item not found: %v", dest.itemID)
					}
				}

				for a := range srcItem.lookAhead.symbols {
					if _, ok := destItem.lookAhead.symbols[a]; ok {
						continue
					}

					if destItem.lookAhead.symbols == nil {
						destItem.lookAhead.symbols = map[symbol.Symbol]struct{}{}
					}

					destItem.lookAhead.symbols[a] = struct{}{}
					changed = true
				}
			}
		}
		if !changed {
			break
		}
	}

	return nil
}
