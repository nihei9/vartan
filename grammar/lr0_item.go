package grammar

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"strconv"
)

type lr0ItemID [32]byte

func (id lr0ItemID) String() string {
	return fmt.Sprintf("%x", id.num())
}

func (id lr0ItemID) num() uint32 {
	return binary.LittleEndian.Uint32(id[:])
}

type lr0Item struct {
	id   lr0ItemID
	prod productionID

	// E → E + T
	//
	// Dot | Dotted Symbol | Item
	// ----+---------------+------------
	// 0   | E             | E →・E + T
	// 1   | +             | E → E・+ T
	// 2   | T             | E → E +・T
	// 3   | Nil           | E → E + T・
	dot          int
	dottedSymbol symbol

	// When initial is true, the LHS of the production is the augmented start symbol and dot is 0.
	// It looks like S' →・S.
	initial bool

	// When reducible is true, the item looks like E → E + T・.
	reducible bool

	// When kernel is true, the item is kernel item.
	kernel bool
}

func newLR0Item(prod *production, dot int) (*lr0Item, error) {
	if prod == nil {
		return nil, fmt.Errorf("production must be non-nil")
	}

	if dot < 0 || dot > prod.rhsLen {
		return nil, fmt.Errorf("dot must be between 0 and %v", prod.rhsLen)
	}

	var id lr0ItemID
	{
		b := []byte{}
		b = append(b, prod.id[:]...)
		bDot := make([]byte, 8)
		binary.LittleEndian.PutUint64(bDot, uint64(dot))
		b = append(b, bDot...)
		id = sha256.Sum256(b)
	}

	dottedSymbol := symbolNil
	if dot < prod.rhsLen {
		dottedSymbol = prod.rhs[dot]
	}

	initial := false
	if prod.lhs.isStart() && dot == 0 {
		initial = true
	}

	reducible := false
	if dot == prod.rhsLen {
		reducible = true
	}

	kernel := false
	if initial || dot > 0 {
		kernel = true
	}

	item := &lr0Item{
		id:           id,
		prod:         prod.id,
		dot:          dot,
		dottedSymbol: dottedSymbol,
		initial:      initial,
		reducible:    reducible,
		kernel:       kernel,
	}

	return item, nil
}

type kernelID [32]byte

func (id kernelID) String() string {
	return fmt.Sprintf("%x", binary.LittleEndian.Uint32(id[:]))
}

type kernel struct {
	id    kernelID
	items []*lr0Item
}

func newKernel(items []*lr0Item) (*kernel, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("a kernel need at least one item")
	}

	// Remove duplicates from items.
	var sortedItems []*lr0Item
	{
		m := map[lr0ItemID]*lr0Item{}
		for _, item := range items {
			if !item.kernel {
				return nil, fmt.Errorf("not a kernel item: %v", item)
			}
			m[item.id] = item
		}
		sortedItems = []*lr0Item{}
		for _, item := range m {
			sortedItems = append(sortedItems, item)
		}
		sort.Slice(sortedItems, func(i, j int) bool {
			return sortedItems[i].id.num() < sortedItems[j].id.num()
		})
	}

	var id kernelID
	{
		b := []byte{}
		for _, item := range sortedItems {
			b = append(b, item.id[:]...)
		}
		id = sha256.Sum256(b)
	}

	return &kernel{
		id:    id,
		items: sortedItems,
	}, nil
}

type stateNum int

const stateNumInitial = stateNum(0)

func (n stateNum) Int() int {
	return int(n)
}

func (n stateNum) String() string {
	return strconv.Itoa(int(n))
}

func (n stateNum) next() stateNum {
	return stateNum(n + 1)
}

type lr0State struct {
	*kernel
	num       stateNum
	next      map[symbol]kernelID
	reducible map[productionID]struct{}
}

type lr0Automaton struct {
	initialState kernelID
	states       map[kernelID]*lr0State
}

func genLR0Automaton(prods *productionSet, startSym symbol) (*lr0Automaton, error) {
	if !startSym.isStart() {
		return nil, fmt.Errorf("passed symbold is not a start symbol")
	}

	automaton := &lr0Automaton{
		states: map[kernelID]*lr0State{},
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

		k, err := newKernel([]*lr0Item{initialItem})
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

func genStateAndNeighbourKernels(k *kernel, prods *productionSet) (*lr0State, []*kernel, error) {
	items, err := genClosure(k, prods)
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

	return &lr0State{
		kernel:    k,
		next:      next,
		reducible: reducible,
	}, kernels, nil
}

func genClosure(k *kernel, prods *productionSet) ([]*lr0Item, error) {
	items := []*lr0Item{}
	knownItems := map[lr0ItemID]struct{}{}
	uncheckedItems := []*lr0Item{}
	for _, item := range k.items {
		items = append(items, item)
		uncheckedItems = append(uncheckedItems, item)
	}
	for len(uncheckedItems) > 0 {
		nextUncheckedItems := []*lr0Item{}
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

func genNeighbourKernels(items []*lr0Item, prods *productionSet) ([]*neighbourKernel, error) {
	kItemMap := map[symbol][]*lr0Item{}
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
