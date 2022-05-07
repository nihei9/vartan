package grammar

import (
	"strings"
	"testing"

	verr "github.com/nihei9/vartan/error"
	"github.com/nihei9/vartan/spec"
)

func TestGrammarBuilderOK(t *testing.T) {
	type okTest struct {
		caption  string
		specSrc  string
		validate func(t *testing.T, g *Grammar)
	}

	nameTests := []*okTest{
		{
			caption: "the `%name` can be the same identifier as a non-terminal symbol",
			specSrc: `
%name s
s
    : foo
    ;

foo
    : 'foo';
`,
			validate: func(t *testing.T, g *Grammar) {
				expected := "s"
				if g.name != expected {
					t.Fatalf("unexpected name: want: %v, got: %v", expected, g.name)
				}
			},
		},
		{
			caption: "the `%name` can be the same identifier as a terminal symbol",
			specSrc: `
%name foo
s
    : foo
    ;

foo
    : 'foo';
`,
			validate: func(t *testing.T, g *Grammar) {
				expected := "foo"
				if g.name != expected {
					t.Fatalf("unexpected name: want: %v, got: %v", expected, g.name)
				}
			},
		},
		{
			caption: "the `%name` can be the same identifier as the error symbol",
			specSrc: `
%name error
s
    : foo
    | error
    ;

foo
    : 'foo';
`,
			validate: func(t *testing.T, g *Grammar) {
				expected := "error"
				if g.name != expected {
					t.Fatalf("unexpected name: want: %v, got: %v", expected, g.name)
				}
			},
		},
		{
			caption: "the `%name` can be the same identifier as a fragment",
			specSrc: `
%name f
s
    : foo
    ;

foo
    : "\f{f}";
fragment f
    : 'foo';
`,
			validate: func(t *testing.T, g *Grammar) {
				expected := "f"
				if g.name != expected {
					t.Fatalf("unexpected name: want: %v, got: %v", expected, g.name)
				}
			},
		},
	}

	modeTests := []*okTest{
		{
			caption: "a `#mode` can be the same identifier as a non-terminal symbol",
			specSrc: `
%name test
s
    : foo bar
    ;

foo #push s
    : 'foo';
bar #mode s
    : 'bar';
`,
			validate: func(t *testing.T, g *Grammar) {
				kind := "bar"
				expectedMode := "s"
				for _, e := range g.lexSpec.Entries {
					if e.Kind.String() == kind && e.Modes[0].String() == expectedMode {
						return
					}
				}
				t.Fatalf("symbol having expected mode was not found: want: %v #mode %v", kind, expectedMode)
			},
		},
		{
			caption: "a `#mode` can be the same identifier as a terminal symbol",
			specSrc: `
%name test
s
    : foo bar
    ;

foo #push bar
    : 'foo';
bar #mode bar
    : 'bar';
`,
			validate: func(t *testing.T, g *Grammar) {
				kind := "bar"
				expectedMode := "bar"
				for _, e := range g.lexSpec.Entries {
					if e.Kind.String() == kind && e.Modes[0].String() == expectedMode {
						return
					}
				}
				t.Fatalf("symbol having expected mode was not found: want: %v #mode %v", kind, expectedMode)
			},
		},
		{
			caption: "a `#mode` can be the same identifier as the error symbol",
			specSrc: `
%name test
s
    : foo bar
    | error
    ;

foo #push error
    : 'foo';
bar #mode error
    : 'bar';
`,
			validate: func(t *testing.T, g *Grammar) {
				kind := "bar"
				expectedMode := "error"
				for _, e := range g.lexSpec.Entries {
					if e.Kind.String() == kind && e.Modes[0].String() == expectedMode {
						return
					}
				}
				t.Fatalf("symbol having expected mode was not found: want: %v #mode %v", kind, expectedMode)
			},
		},
		{
			caption: "a `#mode` can be the same identifier as a fragment",
			specSrc: `
%name test
s
    : foo bar
    ;

foo #push f
    : "\f{f}";
bar #mode f
    : 'bar';
fragment f
    : 'foo';
`,
			validate: func(t *testing.T, g *Grammar) {
				kind := "bar"
				expectedMode := "f"
				for _, e := range g.lexSpec.Entries {
					if e.Kind.String() == kind && e.Modes[0].String() == expectedMode {
						return
					}
				}
				t.Fatalf("symbol having expected mode was not found: want: %v #mode %v", kind, expectedMode)
			},
		},
		{
			caption: "a production has the same precedence and associativity as the right-most terminal symbol",
			specSrc: `
%name test

%left foo

s
    : foo bar // This alternative has the same precedence and associativity as the right-most terminal symbol 'bar', not 'foo'.
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			validate: func(t *testing.T, g *Grammar) {
				var barPrec int
				var barAssoc assocType
				{
					s, _ := g.symbolTable.toSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.num())
				}
				var sPrec int
				var sAssoc assocType
				{
					s, _ := g.symbolTable.toSymbol("s")
					ps, _ := g.productionSet.findByLHS(s)
					sPrec = g.precAndAssoc.productionPredence(ps[0].num)
					sAssoc = g.precAndAssoc.productionAssociativity(ps[0].num)
				}
				if barPrec != precNil || barAssoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", precNil, assocTypeNil, barPrec, barAssoc)
				}
				if sPrec != barPrec || sAssoc != barAssoc {
					t.Fatalf("unexpected production precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", barPrec, barAssoc, sPrec, sAssoc)
				}
			},
		},
		{
			caption: "a production has the same precedence and associativity as the right-most terminal symbol",
			specSrc: `
%name test

%left foo
%right bar

s
    : foo bar // This alternative has the same precedence and associativity as the right-most terminal symbol 'bar'.
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			validate: func(t *testing.T, g *Grammar) {
				var barPrec int
				var barAssoc assocType
				{
					s, _ := g.symbolTable.toSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.num())
				}
				var sPrec int
				var sAssoc assocType
				{
					s, _ := g.symbolTable.toSymbol("s")
					ps, _ := g.productionSet.findByLHS(s)
					sPrec = g.precAndAssoc.productionPredence(ps[0].num)
					sAssoc = g.precAndAssoc.productionAssociativity(ps[0].num)
				}
				if barPrec != 2 || barAssoc != assocTypeRight {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 2, assocTypeRight, barPrec, barAssoc)
				}
				if sPrec != barPrec || sAssoc != barAssoc {
					t.Fatalf("unexpected production precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", barPrec, barAssoc, sPrec, sAssoc)
				}
			},
		},
		{
			caption: "the `#prec` directive changes only precedence, not associativity",
			specSrc: `
%name test

%left foo

s
    : foo bar #prec foo
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			validate: func(t *testing.T, g *Grammar) {
				var fooPrec int
				var fooAssoc assocType
				{
					s, _ := g.symbolTable.toSymbol("foo")
					fooPrec = g.precAndAssoc.terminalPrecedence(s.num())
					fooAssoc = g.precAndAssoc.terminalAssociativity(s.num())
				}
				var sPrec int
				var sAssoc assocType
				{
					s, _ := g.symbolTable.toSymbol("s")
					ps, _ := g.productionSet.findByLHS(s)
					sPrec = g.precAndAssoc.productionPredence(ps[0].num)
					sAssoc = g.precAndAssoc.productionAssociativity(ps[0].num)
				}
				if fooPrec != 1 || fooAssoc != assocTypeLeft {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeLeft, fooPrec, fooAssoc)
				}
				if sPrec != fooPrec || sAssoc != assocTypeNil {
					t.Fatalf("unexpected production precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", fooPrec, assocTypeNil, sPrec, sAssoc)
				}
			},
		},
		{
			caption: "the `#prec` directive changes only precedence, not associativity",
			specSrc: `
%name test

%left foo
%right bar

s
    : foo bar #prec foo
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			validate: func(t *testing.T, g *Grammar) {
				var fooPrec int
				var fooAssoc assocType
				{
					s, _ := g.symbolTable.toSymbol("foo")
					fooPrec = g.precAndAssoc.terminalPrecedence(s.num())
					fooAssoc = g.precAndAssoc.terminalAssociativity(s.num())
				}
				var barPrec int
				var barAssoc assocType
				{
					s, _ := g.symbolTable.toSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.num())
				}
				var sPrec int
				var sAssoc assocType
				{
					s, _ := g.symbolTable.toSymbol("s")
					ps, _ := g.productionSet.findByLHS(s)
					sPrec = g.precAndAssoc.productionPredence(ps[0].num)
					sAssoc = g.precAndAssoc.productionAssociativity(ps[0].num)
				}
				if fooPrec != 1 || fooAssoc != assocTypeLeft {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeLeft, fooPrec, fooAssoc)
				}
				if barPrec != 2 || barAssoc != assocTypeRight {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 2, assocTypeRight, barPrec, barAssoc)
				}
				if sPrec != fooPrec || sAssoc != barAssoc {
					t.Fatalf("unexpected production precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", fooPrec, barAssoc, sPrec, sAssoc)
				}
			},
		},
	}

	var tests []*okTest
	tests = append(tests, nameTests...)
	tests = append(tests, modeTests...)

	for _, test := range tests {
		t.Run(test.caption, func(t *testing.T) {
			ast, err := spec.Parse(strings.NewReader(test.specSrc))
			if err != nil {
				t.Fatal(err)
			}

			b := GrammarBuilder{
				AST: ast,
			}
			g, err := b.Build()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			test.validate(t, g)
		})
	}
}

func TestGrammarBuilderSpecError(t *testing.T) {
	type specErrTest struct {
		caption string
		specSrc string
		errs    []*SemanticError
	}

	prodTests := []*specErrTest{
		{
			caption: "a production `b` is unused",
			specSrc: `
%name test

a
    : foo
    ;
b
    : foo
    ;

foo
    : "foo";
`,
			errs: []*SemanticError{semErrUnusedProduction},
		},
		{
			caption: "a terminal symbol `bar` is unused",
			specSrc: `
%name test

s
    : foo
    ;

foo
    : "foo";
bar
    : "bar";
`,
			errs: []*SemanticError{semErrUnusedTerminal},
		},
		{
			caption: "a production `b` and terminal symbol `bar` is unused",
			specSrc: `
%name test

a
    : foo
    ;
b
    : bar
    ;

foo
    : "foo";
bar
    : "bar";
`,
			errs: []*SemanticError{
				semErrUnusedProduction,
				semErrUnusedTerminal,
			},
		},
		{
			caption: "a production cannot have production directives",
			specSrc: `
%name test

s #prec foo
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrInvalidProdDir},
		},
		{
			caption: "a lexical production cannot have alternative directives",
			specSrc: `
%name test

s
    : foo
    ;

foo
    : 'foo' #skip;
`,
			errs: []*SemanticError{semErrInvalidAltDir},
		},
		{
			caption: "a production directive must not be duplicated",
			specSrc: `
%name test

s
    : foo
    ;

foo #skip #skip
    : 'foo';
`,
			errs: []*SemanticError{semErrDuplicateDir},
		},
		{
			caption: "an alternative directive must not be duplicated",
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
			errs: []*SemanticError{semErrDuplicateDir},
		},
		{
			caption: "a production must not have a duplicate alternative (non-empty alternatives)",
			specSrc: `
%name test

s
    : foo
    | foo
    ;

foo
    : "foo";
`,
			errs: []*SemanticError{semErrDuplicateProduction},
		},
		{
			caption: "a production must not have a duplicate alternative (non-empty and split alternatives)",
			specSrc: `
%name test

s
    : foo
    | a
    ;
a
    : bar
    ;
s
    : foo
    ;

foo
    : "foo";
bar
    : "bar";
`,
			errs: []*SemanticError{semErrDuplicateProduction},
		},
		{
			caption: "a production must not have a duplicate alternative (empty alternatives)",
			specSrc: `
%name test

s
    : foo
    | a
    ;
a
    :
    |
    ;

foo
    : "foo";
`,
			errs: []*SemanticError{semErrDuplicateProduction},
		},
		{
			caption: "a production must not have a duplicate alternative (empty and split alternatives)",
			specSrc: `
%name test

s
    : foo
    | a
    ;
a
    :
    | foo
    ;
a
    :
    ;

foo
    : "foo";
`,
			errs: []*SemanticError{semErrDuplicateProduction},
		},
		{
			caption: "a terminal symbol and a non-terminal symbol (start symbol) are duplicates",
			specSrc: `
%name test

s
    : foo
    ;

foo
    : "foo";
s
    : "a";
`,
			errs: []*SemanticError{semErrDuplicateName},
		},
		{
			caption: "a terminal symbol and a non-terminal symbol (not start symbol) are duplicates",
			specSrc: `
%name test

s
    : foo
    | a
    ;
a
    : bar
    ;

foo
    : "foo";
bar
    : "bar";
a
    : "a";
`,
			errs: []*SemanticError{semErrDuplicateName},
		},
		{
			caption: "an invalid associativity type",
			specSrc: `
%name test

%foo

s
    : a
    ;

a
    : 'a';
`,
			errs: []*SemanticError{semErrMDInvalidName},
		},
		{
			caption: "a label must be unique in an alternative",
			specSrc: `
%name test

s
    : foo@x bar@x
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			errs: []*SemanticError{semErrDuplicateLabel},
		},
		{
			caption: "a label cannot be the same name as terminal symbols",
			specSrc: `
%name test

s
    : foo bar@foo
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			errs: []*SemanticError{semErrDuplicateLabel},
		},
		{
			caption: "a label cannot be the same name as non-terminal symbols",
			specSrc: `
%name test

s
    : foo@a
    | a
    ;
a
    : bar
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			errs: []*SemanticError{
				semErrInvalidLabel,
			},
		},
	}

	nameTests := []*specErrTest{
		{
			caption: "the `%name` is required",
			specSrc: `
s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrMDMissingName},
		},
		{
			caption: "the `%name` needs an ID parameter",
			specSrc: `
%name

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "the `%name` takes just one parameter",
			specSrc: `
%name test1 test2

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
	}

	leftTests := []*specErrTest{
		{
			caption: "the `%left` needs ID parameters",
			specSrc: `
%name test

%left

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "the `%left` cannot take an undefined symbol",
			specSrc: `
%name test

%left x

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "the `%left` cannot take a non-terminal symbol",
			specSrc: `
%name test

%left s

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "the `%left` cannot take a pattern parameter",
			specSrc: `
%name test

%left "foo"

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "the `%left` cannot take a string parameter",
			specSrc: `
%name test

%left 'foo'

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "the `%left` cannot be specified multiple times for a symbol",
			specSrc: `
%name test

%left foo foo

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDuplicateAssoc},
		},
		{
			caption: "a symbol cannot have different precedence",
			specSrc: `
%name test

%left foo
%left foo

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDuplicateAssoc},
		},
		{
			caption: "a symbol cannot have different associativity",
			specSrc: `
%name test

%right foo
%left foo

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDuplicateAssoc},
		},
	}

	rightTests := []*specErrTest{
		{
			caption: "the `%right` needs ID parameters",
			specSrc: `
%name test

%right

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "the `%right` cannot take an undefined symbol",
			specSrc: `
%name test

%right x

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "the `%right` cannot take a non-terminal symbol",
			specSrc: `
%name test

%right s

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "the `%right` cannot take a pattern parameter",
			specSrc: `
%name test

%right "foo"

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "the `%right` cannot take a string parameter",
			specSrc: `
%name test

%right 'foo'

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "the `%right` cannot be specified multiple times for a symbol",
			specSrc: `
%name test

%right foo foo

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDuplicateAssoc},
		},
		{
			caption: "a symbol cannot have different precedence",
			specSrc: `
%name test

%right foo
%right foo

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDuplicateAssoc},
		},
		{
			caption: "a symbol cannot have different associativity",
			specSrc: `
%name test

%left foo
%right foo

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDuplicateAssoc},
		},
	}

	errorSymTests := []*specErrTest{
		{
			caption: "cannot use the error symbol as a non-terminal symbol",
			specSrc: `
%name test

s
    : error
    ;
error
    : foo
    ;

foo: 'foo';
`,
			errs: []*SemanticError{
				semErrErrSymIsReserved,
				semErrDuplicateName,
			},
		},
		{
			caption: "cannot use the error symbol as a terminal symbol",
			specSrc: `
%name test

s
    : error
    ;

error: 'error';
`,
			errs: []*SemanticError{semErrErrSymIsReserved},
		},
		{
			caption: "cannot use the error symbol as a terminal symbol, even if given the skip directive",
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
			errs: []*SemanticError{semErrErrSymIsReserved},
		},
	}

	astDirTests := []*specErrTest{
		{
			caption: "the `#ast` directive needs ID or label prameters",
			specSrc: `
%name test

s
    : foo #ast
    ;

foo
    : "foo";
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#ast` directive cannot take a pattern parameter",
			specSrc: `
%name test

s
    : foo #ast "foo"
    ;

foo
    : "foo";
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#ast` directive cannot take a string parameter",
			specSrc: `
%name test

s
    : foo #ast 'foo'
    ;

foo
    : "foo";
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "a parameter of the `#ast` directive must be either a symbol or a label in an alternative",
			specSrc: `
%name test

s
    : foo bar #ast foo x
    ;

foo
    : "foo";
bar
    : "bar";
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "a symbol in a different alternative cannot be a parameter of the `#ast` directive",
			specSrc: `
%name test

s
    : foo #ast bar
    | bar
    ;

foo
    : "foo";
bar
    : "bar";
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "a label in a different alternative cannot be a parameter of the `#ast` directive",
			specSrc: `
%name test

s
    : foo #ast b
    | bar@b
    ;

foo
    : "foo";
bar
    : "bar";
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "a symbol can appear in the `#ast` directive only once",
			specSrc: `
%name test

s
    : foo #ast foo foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDuplicateElem},
		},
		{
			caption: "a label can appear in the `#ast` directive only once",
			specSrc: `
%name test

s
    : foo@x #ast x x
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDuplicateElem},
		},
		{
			caption: "a symbol can appear in the `#ast` directive only once, even if the symbol has a label",
			specSrc: `
%name test

s
    : foo@x #ast foo x
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDuplicateElem},
		},
		{
			caption: "symbol `foo` is ambiguous because it appears in an alternative twice",
			specSrc: `
%name test

s
    : foo foo #ast foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrAmbiguousElem},
		},
		{
			caption: "symbol `foo` is ambiguous because it appears in an alternative twice, even if one of them has a label",
			specSrc: `
%name test

s
    : foo@x foo #ast foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrAmbiguousElem},
		},
		{
			caption: "the expansion operator cannot be applied to a terminal symbol",
			specSrc: `
%name test

s
    : foo #ast foo...
    ;

foo
    : "foo";
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the expansion operator cannot be applied to a pattern",
			specSrc: `
%name test

s
    : foo "bar"@b #ast foo b...
    ;

foo
    : "foo";
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the expansion operator cannot be applied to a string",
			specSrc: `
%name test

s
    : foo 'bar'@b #ast foo b...
    ;

foo
    : "foo";
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
	}

	precDirTests := []*specErrTest{
		{
			caption: "the `#prec` directive needs an ID parameter",
			specSrc: `
%name test

s
    : foo #prec
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take an undefined symbol",
			specSrc: `
%name test

s
    : foo #prec x
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take a non-terminal symbol",
			specSrc: `
%name test

s
    : a #prec b
    | b
    ;
a
    : foo
    ;
b
    : bar
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take a pattern parameter",
			specSrc: `
%name test

s
    : foo #prec "foo"
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take a string parameter",
			specSrc: `
%name test

s
    : foo #prec 'foo'
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "a symbol the `#prec` directive takes must be given precedence explicitly",
			specSrc: `
%name test

s
    : foo bar #prec foo
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			errs: []*SemanticError{semErrUndefinedPrec},
		},
	}

	recoverDirTests := []*specErrTest{
		{
			caption: "the `#recover` directive cannot take an ID parameter",
			specSrc: `
%name test

%name test

s
    : foo #recover foo
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#recover` directive cannot take a pattern parameter",
			specSrc: `
%name test

%name test

s
    : foo #recover "foo"
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#recover` directive cannot take a string parameter",
			specSrc: `
%name test

%name test

s
    : foo #recover 'foo'
    ;

foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
	}

	fragmentTests := []*specErrTest{
		{
			caption: "a production cannot contain a fragment",
			specSrc: `
%name test

s
    : f
    ;

fragment f
    : 'fragment';
`,
			errs: []*SemanticError{semErrUndefinedSym},
		},
		{
			caption: "fragments cannot be duplicated",
			specSrc: `
%name test

s
    : foo
    ;

foo
    : "\f{f}";
fragment f
    : 'fragment 1';
fragment f
    : 'fragment 2';
`,
			errs: []*SemanticError{semErrDuplicateFragment},
		},
	}

	aliasDirTests := []*specErrTest{
		{
			caption: "the `#alias` directive needs a string parameter",
			specSrc: `
%name test

s
    : foo
    ;

foo #alias
    : 'foo';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#alias` directive takes just one string parameter",
			specSrc: `
%name test

s
    : foo
    ;

foo #alias 'Foo' 'FOO'
    : 'foo';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#alias` directive cannot take an ID parameter",
			specSrc: `
%name test

s
    : foo
    ;

foo #alias Foo
    : 'foo';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#alias` directive cannot take a pattern parameter",
			specSrc: `
%name test

s
    : foo
    ;

foo #alias "Foo"
    : 'foo';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
	}

	modeTests := []*specErrTest{
		{
			caption: "the `#mode` directive needs an ID parameter",
			specSrc: `
%name test

s
    : foo bar
    ;

foo #push mode_1
    : 'foo';
bar #mode
    : 'bar';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#mode` directive cannot take a pattern parameter",
			specSrc: `
%name test

s
    : foo bar
    ;

foo #push mode_1
    : 'foo';
bar #mode "mode_1"
    : 'bar';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#mode` directive cannot take a string parameter",
			specSrc: `
%name test

s
    : foo bar
    ;

foo #push mode_1
    : 'foo';
bar #mode 'mode_1'
    : 'bar';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
	}

	pushTests := []*specErrTest{
		{
			caption: "the `#push` directive needs an ID parameter",
			specSrc: `
%name test

s
    : foo bar
    ;

foo #push
    : 'foo';
bar #mode mode_1
    : 'bar';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#push` directive takes just one ID parameter",
			specSrc: `
%name test

s
    : foo bar
    ;

foo #push mode_1 mode_2
    : 'foo';
bar #mode mode_1
    : 'bar';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#push` directive cannot take a pattern parameter",
			specSrc: `
%name test

s
    : foo bar
    ;

foo #push "mode_1"
    : 'foo';
bar #mode mode_1
    : 'bar';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#push` directive cannot take a string parameter",
			specSrc: `
%name test

s
    : foo bar
    ;

foo #push 'mode_1'
    : 'foo';
bar #mode mode_1
    : 'bar';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
	}

	popTests := []*specErrTest{
		{
			caption: "the `#pop` directive cannot take an ID parameter",
			specSrc: `
%name test

s
    : foo bar baz
    ;

foo #push mode_1
    : 'foo';
bar #mode mode_1
    : 'bar';
baz #pop mode_1
    : 'baz';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#pop` directive cannot take a pattern parameter",
			specSrc: `
%name test

s
    : foo bar baz
    ;

foo #push mode_1
    : 'foo';
bar #mode mode_1
    : 'bar';
baz #pop "mode_1"
    : 'baz';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#pop` directive cannot take a string parameter",
			specSrc: `
%name test

s
    : foo bar baz
    ;

foo #push mode_1
    : 'foo';
bar #mode mode_1
    : 'bar';
baz #pop 'mode_1'
    : 'baz';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
	}

	skipDirTests := []*specErrTest{
		{
			caption: "the `#skip` directive cannot take an ID parameter",
			specSrc: `
%name test

s
    : foo bar
    ;

foo #skip bar
    : 'foo';
bar
    : 'bar';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#skip` directive cannot take a pattern parameter",
			specSrc: `
%name test

s
    : foo bar
    ;

foo #skip "bar"
    : 'foo';
bar
    : 'bar';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#skip` directive cannot take a string parameter",
			specSrc: `
%name test

s
    : foo bar
    ;

foo #skip 'bar'
    : 'foo';
bar
    : 'bar';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "a terminal symbol used in productions cannot have the skip directive",
			specSrc: `
%name test

s
    : foo bar
    ;

foo #skip
    : 'foo';
bar
    : 'bar';
`,
			errs: []*SemanticError{semErrTermCannotBeSkipped},
		},
	}

	var tests []*specErrTest
	tests = append(tests, prodTests...)
	tests = append(tests, nameTests...)
	tests = append(tests, leftTests...)
	tests = append(tests, rightTests...)
	tests = append(tests, errorSymTests...)
	tests = append(tests, astDirTests...)
	tests = append(tests, precDirTests...)
	tests = append(tests, recoverDirTests...)
	tests = append(tests, fragmentTests...)
	tests = append(tests, aliasDirTests...)
	tests = append(tests, modeTests...)
	tests = append(tests, pushTests...)
	tests = append(tests, popTests...)
	tests = append(tests, skipDirTests...)
	for _, test := range tests {
		t.Run(test.caption, func(t *testing.T) {
			ast, err := spec.Parse(strings.NewReader(test.specSrc))
			if err != nil {
				t.Fatal(err)
			}

			b := GrammarBuilder{
				AST: ast,
			}
			_, err = b.Build()
			if err == nil {
				t.Fatal("an expected error didn't occur")
			}
			specErrs, ok := err.(verr.SpecErrors)
			if !ok {
				t.Fatalf("unexpected error type: want: %T, got: %T: %v", verr.SpecErrors{}, err, err)
			}
			if len(specErrs) != len(test.errs) {
				t.Fatalf("unexpected spec error count: want: %+v, got: %+v", test.errs, specErrs)
			}
			for _, expected := range test.errs {
				for _, actual := range specErrs {
					if actual.Cause == expected {
						return
					}
				}
			}
			t.Fatalf("an expected spec error didn't occur: want: %v, got: %+v", test.errs, specErrs)
		})
	}
}
