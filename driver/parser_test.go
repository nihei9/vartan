package driver

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec"
)

func termNode(kind string, text string, children ...*Node) *Node {
	return &Node{
		KindName: kind,
		Text:     text,
		Children: children,
	}
}

func nonTermNode(kind string, children ...*Node) *Node {
	return &Node{
		KindName: kind,
		Children: children,
	}
}

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		specSrc string
		src     string
		cst     *Node
		ast     *Node
		specErr bool
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
			cst: nonTermNode("expr",
				nonTermNode("expr",
					nonTermNode("term",
						nonTermNode("term",
							nonTermNode("factor",
								termNode("__3__", "("),
								nonTermNode("expr",
									nonTermNode("expr",
										nonTermNode("term",
											nonTermNode("factor",
												termNode("id", "a"),
											),
										),
									),
									termNode("__1__", "+"),
									nonTermNode("term",
										nonTermNode("factor",
											termNode("__3__", "("),
											nonTermNode("expr",
												nonTermNode("expr",
													nonTermNode("term",
														nonTermNode("factor",
															termNode("id", "b"),
														),
													),
												),
												termNode("__1__", "+"),
												nonTermNode("term",
													nonTermNode("factor",
														termNode("id", "c"),
													),
												),
											),
											termNode("__4__", ")"),
										),
									),
								),
								termNode("__4__", ")"),
							),
						),
						termNode("__2__", "*"),
						nonTermNode("factor",
							termNode("id", "d"),
						),
					),
				),
				termNode("__1__", "+"),
				nonTermNode("term",
					nonTermNode("factor",
						termNode("id", "e"),
					),
				),
			),
		},
		// The driver can reduce productions that have the empty alternative and can generate a CST (and AST) node.
		{
			specSrc: `
s
    : foo bar
    ;
foo
    :
    ;
bar
    : bar_text
    |
    ;
bar_text: "bar";
`,
			src: ``,
			cst: nonTermNode("s",
				termNode("foo", ""),
				termNode("bar", ""),
			),
		},
		// The driver can reduce productions that have the empty alternative and can generate a CST (and AST) node.
		{
			specSrc: `
s
    : foo bar
    ;
foo
    :
    ;
bar
    : bar_text
    |
    ;
bar_text: "bar";
`,
			src: `bar`,
			cst: nonTermNode("s",
				termNode("foo", ""),
				nonTermNode("bar",
					termNode("bar_text", "bar"),
				),
			),
		},
		// Production `b` is unused.
		{
			specSrc: `
a
    : foo
    ;
b
    : foo;
foo: "foo";
`,
			src:     `foo`,
			specErr: true,
		},
		// Terminal `bar` is unused.
		{
			specSrc: `
s
    : foo
    ;
foo: "foo";
bar: "bar";
`,
			src:     `foo`,
			specErr: true,
		},
		// Production `b` and terminal `bar` is unused.
		{
			specSrc: `
a
    : foo
    ;
b
    : bar
    ;
foo: "foo";
bar: "bar";
`,
			src:     `foo`,
			specErr: true,
		},
		// A terminal used in productions cannot have the skip directive.
		{
			specSrc: `
a
    : foo
    ;
foo: "foo" #skip;
`,
			src:     `foo`,
			specErr: true,
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
			src: ` -> --> <-- <- `,
		},
		{
			specSrc: `
s
    : foo bar
    ;
foo: "foo";
#mode default
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
white_space: "[\u{0009}\u{0020}]+" #skip;
`,
			src: `foo bar`,
		},
		// A grammar can contain fragments.
		{
			specSrc: `
s
    : tagline
    ;
tagline: "\f{words} IS OUT THERE.";
fragment words: "[A-Za-z\u{0020}]+";
`,
			src: `THE TRUTH IS OUT THERE.`,
		},
		// A grammar can contain ast actions.
		{
			specSrc: `
list
    : "\[" elems "]" #ast #(list $2...)
    ;
elems
    : elems "," id #ast #(elems $1... $3)
    | id
    ;
whitespace: "\u{0020}+" #skip;
id: "[A-Za-z]+";
`,
			src: `[Byers, Frohike, Langly]`,
			cst: nonTermNode("list",
				termNode("__1__", "["),
				nonTermNode("elems",
					nonTermNode("elems",
						nonTermNode("elems",
							termNode("id", "Byers"),
						),
						termNode("__3__", ","),
						termNode("id", "Frohike"),
					),
					termNode("__3__", ","),
					termNode("id", "Langly"),
				),
				termNode("__2__", "]"),
			),
			ast: nonTermNode("list",
				termNode("id", "Byers"),
				termNode("id", "Frohike"),
				termNode("id", "Langly"),
			),
		},
		// The first element of a tree structure must be the same ID as an LHS of a production.
		{
			specSrc: `
s
    : foo #ast #(start $1)
    ;
foo
    : bar
    ;
bar: "bar";
`,
			specErr: true,
		},
		// An ast action cannot be applied to a terminal symbol.
		{
			specSrc: `
s
    : foo
    ;
foo
    : "foo" #ast #(s $1...)
    ;
`,
			specErr: true,
		},
		// The expansion cannot be applied to a terminal symbol.
		{
			specSrc: `
s
    : foo #ast #(s $1...)
    ;
foo: "foo";
`,
			specErr: true,
		},
		// A production must not have a duplicate alternative.
		{
			specSrc: `
s
    : foo
    | foo
    ;
foo: "foo";
`,
			specErr: true,
		},
		// A production must not have a duplicate alternative.
		{
			specSrc: `
a
    : foo
    ;
b
    :
    |
    ;
foo: "foo";
`,
			specErr: true,
		},
		// A production must not have a duplicate alternative.
		{
			specSrc: `
a
    : foo
    ;
b
    : bar
    ;
a
    : foo
    ;
foo: "foo";
bar: "bar";
`,
			specErr: true,
		},
		// A terminal and a non-terminal (start symbol) are duplicates.
		{
			specSrc: `
a
    : foo
    ;
foo: "foo";
a: "a";
`,
			specErr: true,
		},
		// A terminal and a non-terminal (not start symbol) are duplicates.
		{
			specSrc: `
a
    : foo
    ;
b
    : bar
    ;
foo: "foo";
bar: "bar";
b: "a";
`,
			specErr: true,
		},
		// Invalid associativity type
		{
			specSrc: `
%foo

s
    : a
    ;

a: 'a';
`,
			specErr: true,
		},
		// Associativity needs at least one symbol.
		{
			specSrc: `
%left

s
    : a
    ;

a: 'a';
`,
			specErr: true,
		},
		// Associativity cannot take an undefined symbol.
		{
			specSrc: `
%left b

s
    : a
    ;

a: 'a';
`,
			specErr: true,
		},
		// Associativity cannot take a non-terminal symbol.
		{
			specSrc: `
%left s

s
    : a
    ;

a: 'a';
`,
			specErr: true,
		},
		// The grammar can contain the 'error' symbol.
		{
			specSrc: `
s
    : id id id ';'
    | error ';'
    ;

ws: "[\u{0009}\u{0020}]+" #skip;
id: "[A-Za-z_]+";
`,
			src: `foo bar baz ;`,
		},
		// The grammar can contain the 'recover' directive.
		{
			specSrc: `
seq
    : seq elem
    | elem
    ;
elem
    : id id id ';'
    | error ';' #recover
    ;

ws: "[\u{0009}\u{0020}]+" #skip;
id: "[A-Za-z_]+";
`,
			src: `a b c ; d e f ;`,
		},
		// The 'recover' directive cannot take a parameter.
		{
			specSrc: `
seq
    : seq elem
    | elem
    ;
elem
    : id id id ';'
    | error ';' #recover foo
    ;

ws: "[\u{0009}\u{0020}]+" #skip;
id: "[A-Za-z_]+";
`,
			src:     `a b c ; d e f ;`,
			specErr: true,
		},
		// You cannot use the error symbol as a non-terminal symbol.
		{
			specSrc: `
s
    : foo
    ;
error
    : bar
    ;

foo: 'foo';
bar: 'bar';
`,
			specErr: true,
		},
		// You cannot use the error symbol as a terminal symbol.
		{
			specSrc: `
s
    : foo
    | error
    ;

foo: 'foo';
error: 'error';
`,
			specErr: true,
		},
		// You cannot use the error symbol as a terminal symbol, even if given the skip directive.
		{
			specSrc: `
s
    : foo
    ;

foo: 'foo';
error: 'error' #skip;
`,
			specErr: true,
		},
	}

	classes := []grammar.Class{
		grammar.ClassSLR,
		grammar.ClassLALR,
	}

	for i, tt := range tests {
		for _, class := range classes {
			t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
				ast, err := spec.Parse(strings.NewReader(tt.specSrc))
				if err != nil {
					t.Fatal(err)
				}

				b := grammar.GrammarBuilder{
					AST: ast,
				}
				g, err := b.Build()
				if tt.specErr {
					if err == nil {
						t.Fatal("an expected error didn't occur")
					}
					return
				} else {
					if err != nil {
						t.Fatal(err)
					}
				}

				gram, err := grammar.Compile(g, grammar.SpecifyClass(class))
				if err != nil {
					t.Fatal(err)
				}

				p, err := NewParser(gram, strings.NewReader(tt.src), MakeAST(), MakeCST())
				if err != nil {
					t.Fatal(err)
				}

				err = p.Parse()
				if err != nil {
					t.Fatal(err)
				}

				if len(p.SyntaxErrors()) > 0 {
					t.Fatalf("unexpected syntax errors occurred: %+v", p.SyntaxErrors())
				}

				if tt.cst != nil {
					testTree(t, p.CST(), tt.cst)
				}

				if tt.ast != nil {
					testTree(t, p.AST(), tt.ast)
				}

				fmt.Println("CST:")
				PrintTree(os.Stdout, p.CST())
				fmt.Println("AST:")
				PrintTree(os.Stdout, p.AST())
			})
		}
	}
}

func testTree(t *testing.T, node, expected *Node) {
	t.Helper()

	if node.KindName != expected.KindName || node.Text != expected.Text {
		t.Fatalf("unexpected node; want: %+v, got: %+v", expected, node)
	}
	if len(node.Children) != len(expected.Children) {
		t.Fatalf("unexpected children; want: %v, got: %v", len(expected.Children), len(node.Children))
	}
	for i, c := range node.Children {
		testTree(t, c, expected.Children[i])
	}
}
