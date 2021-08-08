package grammar

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"strconv"
)

type lrItemID [32]byte

func (id lrItemID) String() string {
	return fmt.Sprintf("%x", id.num())
}

func (id lrItemID) num() uint32 {
	return binary.LittleEndian.Uint32(id[:])
}

type lookAhead struct {
	symbols map[symbol]struct{}

	// When propagation is true, an item propagates look-ahead symbols to other items.
	propagation bool
}

type lrItem struct {
	id   lrItemID
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

	// lookAhead stores look-ahead symbols, and they are terminal symbols.
	// The item is reducible only when the look-ahead symbols appear as the next input symbol.
	lookAhead lookAhead
}

func newLR0Item(prod *production, dot int) (*lrItem, error) {
	if prod == nil {
		return nil, fmt.Errorf("production must be non-nil")
	}

	if dot < 0 || dot > prod.rhsLen {
		return nil, fmt.Errorf("dot must be between 0 and %v", prod.rhsLen)
	}

	var id lrItemID
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

	item := &lrItem{
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
	items []*lrItem
}

func newKernel(items []*lrItem) (*kernel, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("a kernel need at least one item")
	}

	// Remove duplicates from items.
	var sortedItems []*lrItem
	{
		m := map[lrItemID]*lrItem{}
		for _, item := range items {
			if !item.kernel {
				return nil, fmt.Errorf("not a kernel item: %v", item)
			}
			m[item.id] = item
		}
		sortedItems = []*lrItem{}
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

type lrState struct {
	*kernel
	num       stateNum
	next      map[symbol]kernelID
	reducible map[productionID]struct{}

	// emptyProdItems stores items that have an empty production like `p → ε` and is reducible.
	// Thus the items emptyProdItems stores are like `p → ・ε`. emptyProdItems is needed to store
	// look-ahead symbols because the kernel items don't include these items.
	//
	// For instance, we have the following productions, and A is a terminal symbol.
	//
	// s' → s
	// s → A | ε
	//
	// CLOSURE({s' → ・s}) generates the following closure, but the kernel of this closure doesn't
	// include `s → ・ε`.
	//
	// s' → ・s
	// s → ・A
	// s → ・ε
	emptyProdItems []*lrItem
}
