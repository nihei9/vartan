package driver

import (
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		specSrc string
		src     string
	}{
		{
			specSrc: `
expr
    : expr "\+" term
    | term
    ;
term
    : term "\*" factor
    | factor
    ;
factor
    : "\(" expr "\)"
    | id
    ;
id: "[A-Za-z_][0-9A-Za-z_]*";
`,
			src: `(a+(b+c))*d+e`,
		},
		{
			specSrc: `
mode_tran_seq
    : mode_tran_seq mode_tran
    | mode_tran
    ;
mode_tran
    : push_m1
    | push_m2
    | pop_m1
    | pop_m2
    ;
push_m1: "->" # push m1;
@mode m1
push_m2: "-->" # push m2;
@mode m1
pop_m1 : "<-" # pop;
@mode m2
pop_m2: "<--" # pop;
`,
			src: `->--><--<-`,
		},
		{
			specSrc: `
s
    : foo bar
    ;
foo: "foo";
@mode default
bar: "bar";
`,
			src: `foobar`,
		},
		// The parser can skips specified tokens.
		{
			specSrc: `
s
    : foo bar
    ;
foo: "foo";
bar: "bar";
white_space: "[\u{0009}\u{0020}]+" # skip;
`,
			src: `foo bar`,
		},
	}
	for _, tt := range tests {
		ast, err := spec.Parse(strings.NewReader(tt.specSrc))
		if err != nil {
			t.Fatal(err)
		}
		g, err := grammar.NewGrammar(ast)
		if err != nil {
			t.Fatal(err)
		}
		gram, err := grammar.Compile(g)
		if err != nil {
			t.Fatal(err)
		}
		p, err := NewParser(gram, strings.NewReader(tt.src))
		if err != nil {
			t.Fatal(err)
		}
		err = p.Parse()
		if err != nil {
			t.Fatal(err)
		}

		printCST(p.GetCST(), 0)
	}
}
