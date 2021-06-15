package spec

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	production := func(lhs string, alts ...*AlternativeNode) *ProductionNode {
		return &ProductionNode{
			LHS: lhs,
			RHS: alts,
		}
	}
	alternative := func(elems ...*ElementNode) *AlternativeNode {
		return &AlternativeNode{
			Elements: elems,
		}
	}
	id := func(id string) *ElementNode {
		return &ElementNode{
			ID: id,
		}
	}
	pattern := func(p string) *ElementNode {
		return &ElementNode{
			Pattern: p,
		}
	}

	tests := []struct {
		caption string
		src     string
		ast     *RootNode
		synErr  *SyntaxError
	}{
		{
			caption: "single production is a valid grammar",
			src:     `a: "a";`,
			ast: &RootNode{
				Productions: []*ProductionNode{
					production("a", alternative(pattern("a"))),
				},
			},
		},
		{
			caption: "multiple productions are a valid grammar",
			src: `
e: e "\+|-" t | t;
t: t "\*|/" f | f;
f: "\(" e ")" | id;
id: "[A-Za-z_][0-9A-Za-z_]*";
`,
			ast: &RootNode{
				Productions: []*ProductionNode{
					production("e",
						alternative(id("e"), pattern(`\+|-`), id("t")),
						alternative(id("t")),
					),
					production("t",
						alternative(id("t"), pattern(`\*|/`), id("f")),
						alternative(id("f")),
					),
					production("f",
						alternative(pattern(`\(`), id("e"), pattern(`)`)),
						alternative(id("id")),
					),
					production("id",
						alternative(pattern(`[A-Za-z_][0-9A-Za-z_]*`)),
					),
				},
			},
		},
		{
			caption: "productions can contain the empty alternative",
			src: `
a: "foo" | ;
b: | "bar";
c: ;
`,
			ast: &RootNode{
				Productions: []*ProductionNode{
					production("a",
						alternative(pattern(`foo`)),
						alternative(),
					),
					production("b",
						alternative(),
						alternative(pattern(`bar`)),
					),
					production("c",
						alternative(),
					),
				},
			},
		},
		{
			caption: "when a source contains an unknown token, the parser raises a syntax error",
			src:     `a: !;`,
			synErr:  synErrInvalidToken,
		},
		{
			caption: "a grammar must have at least one production",
			src:     ``,
			synErr:  synErrNoProduction,
		},
		{
			caption: "a production must have its name as the first element",
			src:     `: "a";`,
			synErr:  synErrNoProductionName,
		},
		{
			caption: "':' must precede an alternative",
			src:     `a "a";`,
			synErr:  synErrNoColon,
		},
		{
			caption: "';' must follow a production",
			src:     `a: "a"`,
			synErr:  synErrNoSemicolon,
		},
		{
			caption: "';' can only appear at the end of a production",
			src:     `;`,
			synErr:  synErrNoProductionName,
		},
	}
	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			ast, err := Parse(strings.NewReader(tt.src))
			if tt.synErr != nil {
				if tt.synErr != err {
					t.Fatalf("unexpected error; want: %v, got: %v", tt.synErr, err)
				}
				if ast != nil {
					t.Fatalf("AST must be nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if ast == nil {
					t.Fatalf("AST must be non-nil")
				}
				testRootNode(t, ast, tt.ast)
			}
		})
	}
}

func testRootNode(t *testing.T, root, expected *RootNode) {
	t.Helper()
	if len(root.Productions) != len(expected.Productions) {
		t.Fatalf("unexpected length of productions; want: %v, got: %v", len(expected.Productions), len(root.Productions))
	}
	for i, prod := range root.Productions {
		testProductionNode(t, prod, expected.Productions[i])
	}
}

func testProductionNode(t *testing.T, prod, expected *ProductionNode) {
	t.Helper()
	if prod.LHS != expected.LHS {
		t.Fatalf("unexpected LHS; want: %v, got: %v", expected.LHS, prod.LHS)
	}
	if len(prod.RHS) != len(expected.RHS) {
		t.Fatalf("unexpected length of an RHS; want: %v, got: %v", len(expected.RHS), len(prod.RHS))
	}
	for i, alt := range prod.RHS {
		testAlternativeNode(t, alt, expected.RHS[i])
	}
}

func testAlternativeNode(t *testing.T, alt, expected *AlternativeNode) {
	t.Helper()
	if len(alt.Elements) != len(expected.Elements) {
		t.Fatalf("unexpected length of elements; want: %v, got: %v", len(expected.Elements), len(alt.Elements))
	}
	for i, elem := range alt.Elements {
		testElementNode(t, elem, expected.Elements[i])
	}
}

func testElementNode(t *testing.T, elem, expected *ElementNode) {
	t.Helper()
	if elem.Pattern != expected.Pattern {
		t.Fatalf("unexpected pattern; want: %v, got: %v", expected.Pattern, elem.Pattern)
	}
}
