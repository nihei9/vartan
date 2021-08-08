package grammar

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nihei9/vartan/spec"
)

type expectedLALR1State struct {
	kernelItems    []*lrItem
	nextStates     map[symbol][]*lrItem
	reducibleProds []*production
}

func TestGenLALR1Automaton(t *testing.T) {
	// This grammar belongs to LALR(1) class, not SLR(1).
	src := `
S: L eq R | R;
L: ref R | id;
R: L;
eq: '=';
ref: '*';
id: "[A-Za-z0-9_]+";
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

	lr0, err := genLR0Automaton(gram.productionSet, gram.augmentedStartSymbol)
	if err != nil {
		t.Fatalf("failed to create a LR0 automaton: %v", err)
	}
	if lr0 == nil {
		t.Fatalf("genLR0Automaton returns nil without any error")
	}

	firstSet, err := genFirstSet(gram.productionSet)
	if err != nil {
		t.Fatalf("failed to create a FIRST set: %v", err)
	}

	automaton, err := genLALR1Automaton(lr0, gram.productionSet, firstSet)
	if err != nil {
		t.Fatalf("failed to create a LALR1 automaton: %v", err)
	}
	if lr0 == nil {
		t.Fatalf("genLALR1Automaton returns nil without any error")
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
			withLookAhead(genLR0Item("S'", 0, "S"), symbolEOF),
		},
		1: {
			withLookAhead(genLR0Item("S'", 1, "S"), symbolEOF),
		},
		2: {
			withLookAhead(genLR0Item("S", 1, "L", "eq", "R"), symbolEOF),
			withLookAhead(genLR0Item("R", 1, "L"), symbolEOF),
		},
		3: {
			withLookAhead(genLR0Item("S", 1, "R"), symbolEOF),
		},
		4: {
			withLookAhead(genLR0Item("L", 1, "ref", "R"), genSym("eq"), symbolEOF),
		},
		5: {
			withLookAhead(genLR0Item("L", 1, "id"), genSym("eq"), symbolEOF),
		},
		6: {
			withLookAhead(genLR0Item("S", 2, "L", "eq", "R"), symbolEOF),
		},
		7: {
			withLookAhead(genLR0Item("L", 2, "ref", "R"), genSym("eq"), symbolEOF),
		},
		8: {
			withLookAhead(genLR0Item("R", 1, "L"), genSym("eq"), symbolEOF),
		},
		9: {
			withLookAhead(genLR0Item("S", 3, "L", "eq", "R"), symbolEOF),
		},
	}

	expectedStates := []expectedLALR1State{
		{
			kernelItems: expectedKernels[0],
			nextStates: map[symbol][]*lrItem{
				genSym("S"):   expectedKernels[1],
				genSym("L"):   expectedKernels[2],
				genSym("R"):   expectedKernels[3],
				genSym("ref"): expectedKernels[4],
				genSym("id"):  expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[1],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("S'", "S"),
			},
		},
		{
			kernelItems: expectedKernels[2],
			nextStates: map[symbol][]*lrItem{
				genSym("eq"): expectedKernels[6],
			},
			reducibleProds: []*production{
				genProd("R", "L"),
			},
		},
		{
			kernelItems: expectedKernels[3],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("S", "R"),
			},
		},
		{
			kernelItems: expectedKernels[4],
			nextStates: map[symbol][]*lrItem{
				genSym("R"):   expectedKernels[7],
				genSym("L"):   expectedKernels[8],
				genSym("ref"): expectedKernels[4],
				genSym("id"):  expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[5],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("L", "id"),
			},
		},
		{
			kernelItems: expectedKernels[6],
			nextStates: map[symbol][]*lrItem{
				genSym("R"):   expectedKernels[9],
				genSym("L"):   expectedKernels[8],
				genSym("ref"): expectedKernels[4],
				genSym("id"):  expectedKernels[5],
			},
			reducibleProds: []*production{},
		},
		{
			kernelItems: expectedKernels[7],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("L", "ref", "R"),
			},
		},
		{
			kernelItems: expectedKernels[8],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("R", "L"),
			},
		},
		{
			kernelItems: expectedKernels[9],
			nextStates:  map[symbol][]*lrItem{},
			reducibleProds: []*production{
				genProd("S", "L", "eq", "R"),
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

			// test look-ahead symbols
			{
				if len(state.kernel.items) != len(eState.kernelItems) {
					t.Errorf("kernels is mismatched; want: %v, got: %v", len(eState.kernelItems), len(state.kernel.items))
				}
				for _, eKItem := range eState.kernelItems {
					var kItem *lrItem
					for _, it := range state.kernel.items {
						if it.id != eKItem.id {
							continue
						}
						kItem = it
						break
					}
					if kItem == nil {
						t.Fatalf("kernel item not found; want: %v, got: %v", eKItem.id, kItem.id)
					}

					for eSym := range eKItem.lookAhead.symbols {
						if _, ok := kItem.lookAhead.symbols[eSym]; !ok {
							t.Errorf("look-ahead symbol not found: %v", eSym)
						}
					}
				}
			}

			// test next states
			{
				if len(state.next) != len(eState.nextStates) {
					t.Errorf("next state count is mismatched; want: %v, got: %v", len(eState.nextStates), len(state.next))
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
