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
		specErr bool
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
		// `name` is missing.
		{
			specSrc: `
a
    : foo
    ;
foo: "foo";
`,
			src:     `foo`,
			specErr: true,
		},
		// `name` needs a parameter.
		{
			specSrc: `
%name

a
    : foo
    ;
foo: "foo";
`,
			src:     `foo`,
			specErr: true,
		},
		// `name` takes just one parameter.
		{
			specSrc: `
%name test foo

a
    : foo
    ;
foo: "foo";
`,
			src:     `foo`,
			specErr: true,
		},
		// Production `b` is unused.
		{
			specSrc: `
%name test

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
%name test

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
%name test

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
%name test

a
    : foo
    ;

foo #skip
    : "foo";
`,
			src:     `foo`,
			specErr: true,
		},
		// A production cannot have production directives.
		{
			specSrc: `
%name test

s #prec foo
    : foo
    ;

foo: 'foo' #skip;
`,
			src:     `foo`,
			specErr: true,
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
		// A lexical production cannot have alternative directives.
		{
			specSrc: `
%name test

s
    : foo
    ;

foo: 'foo' #skip;
`,
			src:     `foo`,
			specErr: true,
		},
		// A production directive must not be duplicated.
		{
			specSrc: `
%name test

s
    : foo
    ;

foo #skip #skip
    : 'foo';
`,
			src:     `foo`,
			specErr: true,
		},
		// An alternative directive must not be duplicated.
		{
			specSrc: `
%name test

s
    : foo bar #ast foo bar #ast foo bar
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			src:     `foobar`,
			specErr: true,
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
		// The expansion cannot be applied to a terminal symbol.
		{
			specSrc: `
%name test

s
    : foo #ast foo...
    ;
foo: "foo";
`,
			specErr: true,
		},
		// The expansion cannot be applied to a pattern.
		{
			specSrc: `
%name test

s
    : foo "bar"@b #ast foo b...
    ;
foo: "foo";
`,
			specErr: true,
		},
		// The expansion cannot be applied to a string.
		{
			specSrc: `
%name test

s
    : foo 'bar'@b #ast foo b...
    ;
foo: "foo";
`,
			specErr: true,
		},
		// A parameter of #ast directive must be either a symbol or a label in an alternative.
		{
			specSrc: `
%name test

s
    : foo bar #ast foo x
    ;
foo: "foo";
bar: "bar";
`,
			specErr: true,
		},
		// A symbol in a different alternative cannot be a parameter of #ast directive.
		{
			specSrc: `
%name test

s
    : foo #ast bar
    | bar
    ;
foo: "foo";
bar: "bar";
`,
			specErr: true,
		},
		// A label in a different alternative cannot be a parameter of #ast directive.
		{
			specSrc: `
%name test

s
    : foo #ast b
    | bar@b
    ;
foo: "foo";
bar: "bar";
`,
			specErr: true,
		},
		// A production must not have a duplicate alternative.
		{
			specSrc: `
%name test

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
%name test

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
%name test

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
%name test

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
%name test

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
%name test

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
%name test

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
%name test

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
%name test

%left s

s
    : a
    ;

a: 'a';
`,
			specErr: true,
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
		// The 'prec' directive needs an ID parameter.
		{
			specSrc: `
%name test

s
    : a #prec
    ;

a: 'a';
`,
			specErr: true,
		},
		// The 'prec' directive cannot take an unknown symbol.
		{
			specSrc: `
%name test

s
    : a #prec foo
    ;

a: 'a';
`,
			specErr: true,
		},
		// The 'prec' directive cannot take a non-terminal symbol.
		{
			specSrc: `
%name test

s
    : foo #prec bar
    | bar
    ;
foo
    : a
    ;
bar
    : b
    ;

a: 'a';
b: 'b';
`,
			specErr: true,
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
		// The 'recover' directive cannot take a parameter.
		{
			specSrc: `
%name test

seq
    : seq elem
    | elem
    ;
elem
    : id id id ';'
    | error ';' #recover foo
    ;

ws #skip
    : "[\u{0009}\u{0020}]+";
id
    : "[A-Za-z_]+";
`,
			src:     `a b c ; d e f ;`,
			specErr: true,
		},
		// You cannot use the error symbol as a non-terminal symbol.
		{
			specSrc: `
%name test

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
%name test

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
%name test

s
    : foo
    ;

foo
    : 'foo';
error #skip
    : 'error';
`,
			specErr: true,
		},
		// A label must be unique in an alternative.
		{
			specSrc: `
%name test

s
    : foo@x bar@x
    ;

foo: 'foo';
bar: 'bar';
`,
			specErr: true,
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
		// A label cannot be the same name as terminal symbols.
		{
			specSrc: `
%name test

s
    : foo bar@foo
    ;

foo: 'foo';
bar: 'bar';
`,
			specErr: true,
		},
		// A label cannot be the same name as non-terminal symbols.
		{
			specSrc: `
%name test

s
    : foo@a
    ;
a
    : bar
    ;

foo: 'foo';
bar: 'bar';
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
