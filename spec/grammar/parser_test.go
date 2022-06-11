package grammar

import (
	"strings"
	"testing"

	verr "github.com/nihei9/vartan/error"
)

func TestParse(t *testing.T) {
	name := func(param *ParameterNode) *DirectiveNode {
		return &DirectiveNode{
			Name:       "name",
			Parameters: []*ParameterNode{param},
		}
	}
	prec := func(param *ParameterNode) *DirectiveNode {
		return &DirectiveNode{
			Name:       "prec",
			Parameters: []*ParameterNode{param},
		}
	}
	leftAssoc := func(params ...*ParameterNode) *DirectiveNode {
		return &DirectiveNode{
			Name:       "left",
			Parameters: params,
		}
	}
	rightAssoc := func(params ...*ParameterNode) *DirectiveNode {
		return &DirectiveNode{
			Name:       "right",
			Parameters: params,
		}
	}
	assign := func(params ...*ParameterNode) *DirectiveNode {
		return &DirectiveNode{
			Name:       "assign",
			Parameters: params,
		}
	}
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
	withProdDir := func(prod *ProductionNode, dirs ...*DirectiveNode) *ProductionNode {
		prod.Directives = dirs
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
	withAltDir := func(alt *AlternativeNode, dirs ...*DirectiveNode) *AlternativeNode {
		alt.Directives = dirs
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
	ordSymParam := func(id string) *ParameterNode {
		return &ParameterNode{
			OrderedSymbol: id,
		}
	}
	exp := func(param *ParameterNode) *ParameterNode {
		param.Expansion = true
		return param
	}
	group := func(dirs ...*DirectiveNode) *ParameterNode {
		return &ParameterNode{
			Group: dirs,
		}
	}
	withParamPos := func(param *ParameterNode, pos Position) *ParameterNode {
		param.Pos = pos
		return param
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
	label := func(name string) *LabelNode {
		return &LabelNode{
			Name: name,
		}
	}
	withLabelPos := func(label *LabelNode, pos Position) *LabelNode {
		label.Pos = pos
		return label
	}
	withLabel := func(elem *ElementNode, label *LabelNode) *ElementNode {
		elem.Label = label
		return elem
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
	newPos := func(row int) Position {
		return Position{
			Row: row,
			Col: 0,
		}
	}

	tests := []struct {
		caption       string
		src           string
		checkPosition bool
		ast           *RootNode
		synErr        *SyntaxError
	}{
		{
			caption: "a grammar can contain top-level directives",
			src: `
#name test;

#prec (
    #left a b $x1
    #right c d $x2
    #assign e f $x3
);
`,
			ast: &RootNode{
				Directives: []*DirectiveNode{
					withDirPos(
						name(
							withParamPos(
								idParam("test"),
								newPos(2),
							),
						),
						newPos(2),
					),
					withDirPos(
						prec(
							withParamPos(
								group(
									withDirPos(
										leftAssoc(
											withParamPos(
												idParam("a"),
												newPos(5),
											),
											withParamPos(
												idParam("b"),
												newPos(5),
											),
											withParamPos(
												ordSymParam("x1"),
												newPos(5),
											),
										),
										newPos(5),
									),
									withDirPos(
										rightAssoc(
											withParamPos(
												idParam("c"),
												newPos(6),
											),
											withParamPos(
												idParam("d"),
												newPos(6),
											),
											withParamPos(
												ordSymParam("x2"),
												newPos(6),
											),
										),
										newPos(6),
									),
									withDirPos(
										assign(
											withParamPos(
												idParam("e"),
												newPos(7),
											),
											withParamPos(
												idParam("f"),
												newPos(7),
											),
											withParamPos(
												ordSymParam("x3"),
												newPos(7),
											),
										),
										newPos(7),
									),
								),
								newPos(4),
							),
						),
						newPos(4),
					),
				},
			},
		},
		{
			caption: "a top-level directive must be followed by ';'",
			src: `
#name test
`,
			synErr: synErrTopLevelDirNoSemicolon,
		},
		{
			caption: "a directive group must be closed by ')'",
			src: `
#prec (
    #left a b
;
`,
			synErr: synErrUnclosedDirGroup,
		},
		{
			caption: "an ordered symbol marker '$' must be followed by and ID",
			src: `
#prec (
    #assign $
);
`,
			synErr: synErrNoOrderedSymbolName,
		},
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
e: e '+' t | e '-' t | t;
t: t '*' f | t '/' f | f;
f: '(' e ')' | id;
id: "[A-Za-z_][0-9A-Za-z_]*";
`,
			ast: &RootNode{
				Productions: []*ProductionNode{
					prod("e",
						alt(id("e"), pat(`+`), id("t")),
						alt(id("e"), pat(`-`), id("t")),
						alt(id("t")),
					),
					prod("t",
						alt(id("t"), pat(`*`), id("f")),
						alt(id("t"), pat(`/`), id("f")),
						alt(id("f")),
					),
					prod("f",
						alt(pat(`(`), id("e"), pat(`)`)),
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
a: 'foo' | ;
b: | 'bar';
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
			caption: "a production cannot contain an ordered symbol",
			src: `
a: $x;
`,
			synErr: synErrNoSemicolon,
		},
		{
			caption: "an alternative can contain a string literal without a terminal symbol",
			src: `
s
    : 'foo' bar
    ;

bar
    : 'bar';
`,
			ast: &RootNode{
				Productions: []*ProductionNode{
					prod("s",
						alt(pat(`foo`), id("bar")),
					),
				},
				LexProductions: []*ProductionNode{
					prod("bar",
						alt(pat(`bar`)),
					),
				},
			},
		},
		{
			caption: "an alternative cannot contain a pattern directly",
			src: `
s
    : "foo" bar
    ;

bar
    : "bar";
`,
			synErr: synErrPatternInAlt,
		},
		{
			caption: "a terminal symbol can be defined using a string literal",
			src: `
foo
    : 'foo';
`,
			ast: &RootNode{
				LexProductions: []*ProductionNode{
					prod("foo",
						alt(pat(`foo`)),
					),
				},
			},
		},
		{
			caption: "a terminal symbol can be defined using a pattern",
			src: `
foo
    : "foo";
`,
			ast: &RootNode{
				LexProductions: []*ProductionNode{
					prod("foo",
						alt(pat(`foo`)),
					),
				},
			},
		},
		{
			caption: "`fragment` is a reserved word",
			src:     `fragment: 'fragment';`,
			synErr:  synErrNoProductionName,
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
			caption: "the lexer treats consecutive lines as a single token but can count lines correctly",
			src: `// This line precedes line comments and blank lines.
// This is a line comment.


s
    : foo
    ;


// This line is sandwiched between blank lines.


foo: 'foo';
`,
			checkPosition: true,
			ast: &RootNode{
				Productions: []*ProductionNode{
					withProdPos(
						prod("s",
							withAltPos(
								alt(
									withElemPos(
										id("foo"),
										newPos(6),
									),
								),
								newPos(6),
							),
						),
						newPos(5),
					),
				},
				LexProductions: []*ProductionNode{
					withProdPos(
						prod("foo",
							withAltPos(
								alt(
									withElemPos(
										pat(`foo`),
										newPos(13),
									),
								),
								newPos(13),
							),
						),
						newPos(13),
					),
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

push_m1 #push m1
    : "->";
push_m2 #mode m1 #push m2
    : "-->";
pop_m1 #mode m1 #pop
    : "<-";
pop_m2 #mode m2 #pop
    : "<--";
whitespace #mode default m1 m2 #skip
    : "\u{0020}+";
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
					withProdDir(
						prod("push_m1",
							alt(pat(`->`)),
						),
						dir("push", idParam("m1")),
					),
					withProdDir(
						prod("push_m2",
							alt(pat(`-->`)),
						),
						dir("mode", idParam("m1")),
						dir("push", idParam("m2")),
					),
					withProdDir(
						prod("pop_m1",
							alt(pat(`<-`)),
						),
						dir("mode", idParam("m1")),
						dir("pop"),
					),
					withProdDir(
						prod("pop_m2",
							alt(pat(`<--`)),
						),
						dir("mode", idParam("m2")),
						dir("pop"),
					),
					withProdDir(
						prod("whitespace",
							alt(pat(`\u{0020}+`)),
						),
						dir("mode", idParam("default"), idParam("m1"), idParam("m2")),
						dir("skip"),
					),
				},
			},
		},
		{
			caption: "an alternative of a production can have multiple alternative directives",
			src: `
s
    : foo bar #prec baz #ast foo bar
    ;
`,
			ast: &RootNode{
				Productions: []*ProductionNode{
					prod("s",
						withAltDir(
							alt(id("foo"), id("bar")),
							dir("prec", idParam("baz")),
							dir("ast", idParam("foo"), idParam("bar")),
						),
					),
				},
			},
		},
		{
			caption: "a lexical production can have multiple production directives",
			src: `
foo #mode a #push b
    : 'foo';
`,
			ast: &RootNode{
				LexProductions: []*ProductionNode{
					withProdDir(
						prod("foo",
							alt(pat("foo")),
						),
						dir("mode", idParam("a")),
						dir("push", idParam("b")),
					),
				},
			},
		},
		{
			caption: "a production must be followed by a newline",
			src: `
s: foo; foo: "foo";
`,
			synErr: synErrSemicolonNoNewline,
		},
		{
			caption: "a grammar can contain 'ast' directives and expansion operator",
			src: `
s
    : foo bar_list #ast foo bar_list
    ;
bar_list
    : bar_list bar #ast bar_list... bar
    | bar          #ast bar
    ;
foo: "foo";
bar: "bar";
`,
			ast: &RootNode{
				Productions: []*ProductionNode{
					prod("s",
						withAltDir(
							alt(id("foo"), id("bar_list")),
							dir("ast", idParam("foo"), idParam("bar_list")),
						),
					),
					prod("bar_list",
						withAltDir(
							alt(id("bar_list"), id("bar")),
							dir("ast", exp(idParam("bar_list")), idParam("bar")),
						),
						withAltDir(
							alt(id("bar")),
							dir("ast", idParam("bar")),
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
			caption: "an expansion operator must be preceded by an identifier",
			src: `
s
    : foo #ast ...
    ;
`,
			synErr: synErrStrayExpOp,
		},
		{
			caption: "an expansion operator must be preceded by an identifier",
			src: `
a
    : foo #ast ... foo
    ;
`,
			synErr: synErrStrayExpOp,
		},
		{
			caption: "an expansion operator cannot be applied to a pattern",
			src: `
a
    : "foo" #ast "foo"...
    ;
`,
			synErr: synErrInvalidExpOperand,
		},
		{
			caption: "an expansion operator cannot be applied to a string",
			src: `
a
    : 'foo' #ast 'foo'...
    ;
`,
			synErr: synErrInvalidExpOperand,
		},
		{
			caption: "an expansion operator cannot be applied to an ordered symbol",
			src: `
a
    : foo #ast $foo...
    ;
`,
			synErr: synErrInvalidExpOperand,
		},
		{
			caption: "an expansion operator cannot be applied to a directive group",
			src: `
a
    : foo #ast ()...
    ;
`,
			synErr: synErrInvalidExpOperand,
		},
		{
			caption: "an AST has node positions",
			src: `
exp
    : exp '+' id #ast exp id
    | id
    ;

whitespace #skip
    : "\u{0020}+";
id
    : "\f{letter}(\f{letter}|\f{number})*";
fragment letter
    : "[A-Za-z_]";
fragment number
    : "[0-9]";
`,
			checkPosition: true,
			ast: &RootNode{
				Productions: []*ProductionNode{
					withProdPos(
						prod("exp",
							withAltPos(
								withAltDir(
									alt(
										withElemPos(id("exp"), newPos(3)),
										withElemPos(pat(`+`), newPos(3)),
										withElemPos(id("id"), newPos(3)),
									),
									withDirPos(
										dir("ast",
											withParamPos(idParam("exp"), newPos(3)),
											withParamPos(idParam("id"), newPos(3)),
										),
										newPos(3),
									),
								),
								newPos(3),
							),
							withAltPos(
								alt(
									withElemPos(id("id"), newPos(4)),
								),
								newPos(4),
							),
						),
						newPos(2),
					),
				},
				LexProductions: []*ProductionNode{
					withProdPos(
						withProdDir(
							prod("whitespace",
								withAltPos(
									alt(
										withElemPos(
											pat(`\u{0020}+`),
											newPos(8),
										),
									),
									newPos(8),
								),
							),
							withDirPos(
								dir("skip"),
								newPos(7),
							),
						),
						newPos(7),
					),
					withProdPos(
						prod("id",
							withAltPos(
								alt(
									withElemPos(
										pat(`\f{letter}(\f{letter}|\f{number})*`),
										newPos(10),
									),
								),
								newPos(10),
							),
						),
						newPos(9),
					),
				},
				Fragments: []*FragmentNode{
					withFragmentPos(
						frag("letter", "[A-Za-z_]"),
						newPos(11),
					),
					withFragmentPos(
						frag("number", "[0-9]"),
						newPos(13),
					),
				},
			},
		},
		{
			caption: "a symbol can have a label",
			src: `
expr
    : term@lhs add term@rhs
    ;
`,
			ast: &RootNode{
				Productions: []*ProductionNode{
					withProdPos(
						prod("expr",
							withAltPos(
								alt(
									withElemPos(
										withLabel(
											id("term"),
											withLabelPos(
												label("lhs"),
												newPos(3),
											),
										),
										newPos(3),
									),
									withElemPos(
										id("add"),
										newPos(3),
									),
									withElemPos(
										withLabel(
											id("term"),
											withLabelPos(
												label("rhs"),
												newPos(3),
											),
										),
										newPos(3),
									),
								),
								newPos(3),
							),
						),
						newPos(2),
					),
				},
			},
		},
		{
			caption: "a label must be an identifier, not a string",
			src: `
foo
    : bar@'baz'
    ;
`,
			synErr: synErrNoLabel,
		},
		{
			caption: "a label must be an identifier, not a pattern",
			src: `
foo
    : bar@"baz"
    ;
`,
			synErr: synErrNoLabel,
		},
		{
			caption: "the symbol marker @ must be followed by an identifier",
			src: `
foo
    : bar@
    ;
`,
			synErr: synErrNoLabel,
		},
		{
			caption: "a symbol cannot have more than or equal to two labels",
			src: `
foo
    : bar@baz@bra
    ;
`,
			synErr: synErrLabelWithNoSymbol,
		},
		{
			caption: "a label must follow a symbol",
			src: `
foo
    : @baz
    ;
`,
			synErr: synErrLabelWithNoSymbol,
		},
		{
			caption: "a grammar can contain left and right associativities",
			src: `
#prec (
    #left l1 l2
    #left l3
    #right r1 r2
    #right r3
);

s
    : id l1 id l2 id l3 id
    | id r1 id r2 id r3 id
    ;

whitespaces #skip
    : "[\u{0009}\u{0020}]+";
l1
    : 'l1';
l2
    : 'l2';
l3
    : 'l3';
r1
    : 'r1';
r2
    : 'r2';
r3
    : 'r3';
id
    : "[A-Za-z0-9_]+";
`,
			ast: &RootNode{
				Directives: []*DirectiveNode{
					withDirPos(
						prec(
							withParamPos(
								group(
									withDirPos(
										leftAssoc(
											withParamPos(idParam("l1"), newPos(3)),
											withParamPos(idParam("l2"), newPos(3)),
										),
										newPos(3),
									),
									withDirPos(
										leftAssoc(
											withParamPos(idParam("l3"), newPos(4)),
										),
										newPos(4),
									),
									withDirPos(
										rightAssoc(
											withParamPos(idParam("r1"), newPos(5)),
											withParamPos(idParam("r2"), newPos(5)),
										),
										newPos(5),
									),
									withDirPos(
										rightAssoc(
											withParamPos(idParam("r3"), newPos(6)),
										),
										newPos(6),
									),
								),
								newPos(2),
							),
						),
						newPos(2),
					),
				},
				Productions: []*ProductionNode{
					prod("s",
						alt(id(`id`), id(`l1`), id(`id`), id(`l2`), id(`id`), id(`l3`), id(`id`)),
						alt(id(`id`), id(`r1`), id(`id`), id(`r2`), id(`id`), id(`r3`), id(`id`)),
					),
				},
				LexProductions: []*ProductionNode{
					withProdDir(
						prod("whitespaces",
							alt(pat(`[\u{0009}\u{0020}]+`)),
						),
						dir("skip"),
					),
					prod("l1", alt(pat(`l1`))),
					prod("l2", alt(pat(`l2`))),
					prod("l3", alt(pat(`l3`))),
					prod("r1", alt(pat(`r1`))),
					prod("r2", alt(pat(`r2`))),
					prod("r3", alt(pat(`r3`))),
					prod("id", alt(pat(`[A-Za-z0-9_]+`))),
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
	if len(root.Directives) != len(expected.Directives) {
		t.Fatalf("unexpected length of top-level directives; want: %v, got: %v", len(expected.Directives), len(root.Directives))
	}
	for i, dir := range root.Directives {
		testDirectives(t, []*DirectiveNode{dir}, []*DirectiveNode{expected.Directives[i]}, true)
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
	if len(expected.Directives) != len(prod.Directives) {
		t.Fatalf("unexpected directive count; want: %v directives, got: %v directives", len(expected.Directives), len(prod.Directives))
	}
	if len(expected.Directives) > 0 {
		testDirectives(t, prod.Directives, expected.Directives, checkPosition)
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
	if len(alt.Directives) != len(expected.Directives) {
		t.Fatalf("unexpected alternative directive count; want: %v directive, got: %v directive", len(expected.Directives), len(alt.Directives))
	}
	if len(alt.Directives) > 0 {
		testDirectives(t, alt.Directives, expected.Directives, checkPosition)
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

func testDirectives(t *testing.T, dirs, expected []*DirectiveNode, checkPosition bool) {
	t.Helper()
	for i, exp := range expected {
		dir := dirs[i]

		if exp.Name != dir.Name {
			t.Fatalf("unexpected directive name; want: %+v, got: %+v", exp.Name, dir.Name)
		}
		if len(exp.Parameters) != len(dir.Parameters) {
			t.Fatalf("unexpected directive parameter; want: %+v, got: %+v", exp.Parameters, dir.Parameters)
		}
		for j, expParam := range exp.Parameters {
			testParameter(t, dir.Parameters[j], expParam, checkPosition)
		}
		if checkPosition {
			testPosition(t, dir.Pos, exp.Pos)
		}
	}
}

func testParameter(t *testing.T, param, expected *ParameterNode, checkPosition bool) {
	t.Helper()
	if param.ID != expected.ID {
		t.Fatalf("unexpected ID parameter; want: %v, got: %v", expected.ID, param.ID)
	}
	if param.String != expected.String {
		t.Fatalf("unexpected string parameter; want: %v, got: %v", expected.ID, param.ID)
	}
	if param.Expansion != expected.Expansion {
		t.Fatalf("unexpected expansion; want: %v, got: %v", expected.Expansion, param.Expansion)
	}
	if checkPosition {
		testPosition(t, param.Pos, expected.Pos)
	}
}

func testPosition(t *testing.T, pos, expected Position) {
	t.Helper()
	if pos.Row != expected.Row {
		t.Fatalf("unexpected position want: %+v, got: %+v", expected, pos)
	}
}
