package grammar

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type productionID [32]byte

func (id productionID) String() string {
	return hex.EncodeToString(id[:])
}

func genProductionID(lhs symbol, rhs []symbol) productionID {
	seq := lhs.byte()
	for _, sym := range rhs {
		seq = append(seq, sym.byte()...)
	}
	return productionID(sha256.Sum256(seq))
}

type productionNum uint16

const (
	productionNumNil   = productionNum(0)
	productionNumStart = productionNum(1)
	productionNumMin   = productionNum(2)
)

func (n productionNum) Int() int {
	return int(n)
}

type production struct {
	id     productionID
	num    productionNum
	lhs    symbol
	rhs    []symbol
	rhsLen int
}

func newProduction(lhs symbol, rhs []symbol) (*production, error) {
	if lhs.isNil() {
		return nil, fmt.Errorf("LHS must be a non-nil symbol; LHS: %v, RHS: %v", lhs, rhs)
	}
	for _, sym := range rhs {
		if sym.isNil() {
			return nil, fmt.Errorf("a symbol of RHS must be a non-nil symbol; LHS: %v, RHS: %v", lhs, rhs)
		}
	}

	return &production{
		id:     genProductionID(lhs, rhs),
		lhs:    lhs,
		rhs:    rhs,
		rhsLen: len(rhs),
	}, nil
}

func (p *production) equals(q *production) bool {
	return q.id == p.id
}

func (p *production) isEmpty() bool {
	return p.rhsLen == 0
}

type productionSet struct {
	lhs2Prods map[symbol][]*production
	id2Prod   map[productionID]*production
	num       productionNum
}

func newProductionSet() *productionSet {
	return &productionSet{
		lhs2Prods: map[symbol][]*production{},
		id2Prod:   map[productionID]*production{},
		num:       productionNumMin,
	}
}

func (ps *productionSet) append(prod *production) bool {
	if _, ok := ps.id2Prod[prod.id]; ok {
		return false
	}

	if prod.lhs.isStart() {
		prod.num = productionNumStart
	} else {
		prod.num = ps.num
		ps.num++
	}

	if prods, ok := ps.lhs2Prods[prod.lhs]; ok {
		ps.lhs2Prods[prod.lhs] = append(prods, prod)
	} else {
		ps.lhs2Prods[prod.lhs] = []*production{prod}
	}
	ps.id2Prod[prod.id] = prod

	return true
}

func (ps *productionSet) findByID(id productionID) (*production, bool) {
	prod, ok := ps.id2Prod[id]
	return prod, ok
}

func (ps *productionSet) findByLHS(lhs symbol) ([]*production, bool) {
	if lhs.isNil() {
		return nil, false
	}

	prods, ok := ps.lhs2Prods[lhs]
	return prods, ok
}

func (ps *productionSet) getAllProductions() map[productionID]*production {
	return ps.id2Prod
}
