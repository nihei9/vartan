package grammar

import (
	"strings"
	"testing"

	verr "github.com/nihei9/vartan/error"
	"github.com/nihei9/vartan/spec"
)

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
    : foo;
foo: "foo";
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
foo: "foo";
bar: "bar";
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
foo: "foo";
bar: "bar";
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

foo: 'foo';
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

foo: 'foo' #skip;
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
foo: "foo";
`,
			errs: []*SemanticError{semErrDuplicateProduction},
		},
		{
			caption: "a production must not have a duplicate alternative (non-empty and split alternatives)",
			specSrc: `
%name test

a
    : foo
    | b
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
			errs: []*SemanticError{semErrDuplicateProduction},
		},
		{
			caption: "a production must not have a duplicate alternative (empty alternatives)",
			specSrc: `
%name test

a
    : foo
    | b
    ;
b
    :
    |
    ;
foo: "foo";
`,
			errs: []*SemanticError{semErrDuplicateProduction},
		},
		{
			caption: "a production must not have a duplicate alternative (empty and split alternatives)",
			specSrc: `
%name test

a
    : foo
    | b
    ;
b
    :
    | foo
    ;
b
    :
    ;
foo: "foo";
`,
			errs: []*SemanticError{semErrDuplicateProduction},
		},
		{
			caption: "a terminal symbol and a non-terminal symbol (start symbol) are duplicates",
			specSrc: `
%name test

a
    : foo
    ;
foo: "foo";
a: "a";
`,
			errs: []*SemanticError{semErrDuplicateName},
		},
		{
			caption: "a terminal symbol and a non-terminal symbol (not start symbol) are duplicates",
			specSrc: `
%name test

a
    : foo
    | b
    ;
b
    : bar
    ;
foo: "foo";
bar: "bar";
b: "a";
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

a: 'a';
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

foo: 'foo';
bar: 'bar';
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

foo: 'foo';
bar: 'bar';
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

foo: 'foo';
bar: 'bar';
`,
			errs: []*SemanticError{
				semErrInvalidLabel,
			},
		},
	}

	nameTests := []*specErrTest{
		{
			caption: "the `%name` is missing",
			specSrc: `
a
    : foo
    ;
foo: "foo";
`,
			errs: []*SemanticError{semErrMDMissingName},
		},
		{
			caption: "the `%name` needs a parameter",
			specSrc: `
%name

a
    : foo
    ;
foo: "foo";
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "the `%name` takes just one parameter",
			specSrc: `
%name test foo

a
    : foo
    ;
foo: "foo";
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
	}

	assocTests := []*specErrTest{
		{
			caption: "associativity needs at least one symbol",
			specSrc: `
%name test

%left

s
    : a
    ;

a: 'a';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "associativity cannot take an undefined symbol",
			specSrc: `
%name test

%left b

s
    : a
    ;

a: 'a';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
		{
			caption: "associativity cannot take a non-terminal symbol",
			specSrc: `
%name test

%left s

s
    : a
    ;

a: 'a';
`,
			errs: []*SemanticError{semErrMDInvalidParam},
		},
	}

	errorSymTests := []*specErrTest{
		{
			caption: "cannot use the error symbol as a non-terminal symbol",
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
			errs: []*SemanticError{
				semErrErrSymIsReserved,
				semErrDuplicateName,
				// The compiler determines the symbol `bar` is unreachable because the production rule `error → bar` contains
				// a build error and the compiler doesn't recognize the production rule as a valid one.
				// This error is essentially irrelevant to this test case.
				semErrUnusedTerminal, // This error means `bar` is unreachable.
			},
		},
		{
			caption: "cannot use the error symbol as a terminal symbol",
			specSrc: `
%name test

s
    : foo
    | error
    ;

foo: 'foo';
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
			caption: "a parameter of the `#ast` directive must be either a symbol or a label in an alternative",
			specSrc: `
%name test

s
    : foo bar #ast foo x
    ;
foo: "foo";
bar: "bar";
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
foo: "foo";
bar: "bar";
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
foo: "foo";
bar: "bar";
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the expansion operator cannot be applied to a terminal symbol",
			specSrc: `
%name test

s
    : foo #ast foo...
    ;
foo: "foo";
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
foo: "foo";
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
foo: "foo";
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
    : a #prec
    ;

a: 'a';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take an unknown symbol",
			specSrc: `
%name test

s
    : a #prec foo
    ;

a: 'a';
`,
			errs: []*SemanticError{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take a non-terminal symbol",
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
			errs: []*SemanticError{semErrDirInvalidParam},
		},
	}

	recoverDirTests := []*specErrTest{
		{
			caption: "the `#recover` directive cannot take a parameter",
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
			errs: []*SemanticError{semErrDirInvalidParam},
		},
	}

	skipDirTests := []*specErrTest{
		{
			caption: "a terminal symbol used in productions cannot have the skip directive",
			specSrc: `
%name test

a
    : foo
    ;

foo #skip
    : "foo";
`,
			errs: []*SemanticError{semErrTermCannotBeSkipped},
		},
	}

	var tests []*specErrTest
	tests = append(tests, prodTests...)
	tests = append(tests, nameTests...)
	tests = append(tests, assocTests...)
	tests = append(tests, errorSymTests...)
	tests = append(tests, astDirTests...)
	tests = append(tests, precDirTests...)
	tests = append(tests, recoverDirTests...)
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
