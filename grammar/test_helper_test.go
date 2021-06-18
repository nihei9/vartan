package grammar

import "testing"

type testSymbolGenerator func(text string) symbol

func newTestSymbolGenerator(t *testing.T, symTab *symbolTable) testSymbolGenerator {
	return func(text string) symbol {
		t.Helper()

		sym, ok := symTab.toSymbol(text)
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

		rhsSym := []symbol{}
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

type testLR0ItemGenerator func(lhs string, dot int, rhs ...string) *lr0Item

func newTestLR0ItemGenerator(t *testing.T, genProd testProductionGenerator) testLR0ItemGenerator {
	return func(lhs string, dot int, rhs ...string) *lr0Item {
		t.Helper()

		prod := genProd(lhs, rhs...)
		item, err := newLR0Item(prod, dot)
		if err != nil {
			t.Fatalf("failed to create a LR0 item: %v", err)
		}

		return item
	}
}
