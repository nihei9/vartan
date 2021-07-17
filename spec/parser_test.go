package spec

import (
	"strings"
	"testing"

	verr "github.com/nihei9/vartan/error"
)

func TestParse(t *testing.T) {
	prod := func(lhs string, alts ...*AlternativeNode) *ProductionNode {
		return &ProductionNode{
			LHS: lhs,
			RHS: alts,
		}
	}
	withProdDir := func(prod *ProductionNode, dir *DirectiveNode) *ProductionNode {
		prod.Directive = dir
		return prod
	}
	alt := func(elems ...*ElementNode) *AlternativeNode {
		return &AlternativeNode{
			Elements: elems,
		}
	}
	withAltDir := func(alt *AlternativeNode, dir *DirectiveNode) *AlternativeNode {
		alt.Directive = dir
		return alt
	}
	dir := func(name string, params ...*ParameterNode) *DirectiveNode {
		return &DirectiveNode{
			Name:       name,
			Parameters: params,
		}
	}
	idParam := func(id string) *ParameterNode {
		return &ParameterNode{
			ID: id,
		}
	}
	treeParam := func(name string, children ...*TreeChildNode) *ParameterNode {
		return &ParameterNode{
			Tree: &TreeStructNode{
				Name:     name,
				Children: children,
			},
		}
	}
	pos := func(pos int) *TreeChildNode {
		return &TreeChildNode{
			Position: pos,
		}
	}
	exp := func(c *TreeChildNode) *TreeChildNode {
		c.Expansion = true
		return c
	}
	id := func(id string) *ElementNode {
		return &ElementNode{
			ID: id,
		}
	}
	pat := func(p string) *ElementNode {
		return &ElementNode{
			Pattern: p,
		}
	}
	frag := func(lhs string, rhs string) *FragmentNode {
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
				LexProductions: []*ProductionNode{
					prod("a", alt(pat("a"))),
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
					prod("e",
						alt(id("e"), pat(`\+|-`), id("t")),
						alt(id("t")),
					),
					prod("t",
						alt(id("t"), pat(`\*|/`), id("f")),
						alt(id("f")),
					),
					prod("f",
						alt(pat(`\(`), id("e"), pat(`)`)),
						alt(id("id")),
					),
				},
				LexProductions: []*ProductionNode{
					prod("id",
						alt(pat(`[A-Za-z_][0-9A-Za-z_]*`)),
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
					prod("a",
						alt(pat(`foo`)),
						alt(),
					),
					prod("b",
						alt(),
						alt(pat(`bar`)),
					),
					prod("c",
						alt(),
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
					prod("s",
						alt(id("tagline")),
					),
				},
				LexProductions: []*ProductionNode{
					prod("tagline",
						alt(pat(`\f{words} IS OUT THERE.`)),
					),
				},
				Fragments: []*FragmentNode{
					frag("words", `[A-Za-z\u{0020}]+`),
				},
			},
		},
		{
			caption: "a grammar can contain production directives and alternative directives",
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
push_m1: "->" #push m1;
#mode m1
push_m2: "-->" #push m2;
#mode m1
pop_m1 : "<-" #pop;
#mode m2
pop_m2: "<--" #pop;
#mode default m1 m2
whitespace: "\u{0020}+" #skip;
`,
			ast: &RootNode{
				Productions: []*ProductionNode{
					prod("mode_tran_seq",
						alt(id("mode_tran_seq"), id("mode_tran")),
						alt(id("mode_tran")),
					),
					prod("mode_tran",
						alt(id("push_m1")),
						alt(id("push_m2")),
						alt(id("pop_m1")),
						alt(id("pop_m2")),
					),
				},
				LexProductions: []*ProductionNode{
					prod("push_m1",
						withAltDir(
							alt(pat(`->`)),
							dir("push", idParam("m1")),
						),
					),
					withProdDir(
						prod("push_m2",
							withAltDir(
								alt(pat(`-->`)),
								dir("push", idParam("m2")),
							),
						),
						dir("mode", idParam("m1")),
					),
					withProdDir(
						prod("pop_m1",
							withAltDir(
								alt(pat(`<-`)),
								dir("pop"),
							),
						),
						dir("mode", idParam("m1")),
					),
					withProdDir(
						prod("pop_m2",
							withAltDir(
								alt(pat(`<--`)),
								dir("pop"),
							),
						),
						dir("mode", idParam("m2")),
					),
					withProdDir(
						prod("whitespace",
							withAltDir(
								alt(pat(`\u{0020}+`)),
								dir("skip"),
							),
						),
						dir("mode", idParam("default"), idParam("m1"), idParam("m2")),
					),
				},
			},
		},
		{
			caption: "a production directive must be followed by a newline",
			src: `
#mode foo;
`,
			synErr: synErrProdDirNoNewline,
		},
		{
			caption: "a production must be followed by a newline",
			src: `
s: foo; foo: "foo";
`,
			synErr: synErrSemicolonNoNewline,
		},
		{
			caption: "a grammar can contain 'ast' directives",
			src: `
s
    : foo bar_list #ast '(s $1 $2)
    ;
bar_list
    : bar_list bar #ast '(bar_list $1... $2)
    | bar          #ast '(bar_list $1)
    ;
foo: "foo";
bar: "bar";
`,
			ast: &RootNode{
				Productions: []*ProductionNode{
					prod("s",
						withAltDir(
							alt(id("foo"), id("bar_list")),
							dir("ast", treeParam("s", pos(1), pos(2))),
						),
					),
					prod("bar_list",
						withAltDir(
							alt(id("bar_list"), id("bar")),
							dir("ast", treeParam("bar_list", exp(pos(1)), pos(2))),
						),
						withAltDir(
							alt(id("bar")),
							dir("ast", treeParam("bar_list", pos(1))),
						),
					),
				},
				LexProductions: []*ProductionNode{
					prod("foo",
						alt(pat("foo")),
					),
					prod("bar",
						alt(pat("bar")),
					),
				},
			},
		},
		{
			caption: "the first element of a tree structure must be an ID",
			src: `
s
    : foo #ast '($1)
    ;
foo: "foo";
`,
			synErr: synErrTreeInvalidFirstElem,
		},
		{
			caption: "a tree structure must be closed by ')'",
			src: `
s
    : foo #ast '(s $1
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
				synErrs, ok := err.(verr.SpecErrors)
				if !ok {
					t.Fatalf("unexpected error; want: %v, got: %v", tt.synErr, err)
				}
				synErr := synErrs[0]
				if tt.synErr != synErr.Cause {
					t.Fatalf("unexpected error; want: %v, got: %v", tt.synErr, synErr.Cause)
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
	for i, prod := range root.LexProductions {
		testProductionNode(t, prod, expected.LexProductions[i])
	}
}

func testProductionNode(t *testing.T, prod, expected *ProductionNode) {
	t.Helper()
	if expected.Directive == nil && prod.Directive != nil {
		t.Fatalf("unexpected directive; want: nil, got: %+v", prod.Directive)
	}
	if expected.Directive != nil {
		if prod.Directive == nil {
			t.Fatalf("a directive is not set; want: %+v, got: nil", expected.Directive)
		}
		testDirective(t, prod.Directive, expected.Directive)
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
	if expected.Directive == nil && alt.Directive != nil {
		t.Fatalf("unexpected directive; want: nil, got: %+v", alt.Directive)
	}
	if expected.Directive != nil {
		if alt.Directive == nil {
			t.Fatalf("a directive is not set; want: %+v, got: nil", expected.Directive)
		}
		testDirective(t, alt.Directive, expected.Directive)
	}
}

func testElementNode(t *testing.T, elem, expected *ElementNode) {
	t.Helper()
	if elem.Pattern != expected.Pattern {
		t.Fatalf("unexpected pattern; want: %v, got: %v", expected.Pattern, elem.Pattern)
	}
}

func testDirective(t *testing.T, dir, expected *DirectiveNode) {
	t.Helper()
	if expected.Name != dir.Name {
		t.Fatalf("unexpected directive name; want: %+v, got: %+v", expected.Name, dir.Name)
	}
	if len(expected.Parameters) != len(dir.Parameters) {
		t.Fatalf("unexpected directive parameter; want: %+v, got: %+v", expected.Parameters, dir.Parameters)
	}
	for i, param := range dir.Parameters {
		testParameter(t, param, expected.Parameters[i])
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
