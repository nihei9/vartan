package grammar

import (
	"fmt"
	"sort"
)

type lr0Automaton struct {
	initialState kernelID
	states       map[kernelID]*lrState
}

func genLR0Automaton(prods *productionSet, startSym symbol) (*lr0Automaton, error) {
	if !startSym.isStart() {
		return nil, fmt.Errorf("passed symbold is not a start symbol")
	}

	automaton := &lr0Automaton{
		states: map[kernelID]*lrState{},
	}

	currentState := stateNumInitial
	knownKernels := map[kernelID]struct{}{}
	uncheckedKernels := []*kernel{}

	// Generate an initial kernel.
	{
		prods, _ := prods.findByLHS(startSym)
		initialItem, err := newLR0Item(prods[0], 0)
		if err != nil {
			return nil, err
		}

		k, err := newKernel([]*lrItem{initialItem})
		if err != nil {
			return nil, err
		}

		automaton.initialState = k.id
		knownKernels[k.id] = struct{}{}
		uncheckedKernels = append(uncheckedKernels, k)
	}

	for len(uncheckedKernels) > 0 {
		nextUncheckedKernels := []*kernel{}
		for _, k := range uncheckedKernels {
			state, neighbours, err := genStateAndNeighbourKernels(k, prods)
			if err != nil {
				return nil, err
			}
			state.num = currentState
			currentState = currentState.next()

			automaton.states[state.id] = state

			for _, k := range neighbours {
				if _, known := knownKernels[k.id]; known {
					continue
				}
				knownKernels[k.id] = struct{}{}
				nextUncheckedKernels = append(nextUncheckedKernels, k)
			}
		}
		uncheckedKernels = nextUncheckedKernels
	}

	return automaton, nil
}

func genStateAndNeighbourKernels(k *kernel, prods *productionSet) (*lrState, []*kernel, error) {
	items, err := genLR0Closure(k, prods)
	if err != nil {
		return nil, nil, err
	}
	neighbours, err := genNeighbourKernels(items, prods)
	if err != nil {
		return nil, nil, err
	}

	next := map[symbol]kernelID{}
	kernels := []*kernel{}
	for _, n := range neighbours {
		next[n.symbol] = n.kernel.id
		kernels = append(kernels, n.kernel)
	}

	reducible := map[productionID]struct{}{}
	for _, item := range items {
		if item.reducible {
			reducible[item.prod] = struct{}{}
		}
	}

	return &lrState{
		kernel:    k,
		next:      next,
		reducible: reducible,
	}, kernels, nil
}

func genLR0Closure(k *kernel, prods *productionSet) ([]*lrItem, error) {
	items := []*lrItem{}
	knownItems := map[lrItemID]struct{}{}
	uncheckedItems := []*lrItem{}
	for _, item := range k.items {
		items = append(items, item)
		uncheckedItems = append(uncheckedItems, item)
	}
	for len(uncheckedItems) > 0 {
		nextUncheckedItems := []*lrItem{}
		for _, item := range uncheckedItems {
			if item.dottedSymbol.isTerminal() {
				continue
			}

			ps, _ := prods.findByLHS(item.dottedSymbol)
			for _, prod := range ps {
				item, err := newLR0Item(prod, 0)
				if err != nil {
					return nil, err
				}
				if _, exist := knownItems[item.id]; exist {
					continue
				}
				items = append(items, item)
				knownItems[item.id] = struct{}{}
				nextUncheckedItems = append(nextUncheckedItems, item)
			}
		}
		uncheckedItems = nextUncheckedItems
	}

	return items, nil
}

type neighbourKernel struct {
	symbol symbol
	kernel *kernel
}

func genNeighbourKernels(items []*lrItem, prods *productionSet) ([]*neighbourKernel, error) {
	kItemMap := map[symbol][]*lrItem{}
	for _, item := range items {
		if item.dottedSymbol.isNil() {
			continue
		}
		prod, ok := prods.findByID(item.prod)
		if !ok {
			return nil, fmt.Errorf("a production was not found: %v", item.prod)
		}
		kItem, err := newLR0Item(prod, item.dot+1)
		if err != nil {
			return nil, err
		}
		kItemMap[item.dottedSymbol] = append(kItemMap[item.dottedSymbol], kItem)
	}

	nextSyms := []symbol{}
	for sym := range kItemMap {
		nextSyms = append(nextSyms, sym)
	}
	sort.Slice(nextSyms, func(i, j int) bool {
		return nextSyms[i] < nextSyms[j]
	})

	kernels := []*neighbourKernel{}
	for _, sym := range nextSyms {
		k, err := newKernel(kItemMap[sym])
		if err != nil {
			return nil, err
		}
		kernels = append(kernels, &neighbourKernel{
			symbol: sym,
			kernel: k,
		})
	}

	return kernels, nil
}
