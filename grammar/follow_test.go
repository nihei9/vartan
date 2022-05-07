package grammar

import (
	"strings"
	"testing"

	"github.com/nihei9/vartan/spec"
)

type follow struct {
	nonTermText string
	symbols     []string
	eof         bool
}

func TestFollowSet(t *testing.T) {
	tests := []struct {
		caption string
		src     string
		follow  []follow
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
			follow: []follow{
				{nonTermText: "expr'", symbols: []string{}, eof: true},
				{nonTermText: "expr", symbols: []string{"add", "r_paren"}, eof: true},
				{nonTermText: "term", symbols: []string{"add", "mul", "r_paren"}, eof: true},
				{nonTermText: "factor", symbols: []string{"add", "mul", "r_paren"}, eof: true},
			},
		},
		{
			caption: "productions contain an empty start production",
			src: `
#name test;

s
    :
    ;
`,
			follow: []follow{
				{nonTermText: "s'", symbols: []string{}, eof: true},
				{nonTermText: "s", symbols: []string{}, eof: true},
			},
		},
		{
			caption: "productions contain an empty production",
			src: `
#name test;

s
    : foo
    ;
foo
    :
;
`,
			follow: []follow{
				{nonTermText: "s'", symbols: []string{}, eof: true},
				{nonTermText: "s", symbols: []string{}, eof: true},
				{nonTermText: "foo", symbols: []string{}, eof: true},
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
			follow: []follow{
				{nonTermText: "s'", symbols: []string{}, eof: true},
				{nonTermText: "s", symbols: []string{}, eof: true},
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
			follow: []follow{
				{nonTermText: "s'", symbols: []string{}, eof: true},
				{nonTermText: "s", symbols: []string{}, eof: true},
				{nonTermText: "foo", symbols: []string{}, eof: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			flw, gram := genActualFollow(t, tt.src)

			for _, ttFollow := range tt.follow {
				sym, ok := gram.symbolTable.toSymbol(ttFollow.nonTermText)
				if !ok {
					t.Fatalf("a symbol '%v' was not found", ttFollow.nonTermText)
				}

				actualFollow, err := flw.find(sym)
				if err != nil {
					t.Fatalf("failed to get a FOLLOW entry; non-terminal symbol: %v (%v), error: %v", ttFollow.nonTermText, sym, err)
				}

				expectedFollow := genExpectedFollowEntry(t, ttFollow.symbols, ttFollow.eof, gram.symbolTable)

				testFollow(t, actualFollow, expectedFollow)
			}
		})
	}
}

func genActualFollow(t *testing.T, src string) (*followSet, *Grammar) {
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
	flw, err := genFollowSet(gram.productionSet, fst)
	if err != nil {
		t.Fatal(err)
	}
	if flw == nil {
		t.Fatal("genFollow returned nil without any error")
	}

	return flw, gram
}

func genExpectedFollowEntry(t *testing.T, symbols []string, eof bool, symTab *symbolTable) *followEntry {
	t.Helper()

	entry := newFollowEntry()
	if eof {
		entry.addEOF()
	}
	for _, sym := range symbols {
		symID, _ := symTab.toSymbol(sym)
		if symID.isNil() {
			t.Fatalf("a symbol '%v' was not found", sym)
		}

		entry.add(symID)
	}

	return entry
}

func testFollow(t *testing.T, actual, expected *followEntry) {
	if actual.eof != expected.eof {
		t.Errorf("eof is mismatched; want: %v, got: %v", expected.eof, actual.eof)
	}

	if len(actual.symbols) != len(expected.symbols) {
		t.Fatalf("unexpected symbol count of a FOLLOW entry; want: %v, got: %v", expected.symbols, actual.symbols)
	}

	for eSym := range expected.symbols {
		if _, ok := actual.symbols[eSym]; !ok {
			t.Fatalf("invalid FOLLOW entry; want: %v, got: %v", expected.symbols, actual.symbols)
		}
	}
}
