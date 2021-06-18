package grammar

import (
	"fmt"
)

type followEntry struct {
	symbols map[symbol]struct{}
	eof     bool
}

func newFollowEntry() *followEntry {
	return &followEntry{
		symbols: map[symbol]struct{}{},
		eof:     false,
	}
}

func (e *followEntry) add(sym symbol) bool {
	if _, ok := e.symbols[sym]; ok {
		return false
	}
	e.symbols[sym] = struct{}{}
	return true
}

func (e *followEntry) addEOF() bool {
	if !e.eof {
		e.eof = true
		return true
	}
	return false
}

func (e *followEntry) merge(fst *firstEntry, flw *followEntry) bool {
	changed := false

	if fst != nil {
		for sym := range fst.symbols {
			added := e.add(sym)
			if added {
				changed = true
			}
		}
	}

	if flw != nil {
		for sym := range flw.symbols {
			added := e.add(sym)
			if added {
				changed = true
			}
		}
		if flw.eof {
			added := e.addEOF()
			if added {
				changed = true
			}
		}
	}

	return changed
}

type followSet struct {
	set map[symbol]*followEntry
}

func newFollow(prods *productionSet) *followSet {
	flw := &followSet{
		set: map[symbol]*followEntry{},
	}
	for _, prod := range prods.getAllProductions() {
		if _, ok := flw.set[prod.lhs]; ok {
			continue
		}
		flw.set[prod.lhs] = newFollowEntry()
	}
	return flw
}

func (flw *followSet) find(sym symbol) (*followEntry, error) {
	e, ok := flw.set[sym]
	if !ok {
		return nil, fmt.Errorf("an entry of FOLLOW was not found; symbol: %s", sym)
	}
	return e, nil
}

type followComContext struct {
	prods  *productionSet
	first  *firstSet
	follow *followSet
}

func newFollowComContext(prods *productionSet, first *firstSet) *followComContext {
	return &followComContext{
		prods:  prods,
		first:  first,
		follow: newFollow(prods),
	}
}

func genFollowSet(prods *productionSet, first *firstSet) (*followSet, error) {
	ntsyms := map[symbol]struct{}{}
	for _, prod := range prods.getAllProductions() {
		if _, ok := ntsyms[prod.lhs]; ok {
			continue
		}
		ntsyms[prod.lhs] = struct{}{}
	}

	cc := newFollowComContext(prods, first)
	for {
		more := false
		for ntsym := range ntsyms {
			e, err := cc.follow.find(ntsym)
			if err != nil {
				return nil, err
			}
			if ntsym.isStart() {
				changed := e.addEOF()
				if changed {
					more = true
				}
			}
			for _, prod := range prods.getAllProductions() {
				for i, sym := range prod.rhs {
					if sym != ntsym {
						continue
					}
					fst, err := first.find(prod, i+1)
					if err != nil {
						return nil, err
					}
					changed := e.merge(fst, nil)
					if changed {
						more = true
					}
					if fst.empty {
						flw, err := cc.follow.find(prod.lhs)
						if err != nil {
							return nil, err
						}
						changed := e.merge(nil, flw)
						if changed {
							more = true
						}
					}
				}
			}
		}
		if !more {
			break
		}
	}

	return cc.follow, nil
}

func genFollowEntry(cc *followComContext, acc *followEntry, ntsym symbol) (bool, error) {
	changed := false

	if ntsym.isStart() {
		added := acc.addEOF()
		if added {
			changed = true
		}
	}
	for _, prod := range cc.prods.getAllProductions() {
		for i, sym := range prod.rhs {
			if sym != ntsym {
				continue
			}
			fst, err := cc.first.find(prod, i+1)
			if err != nil {
				return false, err
			}
			added := acc.merge(fst, nil)
			if added {
				changed = true
			}
			if fst.empty {
				flw, err := cc.follow.find(prod.lhs)
				if err != nil {
					return false, err
				}
				added := acc.merge(nil, flw)
				if added {
					changed = true
				}
			}
		}
	}

	return changed, nil
}
