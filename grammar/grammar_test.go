package grammar

import (
	"strings"
	"testing"

	verr "github.com/nihei9/vartan/error"
	"github.com/nihei9/vartan/spec/grammar/parser"
)

func TestGrammarBuilderOK(t *testing.T) {
	type okTest struct {
		caption  string
		specSrc  string
		validate func(t *testing.T, g *Grammar)
	}

	nameTests := []*okTest{
		{
			caption: "the `#name` can be the same identifier as a non-terminal symbol",
			specSrc: `
#name s;

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
			caption: "the `#name` can be the same identifier as a terminal symbol",
			specSrc: `
#name foo;

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
			caption: "the `#name` can be the same identifier as the error symbol",
			specSrc: `
#name error;

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
			caption: "the `#name` can be the same identifier as a fragment",
			specSrc: `
#name f;

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
#name test;

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
#name test;

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
#name test;

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
#name test;

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
	}

	precTests := []*okTest{
		{
			caption: "a `#prec` allows the empty directive group",
			specSrc: `
#name test;

#prec ();

s
    : foo
    ;

foo
    : 'foo';
`,
		},
		{
			caption: "a `#left` directive gives a precedence and the left associativity to specified terminal symbols",
			specSrc: `
#name test;

#prec (
    #left foo bar
);

s
    : foo bar baz
    ;

foo
    : 'foo';
bar
    : 'bar';
baz
    : 'baz';
`,
			validate: func(t *testing.T, g *Grammar) {
				var fooPrec int
				var fooAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("foo")
					fooPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					fooAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if fooPrec != 1 || fooAssoc != assocTypeLeft {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeLeft, fooPrec, fooAssoc)
				}
				var barPrec int
				var barAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if barPrec != 1 || barAssoc != assocTypeLeft {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeLeft, barPrec, barAssoc)
				}
				var bazPrec int
				var bazAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("baz")
					bazPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					bazAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if bazPrec != precNil || bazAssoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", precNil, assocTypeNil, bazPrec, bazAssoc)
				}
			},
		},
		{
			caption: "a `#right` directive gives a precedence and the right associativity to specified terminal symbols",
			specSrc: `
#name test;

#prec (
    #right foo bar
);

s
    : foo bar baz
    ;

foo
    : 'foo';
bar
    : 'bar';
baz
    : 'baz';
`,
			validate: func(t *testing.T, g *Grammar) {
				var fooPrec int
				var fooAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("foo")
					fooPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					fooAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if fooPrec != 1 || fooAssoc != assocTypeRight {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeRight, fooPrec, fooAssoc)
				}
				var barPrec int
				var barAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if barPrec != 1 || barAssoc != assocTypeRight {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeRight, barPrec, barAssoc)
				}
				var bazPrec int
				var bazAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("baz")
					bazPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					bazAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if bazPrec != precNil || bazAssoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", precNil, assocTypeNil, bazPrec, bazAssoc)
				}
			},
		},
		{
			caption: "an `#assign` directive gives only a precedence to specified terminal symbols",
			specSrc: `
#name test;

#prec (
    #assign foo bar
);

s
    : foo bar baz
    ;

foo
    : 'foo';
bar
    : 'bar';
baz
    : 'baz';
`,
			validate: func(t *testing.T, g *Grammar) {
				var fooPrec int
				var fooAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("foo")
					fooPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					fooAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if fooPrec != 1 || fooAssoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeNil, fooPrec, fooAssoc)
				}
				var barPrec int
				var barAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if barPrec != 1 || barAssoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeNil, barPrec, barAssoc)
				}
				var bazPrec int
				var bazAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("baz")
					bazPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					bazAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if bazPrec != precNil || bazAssoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", precNil, assocTypeNil, bazPrec, bazAssoc)
				}
			},
		},
		{
			caption: "a production has the same precedence and associativity as the right-most terminal symbol",
			specSrc: `
#name test;

#prec (
    #left foo
);

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
					s, _ := g.symbolTable.ToSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				var sPrec int
				var sAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("s")
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
#name test;

#prec (
    #left foo
    #right bar
);

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
					s, _ := g.symbolTable.ToSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				var sPrec int
				var sAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("s")
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
			caption: "even if a non-terminal symbol apears to a terminal symbol, a production inherits precedence and associativity from the right-most terminal symbol, not from the non-terminal symbol",
			specSrc: `
#name test;

#prec (
    #left foo
    #right bar
);

s
    : foo a // This alternative has the same precedence and associativity as the right-most terminal symbol 'foo', not 'a'.
    ;
a
    : bar
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
					s, _ := g.symbolTable.ToSymbol("foo")
					fooPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					fooAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				var barPrec int
				var barAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				var aPrec int
				var aAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("a")
					ps, _ := g.productionSet.findByLHS(s)
					aPrec = g.precAndAssoc.productionPredence(ps[0].num)
					aAssoc = g.precAndAssoc.productionAssociativity(ps[0].num)
				}
				var sPrec int
				var sAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("s")
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
				if aPrec != barPrec || aAssoc != barAssoc {
					t.Fatalf("unexpected production precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", barPrec, barAssoc, aPrec, aAssoc)
				}
				if sPrec != fooPrec || sAssoc != fooAssoc {
					t.Fatalf("unexpected production precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", fooPrec, fooAssoc, sPrec, sAssoc)
				}
			},
		},
		{
			caption: "each alternative in the same production can have its own precedence and associativity",
			specSrc: `
#name test;

#prec (
    #left foo
    #right bar
    #assign baz
);

s
    : foo
    | bar
    | baz
    | bra
    ;

foo
    : 'foo';
bar
    : 'bar';
baz
    : 'baz';
bra
    : 'bra';
`,
			validate: func(t *testing.T, g *Grammar) {
				var alt1Prec int
				var alt1Assoc assocType
				var alt2Prec int
				var alt2Assoc assocType
				var alt3Prec int
				var alt3Assoc assocType
				var alt4Prec int
				var alt4Assoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("s")
					ps, _ := g.productionSet.findByLHS(s)
					alt1Prec = g.precAndAssoc.productionPredence(ps[0].num)
					alt1Assoc = g.precAndAssoc.productionAssociativity(ps[0].num)
					alt2Prec = g.precAndAssoc.productionPredence(ps[1].num)
					alt2Assoc = g.precAndAssoc.productionAssociativity(ps[1].num)
					alt3Prec = g.precAndAssoc.productionPredence(ps[2].num)
					alt3Assoc = g.precAndAssoc.productionAssociativity(ps[2].num)
					alt4Prec = g.precAndAssoc.productionPredence(ps[3].num)
					alt4Assoc = g.precAndAssoc.productionAssociativity(ps[3].num)
				}
				if alt1Prec != 1 || alt1Assoc != assocTypeLeft {
					t.Fatalf("unexpected production precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeLeft, alt1Prec, alt1Assoc)
				}
				if alt2Prec != 2 || alt2Assoc != assocTypeRight {
					t.Fatalf("unexpected production precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 2, assocTypeRight, alt2Prec, alt2Assoc)
				}
				if alt3Prec != 3 || alt3Assoc != assocTypeNil {
					t.Fatalf("unexpected production precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 3, assocTypeNil, alt3Prec, alt3Assoc)
				}
				if alt4Prec != precNil || alt4Assoc != assocTypeNil {
					t.Fatalf("unexpected production precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", precNil, assocTypeNil, alt4Prec, alt4Assoc)
				}
			},
		},
		{
			caption: "when a production contains no terminal symbols, the production will not have precedence and associativiry",
			specSrc: `
#name test;

#prec (
    #left foo
);

s
    : a
    ;
a
    : foo
    ;

foo
    : 'foo';
`,
			validate: func(t *testing.T, g *Grammar) {
				var fooPrec int
				var fooAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("foo")
					fooPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					fooAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				var aPrec int
				var aAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("a")
					ps, _ := g.productionSet.findByLHS(s)
					aPrec = g.precAndAssoc.productionPredence(ps[0].num)
					aAssoc = g.precAndAssoc.productionAssociativity(ps[0].num)
				}
				var sPrec int
				var sAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("s")
					ps, _ := g.productionSet.findByLHS(s)
					sPrec = g.precAndAssoc.productionPredence(ps[0].num)
					sAssoc = g.precAndAssoc.productionAssociativity(ps[0].num)
				}
				if fooPrec != 1 || fooAssoc != assocTypeLeft {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeLeft, fooPrec, fooAssoc)
				}
				if aPrec != fooPrec || aAssoc != fooAssoc {
					t.Fatalf("unexpected production precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", fooPrec, fooAssoc, aPrec, aAssoc)
				}
				if sPrec != precNil || sAssoc != assocTypeNil {
					t.Fatalf("unexpected production precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", precNil, assocTypeNil, sPrec, sAssoc)
				}
			},
		},
		{
			caption: "the `#prec` directive applied to an alternative changes only precedence, not associativity",
			specSrc: `
#name test;

#prec (
    #left foo
);

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
					s, _ := g.symbolTable.ToSymbol("foo")
					fooPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					fooAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				var sPrec int
				var sAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("s")
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
			caption: "the `#prec` directive applied to an alternative changes only precedence, not associativity",
			specSrc: `
#name test;

#prec (
    #left foo
    #right bar
);

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
					s, _ := g.symbolTable.ToSymbol("foo")
					fooPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					fooAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				var barPrec int
				var barAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				var sPrec int
				var sAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("s")
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
				if sPrec != fooPrec || sAssoc != assocTypeNil {
					t.Fatalf("unexpected production precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", fooPrec, assocTypeNil, sPrec, sAssoc)
				}
			},
		},
		{
			caption: "an ordered symbol can appear in a `#left` directive",
			specSrc: `
#name test;

#prec (
    #left $high
    #right foo bar
    #left $low
);

s
    : foo #prec $high
    | bar #prec $low
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
					s, _ := g.symbolTable.ToSymbol("foo")
					fooPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					fooAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if fooPrec != 2 || fooAssoc != assocTypeRight {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 2, assocTypeRight, fooPrec, fooAssoc)
				}
				var barPrec int
				var barAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if barPrec != 2 || barAssoc != assocTypeRight {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 2, assocTypeRight, barPrec, barAssoc)
				}
				var alt1Prec int
				var alt1Assoc assocType
				var alt2Prec int
				var alt2Assoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("s")
					ps, _ := g.productionSet.findByLHS(s)
					alt1Prec = g.precAndAssoc.productionPredence(ps[0].num)
					alt1Assoc = g.precAndAssoc.productionAssociativity(ps[0].num)
					alt2Prec = g.precAndAssoc.productionPredence(ps[1].num)
					alt2Assoc = g.precAndAssoc.productionAssociativity(ps[1].num)
				}
				if alt1Prec != 1 || alt1Assoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeNil, alt1Prec, alt1Assoc)
				}
				if alt2Prec != 3 || alt2Assoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 3, assocTypeNil, alt2Prec, alt2Assoc)
				}
			},
		},
		{
			caption: "an ordered symbol can appear in a `#right` directive",
			specSrc: `
#name test;

#prec (
    #right $high
    #left foo bar
    #right $low
);

s
    : foo #prec $high
    | bar #prec $low
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
					s, _ := g.symbolTable.ToSymbol("foo")
					fooPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					fooAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if fooPrec != 2 || fooAssoc != assocTypeLeft {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 2, assocTypeLeft, fooPrec, fooAssoc)
				}
				var barPrec int
				var barAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if barPrec != 2 || barAssoc != assocTypeLeft {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 2, assocTypeLeft, barPrec, barAssoc)
				}
				var alt1Prec int
				var alt1Assoc assocType
				var alt2Prec int
				var alt2Assoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("s")
					ps, _ := g.productionSet.findByLHS(s)
					alt1Prec = g.precAndAssoc.productionPredence(ps[0].num)
					alt1Assoc = g.precAndAssoc.productionAssociativity(ps[0].num)
					alt2Prec = g.precAndAssoc.productionPredence(ps[1].num)
					alt2Assoc = g.precAndAssoc.productionAssociativity(ps[1].num)
				}
				if alt1Prec != 1 || alt1Assoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeNil, alt1Prec, alt1Assoc)
				}
				if alt2Prec != 3 || alt2Assoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 3, assocTypeNil, alt2Prec, alt2Assoc)
				}
			},
		},
		{
			caption: "an ordered symbol can appear in a `#assign` directive",
			specSrc: `
#name test;

#prec (
    #assign $high
    #left foo
    #right bar
    #assign $low
);

s
    : foo #prec $high
    | bar #prec $low
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
					s, _ := g.symbolTable.ToSymbol("foo")
					fooPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					fooAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if fooPrec != 2 || fooAssoc != assocTypeLeft {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 2, assocTypeLeft, fooPrec, fooAssoc)
				}
				var barPrec int
				var barAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if barPrec != 3 || barAssoc != assocTypeRight {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 3, assocTypeRight, barPrec, barAssoc)
				}
				var alt1Prec int
				var alt1Assoc assocType
				var alt2Prec int
				var alt2Assoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("s")
					ps, _ := g.productionSet.findByLHS(s)
					alt1Prec = g.precAndAssoc.productionPredence(ps[0].num)
					alt1Assoc = g.precAndAssoc.productionAssociativity(ps[0].num)
					alt2Prec = g.precAndAssoc.productionPredence(ps[1].num)
					alt2Assoc = g.precAndAssoc.productionAssociativity(ps[1].num)
				}
				if alt1Prec != 1 || alt1Assoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeNil, alt1Prec, alt1Assoc)
				}
				if alt2Prec != 4 || alt2Assoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 4, assocTypeNil, alt2Prec, alt2Assoc)
				}
			},
		},
		{
			caption: "names of an ordered symbol and a terminal symbol can duplicate",
			specSrc: `
#name test;

#prec (
    #left foo bar
    #right $foo
);

s
    : foo
    | bar #prec $foo
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
					s, _ := g.symbolTable.ToSymbol("foo")
					fooPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					fooAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				var barPrec int
				var barAssoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if fooPrec != 1 || fooAssoc != assocTypeLeft {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeLeft, fooPrec, fooAssoc)
				}
				if barPrec != 1 || barAssoc != assocTypeLeft {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeLeft, barPrec, barAssoc)
				}
				var alt1Prec int
				var alt1Assoc assocType
				var alt2Prec int
				var alt2Assoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("s")
					ps, _ := g.productionSet.findByLHS(s)
					alt1Prec = g.precAndAssoc.productionPredence(ps[0].num)
					alt1Assoc = g.precAndAssoc.productionAssociativity(ps[0].num)
					alt2Prec = g.precAndAssoc.productionPredence(ps[1].num)
					alt2Assoc = g.precAndAssoc.productionAssociativity(ps[1].num)
				}
				if alt1Prec != fooPrec || alt1Assoc != fooAssoc {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", fooPrec, fooAssoc, alt1Prec, alt1Assoc)
				}
				if alt2Prec != 2 || alt2Assoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 2, assocTypeNil, alt2Prec, alt2Assoc)
				}
			},
		},
		{
			caption: "names of an ordered symbol and a non-terminal symbol can duplicate",
			specSrc: `
#name test;

#prec (
    #left foo bar
    #right $a
);

s
    : a
    | bar #prec $a
    ;
a
    : foo
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
					s, _ := g.symbolTable.ToSymbol("bar")
					barPrec = g.precAndAssoc.terminalPrecedence(s.Num())
					barAssoc = g.precAndAssoc.terminalAssociativity(s.Num())
				}
				if barPrec != 1 || barAssoc != assocTypeLeft {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 1, assocTypeLeft, barPrec, barAssoc)
				}
				var alt1Prec int
				var alt1Assoc assocType
				var alt2Prec int
				var alt2Assoc assocType
				{
					s, _ := g.symbolTable.ToSymbol("s")
					ps, _ := g.productionSet.findByLHS(s)
					alt1Prec = g.precAndAssoc.productionPredence(ps[0].num)
					alt1Assoc = g.precAndAssoc.productionAssociativity(ps[0].num)
					alt2Prec = g.precAndAssoc.productionPredence(ps[1].num)
					alt2Assoc = g.precAndAssoc.productionAssociativity(ps[1].num)
				}
				if alt1Prec != precNil || alt1Assoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", precNil, assocTypeNil, alt1Prec, alt1Assoc)
				}
				if alt2Prec != 2 || alt2Assoc != assocTypeNil {
					t.Fatalf("unexpected terminal precedence and associativity: want: (prec: %v, assoc: %v), got: (prec: %v, assoc: %v)", 2, assocTypeNil, alt2Prec, alt2Assoc)
				}
			},
		},
	}

	var tests []*okTest
	tests = append(tests, nameTests...)
	tests = append(tests, modeTests...)
	tests = append(tests, precTests...)

	for _, test := range tests {
		t.Run(test.caption, func(t *testing.T) {
			ast, err := parser.Parse(strings.NewReader(test.specSrc))
			if err != nil {
				t.Fatal(err)
			}

			b := GrammarBuilder{
				AST: ast,
			}
			g, err := b.build()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if test.validate != nil {
				test.validate(t, g)
			}
		})
	}
}

func TestGrammarBuilderSpecError(t *testing.T) {
	type specErrTest struct {
		caption string
		specSrc string
		errs    []error
	}

	spellingInconsistenciesTests := []*specErrTest{
		{
			caption: "a spelling inconsistency appears among non-terminal symbols",
			specSrc: `
#name test;

a1
    : a_1
    ;
a_1
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrSpellingInconsistency},
		},
		{
			caption: "a spelling inconsistency appears among terminal symbols",
			specSrc: `
#name test;

s
    : foo1 foo_1
    ;

foo1
    : 'foo1';
foo_1
    : 'foo_1';
`,
			errs: []error{semErrSpellingInconsistency},
		},
		{
			caption: "a spelling inconsistency appears among non-terminal and terminal symbols",
			specSrc: `
#name test;

a1
    : a_1
    ;

a_1
    : 'a_1';
`,
			errs: []error{semErrSpellingInconsistency},
		},
		{
			caption: "a spelling inconsistency appears among ordered symbols whose precedence is the same",
			specSrc: `
#name test;

#prec (
    #assign $p1 $p_1
);

s
    : foo #prec $p1
    | bar #prec $p_1
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrSpellingInconsistency},
		},
		{
			caption: "a spelling inconsistency appears among ordered symbols whose precedence is not the same",
			specSrc: `
#name test;

#prec (
    #assign $p1
    #assign $p_1
);

s
    : foo #prec $p1
    | bar #prec $p_1
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrSpellingInconsistency},
		},
		{
			caption: "a spelling inconsistency appears among labels the same alternative contains",
			specSrc: `
#name test;

s
    : foo@l1 foo@l_1
    ;

foo
    : 'foo';
`,
			errs: []error{semErrSpellingInconsistency},
		},
		{
			caption: "a spelling inconsistency appears among labels the same production contains",
			specSrc: `
#name test;

s
    : foo@l1
    | bar@l_1
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrSpellingInconsistency},
		},
		{
			caption: "a spelling inconsistency appears among labels different productions contain",
			specSrc: `
#name test;

s
    : foo@l1
    ;
a
    : bar@l_1
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrSpellingInconsistency},
		},
	}

	prodTests := []*specErrTest{
		{
			caption: "a production `b` is unused",
			specSrc: `
#name test;

a
    : foo
    ;
b
    : foo
    ;

foo
    : "foo";
`,
			errs: []error{semErrUnusedProduction},
		},
		{
			caption: "a terminal symbol `bar` is unused",
			specSrc: `
#name test;

s
    : foo
    ;

foo
    : "foo";
bar
    : "bar";
`,
			errs: []error{semErrUnusedTerminal},
		},
		{
			caption: "a production `b` and terminal symbol `bar` is unused",
			specSrc: `
#name test;

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
			errs: []error{
				semErrUnusedProduction,
				semErrUnusedTerminal,
			},
		},
		{
			caption: "a production cannot have production directives",
			specSrc: `
#name test;

s #prec foo
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrInvalidProdDir},
		},
		{
			caption: "a lexical production cannot have alternative directives",
			specSrc: `
#name test;

s
    : foo
    ;

foo
    : 'foo' #skip;
`,
			errs: []error{semErrInvalidAltDir},
		},
		{
			caption: "a production directive must not be duplicated",
			specSrc: `
#name test;

s
    : foo
    ;

foo #skip #skip
    : 'foo';
`,
			errs: []error{semErrDuplicateDir},
		},
		{
			caption: "an alternative directive must not be duplicated",
			specSrc: `
#name test;

s
    : foo bar #ast foo bar #ast foo bar
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrDuplicateDir},
		},
		{
			caption: "a production must not have a duplicate alternative (non-empty alternatives)",
			specSrc: `
#name test;

s
    : foo
    | foo
    ;

foo
    : "foo";
`,
			errs: []error{semErrDuplicateProduction},
		},
		{
			caption: "a production must not have a duplicate alternative (non-empty and split alternatives)",
			specSrc: `
#name test;

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
			errs: []error{semErrDuplicateProduction},
		},
		{
			caption: "a production must not have a duplicate alternative (empty alternatives)",
			specSrc: `
#name test;

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
			errs: []error{semErrDuplicateProduction},
		},
		{
			caption: "a production must not have a duplicate alternative (empty and split alternatives)",
			specSrc: `
#name test;

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
			errs: []error{semErrDuplicateProduction},
		},
		{
			caption: "a terminal symbol and a non-terminal symbol (start symbol) are duplicates",
			specSrc: `
#name test;

s
    : foo
    ;

foo
    : "foo";
s
    : "a";
`,
			errs: []error{semErrDuplicateName},
		},
		{
			caption: "a terminal symbol and a non-terminal symbol (not start symbol) are duplicates",
			specSrc: `
#name test;

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
			errs: []error{semErrDuplicateName},
		},
		{
			caption: "an invalid top-level directive",
			specSrc: `
#name test;

#foo;

s
    : a
    ;

a
    : 'a';
`,
			errs: []error{semErrDirInvalidName},
		},
		{
			caption: "a label must be unique in an alternative",
			specSrc: `
#name test;

s
    : foo@x bar@x
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrDuplicateLabel},
		},
		{
			caption: "a label cannot be the same name as terminal symbols",
			specSrc: `
#name test;

s
    : foo bar@foo
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrDuplicateLabel},
		},
		{
			caption: "a label cannot be the same name as non-terminal symbols",
			specSrc: `
#name test;

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
			errs: []error{
				semErrInvalidLabel,
			},
		},
	}

	nameDirTests := []*specErrTest{
		{
			caption: "the `#name` directive is required",
			specSrc: `
s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrNoGrammarName},
		},
		{
			caption: "the `#name` directive needs an ID parameter",
			specSrc: `
#name;

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#name` directive cannot take a pattern parameter",
			specSrc: `
#name "test";

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#name` directive cannot take a string parameter",
			specSrc: `
#name 'test';

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#name` directive takes just one parameter",
			specSrc: `
#name test1 test2;

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
	}

	precDirTests := []*specErrTest{
		{
			caption: "the `#prec` directive needs a directive group parameter",
			specSrc: `
#name test;

#prec;

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take an ID parameter",
			specSrc: `
#name test;

#prec foo;

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take an ordered symbol parameter",
			specSrc: `
#name test;

#prec $x;

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take a pattern parameter",
			specSrc: `
#name test;

#prec "foo";

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take a string parameter",
			specSrc: `
#name test;

#prec 'foo';

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive takes just one directive group parameter",
			specSrc: `
#name test;

#prec () ();

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
	}

	leftDirTests := []*specErrTest{
		{
			caption: "the `#left` directive needs ID parameters",
			specSrc: `
#name test;

#prec (
    #left
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#left` directive cannot be applied to an error symbol",
			specSrc: `
#name test;

#prec (
    #left error
);

s
    : foo semi_colon
    | error semi_colon
    ;

foo
    : 'foo';
semi_colon
    : ';';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#left` directive cannot take an undefined symbol",
			specSrc: `
#name test;

#prec (
    #left x
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#left` directive cannot take a non-terminal symbol",
			specSrc: `
#name test;

#prec (
    #left s
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#left` directive cannot take a pattern parameter",
			specSrc: `
#name test;

#prec (
    #left "foo"
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#left` directive cannot take a string parameter",
			specSrc: `
#name test;

#prec (
    #left 'foo'
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#left` directive cannot take a directive parameter",
			specSrc: `
#name test;

#prec (
    #left ()
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#left` dirctive cannot be specified multiple times for a terminal symbol",
			specSrc: `
#name test;

#prec (
    #left foo foo
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "the `#left` dirctive cannot be specified multiple times for an ordered symbol",
			specSrc: `
#name test;

#prec (
    #left $x $x
);

s
    : foo #prec $x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "a terminal symbol cannot have different precedence",
			specSrc: `
#name test;

#prec (
    #left foo
    #left foo
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "an ordered symbol cannot have different precedence",
			specSrc: `
#name test;

#prec (
    #left $x
    #left $x
);

s
    : foo #prec $x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "a terminal symbol cannot have different associativity",
			specSrc: `
#name test;

#prec (
    #right foo
    #left foo
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "an ordered symbol cannot have different associativity",
			specSrc: `
#name test;

#prec (
    #right $x
    #left $x
);

s
    : foo #prec $x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
	}

	rightDirTests := []*specErrTest{
		{
			caption: "the `#right` directive needs ID parameters",
			specSrc: `
#name test;

#prec (
    #right
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#right` directive cannot be applied to an error symbol",
			specSrc: `
#name test;

#prec (
    #right error
);

s
    : foo semi_colon
    | error semi_colon
    ;

foo
    : 'foo';
semi_colon
    : ';';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#right` directive cannot take an undefined symbol",
			specSrc: `
#name test;

#prec (
    #right x
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#right` directive cannot take a non-terminal symbol",
			specSrc: `
#name test;

#prec (
    #right s
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#right` directive cannot take a pattern parameter",
			specSrc: `
#name test;

#prec (
    #right "foo"
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#right` directive cannot take a string parameter",
			specSrc: `
#name test;

#prec (
    #right 'foo'
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#right` directive cannot take a directive group parameter",
			specSrc: `
#name test;

#prec (
    #right ()
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#right` directive cannot be specified multiple times for a terminal symbol",
			specSrc: `
#name test;

#prec (
    #right foo foo
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "the `#right` directive cannot be specified multiple times for an ordered symbol",
			specSrc: `
#name test;

#prec (
    #right $x $x
);

s
    : foo #prec $x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "a terminal symbol cannot have different precedence",
			specSrc: `
#name test;

#prec (
    #right foo
    #right foo
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "an ordered symbol cannot have different precedence",
			specSrc: `
#name test;

#prec (
    #right $x
    #right $x
);

s
    : foo #prec $x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "a terminal symbol cannot have different associativity",
			specSrc: `
#name test;

#prec (
    #left foo
    #right foo
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "an ordered symbol cannot have different associativity",
			specSrc: `
#name test;

#prec (
    #left $x
    #right $x
);

s
    : foo #prec $x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
	}

	assignDirTests := []*specErrTest{
		{
			caption: "the `#assign` directive needs ID parameters",
			specSrc: `
#name test;

#prec (
    #assign
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#assign` directive cannot be applied to an error symbol",
			specSrc: `
#name test;

#prec (
    #assign error
);

s
    : foo semi_colon
    | error semi_colon
    ;

foo
    : 'foo';
semi_colon
    : ';';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#assign` directive cannot take an undefined symbol",
			specSrc: `
#name test;

#prec (
    #assign x
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#assign` directive cannot take a non-terminal symbol",
			specSrc: `
#name test;

#prec (
    #assign s
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#assign` directive cannot take a pattern parameter",
			specSrc: `
#name test;

#prec (
    #assign "foo"
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#assign` directive cannot take a string parameter",
			specSrc: `
#name test;

#prec (
    #assign 'foo'
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#assign` directive cannot take a directive parameter",
			specSrc: `
#name test;

#prec (
    #assign ()
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#assign` dirctive cannot be specified multiple times for a terminal symbol",
			specSrc: `
#name test;

#prec (
    #assign foo foo
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "the `#assign` dirctive cannot be specified multiple times for an ordered symbol",
			specSrc: `
#name test;

#prec (
    #assign $x $x
);

s
    : foo #prec $x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "a terminal symbol cannot have different precedence",
			specSrc: `
#name test;

#prec (
    #assign foo
    #assign foo
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "an ordered symbol cannot have different precedence",
			specSrc: `
#name test;

#prec (
    #assign $x
    #assign $x
);

s
    : foo #prec $x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "a terminal symbol cannot have different associativity",
			specSrc: `
#name test;

#prec (
    #assign foo
    #left foo
);

s
    : foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
		{
			caption: "an ordered symbol cannot have different associativity",
			specSrc: `
#name test;

#prec (
    #assign $x
    #left $x
);

s
    : foo #prec $x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateAssoc},
		},
	}

	errorSymTests := []*specErrTest{
		{
			caption: "cannot use the error symbol as a non-terminal symbol",
			specSrc: `
#name test;

s
    : error
    ;
error
    : foo
    ;

foo: 'foo';
`,
			errs: []error{
				semErrErrSymIsReserved,
				semErrDuplicateName,
			},
		},
		{
			caption: "cannot use the error symbol as a terminal symbol",
			specSrc: `
#name test;

s
    : error
    ;

error: 'error';
`,
			errs: []error{semErrErrSymIsReserved},
		},
		{
			caption: "cannot use the error symbol as a terminal symbol, even if given the skip directive",
			specSrc: `
#name test;

s
    : foo
    ;

foo
    : 'foo';
error #skip
    : 'error';
`,
			errs: []error{semErrErrSymIsReserved},
		},
	}

	astDirTests := []*specErrTest{
		{
			caption: "the `#ast` directive needs ID or label prameters",
			specSrc: `
#name test;

s
    : foo #ast
    ;

foo
    : "foo";
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#ast` directive cannot take an ordered symbol parameter",
			specSrc: `
#name test;

#prec (
    #assign $x
);

s
    : foo #ast $x
    ;

foo
    : "foo";
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#ast` directive cannot take a pattern parameter",
			specSrc: `
#name test;

s
    : foo #ast "foo"
    ;

foo
    : "foo";
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#ast` directive cannot take a string parameter",
			specSrc: `
#name test;

s
    : foo #ast 'foo'
    ;

foo
    : "foo";
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#ast` directive cannot take a directive group parameter",
			specSrc: `
#name test;

s
    : foo #ast ()
    ;

foo
    : "foo";
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "a parameter of the `#ast` directive must be either a symbol or a label in an alternative",
			specSrc: `
#name test;

s
    : foo bar #ast foo x
    ;

foo
    : "foo";
bar
    : "bar";
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "a symbol in a different alternative cannot be a parameter of the `#ast` directive",
			specSrc: `
#name test;

s
    : foo #ast bar
    | bar
    ;

foo
    : "foo";
bar
    : "bar";
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "a label in a different alternative cannot be a parameter of the `#ast` directive",
			specSrc: `
#name test;

s
    : foo #ast b
    | bar@b
    ;

foo
    : "foo";
bar
    : "bar";
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "a symbol can appear in the `#ast` directive only once",
			specSrc: `
#name test;

s
    : foo #ast foo foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateElem},
		},
		{
			caption: "a label can appear in the `#ast` directive only once",
			specSrc: `
#name test;

s
    : foo@x #ast x x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateElem},
		},
		{
			caption: "a symbol can appear in the `#ast` directive only once, even if the symbol has a label",
			specSrc: `
#name test;

s
    : foo@x #ast foo x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDuplicateElem},
		},
		{
			caption: "symbol `foo` is ambiguous because it appears in an alternative twice",
			specSrc: `
#name test;

s
    : foo foo #ast foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrAmbiguousElem},
		},
		{
			caption: "symbol `foo` is ambiguous because it appears in an alternative twice, even if one of them has a label",
			specSrc: `
#name test;

s
    : foo@x foo #ast foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrAmbiguousElem},
		},
		{
			caption: "the expansion operator cannot be applied to a terminal symbol",
			specSrc: `
#name test;

s
    : foo #ast foo...
    ;

foo
    : "foo";
`,
			errs: []error{semErrDirInvalidParam},
		},
	}

	altPrecDirTests := []*specErrTest{
		{
			caption: "the `#prec` directive needs an ID parameter or an ordered symbol parameter",
			specSrc: `
#name test;

s
    : foo #prec
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot be applied to an error symbol",
			specSrc: `
#name test;

s
    : foo #prec error
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take an undefined symbol",
			specSrc: `
#name test;

s
    : foo #prec x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take a non-terminal symbol",
			specSrc: `
#name test;

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
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take an undefined ordered symbol parameter",
			specSrc: `
#name test;

s
    : foo #prec $x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrUndefinedOrdSym},
		},
		{
			caption: "the `#prec` directive cannot take a pattern parameter",
			specSrc: `
#name test;

s
    : foo #prec "foo"
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take a string parameter",
			specSrc: `
#name test;

s
    : foo #prec 'foo'
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#prec` directive cannot take a directive parameter",
			specSrc: `
#name test;

s
    : foo #prec ()
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "a symbol the `#prec` directive takes must be given precedence explicitly",
			specSrc: `
#name test;

s
    : foo bar #prec foo
    ;

foo
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrUndefinedPrec},
		},
	}

	recoverDirTests := []*specErrTest{
		{
			caption: "the `#recover` directive cannot take an ID parameter",
			specSrc: `
#name test;

s
    : foo #recover foo
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#recover` directive cannot take an ordered symbol parameter",
			specSrc: `
#name test;

#prec (
    #assign $x
);

s
    : foo #recover $x
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#recover` directive cannot take a pattern parameter",
			specSrc: `
#name test;

s
    : foo #recover "foo"
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#recover` directive cannot take a string parameter",
			specSrc: `
#name test;

s
    : foo #recover 'foo'
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#recover` directive cannot take a directive group parameter",
			specSrc: `
#name test;

s
    : foo #recover ()
    ;

foo
    : 'foo';
`,
			errs: []error{semErrDirInvalidParam},
		},
	}

	fragmentTests := []*specErrTest{
		{
			caption: "a production cannot contain a fragment",
			specSrc: `
#name test;

s
    : f
    ;

fragment f
    : 'fragment';
`,
			errs: []error{semErrUndefinedSym},
		},
		{
			caption: "fragments cannot be duplicated",
			specSrc: `
#name test;

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
			errs: []error{semErrDuplicateFragment},
		},
	}

	modeDirTests := []*specErrTest{
		{
			caption: "the `#mode` directive needs an ID parameter",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #push mode_1
    : 'foo';
bar #mode
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#mode` directive cannot take an ordered symbol parameter",
			specSrc: `
#name test;

#prec (
    #assign $x
);

s
    : foo bar
    ;

foo
    : 'foo';
bar #mode $x
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#mode` directive cannot take a pattern parameter",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #push mode_1
    : 'foo';
bar #mode "mode_1"
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#mode` directive cannot take a string parameter",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #push mode_1
    : 'foo';
bar #mode 'mode_1'
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#mode` directive cannot take a directive group parameter",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #push mode_1
    : 'foo';
bar #mode ()
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
	}

	pushDirTests := []*specErrTest{
		{
			caption: "the `#push` directive needs an ID parameter",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #push
    : 'foo';
bar #mode mode_1
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#push` directive takes just one ID parameter",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #push mode_1 mode_2
    : 'foo';
bar #mode mode_1
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#push` directive cannot take an ordered symbol parameter",
			specSrc: `
#name test;

#prec (
    #assign $x
);

s
    : foo bar
    ;

foo #push $x
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#push` directive cannot take a pattern parameter",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #push "mode_1"
    : 'foo';
bar #mode mode_1
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#push` directive cannot take a string parameter",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #push 'mode_1'
    : 'foo';
bar #mode mode_1
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#push` directive cannot take a directive group parameter",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #push ()
    : 'foo';
bar #mode mode_1
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
	}

	popDirTests := []*specErrTest{
		{
			caption: "the `#pop` directive cannot take an ID parameter",
			specSrc: `
#name test;

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
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#pop` directive cannot take an ordered symbol parameter",
			specSrc: `
#name test;

#prec (
    #assign $x
);

s
    : foo bar baz
    ;

foo #push mode_1
    : 'foo';
bar #mode mode_1
    : 'bar';
baz #pop $x
    : 'baz';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#pop` directive cannot take a pattern parameter",
			specSrc: `
#name test;

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
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#pop` directive cannot take a string parameter",
			specSrc: `
#name test;

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
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#pop` directive cannot take a directive parameter",
			specSrc: `
#name test;

s
    : foo bar baz
    ;

foo #push mode_1
    : 'foo';
bar #mode mode_1
    : 'bar';
baz #pop ()
    : 'baz';
`,
			errs: []error{semErrDirInvalidParam},
		},
	}

	skipDirTests := []*specErrTest{
		{
			caption: "the `#skip` directive cannot take an ID parameter",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #skip bar
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#skip` directive cannot take an ordered symbol parameter",
			specSrc: `
#name test;

#prec (
    #assign $x
);

s
    : foo bar
    ;

foo #skip $x
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#skip` directive cannot take a pattern parameter",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #skip "bar"
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#skip` directive cannot take a string parameter",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #skip 'bar'
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "the `#skip` directive cannot take a directive group parameter",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #skip ()
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrDirInvalidParam},
		},
		{
			caption: "a terminal symbol used in productions cannot have the skip directive",
			specSrc: `
#name test;

s
    : foo bar
    ;

foo #skip
    : 'foo';
bar
    : 'bar';
`,
			errs: []error{semErrTermCannotBeSkipped},
		},
	}

	var tests []*specErrTest
	tests = append(tests, spellingInconsistenciesTests...)
	tests = append(tests, prodTests...)
	tests = append(tests, nameDirTests...)
	tests = append(tests, precDirTests...)
	tests = append(tests, leftDirTests...)
	tests = append(tests, rightDirTests...)
	tests = append(tests, assignDirTests...)
	tests = append(tests, errorSymTests...)
	tests = append(tests, astDirTests...)
	tests = append(tests, altPrecDirTests...)
	tests = append(tests, recoverDirTests...)
	tests = append(tests, fragmentTests...)
	tests = append(tests, modeDirTests...)
	tests = append(tests, pushDirTests...)
	tests = append(tests, popDirTests...)
	tests = append(tests, skipDirTests...)
	for _, test := range tests {
		t.Run(test.caption, func(t *testing.T) {
			ast, err := parser.Parse(strings.NewReader(test.specSrc))
			if err != nil {
				t.Fatal(err)
			}

			b := GrammarBuilder{
				AST: ast,
			}
			_, err = b.build()
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
