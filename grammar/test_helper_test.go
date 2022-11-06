package grammar

import (
	"testing"

	"github.com/nihei9/vartan/grammar/symbol"
)

type testSymbolGenerator func(text string) symbol.Symbol

func newTestSymbolGenerator(t *testing.T, symTab *symbol.SymbolTableReader) testSymbolGenerator {
	return func(text string) symbol.Symbol {
		t.Helper()

		sym, ok := symTab.ToSymbol(text)
		if !ok {
			t.Fatalf("symbol was not found: %v", text)
		}
		return sym
	}
}

type testProductionGenerator func(lhs string, rhs ...string) *production

func newTestProductionGenerator(t *testing.T, genSym testSymbolGenerator) testProductionGenerator {
	return func(lhs string, rhs ...string) *production {
		t.Helper()

		rhsSym := []symbol.Symbol{}
		for _, text := range rhs {
			rhsSym = append(rhsSym, genSym(text))
		}
		prod, err := newProduction(genSym(lhs), rhsSym)
		if err != nil {
			t.Fatalf("failed to create a production: %v", err)
		}

		return prod
	}
}

type testLR0ItemGenerator func(lhs string, dot int, rhs ...string) *lrItem

func newTestLR0ItemGenerator(t *testing.T, genProd testProductionGenerator) testLR0ItemGenerator {
	return func(lhs string, dot int, rhs ...string) *lrItem {
		t.Helper()

		prod := genProd(lhs, rhs...)
		item, err := newLR0Item(prod, dot)
		if err != nil {
			t.Fatalf("failed to create a LR0 item: %v", err)
		}

		return item
	}
}

func withLookAhead(item *lrItem, lookAhead ...symbol.Symbol) *lrItem {
	if item.lookAhead.symbols == nil {
		item.lookAhead.symbols = map[symbol.Symbol]struct{}{}
	}

	for _, a := range lookAhead {
		item.lookAhead.symbols[a] = struct{}{}
	}

	return item
}
