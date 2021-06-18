package driver

import (
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec"
)

func TestParser_Parse(t *testing.T) {
	specSrc := `
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
`
	ast, err := spec.Parse(strings.NewReader(specSrc))
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
	src := `(a+(b+c))*d+e`
	p, err := NewParser(gram, strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	err = p.Parse()
	if err != nil {
		t.Fatal(err)
	}

	printCST(p.GetCST(), 0)
}
