package parser

import (
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec/grammar/parser"
)

func TestParserWithConflicts(t *testing.T) {
	tests := []struct {
		caption string
		specSrc string
		src     string
		cst     *Node
	}{
		{
			caption: "when a shift/reduce conflict occurred, we prioritize the shift action",
			specSrc: `
#name test;

expr
    : expr assign expr
	| id
	;

id: "[A-Za-z0-9_]+";
assign: '=';
`,
			src: `foo=bar=baz`,
			cst: nonTermNode("expr",
				nonTermNode("expr",
					termNode("id", "foo"),
				),
				termNode("assign", "="),
				nonTermNode("expr",
					nonTermNode("expr",
						termNode("id", "bar"),
					),
					termNode("assign", "="),
					nonTermNode("expr",
						termNode("id", "baz"),
					),
				),
			),
		},
		{
			caption: "when a reduce/reduce conflict occurred, we prioritize the production defined earlier in the grammar",
			specSrc: `
#name test;

s
    : a
	| b
	;
a
    : id
	;
b
    : id
	;

id: "[A-Za-z0-9_]+";
`,
			src: `foo`,
			cst: nonTermNode("s",
				nonTermNode("a",
					termNode("id", "foo"),
				),
			),
		},
		{
			caption: "left associativities defined earlier in the grammar have higher precedence",
			specSrc: `
#name test;

#prec (
    #left mul
    #left add
);

expr
    : expr add expr
    | expr mul expr
	| id
    ;

id: "[A-Za-z0-9_]+";
add: '+';
mul: '*';
`,
			src: `a+b*c*d+e`,
			cst: nonTermNode("expr",
				nonTermNode("expr",
					nonTermNode("expr",
						termNode("id", "a"),
					),
					termNode("add", "+"),
					nonTermNode("expr",
						nonTermNode("expr",
							nonTermNode("expr",
								termNode("id", "b"),
							),
							termNode("mul", "*"),
							nonTermNode("expr",
								termNode("id", "c"),
							),
						),
						termNode("mul", "*"),
						nonTermNode("expr",
							termNode("id", "d"),
						),
					),
				),
				termNode("add", "+"),
				nonTermNode("expr",
					termNode("id", "e"),
				),
			),
		},
		{
			caption: "left associativities defined in the same line have the same precedence",
			specSrc: `
#name test;

#prec (
    #left add sub
);

expr
    : expr add expr
    | expr sub expr
	| id
    ;

id: "[A-Za-z0-9_]+";
add: '+';
sub: '-';
`,
			src: `a-b+c+d-e`,
			cst: nonTermNode("expr",
				nonTermNode("expr",
					nonTermNode("expr",
						nonTermNode("expr",
							nonTermNode("expr",
								termNode("id", "a"),
							),
							termNode("sub", "-"),
							nonTermNode("expr",
								termNode("id", "b"),
							),
						),
						termNode("add", "+"),
						nonTermNode("expr",
							termNode("id", "c"),
						),
					),
					termNode("add", "+"),
					nonTermNode("expr",
						termNode("id", "d"),
					),
				),
				termNode("sub", "-"),
				nonTermNode("expr",
					termNode("id", "e"),
				),
			),
		},
		{
			caption: "right associativities defined earlier in the grammar have higher precedence",
			specSrc: `
#name test;

#prec (
    #right r1
    #right r2
);

expr
    : expr r2 expr
    | expr r1 expr
	| id
    ;

whitespaces #skip
    : "[\u{0009}\u{0020}]+";
r1
    : 'r1';
r2
    : 'r2';
id
    : "[A-Za-z0-9_]+";
`,
			src: `a r2 b r1 c r1 d r2 e`,
			cst: nonTermNode("expr",
				nonTermNode("expr",
					termNode("id", "a"),
				),
				termNode("r2", "r2"),
				nonTermNode("expr",
					nonTermNode("expr",
						nonTermNode("expr",
							termNode("id", "b"),
						),
						termNode("r1", "r1"),
						nonTermNode("expr",
							nonTermNode("expr",
								termNode("id", "c"),
							),
							termNode("r1", "r1"),
							nonTermNode("expr",
								termNode("id", "d"),
							),
						),
					),
					termNode("r2", "r2"),
					nonTermNode("expr",
						termNode("id", "e"),
					),
				),
			),
		},
		{
			caption: "right associativities defined in the same line have the same precedence",
			specSrc: `
#name test;

#prec (
    #right r1 r2
);

expr
    : expr r2 expr
    | expr r1 expr
	| id
    ;

whitespaces #skip
    : "[\u{0009}\u{0020}]+";
r1
    : 'r1';
r2
    : 'r2';
id
    : "[A-Za-z0-9_]+";
`,
			src: `a r2 b r1 c r1 d r2 e`,
			cst: nonTermNode("expr",
				nonTermNode("expr",
					termNode("id", "a"),
				),
				termNode("r2", "r2"),
				nonTermNode("expr",
					nonTermNode("expr",
						termNode("id", "b"),
					),
					termNode("r1", "r1"),
					nonTermNode("expr",
						nonTermNode("expr",
							termNode("id", "c"),
						),
						termNode("r1", "r1"),
						nonTermNode("expr",
							nonTermNode("expr",
								termNode("id", "d"),
							),
							termNode("r2", "r2"),
							nonTermNode("expr",
								termNode("id", "e"),
							),
						),
					),
				),
			),
		},
		{
			caption: "terminal symbols with an #assign directive defined earlier in the grammar have higher precedence",
			specSrc: `
#name test;

#prec (
    #assign a1
    #assign a2
);

expr
    : expr a2 expr
    | expr a1 expr
	| id
    ;

whitespaces #skip
    : "[\u{0009}\u{0020}]+";
a1
    : 'a1';
a2
    : 'a2';
id
    : "[A-Za-z0-9_]+";
`,
			src: `a a2 b a1 c a1 d a2 e`,
			cst: nonTermNode("expr",
				nonTermNode("expr",
					termNode("id", "a"),
				),
				termNode("a2", "a2"),
				nonTermNode("expr",
					nonTermNode("expr",
						nonTermNode("expr",
							termNode("id", "b"),
						),
						termNode("a1", "a1"),
						nonTermNode("expr",
							nonTermNode("expr",
								termNode("id", "c"),
							),
							termNode("a1", "a1"),
							nonTermNode("expr",
								termNode("id", "d"),
							),
						),
					),
					termNode("a2", "a2"),
					nonTermNode("expr",
						termNode("id", "e"),
					),
				),
			),
		},
		{
			caption: "terminal symbols with an #assign directive defined in the same line have the same precedence",
			specSrc: `
#name test;

#prec (
    #assign a1 a2
);

expr
    : expr a2 expr
    | expr a1 expr
	| id
    ;

whitespaces #skip
    : "[\u{0009}\u{0020}]+";
a1
    : 'a1';
a2
    : 'a2';
id
    : "[A-Za-z0-9_]+";
`,
			src: `a a2 b a1 c a1 d a2 e`,
			cst: nonTermNode("expr",
				nonTermNode("expr",
					termNode("id", "a"),
				),
				termNode("a2", "a2"),
				nonTermNode("expr",
					nonTermNode("expr",
						termNode("id", "b"),
					),
					termNode("a1", "a1"),
					nonTermNode("expr",
						nonTermNode("expr",
							termNode("id", "c"),
						),
						termNode("a1", "a1"),
						nonTermNode("expr",
							nonTermNode("expr",
								termNode("id", "d"),
							),
							termNode("a2", "a2"),
							nonTermNode("expr",
								termNode("id", "e"),
							),
						),
					),
				),
			),
		},
		{
			caption: "#left, #right, and #assign can be mixed",
			specSrc: `
#name test;

#prec (
    #left mul div
    #left add sub
    #assign else
    #assign then
    #right assign
);

expr
    : expr add expr
    | expr sub expr
    | expr mul expr
    | expr div expr
    | expr assign expr
    | if expr then expr
    | if expr then expr else expr
    | id
    ;

ws #skip: "[\u{0009}\u{0020}]+";
if: 'if';
then: 'then';
else: 'else';
id: "[A-Za-z0-9_]+";
add: '+';
sub: '-';
mul: '*';
div: '/';
assign: '=';
`,
			src: `x = y = a + b * c - d / e + if f then if g then h else i`,
			cst: nonTermNode(
				"expr",
				nonTermNode("expr",
					termNode("id", "x"),
				),
				termNode("assign", "="),
				nonTermNode("expr",
					nonTermNode("expr",
						termNode("id", "y"),
					),
					termNode("assign", "="),
					nonTermNode("expr",
						nonTermNode("expr",
							nonTermNode("expr",
								nonTermNode("expr",
									termNode("id", "a"),
								),
								termNode("add", "+"),
								nonTermNode("expr",
									nonTermNode("expr",
										termNode("id", "b"),
									),
									termNode("mul", "*"),
									nonTermNode("expr",
										termNode("id", "c"),
									),
								),
							),
							termNode("sub", "-"),
							nonTermNode("expr",
								nonTermNode("expr",
									termNode("id", "d"),
								),
								termNode("div", "/"),
								nonTermNode("expr",
									termNode("id", "e"),
								),
							),
						),
						termNode("add", "+"),
						nonTermNode("expr",
							termNode("if", "if"),
							nonTermNode("expr",
								termNode("id", "f"),
							),
							termNode("then", "then"),
							nonTermNode("expr",
								termNode("if", "if"),
								nonTermNode("expr",
									termNode("id", "g"),
								),
								termNode("then", "then"),
								nonTermNode("expr",
									termNode("id", "h"),
								),
								termNode("else", "else"),
								nonTermNode("expr",
									termNode("id", "i"),
								),
							),
						),
					),
				),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			ast, err := parser.Parse(strings.NewReader(tt.specSrc))
			if err != nil {
				t.Fatal(err)
			}

			b := grammar.GrammarBuilder{
				AST: ast,
			}
			cg, _, err := b.Build()
			if err != nil {
				t.Fatal(err)
			}

			toks, err := NewTokenStream(cg, strings.NewReader(tt.src))
			if err != nil {
				t.Fatal(err)
			}

			gram := NewGrammar(cg)
			tb := NewDefaultSyntaxTreeBuilder()
			p, err := NewParser(toks, gram, SemanticAction(NewCSTActionSet(gram, tb)))
			if err != nil {
				t.Fatal(err)
			}

			err = p.Parse()
			if err != nil {
				t.Fatal(err)
			}

			if tt.cst != nil {
				testTree(t, tb.Tree(), tt.cst)
			}
		})
	}
}
