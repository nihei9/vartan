package driver

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec"
)

func TestParser_Parse(t *testing.T) {
	termNode := func(kind string, text string, children ...*Node) *Node {
		return &Node{
			KindName: kind,
			Text:     text,
			Children: children,
		}
	}

	nonTermNode := func(kind string, children ...*Node) *Node {
		return &Node{
			KindName: kind,
			Children: children,
		}
	}

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
    : "\[" elems "]" # ast '(list $2...)
    ;
elems
    : elems "," id # ast '(elems $1... $3)
    | id
    ;
whitespace: "\u{0020}+" # skip;
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
    : foo # ast '(start $1)
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
    : "foo" # ast '(s $1...)
    ;
`,
			specErr: true,
		},
		// The expansion cannot be applied to a terminal symbol.
		{
			specSrc: `
s
    : foo # ast '(s $1...)
    ;
foo: "foo";
`,
			specErr: true,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			ast, err := spec.Parse(strings.NewReader(tt.specSrc))
			if err != nil {
				t.Fatal(err)
			}

			g, err := grammar.NewGrammar(ast)
			if tt.specErr {
				if err == nil {
					t.Fatal("an expected error didn't occur")
				}
				// fmt.Printf("error: %v\n", err)
				return
			} else {
				if err != nil {
					t.Fatal(err)
				}
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

			if tt.cst != nil {
				testTree(t, p.CST(), tt.cst)
			}

			if tt.ast != nil {
				testTree(t, p.AST(), tt.ast)
			}

			fmt.Println("CST:")
			PrintTree(p.CST(), 0)
			fmt.Println("AST:")
			PrintTree(p.AST(), 0)
		})
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
