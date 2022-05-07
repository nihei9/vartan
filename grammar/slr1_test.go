package grammar

import (
	"strings"
	"testing"

	"github.com/nihei9/vartan/spec"
)

func TestGenSLR1Automaton(t *testing.T) {
	src := `
#name test;

expr
    : expr add term
    | term
    ;
term
    : term mul factor
    | factor
    ;
factor
    : l_paren expr r_paren
    | id
    ;
add: "\+";
mul: "\*";
l_paren: "\(";
r_paren: "\)";
id: "[A-Za-z_][0-9A-Za-z_]*";
`

	var gram *Grammar
	var automaton *slr1Automaton
	{
		ast, err := spec.Parse(strings.NewReader(src))
		if err != nil {
			t.Fatal(err)
		}
		b := GrammarBuilder{
			AST: ast,
		}

		gram, err = b.Build()
		if err != nil {
			t.Fatal(err)
		}

		lr0, err := genLR0Automaton(gram.productionSet, gram.augmentedStartSymbol, gram.errorSymbol)
		if err != nil {
			t.Fatalf("failed to create a LR0 automaton: %v", err)
		}

		firstSet, err := genFirstSet(gram.productionSet)
		if err != nil {
			t.Fatalf("failed to create a FIRST set: %v", err)
		}

		followSet, err := genFollowSet(gram.productionSet, firstSet)
		if err != nil {
			t.Fatalf("failed to create a FOLLOW set: %v", err)
		}

		automaton, err = genSLR1Automaton(lr0, gram.productionSet, followSet)
		if err != nil {
			t.Fatalf("failed to create a SLR1 automaton: %v", err)
		}
		if automaton == nil {
			t.Fatalf("genSLR1Automaton returns nil without any error")
		}
	}

	initialState := automaton.states[automaton.initialState]
	if initialState == nil {
		t.Errorf("failed to get an initial status: %v", automaton.initialState)
	}

	genSym := newTestSymbolGenerator(t, gram.symbolTable)
	genProd := newTestProductionGenerator(t, genSym)
	genLR0Item := newTestLR0ItemGenerator(t, genProd)

	expectedKernels := map[int][]*lrItem{
		0: {
			genLR0Item("expr'", 0, "expr"),
		},
		1: {
			withLookAhead(genLR0Item("expr'", 1, "expr"), symbolEOF),
			genLR0Item("expr", 1, "expr", "add", "term"),
		},
		2: {
			withLookAhead(genLR0Item("expr", 1, "term"), genSym("add"), genSym("r_paren"), symbolEOF),
			genLR0Item("term", 1, "term", "mul", "factor"),
		},
		3: {
			withLookAhead(genLR0Item("term", 1, "factor"), genSym("add"), genSym("mul"), genSym("r_paren"), symbolEOF),
		},
		4: {
			genLR0Item("factor", 1, "l_paren", "expr", "r_paren"),
		},
		5: {
			withLookAhead(genLR0Item("factor", 1, "id"), genSym("add"), genSym("mul"), genSym("r_paren"), symbolEOF),
		},
		6: {
			genLR0Item("expr", 2, "expr", "add", "term"),
		},
		7: {
			genLR0Item("term", 2, "term", "mul", "factor"),
		},
		8: {
			genLR0Item("expr", 1, "expr", "add", "term"),
			genLR0Item("factor", 2, "l_paren", "expr", "r_paren"),
		},
		9: {
			withLookAhead(genLR0Item("expr", 3, "expr", "add", "term"), genSym("add"), genSym("r_paren"), symbolEOF),
			genLR0Item("term", 1, "term", "mul", "factor"),
		},
		10: {
			withLookAhead(genLR0Item("term", 3, "term", "mul", "factor"), genSym("add"), genSym("mul"), genSym("r_paren"), symbolEOF),
		},
		11: {
			withLookAhead(genLR0Item("factor", 3, "l_paren", "expr", "r_paren"), genSym("add"), genSym("mul"), genSym("r_paren"), symbolEOF),
		},
	}

	expectedStates := []*expectedLRState{
		{
			kernelItems: expectedKernels[0],
			nextStates: map[symbol][]*lrItem{
				genSym("expr"):    expectedKernels[1],
				genSym("term"):    expectedKernels[2],
				genSym("factor"):  expectedKernels[3],
				genSym("l_paren"): expectedKernels[4],
				genSym("id"):      expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[1],
			nextStates: map[symbol][]*lrItem{
				genSym("add"): expectedKernels[6],
			},
			reducibleProds: []*production{
				genProd("expr'", "expr"),
			},
		},
		{
			kernelItems: expectedKernels[2],
			nextStates: map[symbol][]*lrItem{
				genSym("mul"): expectedKernels[7],
			},
			reducibleProds: []*production{
				genProd("expr", "term"),
			},
		},
		{
			kernelItems: expectedKernels[3],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("term", "factor"),
			},
		},
		{
			kernelItems: expectedKernels[4],
			nextStates: map[symbol][]*lrItem{
				genSym("expr"):    expectedKernels[8],
				genSym("term"):    expectedKernels[2],
				genSym("factor"):  expectedKernels[3],
				genSym("l_paren"): expectedKernels[4],
				genSym("id"):      expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[5],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("factor", "id"),
			},
		},
		{
			kernelItems: expectedKernels[6],
			nextStates: map[symbol][]*lrItem{
				genSym("term"):    expectedKernels[9],
				genSym("factor"):  expectedKernels[3],
				genSym("l_paren"): expectedKernels[4],
				genSym("id"):      expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[7],
			nextStates: map[symbol][]*lrItem{
				genSym("factor"):  expectedKernels[10],
				genSym("l_paren"): expectedKernels[4],
				genSym("id"):      expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[8],
			nextStates: map[symbol][]*lrItem{
				genSym("add"):     expectedKernels[6],
				genSym("r_paren"): expectedKernels[11],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[9],
			nextStates: map[symbol][]*lrItem{
				genSym("mul"): expectedKernels[7],
			},
			reducibleProds: []*production{
				genProd("expr", "expr", "add", "term"),
			},
		},
		{
			kernelItems: expectedKernels[10],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("term", "term", "mul", "factor"),
			},
		},
		{
			kernelItems: expectedKernels[11],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("factor", "l_paren", "expr", "r_paren"),
			},
		},
	}

	testLRAutomaton(t, expectedStates, automaton.lr0Automaton)
}
