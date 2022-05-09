package grammar

import (
	"strings"
	"testing"

	"github.com/nihei9/vartan/spec"
)

func TestGenLALR1Automaton(t *testing.T) {
	// This grammar belongs to LALR(1) class, not SLR(1).
	src := `
#name test;

s: l eq r | r;
l: ref r | id;
r: l;
eq: '=';
ref: '*';
id: "[A-Za-z0-9_]+";
`

	var gram *Grammar
	var automaton *lalr1Automaton
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

		automaton, err = genLALR1Automaton(lr0, gram.productionSet, firstSet)
		if err != nil {
			t.Fatalf("failed to create a LALR1 automaton: %v", err)
		}
		if automaton == nil {
			t.Fatalf("genLALR1Automaton returns nil without any error")
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
			withLookAhead(genLR0Item("s'", 0, "s"), symbolEOF),
		},
		1: {
			withLookAhead(genLR0Item("s'", 1, "s"), symbolEOF),
		},
		2: {
			withLookAhead(genLR0Item("s", 1, "l", "eq", "r"), symbolEOF),
			withLookAhead(genLR0Item("r", 1, "l"), symbolEOF),
		},
		3: {
			withLookAhead(genLR0Item("s", 1, "r"), symbolEOF),
		},
		4: {
			withLookAhead(genLR0Item("l", 1, "ref", "r"), genSym("eq"), symbolEOF),
		},
		5: {
			withLookAhead(genLR0Item("l", 1, "id"), genSym("eq"), symbolEOF),
		},
		6: {
			withLookAhead(genLR0Item("s", 2, "l", "eq", "r"), symbolEOF),
		},
		7: {
			withLookAhead(genLR0Item("l", 2, "ref", "r"), genSym("eq"), symbolEOF),
		},
		8: {
			withLookAhead(genLR0Item("r", 1, "l"), genSym("eq"), symbolEOF),
		},
		9: {
			withLookAhead(genLR0Item("s", 3, "l", "eq", "r"), symbolEOF),
		},
	}

	expectedStates := []*expectedLRState{
		{
			kernelItems: expectedKernels[0],
			nextStates: map[symbol][]*lrItem{
				genSym("s"):   expectedKernels[1],
				genSym("l"):   expectedKernels[2],
				genSym("r"):   expectedKernels[3],
				genSym("ref"): expectedKernels[4],
				genSym("id"):  expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[1],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("s'", "s"),
			},
		},
		{
			kernelItems: expectedKernels[2],
			nextStates: map[symbol][]*lrItem{
				genSym("eq"): expectedKernels[6],
			},
			reducibleProds: []*production{
				genProd("r", "l"),
			},
		},
		{
			kernelItems: expectedKernels[3],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("s", "r"),
			},
		},
		{
			kernelItems: expectedKernels[4],
			nextStates: map[symbol][]*lrItem{
				genSym("r"):   expectedKernels[7],
				genSym("l"):   expectedKernels[8],
				genSym("ref"): expectedKernels[4],
				genSym("id"):  expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[5],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("l", "id"),
			},
		},
		{
			kernelItems: expectedKernels[6],
			nextStates: map[symbol][]*lrItem{
				genSym("r"):   expectedKernels[9],
				genSym("l"):   expectedKernels[8],
				genSym("ref"): expectedKernels[4],
				genSym("id"):  expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[7],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("l", "ref", "r"),
			},
		},
		{
			kernelItems: expectedKernels[8],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("r", "l"),
			},
		},
		{
			kernelItems: expectedKernels[9],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("s", "l", "eq", "r"),
			},
		},
	}

	testLRAutomaton(t, expectedStates, automaton.lr0Automaton)
}
