package driver

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec"
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
%name test

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
%name test

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
%name test

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
%name test

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

			gram, err := grammar.Compile(g, grammar.SpecifyClass(grammar.ClassLALR))
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
