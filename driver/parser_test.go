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
		Type:     NodeTypeTerminal,
		KindName: kind,
		Text:     text,
		Children: children,
	}
}

func errorNode() *Node {
	return &Node{
		Type:     NodeTypeError,
		KindName: "error",
	}
}

func nonTermNode(kind string, children ...*Node) *Node {
	return &Node{
		Type:     NodeTypeNonTerminal,
		KindName: kind,
		Children: children,
	}
}

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		specSrc string
		src     string
		synErr  bool
		cst     *Node
		ast     *Node
	}{
		{
			specSrc: `
#name test;

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
#name test;

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
				nonTermNode("foo"),
				nonTermNode("bar"),
			),
		},
		// The driver can reduce productions that have the empty alternative and can generate a CST (and AST) node.
		{
			specSrc: `
#name test;

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
				nonTermNode("foo"),
				nonTermNode("bar",
					termNode("bar_text", "bar"),
				),
			),
		},
		// A production can have multiple alternative productions.
		{
			specSrc: `
#name test;

#prec (
    #assign $uminus
    #left mul div
    #left add sub
);

expr
    : expr add expr
    | expr sub expr
    | expr mul expr
    | expr div expr
    | int
    | sub int #prec $uminus // This 'sub' means the unary minus symbol.
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
							termNode("sub", "-"),
							termNode("int", "1"),
						),
						termNode("mul", "*"),
						nonTermNode("expr",
							termNode("sub", "-"),
							termNode("int", "2"),
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
#name test;

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
#name test;

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
#name test;

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
		// When #push and #pop are applied to the same symbol, #pop will run first, then #push.
		{
			specSrc: `
#name test;

s
    : foo bar baz
    ;

foo #push m1
    : 'foo';
bar #mode m1 #pop #push m2
    : 'bar';
baz #mode m2
    : 'baz';
`,
			src: `foobarbaz`,
			ast: nonTermNode("s",
				termNode("foo", "foo"),
				termNode("bar", "bar"),
				termNode("baz", "baz"),
			),
		},
		// When #push and #pop are applied to the same symbol, #pop will run first, then #push, even if #push appears first
		// in a definition. That is, the order in which #push and #pop appear in grammar has nothing to do with the order in which
		// they are executed.
		{
			specSrc: `
#name test;

s
    : foo bar baz
    ;

foo #push m1
    : 'foo';
bar #mode m1 #push m2 #pop
    : 'bar';
baz #mode m2
    : 'baz';
`,
			src: `foobarbaz`,
			ast: nonTermNode("s",
				termNode("foo", "foo"),
				termNode("bar", "bar"),
				termNode("baz", "baz"),
			),
		},
		// The parser can skips specified tokens.
		{
			specSrc: `
#name test;

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
#name test;

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
#name test;

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
		// The '...' operator can expand child nodes.
		{
			specSrc: `
#name test;

s
    : a #ast a...
    ;
a
    : a ',' foo #ast a... foo
    | foo
    ;

foo
    : 'foo';
`,
			src: `foo,foo,foo`,
			ast: nonTermNode("s",
				termNode("foo", "foo"),
				termNode("foo", "foo"),
				termNode("foo", "foo"),
			),
		},
		// The '...' operator also can applied to an element having no children.
		{
			specSrc: `
#name test;

s
    : a ';' #ast a...
    ;

a
    :
    ;
`,
			src: `;`,
			ast: nonTermNode("s"),
		},
		// A label can be a parameter of #ast directive.
		{
			specSrc: `
#name test;

#prec (
    #left add sub
);

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
		// An AST can contain a symbol name, even if the symbol has a label. That is, unused labels are allowed.
		{
			specSrc: `
#name test;

s
    : foo@x ';' #ast foo
    ;

foo
    : 'foo';
`,
			src: `foo;`,
			ast: nonTermNode("s",
				termNode("foo", "foo"),
			),
		},
		// A production has the same precedence and associativity as the right-most terminal symbol.
		{
			specSrc: `
#name test;

#prec (
    #left add
);

expr
    : expr add expr // This alternative has the same precedence and associativiry as 'add'.
    | int
    ;

ws #skip
    : "[\u{0009}\u{0020}]+";
int
    : "0|[1-9][0-9]*";
add
    : '+';
`,
			// This source is recognized as the following structure because the production `expr → expr add expr` has the same
			// precedence and associativity as the symbol 'add'.
			//
			// ((1+2)+3)
			//
			// If the symbol doesn't have the precedence and left associativity, the production also doesn't have the precedence
			// and associativity and this source will be recognized as the following structure.
			//
			// (1+(2+3))
			src: `1+2+3`,
			ast: nonTermNode("expr",
				nonTermNode("expr",
					nonTermNode("expr",
						termNode("int", "1"),
					),
					termNode("add", "+"),
					nonTermNode("expr",
						termNode("int", "2"),
					),
				),
				termNode("add", "+"),
				nonTermNode("expr",
					termNode("int", "3"),
				),
			),
		},
		// The 'prec' directive can set precedence of a production.
		{
			specSrc: `
#name test;

#prec (
    #assign $uminus
    #left mul div
    #left add sub
);

expr
    : expr add expr
    | expr sub expr
    | expr mul expr
    | expr div expr
    | int
    | sub int #prec $uminus // This 'sub' means a unary minus symbol.
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
			// has the `#prec mul` directive and has the same precedence of the symbol `mul`.
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
						termNode("int", "1"),
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
#name test;

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
		// The 'error' symbol can appear in an #ast directive.
		{
			specSrc: `
#name test;

s
    : foo ';'
    | error ';' #ast error
    ;

foo
    : 'foo';
`,
			src:    `bar;`,
			synErr: true,
			ast: nonTermNode("s",
				errorNode(),
			),
		},
		// The 'error' symbol can have a label, and an #ast can reference it.
		{
			specSrc: `
#name test;

s
    : foo ';'
    | error@e ';' #ast e
    ;

foo
    : 'foo';
`,
			src:    `bar;`,
			synErr: true,
			ast: nonTermNode("s",
				errorNode(),
			),
		},
		// The grammar can contain the 'recover' directive.
		{
			specSrc: `
#name test;

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
#name test;

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

				if !tt.synErr && len(p.SyntaxErrors()) > 0 {
					for _, synErr := range p.SyntaxErrors() {
						t.Fatalf("unexpected syntax errors occurred: %v", synErr)
					}
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

	if node.Type != expected.Type || node.KindName != expected.KindName || node.Text != expected.Text {
		t.Fatalf("unexpected node; want: %+v, got: %+v", expected, node)
	}
	if len(node.Children) != len(expected.Children) {
		t.Fatalf("unexpected children; want: %v, got: %v", len(expected.Children), len(node.Children))
	}
	for i, c := range node.Children {
		testTree(t, c, expected.Children[i])
	}
}
