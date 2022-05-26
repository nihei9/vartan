package grammar

import (
	"strings"
	"testing"

	spec "github.com/nihei9/vartan/spec/grammar"
)

type first struct {
	lhs     string
	num     int
	dot     int
	symbols []string
	empty   bool
}

func TestGenFirst(t *testing.T) {
	tests := []struct {
		caption string
		src     string
		first   []first
	}{
		{
			caption: "productions contain only non-empty productions",
			src: `
#name test;

expr
    : expr add term
    | term
    ;
term
    : term mul factor
    | factor
    ;
factor
    : l_paren expr r_paren
    | id
    ;
add: "\+";
mul: "\*";
l_paren: "\(";
r_paren: "\)";
id: "[A-Za-z_][0-9A-Za-z_]*";
`,
			first: []first{
				{lhs: "expr'", num: 0, dot: 0, symbols: []string{"l_paren", "id"}},
				{lhs: "expr", num: 0, dot: 0, symbols: []string{"l_paren", "id"}},
				{lhs: "expr", num: 0, dot: 1, symbols: []string{"add"}},
				{lhs: "expr", num: 0, dot: 2, symbols: []string{"l_paren", "id"}},
				{lhs: "expr", num: 1, dot: 0, symbols: []string{"l_paren", "id"}},
				{lhs: "term", num: 0, dot: 0, symbols: []string{"l_paren", "id"}},
				{lhs: "term", num: 0, dot: 1, symbols: []string{"mul"}},
				{lhs: "term", num: 0, dot: 2, symbols: []string{"l_paren", "id"}},
				{lhs: "term", num: 1, dot: 0, symbols: []string{"l_paren", "id"}},
				{lhs: "factor", num: 0, dot: 0, symbols: []string{"l_paren"}},
				{lhs: "factor", num: 0, dot: 1, symbols: []string{"l_paren", "id"}},
				{lhs: "factor", num: 0, dot: 2, symbols: []string{"r_paren"}},
				{lhs: "factor", num: 1, dot: 0, symbols: []string{"id"}},
			},
		},
		{
			caption: "productions contain the empty start production",
			src: `
#name test;

s
    :
    ;
`,
			first: []first{
				{lhs: "s'", num: 0, dot: 0, symbols: []string{}, empty: true},
				{lhs: "s", num: 0, dot: 0, symbols: []string{}, empty: true},
			},
		},
		{
			caption: "productions contain an empty production",
			src: `
#name test;

s
    : foo bar
    ;
foo
    :
    ;
bar: "bar";
`,
			first: []first{
				{lhs: "s'", num: 0, dot: 0, symbols: []string{"bar"}, empty: false},
				{lhs: "s", num: 0, dot: 0, symbols: []string{"bar"}, empty: false},
				{lhs: "foo", num: 0, dot: 0, symbols: []string{}, empty: true},
			},
		},
		{
			caption: "a start production contains a non-empty alternative and empty alternative",
			src: `
#name test;

s
    : foo
    |
    ;
foo: "foo";
`,
			first: []first{
				{lhs: "s'", num: 0, dot: 0, symbols: []string{"foo"}, empty: true},
				{lhs: "s", num: 0, dot: 0, symbols: []string{"foo"}},
				{lhs: "s", num: 1, dot: 0, symbols: []string{}, empty: true},
			},
		},
		{
			caption: "a production contains non-empty alternative and empty alternative",
			src: `
#name test;

s
    : foo
    ;
foo
    : bar
    |
    ;
bar: "bar";
`,
			first: []first{
				{lhs: "s'", num: 0, dot: 0, symbols: []string{"bar"}, empty: true},
				{lhs: "s", num: 0, dot: 0, symbols: []string{"bar"}, empty: true},
				{lhs: "foo", num: 0, dot: 0, symbols: []string{"bar"}},
				{lhs: "foo", num: 1, dot: 0, symbols: []string{}, empty: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			fst, gram := genActualFirst(t, tt.src)

			for _, ttFirst := range tt.first {
				lhsSym, ok := gram.symbolTable.toSymbol(ttFirst.lhs)
				if !ok {
					t.Fatalf("a symbol was not found; symbol: %v", ttFirst.lhs)
				}

				prod, ok := gram.productionSet.findByLHS(lhsSym)
				if !ok {
					t.Fatalf("a production was not found; LHS: %v (%v)", ttFirst.lhs, lhsSym)
				}

				actualFirst, err := fst.find(prod[ttFirst.num], ttFirst.dot)
				if err != nil {
					t.Fatalf("failed to get a FIRST set; LHS: %v (%v), num: %v, dot: %v, error: %v", ttFirst.lhs, lhsSym, ttFirst.num, ttFirst.dot, err)
				}

				expectedFirst := genExpectedFirstEntry(t, ttFirst.symbols, ttFirst.empty, gram.symbolTable)

				testFirst(t, actualFirst, expectedFirst)
			}
		})
	}
}

func genActualFirst(t *testing.T, src string) (*firstSet, *Grammar) {
	ast, err := spec.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	b := GrammarBuilder{
		AST: ast,
	}
	gram, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	fst, err := genFirstSet(gram.productionSet)
	if err != nil {
		t.Fatal(err)
	}
	if fst == nil {
		t.Fatal("genFiest returned nil without any error")
	}

	return fst, gram
}

func genExpectedFirstEntry(t *testing.T, symbols []string, empty bool, symTab *symbolTable) *firstEntry {
	t.Helper()

	entry := newFirstEntry()
	if empty {
		entry.addEmpty()
	}
	for _, sym := range symbols {
		symSym, ok := symTab.toSymbol(sym)
		if !ok {
			t.Fatalf("a symbol was not found; symbol: %v", sym)
		}
		entry.add(symSym)
	}

	return entry
}

func testFirst(t *testing.T, actual, expected *firstEntry) {
	if actual.empty != expected.empty {
		t.Errorf("empty is mismatched\nwant: %v\ngot: %v", expected.empty, actual.empty)
	}

	if len(actual.symbols) != len(expected.symbols) {
		t.Fatalf("invalid FIRST set\nwant: %+v\ngot: %+v", expected.symbols, actual.symbols)
	}

	for eSym := range expected.symbols {
		if _, ok := actual.symbols[eSym]; !ok {
			t.Fatalf("invalid FIRST set\nwant: %+v\ngot: %+v", expected.symbols, actual.symbols)
		}
	}
}
