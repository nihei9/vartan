package grammar

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nihei9/vartan/spec"
)

type expectedState struct {
	kernelItems []*lrItem
	acts        map[symbol]testActionEntry
	goTos       map[symbol][]*lrItem
}

func TestGenLALRParsingTable(t *testing.T) {
	src := `
#name test;

s: l eq r | r;
l: ref r | id;
r: l;
eq: '=';
ref: '*';
id: "[A-Za-z0-9_]+";
`

	var ptab *ParsingTable
	var automaton *lalr1Automaton
	var gram *Grammar
	var nonTermCount int
	var termCount int
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
		first, err := genFirstSet(gram.productionSet)
		if err != nil {
			t.Fatal(err)
		}
		lr0, err := genLR0Automaton(gram.productionSet, gram.augmentedStartSymbol, gram.errorSymbol)
		if err != nil {
			t.Fatal(err)
		}
		automaton, err = genLALR1Automaton(lr0, gram.productionSet, first)
		if err != nil {
			t.Fatal(err)
		}

		nonTermTexts, err := gram.symbolTable.nonTerminalTexts()
		if err != nil {
			t.Fatal(err)
		}
		termTexts, err := gram.symbolTable.terminalTexts()
		if err != nil {
			t.Fatal(err)
		}
		nonTermCount = len(nonTermTexts)
		termCount = len(termTexts)

		lalr := &lrTableBuilder{
			automaton:    automaton.lr0Automaton,
			prods:        gram.productionSet,
			termCount:    termCount,
			nonTermCount: nonTermCount,
			symTab:       gram.symbolTable,
		}
		ptab, err = lalr.build()
		if err != nil {
			t.Fatalf("failed to create a LALR parsing table: %v", err)
		}
		if ptab == nil {
			t.Fatal("genLALRParsingTable returns nil without any error")
		}
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

	expectedStates := []expectedState{
		{
			kernelItems: expectedKernels[0],
			acts: map[symbol]testActionEntry{
				genSym("ref"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[4],
				},
				genSym("id"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[5],
				},
			},
			goTos: map[symbol][]*lrItem{
				genSym("s"): expectedKernels[1],
				genSym("l"): expectedKernels[2],
				genSym("r"): expectedKernels[3],
			},
		},
		{
			kernelItems: expectedKernels[1],
			acts: map[symbol]testActionEntry{
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("s'", "s"),
				},
			},
		},
		{
			kernelItems: expectedKernels[2],
			acts: map[symbol]testActionEntry{
				genSym("eq"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[6],
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("r", "l"),
				},
			},
		},
		{
			kernelItems: expectedKernels[3],
			acts: map[symbol]testActionEntry{
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("s", "r"),
				},
			},
		},
		{
			kernelItems: expectedKernels[4],
			acts: map[symbol]testActionEntry{
				genSym("ref"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[4],
				},
				genSym("id"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[5],
				},
			},
			goTos: map[symbol][]*lrItem{
				genSym("r"): expectedKernels[7],
				genSym("l"): expectedKernels[8],
			},
		},
		{
			kernelItems: expectedKernels[5],
			acts: map[symbol]testActionEntry{
				genSym("eq"): {
					ty:         ActionTypeReduce,
					production: genProd("l", "id"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("l", "id"),
				},
			},
		},
		{
			kernelItems: expectedKernels[6],
			acts: map[symbol]testActionEntry{
				genSym("ref"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[4],
				},
				genSym("id"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[5],
				},
			},
			goTos: map[symbol][]*lrItem{
				genSym("l"): expectedKernels[8],
				genSym("r"): expectedKernels[9],
			},
		},
		{
			kernelItems: expectedKernels[7],
			acts: map[symbol]testActionEntry{
				genSym("eq"): {
					ty:         ActionTypeReduce,
					production: genProd("l", "ref", "r"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("l", "ref", "r"),
				},
			},
		},
		{
			kernelItems: expectedKernels[8],
			acts: map[symbol]testActionEntry{
				genSym("eq"): {
					ty:         ActionTypeReduce,
					production: genProd("r", "l"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("r", "l"),
				},
			},
		},
		{
			kernelItems: expectedKernels[9],
			acts: map[symbol]testActionEntry{
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("s", "l", "eq", "r"),
				},
			},
		},
	}

	t.Run("initial state", func(t *testing.T) {
		iniState := findStateByNum(automaton.states, ptab.InitialState)
		if iniState == nil {
			t.Fatalf("the initial state was not found: #%v", ptab.InitialState)
		}
		eIniState, err := newKernel(expectedKernels[0])
		if err != nil {
			t.Fatalf("failed to create a kernel item: %v", err)
		}
		if iniState.id != eIniState.id {
			t.Fatalf("the initial state is mismatched; want: %v, got: %v", eIniState.id, iniState.id)
		}
	})

	for i, eState := range expectedStates {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			k, err := newKernel(eState.kernelItems)
			if err != nil {
				t.Fatalf("failed to create a kernel item: %v", err)
			}
			state, ok := automaton.states[k.id]
			if !ok {
				t.Fatalf("state was not found: #%v", 0)
			}

			testAction(t, &eState, state, ptab, automaton.lr0Automaton, gram, termCount)
			testGoTo(t, &eState, state, ptab, automaton.lr0Automaton, nonTermCount)
		})
	}
}

func TestGenSLRParsingTable(t *testing.T) {
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

	var ptab *ParsingTable
	var automaton *slr1Automaton
	var gram *Grammar
	var nonTermCount int
	var termCount int
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
		first, err := genFirstSet(gram.productionSet)
		if err != nil {
			t.Fatal(err)
		}
		follow, err := genFollowSet(gram.productionSet, first)
		if err != nil {
			t.Fatal(err)
		}
		lr0, err := genLR0Automaton(gram.productionSet, gram.augmentedStartSymbol, gram.errorSymbol)
		if err != nil {
			t.Fatal(err)
		}
		automaton, err = genSLR1Automaton(lr0, gram.productionSet, follow)
		if err != nil {
			t.Fatal(err)
		}

		nonTermTexts, err := gram.symbolTable.nonTerminalTexts()
		if err != nil {
			t.Fatal(err)
		}
		termTexts, err := gram.symbolTable.terminalTexts()
		if err != nil {
			t.Fatal(err)
		}
		nonTermCount = len(nonTermTexts)
		termCount = len(termTexts)

		slr := &lrTableBuilder{
			automaton:    automaton.lr0Automaton,
			prods:        gram.productionSet,
			termCount:    termCount,
			nonTermCount: nonTermCount,
			symTab:       gram.symbolTable,
		}
		ptab, err = slr.build()
		if err != nil {
			t.Fatalf("failed to create a SLR parsing table: %v", err)
		}
		if ptab == nil {
			t.Fatal("genSLRParsingTable returns nil without any error")
		}
	}

	genSym := newTestSymbolGenerator(t, gram.symbolTable)
	genProd := newTestProductionGenerator(t, genSym)
	genLR0Item := newTestLR0ItemGenerator(t, genProd)

	expectedKernels := map[int][]*lrItem{
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

	expectedStates := []expectedState{
		{
			kernelItems: expectedKernels[0],
			acts: map[symbol]testActionEntry{
				genSym("l_paren"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[4],
				},
				genSym("id"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[5],
				},
			},
			goTos: map[symbol][]*lrItem{
				genSym("expr"):   expectedKernels[1],
				genSym("term"):   expectedKernels[2],
				genSym("factor"): expectedKernels[3],
			},
		},
		{
			kernelItems: expectedKernels[1],
			acts: map[symbol]testActionEntry{
				genSym("add"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[6],
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("expr'", "expr"),
				},
			},
		},
		{
			kernelItems: expectedKernels[2],
			acts: map[symbol]testActionEntry{
				genSym("add"): {
					ty:         ActionTypeReduce,
					production: genProd("expr", "term"),
				},
				genSym("mul"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[7],
				},
				genSym("r_paren"): {
					ty:         ActionTypeReduce,
					production: genProd("expr", "term"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("expr", "term"),
				},
			},
		},
		{
			kernelItems: expectedKernels[3],
			acts: map[symbol]testActionEntry{
				genSym("add"): {
					ty:         ActionTypeReduce,
					production: genProd("term", "factor"),
				},
				genSym("mul"): {
					ty:         ActionTypeReduce,
					production: genProd("term", "factor"),
				},
				genSym("r_paren"): {
					ty:         ActionTypeReduce,
					production: genProd("term", "factor"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("term", "factor"),
				},
			},
		},
		{
			kernelItems: expectedKernels[4],
			acts: map[symbol]testActionEntry{
				genSym("l_paren"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[4],
				},
				genSym("id"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[5],
				},
			},
			goTos: map[symbol][]*lrItem{
				genSym("expr"):   expectedKernels[8],
				genSym("term"):   expectedKernels[2],
				genSym("factor"): expectedKernels[3],
			},
		},
		{
			kernelItems: expectedKernels[5],
			acts: map[symbol]testActionEntry{
				genSym("add"): {
					ty:         ActionTypeReduce,
					production: genProd("factor", "id"),
				},
				genSym("mul"): {
					ty:         ActionTypeReduce,
					production: genProd("factor", "id"),
				},
				genSym("r_paren"): {
					ty:         ActionTypeReduce,
					production: genProd("factor", "id"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("factor", "id"),
				},
			},
		},
		{
			kernelItems: expectedKernels[6],
			acts: map[symbol]testActionEntry{
				genSym("l_paren"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[4],
				},
				genSym("id"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[5],
				},
			},
			goTos: map[symbol][]*lrItem{
				genSym("term"):   expectedKernels[9],
				genSym("factor"): expectedKernels[3],
			},
		},
		{
			kernelItems: expectedKernels[7],
			acts: map[symbol]testActionEntry{
				genSym("l_paren"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[4],
				},
				genSym("id"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[5],
				},
			},
			goTos: map[symbol][]*lrItem{
				genSym("factor"): expectedKernels[10],
			},
		},
		{
			kernelItems: expectedKernels[8],
			acts: map[symbol]testActionEntry{
				genSym("add"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[6],
				},
				genSym("r_paren"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[11],
				},
			},
		},
		{
			kernelItems: expectedKernels[9],
			acts: map[symbol]testActionEntry{
				genSym("add"): {
					ty:         ActionTypeReduce,
					production: genProd("expr", "expr", "add", "term"),
				},
				genSym("mul"): {
					ty:        ActionTypeShift,
					nextState: expectedKernels[7],
				},
				genSym("r_paren"): {
					ty:         ActionTypeReduce,
					production: genProd("expr", "expr", "add", "term"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("expr", "expr", "add", "term"),
				},
			},
		},
		{
			kernelItems: expectedKernels[10],
			acts: map[symbol]testActionEntry{
				genSym("add"): {
					ty:         ActionTypeReduce,
					production: genProd("term", "term", "mul", "factor"),
				},
				genSym("mul"): {
					ty:         ActionTypeReduce,
					production: genProd("term", "term", "mul", "factor"),
				},
				genSym("r_paren"): {
					ty:         ActionTypeReduce,
					production: genProd("term", "term", "mul", "factor"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("term", "term", "mul", "factor"),
				},
			},
		},
		{
			kernelItems: expectedKernels[11],
			acts: map[symbol]testActionEntry{
				genSym("add"): {
					ty:         ActionTypeReduce,
					production: genProd("factor", "l_paren", "expr", "r_paren"),
				},
				genSym("mul"): {
					ty:         ActionTypeReduce,
					production: genProd("factor", "l_paren", "expr", "r_paren"),
				},
				genSym("r_paren"): {
					ty:         ActionTypeReduce,
					production: genProd("factor", "l_paren", "expr", "r_paren"),
				},
				symbolEOF: {
					ty:         ActionTypeReduce,
					production: genProd("factor", "l_paren", "expr", "r_paren"),
				},
			},
		},
	}

	t.Run("initial state", func(t *testing.T) {
		iniState := findStateByNum(automaton.states, ptab.InitialState)
		if iniState == nil {
			t.Fatalf("the initial state was not found: #%v", ptab.InitialState)
		}
		eIniState, err := newKernel(expectedKernels[0])
		if err != nil {
			t.Fatalf("failed to create a kernel item: %v", err)
		}
		if iniState.id != eIniState.id {
			t.Fatalf("the initial state is mismatched; want: %v, got: %v", eIniState.id, iniState.id)
		}
	})

	for i, eState := range expectedStates {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			k, err := newKernel(eState.kernelItems)
			if err != nil {
				t.Fatalf("failed to create a kernel item: %v", err)
			}
			state, ok := automaton.states[k.id]
			if !ok {
				t.Fatalf("state was not found: #%v", 0)
			}

			testAction(t, &eState, state, ptab, automaton.lr0Automaton, gram, termCount)
			testGoTo(t, &eState, state, ptab, automaton.lr0Automaton, nonTermCount)
		})
	}
}

func testAction(t *testing.T, expectedState *expectedState, state *lrState, ptab *ParsingTable, automaton *lr0Automaton, gram *Grammar, termCount int) {
	nonEmptyEntries := map[symbolNum]struct{}{}
	for eSym, eAct := range expectedState.acts {
		nonEmptyEntries[eSym.num()] = struct{}{}

		ty, stateNum, prodNum := ptab.getAction(state.num, eSym.num())
		if ty != eAct.ty {
			t.Fatalf("action type is mismatched; want: %v, got: %v", eAct.ty, ty)
		}
		switch eAct.ty {
		case ActionTypeShift:
			eNextState, err := newKernel(eAct.nextState)
			if err != nil {
				t.Fatal(err)
			}
			nextState := findStateByNum(automaton.states, stateNum)
			if nextState == nil {
				t.Fatalf("state was not found; state: #%v", stateNum)
			}
			if nextState.id != eNextState.id {
				t.Fatalf("next state is mismatched; symbol: %v, want: %v, got: %v", eSym, eNextState.id, nextState.id)
			}
		case ActionTypeReduce:
			prod := findProductionByNum(gram.productionSet, prodNum)
			if prod == nil {
				t.Fatalf("production was not found: #%v", prodNum)
			}
			if prod.id != eAct.production.id {
				t.Fatalf("production is mismatched; symbol: %v, want: %v, got: %v", eSym, eAct.production.id, prod.id)
			}
		}
	}
	for symNum := 0; symNum < termCount; symNum++ {
		if _, checked := nonEmptyEntries[symbolNum(symNum)]; checked {
			continue
		}
		ty, stateNum, prodNum := ptab.getAction(state.num, symbolNum(symNum))
		if ty != ActionTypeError {
			t.Errorf("unexpected ACTION entry; state: #%v, symbol: #%v, action type: %v, next state: #%v, prodction: #%v", state.num, symNum, ty, stateNum, prodNum)
		}
	}
}

func testGoTo(t *testing.T, expectedState *expectedState, state *lrState, ptab *ParsingTable, automaton *lr0Automaton, nonTermCount int) {
	nonEmptyEntries := map[symbolNum]struct{}{}
	for eSym, eGoTo := range expectedState.goTos {
		nonEmptyEntries[eSym.num()] = struct{}{}

		eNextState, err := newKernel(eGoTo)
		if err != nil {
			t.Fatal(err)
		}
		ty, stateNum := ptab.getGoTo(state.num, eSym.num())
		if ty != GoToTypeRegistered {
			t.Fatalf("GOTO entry was not found; state: #%v, symbol: #%v", state.num, eSym)
		}
		nextState := findStateByNum(automaton.states, stateNum)
		if nextState == nil {
			t.Fatalf("state was not found: #%v", stateNum)
		}
		if nextState.id != eNextState.id {
			t.Fatalf("next state is mismatched; symbol: %v, want: %v, got: %v", eSym, eNextState.id, nextState.id)
		}
	}
	for symNum := 0; symNum < nonTermCount; symNum++ {
		if _, checked := nonEmptyEntries[symbolNum(symNum)]; checked {
			continue
		}
		ty, _ := ptab.getGoTo(state.num, symbolNum(symNum))
		if ty != GoToTypeError {
			t.Errorf("unexpected GOTO entry; state: #%v, symbol: #%v", state.num, symNum)
		}
	}
}

type testActionEntry struct {
	ty         ActionType
	nextState  []*lrItem
	production *production
}

func findStateByNum(states map[kernelID]*lrState, num stateNum) *lrState {
	for _, state := range states {
		if state.num == num {
			return state
		}
	}
	return nil
}

func findProductionByNum(prods *productionSet, num productionNum) *production {
	for _, prod := range prods.getAllProductions() {
		if prod.num == num {
			return prod
		}
	}
	return nil
}
