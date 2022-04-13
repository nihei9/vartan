package driver

import (
	"fmt"
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
	}{
		{
			specSrc: `
%name test

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
								termNode("x_3", "("),
								nonTermNode("expr",
									nonTermNode("expr",
										nonTermNode("term",
											nonTermNode("factor",
												termNode("id", "a"),
											),
										),
									),
									termNode("x_1", "+"),
									nonTermNode("term",
										nonTermNode("factor",
											termNode("x_3", "("),
											nonTermNode("expr",
												nonTermNode("expr",
													nonTermNode("term",
														nonTermNode("factor",
															termNode("id", "b"),
														),
													),
												),
												termNode("x_1", "+"),
												nonTermNode("term",
													nonTermNode("factor",
														termNode("id", "c"),
													),
												),
											),
											termNode("x_4", ")"),
										),
									),
								),
								termNode("x_4", ")"),
							),
						),
						termNode("x_2", "*"),
						nonTermNode("factor",
							termNode("id", "d"),
						),
					),
				),
				termNode("x_1", "+"),
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
%name test

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
%name test

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
		// A production can have multiple alternative productions.
		{
			specSrc: `
%name test

%left mul div
%left add sub

expr
    : expr add expr
    | expr sub expr
    | expr mul expr
    | expr div expr
    | int
    | sub int #prec mul #ast int sub // This 'sub' means the unary minus symbol.
    ;

int
    : "0|[1-9][0-9]*";
add
    : '+';
sub
    : '-';
mul
    : '*';
div
    : '/';
`,
			src: `-1*-2+3-4/5`,
			ast: nonTermNode("expr",
				nonTermNode("expr",
					nonTermNode("expr",
						nonTermNode("expr",
							termNode("int", "1"),
							termNode("sub", "-"),
						),
						termNode("mul", "*"),
						nonTermNode("expr",
							termNode("int", "2"),
							termNode("sub", "-"),
						),
					),
					termNode("add", "+"),
					nonTermNode("expr",
						termNode("int", "3"),
					),
				),
				termNode("sub", "-"),
				nonTermNode("expr",
					nonTermNode("expr",
						termNode("int", "4"),
					),
					termNode("div", "/"),
					nonTermNode("expr",
						termNode("int", "5"),
					),
				),
			),
		},
		// A lexical production can have multiple production directives.
		{
			specSrc: `
%name test

s
    : push_a push_b pop pop
    ;

push_a #mode default #push a
    : '->a';
push_b #mode a #push b
    : '->b';
pop #mode a b #pop
    : '<-';
`,
			src: `->a->b<-<-`,
			ast: nonTermNode("s",
				termNode("push_a", "->a"),
				termNode("push_b", "->b"),
				termNode("pop", "<-"),
				termNode("pop", "<-"),
			),
		},
		{
			specSrc: `
%name test

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
			src: ` -> --> <-- <- `,
		},
		{
			specSrc: `
%name test

s
    : foo bar
    ;

foo
    : "foo";
bar #mode default
    : "bar";
`,
			src: `foobar`,
		},
		// The parser can skips specified tokens.
		{
			specSrc: `
%name test

s
    : foo bar
    ;

foo
    : "foo";
bar
    : "bar";
white_space #skip
    : "[\u{0009}\u{0020}]+";
`,
			src: `foo bar`,
		},
		// A grammar can contain fragments.
		{
			specSrc: `
%name test

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
%name test

list
    : "\[" elems "]" #ast elems...
    ;
elems
    : elems "," id #ast elems... id
    | id
    ;

whitespace #skip
    : "\u{0020}+";
id
    : "[A-Za-z]+";
`,
			src: `[Byers, Frohike, Langly]`,
			cst: nonTermNode("list",
				termNode("x_1", "["),
				nonTermNode("elems",
					nonTermNode("elems",
						nonTermNode("elems",
							termNode("id", "Byers"),
						),
						termNode("x_3", ","),
						termNode("id", "Frohike"),
					),
					termNode("x_3", ","),
					termNode("id", "Langly"),
				),
				termNode("x_2", "]"),
			),
			ast: nonTermNode("list",
				termNode("id", "Byers"),
				termNode("id", "Frohike"),
				termNode("id", "Langly"),
			),
		},
		// A label can be a parameter of #ast directive.
		{
			specSrc: `
%name test

%left add sub

expr
    : expr@lhs add expr@rhs #ast add lhs rhs
    | expr@lhs sub expr@rhs #ast sub lhs rhs
    | num
    ;
add: '+';
sub: '-';
num: "0|[1-9][0-9]*";
`,
			src: `1+2-3`,
			ast: nonTermNode("expr",
				termNode("sub", "-"),
				nonTermNode("expr",
					termNode("add", "+"),
					nonTermNode("expr",
						termNode("num", "1"),
					),
					nonTermNode("expr",
						termNode("num", "2"),
					),
				),
				nonTermNode("expr",
					termNode("num", "3"),
				),
			),
		},
		// The 'prec' directive can set precedence and associativity of a production.
		{
			specSrc: `
%name test

%left mul div
%left add sub

expr
    : expr add expr
    | expr sub expr
    | expr mul expr
    | expr div expr
    | sub expr #prec mul // This 'sub' means a unary minus symbol.
    | int
    ;

ws #skip
    : "[\u{0009}\u{0020}]+";
int
    : "0|[1-9][0-9]*";
add
    : '+';
sub
    : '-';
mul
    : '*';
div
    : '/';
`,
			// This source is recognized as the following structure because the production `expr → sub expr`
			// has the `#prec mul` directive and has the same precedence and associativity of the symbol `mul`.
			//
			// (((-1) * 20) / 5)
			//
			// If the production doesn't have the `#prec` directive, this source will be recognized as
			// the following structure.
			//
			// (- ((1 * 20) / 5))
			src: `-1*20/5`,
			cst: nonTermNode("expr",
				nonTermNode("expr",
					nonTermNode("expr",
						termNode("sub", "-"),
						nonTermNode("expr",
							termNode("int", "1"),
						),
					),
					termNode("mul", "*"),
					nonTermNode("expr",
						termNode("int", "20"),
					),
				),
				termNode("div", "/"),
				nonTermNode("expr",
					termNode("int", "5"),
				),
			),
		},
		// The grammar can contain the 'error' symbol.
		{
			specSrc: `
%name test

s
    : id id id ';'
    | error ';'
    ;

ws #skip
    : "[\u{0009}\u{0020}]+";
id
    : "[A-Za-z_]+";
`,
			src: `foo bar baz ;`,
		},
		// The grammar can contain the 'recover' directive.
		{
			specSrc: `
%name test

seq
    : seq elem
    | elem
    ;
elem
    : id id id ';'
    | error ';' #recover
    ;

ws #skip
    : "[\u{0009}\u{0020}]+";
id
    : "[A-Za-z_]+";
`,
			src: `a b c ; d e f ;`,
		},
		// The same label can be used between different alternatives.
		{
			specSrc: `
%name test

s
    : foo@x bar
    | foo@x
    ;

foo: 'foo';
bar: 'bar';
`,
			src: `foo`,
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
				if err != nil {
					t.Fatal(err)
				}

				cg, err := grammar.Compile(g, grammar.SpecifyClass(class))
				if err != nil {
					t.Fatal(err)
				}

				toks, err := NewTokenStream(cg, strings.NewReader(tt.src))
				if err != nil {
					t.Fatal(err)
				}

				gram := NewGrammar(cg)
				tb := NewDefaultSyntaxTreeBuilder()
				var opt []ParserOption
				switch {
				case tt.ast != nil:
					opt = append(opt, SemanticAction(NewASTActionSet(gram, tb)))
				case tt.cst != nil:
					opt = append(opt, SemanticAction(NewCSTActionSet(gram, tb)))
				}
				p, err := NewParser(toks, gram, opt...)
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

				switch {
				case tt.ast != nil:
					testTree(t, tb.Tree(), tt.ast)
				case tt.cst != nil:
					testTree(t, tb.Tree(), tt.cst)
				}
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
