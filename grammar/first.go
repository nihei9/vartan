package grammar

import (
	"fmt"

	"github.com/nihei9/vartan/grammar/symbol"
)

type firstEntry struct {
	symbols map[symbol.Symbol]struct{}
	empty   bool
}

func newFirstEntry() *firstEntry {
	return &firstEntry{
		symbols: map[symbol.Symbol]struct{}{},
		empty:   false,
	}
}

func (e *firstEntry) add(sym symbol.Symbol) bool {
	if _, ok := e.symbols[sym]; ok {
		return false
	}
	e.symbols[sym] = struct{}{}
	return true
}

func (e *firstEntry) addEmpty() bool {
	if !e.empty {
		e.empty = true
		return true
	}
	return false
}

func (e *firstEntry) mergeExceptEmpty(target *firstEntry) bool {
	if target == nil {
		return false
	}
	changed := false
	for sym := range target.symbols {
		added := e.add(sym)
		if added {
			changed = true
		}
	}
	return changed
}

type firstSet struct {
	set map[symbol.Symbol]*firstEntry
}

func newFirstSet(prods *productionSet) *firstSet {
	fst := &firstSet{
		set: map[symbol.Symbol]*firstEntry{},
	}
	for _, prod := range prods.getAllProductions() {
		if _, ok := fst.set[prod.lhs]; ok {
			continue
		}
		fst.set[prod.lhs] = newFirstEntry()
	}

	return fst
}

func (fst *firstSet) find(prod *production, head int) (*firstEntry, error) {
	entry := newFirstEntry()
	if prod.rhsLen <= head {
		entry.addEmpty()
		return entry, nil
	}
	for _, sym := range prod.rhs[head:] {
		if sym.IsTerminal() {
			entry.add(sym)
			return entry, nil
		}

		e := fst.findBySymbol(sym)
		if e == nil {
			return nil, fmt.Errorf("an entry of FIRST was not found; symbol: %s", sym)
		}
		for s := range e.symbols {
			entry.add(s)
		}
		if !e.empty {
			return entry, nil
		}
	}
	entry.addEmpty()
	return entry, nil
}

func (fst *firstSet) findBySymbol(sym symbol.Symbol) *firstEntry {
	return fst.set[sym]
}

type firstComContext struct {
	first *firstSet
}

func newFirstComContext(prods *productionSet) *firstComContext {
	return &firstComContext{
		first: newFirstSet(prods),
	}
}

func genFirstSet(prods *productionSet) (*firstSet, error) {
	cc := newFirstComContext(prods)
	for {
		more := false
		for _, prod := range prods.getAllProductions() {
			e := cc.first.findBySymbol(prod.lhs)
			changed, err := genProdFirstEntry(cc, e, prod)
			if err != nil {
				return nil, err
			}
			if changed {
				more = true
			}
		}
		if !more {
			break
		}
	}
	return cc.first, nil
}

func genProdFirstEntry(cc *firstComContext, acc *firstEntry, prod *production) (bool, error) {
	if prod.isEmpty() {
		return acc.addEmpty(), nil
	}

	for _, sym := range prod.rhs {
		if sym.IsTerminal() {
			return acc.add(sym), nil
		}

		e := cc.first.findBySymbol(sym)
		changed := acc.mergeExceptEmpty(e)
		if !e.empty {
			return changed, nil
		}
	}
	return acc.addEmpty(), nil
}
