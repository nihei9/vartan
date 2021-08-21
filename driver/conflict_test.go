package driver

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec"
)

func TestParserWithConflicts(t *testing.T) {
	tests := []struct {
		caption string
		specSrc string
		src     string
		cst     *Node
	}{
		{
			caption: "when a shift/reduce conflict occurred, we prioritize the shift action",
			specSrc: `
expr
    : expr assign expr
	| id
	;

id: "[A-Za-z0-9_]+";
assign: '=';
`,
			src: `foo=bar=baz`,
			cst: nonTermNode("expr",
				nonTermNode("expr",
					termNode("id", "foo"),
				),
				termNode("assign", "="),
				nonTermNode("expr",
					nonTermNode("expr",
						termNode("id", "bar"),
					),
					termNode("assign", "="),
					nonTermNode("expr",
						termNode("id", "baz"),
					),
				),
			),
		},
		{
			caption: "when a reduce/reduce conflict occurred, we prioritize the production defined earlier in the grammar",
			specSrc: `
s
    : a
	| b
	;
a
    : id
	;
b
    : id
	;

id: "[A-Za-z0-9_]+";
`,
			src: `foo`,
			cst: nonTermNode("s",
				nonTermNode("a",
					termNode("id", "foo"),
				),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
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

			gram, err := grammar.Compile(g, grammar.SpecifyClass(grammar.ClassSLR))
			if err != nil {
				t.Fatal(err)
			}

			p, err := NewParser(gram, strings.NewReader(tt.src), MakeCST())
			if err != nil {
				t.Fatal(err)
			}

			err = p.Parse()
			if err != nil {
				t.Fatal(err)
			}

			fmt.Printf("CST:\n")
			PrintTree(os.Stdout, p.CST())

			testTree(t, p.CST(), tt.cst)
		})
	}
}
