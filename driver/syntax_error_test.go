package driver

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar"
	spec "github.com/nihei9/vartan/spec/grammar"
)

func TestParserWithSyntaxErrors(t *testing.T) {
	tests := []struct {
		caption     string
		specSrc     string
		src         string
		synErrCount int
	}{
		{
			caption: "the parser can report a syntax error",
			specSrc: `
#name test;

s
    : foo
    ;

foo: 'foo';
`,
			src:         `bar`,
			synErrCount: 1,
		},
		{
			caption: "when the parser reduced a production having the reduce directive, the parser will recover from an error state",
			specSrc: `
#name test;

seq
    : seq elem ';'
	| elem ';'
	| error ';' #recover
	;
elem
    : a b c
	;

ws #skip
    : "[\u{0009}\u{0020}]+";
a
    : 'a';
b
    : 'b';
c
    : 'c';
`,
			src:         `!; a!; ab!;`,
			synErrCount: 3,
		},
		{
			caption: "After the parser shifts the error symbol, symbols are ignored until a symbol the parser can perform shift appears",
			specSrc: `
#name test;

seq
    : seq elem ';'
	| elem ';'
	| error ';' #recover
	;
elem
    : a b c
	;

ws #skip
    : "[\u{0009}\u{0020}]+";
a
    : 'a';
b
    : 'b';
c
    : 'c';
`,
			// After the parser trasits to the error state reading the first invalid symbol ('!'),
			// the second and third invalid symbols ('!') are ignored.
			src:         `! ! !; a!; ab!;`,
			synErrCount: 3,
		},
		{
			caption: "when the parser performs shift three times, the parser recovers from the error state",
			specSrc: `
#name test;

seq
    : seq elem ';'
	| elem ';'
	| error '*' '*' ';'
	;
elem
    : a b c
	;

ws #skip
    : "[\u{0009}\u{0020}]+";
a
    : 'a';
b
    : 'b';
c
    : 'c';
`,
			src:         `!**; a!**; ab!**; abc!`,
			synErrCount: 4,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			ast, err := spec.Parse(strings.NewReader(tt.specSrc))
			if err != nil {
				t.Fatal(err)
			}

			b := grammar.GrammarBuilder{
				AST: ast,
			}
			g, err := b.Build()
			if err != nil {
				t.Fatal(err)
			}

			gram, _, err := grammar.Compile(g)
			if err != nil {
				t.Fatal(err)
			}

			toks, err := NewTokenStream(gram, strings.NewReader(tt.src))
			if err != nil {
				t.Fatal(err)
			}

			p, err := NewParser(toks, NewGrammar(gram))
			if err != nil {
				t.Fatal(err)
			}

			err = p.Parse()
			if err != nil {
				t.Fatal(err)
			}

			synErrs := p.SyntaxErrors()
			if len(synErrs) != tt.synErrCount {
				t.Fatalf("unexpected syntax error; want: %v error(s), got: %v error(s)", tt.synErrCount, len(synErrs))
			}
		})
	}
}

func TestParserWithSyntaxErrorAndExpectedLookahead(t *testing.T) {
	tests := []struct {
		caption  string
		specSrc  string
		src      string
		cause    string
		expected []string
	}{
		{
			caption: "the parser reports an expected lookahead symbol",
			specSrc: `
#name test;

s
    : foo
    ;

foo
    : 'foo';
`,
			src:   `bar`,
			cause: `bar`,
			expected: []string{
				"foo",
			},
		},
		{
			caption: "the parser reports expected lookahead symbols",
			specSrc: `
#name test;

s
    : foo
    | bar
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			src:   `baz`,
			cause: `baz`,
			expected: []string{
				"foo",
				"bar",
			},
		},
		{
			caption: "the parser may report the EOF as an expected lookahead symbol",
			specSrc: `
#name test;

s
    : foo
    ;

foo
    : 'foo';
`,
			src:   `foobar`,
			cause: `bar`,
			expected: []string{
				"<eof>",
			},
		},
		{
			caption: "the parser may report the EOF and others as expected lookahead symbols",
			specSrc: `
#name test;

s
    : foo
    |
    ;

foo
    : 'foo';
`,
			src:   `bar`,
			cause: `bar`,
			expected: []string{
				"foo",
				"<eof>",
			},
		},
		{
			caption: "when an anonymous symbol is expected, an expected symbol list contains an alias of the anonymous symbol",
			specSrc: `
#name test;

s
    : foo 'bar'
    ;

foo
    : 'foo';
`,
			src:   `foobaz`,
			cause: `baz`,
			expected: []string{
				"bar",
			},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			ast, err := spec.Parse(strings.NewReader(tt.specSrc))
			if err != nil {
				t.Fatal(err)
			}

			b := grammar.GrammarBuilder{
				AST: ast,
			}
			g, err := b.Build()
			if err != nil {
				t.Fatal(err)
			}

			gram, _, err := grammar.Compile(g)
			if err != nil {
				t.Fatal(err)
			}

			toks, err := NewTokenStream(gram, strings.NewReader(tt.src))
			if err != nil {
				t.Fatal(err)
			}

			p, err := NewParser(toks, NewGrammar(gram))
			if err != nil {
				t.Fatal(err)
			}

			err = p.Parse()
			if err != nil {
				t.Fatal(err)
			}

			synErrs := p.SyntaxErrors()
			if synErrs == nil {
				t.Fatalf("expected one syntax error, but it didn't occur")
			}
			if len(synErrs) != 1 {
				t.Fatalf("too many syntax errors: %v errors", len(synErrs))
			}
			synErr := synErrs[0]
			if string(synErr.Token.Lexeme()) != tt.cause {
				t.Fatalf("unexpected lexeme: want: %v, got: %v", tt.cause, string(synErr.Token.Lexeme()))
			}
			if len(synErr.ExpectedTerminals) != len(tt.expected) {
				t.Fatalf("unexpected lookahead symbols: want: %v, got: %v", tt.expected, synErr.ExpectedTerminals)
			}
			sort.Slice(tt.expected, func(i, j int) bool {
				return tt.expected[i] < tt.expected[j]
			})
			sort.Slice(synErr.ExpectedTerminals, func(i, j int) bool {
				return synErr.ExpectedTerminals[i] < synErr.ExpectedTerminals[j]
			})
			for i, e := range tt.expected {
				if synErr.ExpectedTerminals[i] != e {
					t.Errorf("unexpected lookahead symbol: want: %v, got: %v", e, synErr.ExpectedTerminals[i])
				}
			}
		})
	}
}
