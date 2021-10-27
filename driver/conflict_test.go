package driver

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec"
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
%name test

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
%name test

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
%name test

%left mul
%left add

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
%name test

%left add sub

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
%name test

%right r1
%right r2

expr
    : expr r2 expr
    | expr r1 expr
	| id
    ;

whitespaces: "[\u{0009}\u{0020}]+" #skip;
r1: 'r1';
r2: 'r2';
id: "[A-Za-z0-9_]+";
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
%name test

%right r1 r2

expr
    : expr r2 expr
    | expr r1 expr
	| id
    ;

whitespaces: "[\u{0009}\u{0020}]+" #skip;
r1: 'r1';
r2: 'r2';
id: "[A-Za-z0-9_]+";
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
			caption: "left and right associativities can be mixed",
			specSrc: `
%name test

%left mul div
%left add sub
%right assign

expr
    : expr add expr
    | expr sub expr
    | expr mul expr
    | expr div expr
	| expr assign expr
	| id
    ;

id: "[A-Za-z0-9_]+";
add: '+';
sub: '-';
mul: '*';
div: '/';
assign: '=';
`,
			src: `x=y=a+b*c-d/e`,
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
				),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
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

			gram, err := grammar.Compile(g, grammar.SpecifyClass(grammar.ClassSLR))
			if err != nil {
				t.Fatal(err)
			}

			treeAct := NewSyntaxTreeActionSet(gram, false, true)
			p, err := NewParser(gram, strings.NewReader(tt.src), SemanticAction(treeAct))
			if err != nil {
				t.Fatal(err)
			}

			err = p.Parse()
			if err != nil {
				t.Fatal(err)
			}

			fmt.Printf("CST:\n")
			PrintTree(os.Stdout, treeAct.CST())

			testTree(t, treeAct.CST(), tt.cst)
		})
	}
}
