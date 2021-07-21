package grammar

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nihei9/vartan/spec"
)

func TestGenLR0Automaton(t *testing.T) {
	src := `
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

	ast, err := spec.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	b := GrammarBuilder{
		AST: ast,
	}
	gram, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}

	automaton, err := genLR0Automaton(gram.productionSet, gram.augmentedStartSymbol)
	if err != nil {
		t.Fatalf("failed to create a LR0 automaton: %v", err)
	}
	if automaton == nil {
		t.Fatalf("genLR0Automaton returns nil without any error")
	}

	initialState := automaton.states[automaton.initialState]
	if initialState == nil {
		t.Errorf("failed to get an initial status: %v", automaton.initialState)
	}

	genSym := newTestSymbolGenerator(t, gram.symbolTable)
	genProd := newTestProductionGenerator(t, genSym)
	genLR0Item := newTestLR0ItemGenerator(t, genProd)

	expectedKernels := map[int][]*lr0Item{
		0: {
			genLR0Item("expr'", 0, "expr"),
		},
		1: {
			genLR0Item("expr'", 1, "expr"),
			genLR0Item("expr", 1, "expr", "add", "term"),
		},
		2: {
			genLR0Item("expr", 1, "term"),
			genLR0Item("term", 1, "term", "mul", "factor"),
		},
		3: {
			genLR0Item("term", 1, "factor"),
		},
		4: {
			genLR0Item("factor", 1, "l_paren", "expr", "r_paren"),
		},
		5: {
			genLR0Item("factor", 1, "id"),
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
			genLR0Item("expr", 3, "expr", "add", "term"),
			genLR0Item("term", 1, "term", "mul", "factor"),
		},
		10: {
			genLR0Item("term", 3, "term", "mul", "factor"),
		},
		11: {
			genLR0Item("factor", 3, "l_paren", "expr", "r_paren"),
		},
	}

	expectedStates := []expectedLR0State{
		{
			kernelItems: expectedKernels[0],
			nextStates: map[symbol][]*lr0Item{
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
			nextStates: map[symbol][]*lr0Item{
				genSym("add"): expectedKernels[6],
			},
			reducibleProds: []*production{
				genProd("expr'", "expr"),
			},
		},
		{
			kernelItems: expectedKernels[2],
			nextStates: map[symbol][]*lr0Item{
				genSym("mul"): expectedKernels[7],
			},
			reducibleProds: []*production{
				genProd("expr", "term"),
			},
		},
		{
			kernelItems: expectedKernels[3],
			nextStates:  map[symbol][]*lr0Item{},
			reducibleProds: []*production{
				genProd("term", "factor"),
			},
		},
		{
			kernelItems: expectedKernels[4],
			nextStates: map[symbol][]*lr0Item{
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
			nextStates:  map[symbol][]*lr0Item{},
			reducibleProds: []*production{
				genProd("factor", "id"),
			},
		},
		{
			kernelItems: expectedKernels[6],
			nextStates: map[symbol][]*lr0Item{
				genSym("term"):    expectedKernels[9],
				genSym("factor"):  expectedKernels[3],
				genSym("l_paren"): expectedKernels[4],
				genSym("id"):      expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[7],
			nextStates: map[symbol][]*lr0Item{
				genSym("factor"):  expectedKernels[10],
				genSym("l_paren"): expectedKernels[4],
				genSym("id"):      expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[8],
			nextStates: map[symbol][]*lr0Item{
				genSym("add"):     expectedKernels[6],
				genSym("r_paren"): expectedKernels[11],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[9],
			nextStates: map[symbol][]*lr0Item{
				genSym("mul"): expectedKernels[7],
			},
			reducibleProds: []*production{
				genProd("expr", "expr", "add", "term"),
			},
		},
		{
			kernelItems: expectedKernels[10],
			nextStates:  map[symbol][]*lr0Item{},
			reducibleProds: []*production{
				genProd("term", "term", "mul", "factor"),
			},
		},
		{
			kernelItems: expectedKernels[11],
			nextStates:  map[symbol][]*lr0Item{},
			reducibleProds: []*production{
				genProd("factor", "l_paren", "expr", "r_paren"),
			},
		},
	}

	if len(automaton.states) != len(expectedStates) {
		t.Errorf("state count is mismatched; want: %v, got: %v", len(expectedStates), len(automaton.states))
	}

	for i, eState := range expectedStates {
		t.Run(fmt.Sprintf("state #%v", i), func(t *testing.T) {
			k, err := newKernel(eState.kernelItems)
			if err != nil {
				t.Fatalf("failed to create a kernel item: %v", err)
			}

			state, ok := automaton.states[k.id]
			if !ok {
				t.Fatalf("a kernel was not found: %v", k.id)
			}

			// test next states
			{
				if len(state.next) != len(eState.nextStates) {
					t.Errorf("next state count is mismcthed; want: %v, got: %v", len(eState.nextStates), len(state.next))
				}
				for eSym, eKItems := range eState.nextStates {
					nextStateKernel, err := newKernel(eKItems)
					if err != nil {
						t.Fatalf("failed to create a kernel item: %v", err)
					}
					nextState, ok := state.next[eSym]
					if !ok {
						t.Fatalf("next state was not found; state: %v, symbol: %v (%v)", state.id, "expr", eSym)
					}
					if nextState != nextStateKernel.id {
						t.Fatalf("a kernel ID of the next state is mismatched; want: %v, got: %v", nextStateKernel.id, nextState)
					}
				}
			}

			// test reducible productions
			{
				if len(state.reducible) != len(eState.reducibleProds) {
					t.Errorf("reducible production count is mismatched; want: %v, got: %v", len(eState.reducibleProds), len(state.reducible))
				}
				for _, eProd := range eState.reducibleProds {
					if _, ok := state.reducible[eProd.id]; !ok {
						t.Errorf("reducible production was not found: %v", eProd.id)
					}
				}
			}
		})
	}
}

type expectedLR0State struct {
	kernelItems    []*lr0Item
	nextStates     map[symbol][]*lr0Item
	reducibleProds []*production
}
