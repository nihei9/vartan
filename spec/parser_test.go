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
	withModifier := func(prod *ProductionNode, mod *ModifierNode) *ProductionNode {
		prod.Modifier = mod
		return prod
	}
	modifier := func(name string, param string) *ModifierNode {
		return &ModifierNode{
			Name:      name,
			Parameter: param,
		}
	}
	alternative := func(elems ...*ElementNode) *AlternativeNode {
		return &AlternativeNode{
			Elements: elems,
		}
	}
	withAction := func(alt *AlternativeNode, act *ActionNode) *AlternativeNode {
		alt.Action = act
		return alt
	}
	action := func(name string, param *ParameterNode) *ActionNode {
		return &ActionNode{
			Name:      name,
			Parameter: param,
		}
	}
	idParameter := func(id string) *ParameterNode {
		return &ParameterNode{
			ID: id,
		}
	}
	treeParameter := func(name string, children ...*TreeChildNode) *ParameterNode {
		return &ParameterNode{
			Tree: &TreeStructNode{
				Name:     name,
				Children: children,
			},
		}
	}
	positionChild := func(pos int) *TreeChildNode {
		return &TreeChildNode{
			Position: pos,
		}
	}
	expandChildren := func(c *TreeChildNode) *TreeChildNode {
		c.Expansion = true
		return c
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
	fragment := func(lhs string, rhs string) *FragmentNode {
		return &FragmentNode{
			LHS: lhs,
			RHS: rhs,
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
		{
			caption: "a grammar can contain fragments",
			src: `
s
    : tagline
    ;
tagline: "\f{words} IS OUT THERE.";
fragment words: "[A-Za-z\u{0020}]+";
`,
			ast: &RootNode{
				Productions: []*ProductionNode{
					production("s",
						alternative(id("tagline")),
					),
					production("tagline",
						alternative(pattern(`\f{words} IS OUT THERE.`)),
					),
				},
				Fragments: []*FragmentNode{
					fragment("words", `[A-Za-z\u{0020}]+`),
				},
			},
		},
		{
			caption: "a grammar can contain production modifiers and semantic actions",
			src: `
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
			ast: &RootNode{
				Productions: []*ProductionNode{
					production("mode_tran_seq",
						alternative(id("mode_tran_seq"), id("mode_tran")),
						alternative(id("mode_tran")),
					),
					production("mode_tran",
						alternative(id("push_m1")),
						alternative(id("push_m2")),
						alternative(id("pop_m1")),
						alternative(id("pop_m2")),
					),
					production("push_m1",
						withAction(
							alternative(pattern(`->`)),
							action("push", idParameter("m1")),
						),
					),
					withModifier(
						production("push_m2",
							withAction(
								alternative(pattern(`-->`)),
								action("push", idParameter("m2")),
							),
						),
						modifier("mode", "m1"),
					),
					withModifier(
						production("pop_m1",
							withAction(
								alternative(pattern(`<-`)),
								action("pop", nil),
							),
						),
						modifier("mode", "m1"),
					),
					withModifier(
						production("pop_m2",
							withAction(
								alternative(pattern(`<--`)),
								action("pop", nil),
							),
						),
						modifier("mode", "m2"),
					),
				},
			},
		},
		{
			caption: "a grammar can contain 'ast' actions",
			src: `
s
    : foo bar_list # ast '(s $1 $2)
    ;
bar_list
    : bar_list bar # ast '(bar_list $1... $2)
    | bar          # ast '(bar_list $1)
    ;
foo: "foo";
bar: "bar";
`,
			ast: &RootNode{
				Productions: []*ProductionNode{
					production("s",
						withAction(
							alternative(id("foo"), id("bar_list")),
							action("ast", treeParameter("s", positionChild(1), positionChild(2))),
						),
					),
					production("bar_list",
						withAction(
							alternative(id("bar_list"), id("bar")),
							action("ast", treeParameter("bar_list", expandChildren(positionChild(1)), positionChild(2))),
						),
						withAction(
							alternative(id("bar")),
							action("ast", treeParameter("bar_list", positionChild(1))),
						),
					),
					production("foo",
						alternative(pattern("foo")),
					),
					production("bar",
						alternative(pattern("bar")),
					),
				},
			},
		},
		{
			caption: "the first element of a tree structure must be an ID",
			src: `
s
    : foo # ast '($1)
    ;
foo: "foo";
`,
			synErr: synErrTreeInvalidFirstElem,
		},
		{
			caption: "a tree structure must be closed by ')'",
			src: `
s
    : foo # ast '(s $1
    ;
foo: "foo";
`,
			synErr: synErrTreeUnclosed,
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
	if expected.Modifier == nil && prod.Modifier != nil {
		t.Fatalf("unexpected modifier; want: nil, got: %+v", prod.Modifier)
	}
	if expected.Modifier != nil {
		if prod.Modifier == nil {
			t.Fatalf("a modifier is not set; want: %+v, got: nil", expected.Modifier)
		}
		if expected.Modifier.Name != prod.Modifier.Name || expected.Modifier.Parameter != prod.Modifier.Parameter {
			t.Fatalf("unexpected modifier; want: %+v, got: %+v", expected.Modifier, prod.Modifier)
		}
	}
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
	if expected.Action == nil && alt.Action != nil {
		t.Fatalf("unexpected action; want: nil, got: %+v", alt.Action)
	}
	if expected.Action != nil {
		if alt.Action == nil {
			t.Fatalf("an action is not set; want: %+v, got: nil", expected.Action)
		}
		testAction(t, alt.Action, expected.Action)
	}
}

func testElementNode(t *testing.T, elem, expected *ElementNode) {
	t.Helper()
	if elem.Pattern != expected.Pattern {
		t.Fatalf("unexpected pattern; want: %v, got: %v", expected.Pattern, elem.Pattern)
	}
}

func testAction(t *testing.T, act, expected *ActionNode) {
	t.Helper()
	if expected.Name != act.Name {
		t.Fatalf("unexpected action name; want: %+v, got: %+v", expected.Name, act.Name)
	}
	if expected.Parameter == nil && act.Parameter != nil {
		t.Fatalf("unexpected action parameter; want: nil, got: %+v", act.Parameter)
	}
	if expected.Parameter != nil {
		if act.Parameter == nil {
			t.Fatalf("unexpected action parameter; want: %+v, got: nil", expected.Parameter)
		}
		testParameter(t, act.Parameter, expected.Parameter)
	}
}

func testParameter(t *testing.T, param, expected *ParameterNode) {
	t.Helper()
	if param.ID != expected.ID {
		t.Fatalf("unexpected ID parameter; want: %v, got: %v", expected.ID, param.ID)
	}
	if expected.Tree == nil && param.Tree != nil {
		t.Fatalf("unexpected tree parameter; want: nil, got: %+v", param.Tree)
	}
	if expected.Tree != nil {
		if param.Tree == nil {
			t.Fatalf("unexpected tree parameter; want: %+v, got: nil", expected.Tree)
		}
		if param.Tree.Name != expected.Tree.Name {
			t.Fatalf("unexpected node name; want: %v, got: %v", expected.Tree.Name, param.Tree.Name)
		}
		if len(param.Tree.Children) != len(expected.Tree.Children) {
			t.Fatalf("unexpected children; want: %v, got: %v", expected.Tree.Children, param.Tree.Children)
		}
		for i, c := range param.Tree.Children {
			e := expected.Tree.Children[i]
			if c.Position != e.Position || c.Expansion != e.Expansion {
				t.Fatalf("unexpected child; want: %+v, got: %+v", e, c)
			}
		}
	}
}
