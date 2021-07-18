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
	withProdPos := func(prod *ProductionNode, pos Position) *ProductionNode {
		prod.Pos = pos
		return prod
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
	withAltPos := func(alt *AlternativeNode, pos Position) *AlternativeNode {
		alt.Pos = pos
		return alt
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
	withDirPos := func(dir *DirectiveNode, pos Position) *DirectiveNode {
		dir.Pos = pos
		return dir
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
	withParamPos := func(param *ParameterNode, pos Position) *ParameterNode {
		param.Pos = pos
		return param
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
	withTreeChildPos := func(child *TreeChildNode, pos Position) *TreeChildNode {
		child.Pos = pos
		return child
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
	withElemPos := func(elem *ElementNode, pos Position) *ElementNode {
		elem.Pos = pos
		return elem
	}
	frag := func(lhs string, rhs string) *FragmentNode {
		return &FragmentNode{
			LHS: lhs,
			RHS: rhs,
		}
	}
	withFragmentPos := func(frag *FragmentNode, pos Position) *FragmentNode {
		frag.Pos = pos
		return frag
	}

	tests := []struct {
		caption       string
		src           string
		checkPosition bool
		ast           *RootNode
		synErr        *SyntaxError
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
		{
			caption: "an AST has node positions",
			src: `
#mode default
exp
    : exp "\+" id #ast '(exp $1 $2)
    | id
    ;
whitespace: "\u{0020}+" #skip;
id: "\f{letter}(\f{letter}|\f{number})*";
fragment letter: "[A-Za-z_]";
fragment number: "[0-9]";
`,
			checkPosition: true,
			ast: &RootNode{
				Productions: []*ProductionNode{
					withProdPos(
						withProdDir(
							prod("exp",
								withAltPos(
									withAltDir(
										alt(
											withElemPos(id("exp"), newPosition(4)),
											withElemPos(pat(`\+`), newPosition(4)),
											withElemPos(id("id"), newPosition(4)),
										),
										withDirPos(
											dir("ast",
												withParamPos(
													treeParam("exp",
														withTreeChildPos(pos(1), newPosition(4)),
														withTreeChildPos(pos(2), newPosition(4))),
													newPosition(4),
												),
											),
											newPosition(4),
										),
									),
									newPosition(4),
								),
								withAltPos(
									alt(
										withElemPos(id("id"), newPosition(5)),
									),
									newPosition(5),
								),
							),
							withDirPos(
								dir("mode",
									withParamPos(
										idParam("default"),
										newPosition(2),
									),
								),
								newPosition(2),
							),
						),
						newPosition(3),
					),
				},
				LexProductions: []*ProductionNode{
					withProdPos(
						prod("whitespace",
							withAltPos(
								withAltDir(
									alt(
										withElemPos(
											pat(`\u{0020}+`),
											newPosition(7),
										),
									),
									withDirPos(
										dir("skip"),
										newPosition(7),
									),
								),
								newPosition(7),
							),
						),
						newPosition(7),
					),
					withProdPos(
						prod("id",
							withAltPos(
								alt(
									withElemPos(
										pat(`\f{letter}(\f{letter}|\f{number})*`),
										newPosition(8),
									),
								),
								newPosition(8),
							),
						),
						newPosition(8),
					),
				},
				Fragments: []*FragmentNode{
					withFragmentPos(
						frag("letter", "[A-Za-z_]"),
						newPosition(9),
					),
					withFragmentPos(
						frag("number", "[0-9]"),
						newPosition(10),
					),
				},
			},
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
				testRootNode(t, ast, tt.ast, tt.checkPosition)
			}
		})
	}
}

func testRootNode(t *testing.T, root, expected *RootNode, checkPosition bool) {
	t.Helper()
	if len(root.Productions) != len(expected.Productions) {
		t.Fatalf("unexpected length of productions; want: %v, got: %v", len(expected.Productions), len(root.Productions))
	}
	for i, prod := range root.Productions {
		testProductionNode(t, prod, expected.Productions[i], checkPosition)
	}
	for i, prod := range root.LexProductions {
		testProductionNode(t, prod, expected.LexProductions[i], checkPosition)
	}
	for i, frag := range root.Fragments {
		testFragmentNode(t, frag, expected.Fragments[i], checkPosition)
	}
}

func testProductionNode(t *testing.T, prod, expected *ProductionNode, checkPosition bool) {
	t.Helper()
	if expected.Directive == nil && prod.Directive != nil {
		t.Fatalf("unexpected directive; want: nil, got: %+v", prod.Directive)
	}
	if expected.Directive != nil {
		if prod.Directive == nil {
			t.Fatalf("a directive is not set; want: %+v, got: nil", expected.Directive)
		}
		testDirective(t, prod.Directive, expected.Directive, checkPosition)
	}
	if prod.LHS != expected.LHS {
		t.Fatalf("unexpected LHS; want: %v, got: %v", expected.LHS, prod.LHS)
	}
	if len(prod.RHS) != len(expected.RHS) {
		t.Fatalf("unexpected length of an RHS; want: %v, got: %v", len(expected.RHS), len(prod.RHS))
	}
	for i, alt := range prod.RHS {
		testAlternativeNode(t, alt, expected.RHS[i], checkPosition)
	}
	if checkPosition {
		testPosition(t, prod.Pos, expected.Pos)
	}
}

func testFragmentNode(t *testing.T, frag, expected *FragmentNode, checkPosition bool) {
	t.Helper()
	if frag.LHS != expected.LHS {
		t.Fatalf("unexpected LHS; want: %v, got: %v", expected.LHS, frag.LHS)
	}
	if frag.RHS != expected.RHS {
		t.Fatalf("unexpected RHS; want: %v, got: %v", expected.RHS, frag.RHS)
	}
	if checkPosition {
		testPosition(t, frag.Pos, expected.Pos)
	}
}

func testAlternativeNode(t *testing.T, alt, expected *AlternativeNode, checkPosition bool) {
	t.Helper()
	if len(alt.Elements) != len(expected.Elements) {
		t.Fatalf("unexpected length of elements; want: %v, got: %v", len(expected.Elements), len(alt.Elements))
	}
	for i, elem := range alt.Elements {
		testElementNode(t, elem, expected.Elements[i], checkPosition)
	}
	if expected.Directive == nil && alt.Directive != nil {
		t.Fatalf("unexpected directive; want: nil, got: %+v", alt.Directive)
	}
	if expected.Directive != nil {
		if alt.Directive == nil {
			t.Fatalf("a directive is not set; want: %+v, got: nil", expected.Directive)
		}
		testDirective(t, alt.Directive, expected.Directive, checkPosition)
	}
	if checkPosition {
		testPosition(t, alt.Pos, expected.Pos)
	}
}

func testElementNode(t *testing.T, elem, expected *ElementNode, checkPosition bool) {
	t.Helper()
	if elem.ID != expected.ID {
		t.Fatalf("unexpected ID; want: %v, got: %v", expected.ID, elem.ID)
	}
	if elem.Pattern != expected.Pattern {
		t.Fatalf("unexpected pattern; want: %v, got: %v", expected.Pattern, elem.Pattern)
	}
	if checkPosition {
		testPosition(t, elem.Pos, expected.Pos)
	}
}

func testDirective(t *testing.T, dir, expected *DirectiveNode, checkPosition bool) {
	t.Helper()
	if expected.Name != dir.Name {
		t.Fatalf("unexpected directive name; want: %+v, got: %+v", expected.Name, dir.Name)
	}
	if len(expected.Parameters) != len(dir.Parameters) {
		t.Fatalf("unexpected directive parameter; want: %+v, got: %+v", expected.Parameters, dir.Parameters)
	}
	for i, param := range dir.Parameters {
		testParameter(t, param, expected.Parameters[i], checkPosition)
	}
	if checkPosition {
		testPosition(t, dir.Pos, expected.Pos)
	}
}

func testParameter(t *testing.T, param, expected *ParameterNode, checkPosition bool) {
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
			if checkPosition {
				testPosition(t, c.Pos, e.Pos)
			}
		}
	}
	if checkPosition {
		testPosition(t, param.Pos, expected.Pos)
	}
}

func testPosition(t *testing.T, pos, expected Position) {
	t.Helper()
	if pos.row != expected.row {
		t.Fatalf("unexpected position want: %+v, got: %+v", expected, pos)
	}
}
