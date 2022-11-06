package grammar

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/nihei9/vartan/grammar/symbol"
)

type productionID [32]byte

func (id productionID) String() string {
	return hex.EncodeToString(id[:])
}

func genProductionID(lhs symbol.Symbol, rhs []symbol.Symbol) productionID {
	seq := lhs.Byte()
	for _, sym := range rhs {
		seq = append(seq, sym.Byte()...)
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
	lhs    symbol.Symbol
	rhs    []symbol.Symbol
	rhsLen int
}

func newProduction(lhs symbol.Symbol, rhs []symbol.Symbol) (*production, error) {
	if lhs.IsNil() {
		return nil, fmt.Errorf("LHS must be a non-nil symbol; LHS: %v, RHS: %v", lhs, rhs)
	}
	for _, sym := range rhs {
		if sym.IsNil() {
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

func (p *production) isEmpty() bool {
	return p.rhsLen == 0
}

type productionSet struct {
	lhs2Prods map[symbol.Symbol][]*production
	id2Prod   map[productionID]*production
	num       productionNum
}

func newProductionSet() *productionSet {
	return &productionSet{
		lhs2Prods: map[symbol.Symbol][]*production{},
		id2Prod:   map[productionID]*production{},
		num:       productionNumMin,
	}
}

func (ps *productionSet) append(prod *production) {
	if _, ok := ps.id2Prod[prod.id]; ok {
		return
	}

	if prod.lhs.IsStart() {
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
}

func (ps *productionSet) findByID(id productionID) (*production, bool) {
	prod, ok := ps.id2Prod[id]
	return prod, ok
}

func (ps *productionSet) findByLHS(lhs symbol.Symbol) ([]*production, bool) {
	if lhs.IsNil() {
		return nil, false
	}

	prods, ok := ps.lhs2Prods[lhs]
	return prods, ok
}

func (ps *productionSet) getAllProductions() map[productionID]*production {
	return ps.id2Prod
}
