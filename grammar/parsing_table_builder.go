package grammar

import (
	"fmt"
	"io"
	"strings"
)

type ActionType string

const (
	ActionTypeShift  = ActionType("shift")
	ActionTypeReduce = ActionType("reduce")
	ActionTypeError  = ActionType("error")
)

type actionEntry int

const actionEntryEmpty = actionEntry(0)

func newShiftActionEntry(state stateNum) actionEntry {
	return actionEntry(state * -1)
}

func newReduceActionEntry(prod productionNum) actionEntry {
	return actionEntry(prod)
}

func (e actionEntry) isEmpty() bool {
	return e == actionEntryEmpty
}

func (e actionEntry) describe() (ActionType, stateNum, productionNum) {
	if e == actionEntryEmpty {
		return ActionTypeError, stateNumInitial, productionNumNil
	}
	if e < 0 {
		return ActionTypeShift, stateNum(e * -1), productionNumNil
	}
	return ActionTypeReduce, stateNumInitial, productionNum(e)
}

type GoToType string

const (
	GoToTypeRegistered = GoToType("registered")
	GoToTypeError      = GoToType("error")
)

type goToEntry uint

const goToEntryEmpty = goToEntry(0)

func newGoToEntry(state stateNum) goToEntry {
	return goToEntry(state)
}

func (e goToEntry) isEmpty() bool {
	return e == goToEntryEmpty
}

func (e goToEntry) describe() (GoToType, stateNum) {
	if e == goToEntryEmpty {
		return GoToTypeError, stateNumInitial
	}
	return GoToTypeRegistered, stateNum(e)
}

type conflict interface {
	conflict()
}

type shiftReduceConflict struct {
	state     stateNum
	sym       symbol
	nextState stateNum
	prodNum   productionNum
}

func (c *shiftReduceConflict) conflict() {
}

type reduceReduceConflict struct {
	state    stateNum
	sym      symbol
	prodNum1 productionNum
	prodNum2 productionNum
}

func (c *reduceReduceConflict) conflict() {
}

var (
	_ conflict = &shiftReduceConflict{}
	_ conflict = &reduceReduceConflict{}
)

type ParsingTable struct {
	actionTable       []actionEntry
	goToTable         []goToEntry
	stateCount        int
	terminalCount     int
	nonTerminalCount  int
	expectedTerminals [][]int

	InitialState stateNum
}

func (t *ParsingTable) getAction(state stateNum, sym symbolNum) (ActionType, stateNum, productionNum) {
	pos := state.Int()*t.terminalCount + sym.Int()
	return t.actionTable[pos].describe()
}

func (t *ParsingTable) getGoTo(state stateNum, sym symbolNum) (GoToType, stateNum) {
	pos := state.Int()*t.nonTerminalCount + sym.Int()
	return t.goToTable[pos].describe()
}

func (t *ParsingTable) writeShiftAction(state stateNum, sym symbol, nextState stateNum) conflict {
	pos := state.Int()*t.terminalCount + sym.num().Int()
	act := t.actionTable[pos]
	if !act.isEmpty() {
		ty, _, p := act.describe()
		if ty == ActionTypeReduce {
			return &shiftReduceConflict{
				state:     state,
				sym:       sym,
				nextState: nextState,
				prodNum:   p,
			}
		}
	}
	t.actionTable[pos] = newShiftActionEntry(nextState)

	return nil
}

func (t *ParsingTable) writeReduceAction(state stateNum, sym symbol, prod productionNum) conflict {
	pos := state.Int()*t.terminalCount + sym.num().Int()
	act := t.actionTable[pos]
	if !act.isEmpty() {
		ty, s, p := act.describe()
		if ty == ActionTypeReduce && p != prod {
			return &reduceReduceConflict{
				state:    state,
				sym:      sym,
				prodNum1: p,
				prodNum2: prod,
			}
		}
		return &shiftReduceConflict{
			state:     state,
			sym:       sym,
			nextState: s,
			prodNum:   prod,
		}
	}
	t.actionTable[pos] = newReduceActionEntry(prod)

	return nil
}

func (t *ParsingTable) writeGoTo(state stateNum, sym symbol, nextState stateNum) {
	pos := state.Int()*t.nonTerminalCount + sym.num().Int()
	t.goToTable[pos] = newGoToEntry(nextState)
}

type lalrTableBuilder struct {
	automaton    *lalr1Automaton
	prods        *productionSet
	termCount    int
	nonTermCount int
	symTab       *symbolTable
	sym2AnonPat  map[symbol]string

	conflicts []conflict
}

func (b *lalrTableBuilder) build() (*ParsingTable, error) {
	var ptab *ParsingTable
	{
		initialState := b.automaton.states[b.automaton.initialState]
		ptab = &ParsingTable{
			actionTable:       make([]actionEntry, len(b.automaton.states)*b.termCount),
			goToTable:         make([]goToEntry, len(b.automaton.states)*b.nonTermCount),
			stateCount:        len(b.automaton.states),
			terminalCount:     b.termCount,
			nonTerminalCount:  b.nonTermCount,
			expectedTerminals: make([][]int, len(b.automaton.states)),
			InitialState:      initialState.num,
		}
	}

	var conflicts []conflict
	for _, state := range b.automaton.states {
		var eTerms []int

		for sym, kID := range state.next {
			nextState := b.automaton.states[kID]
			if sym.isTerminal() {
				eTerms = append(eTerms, sym.num().Int())

				c := ptab.writeShiftAction(state.num, sym, nextState.num)
				if c != nil {
					conflicts = append(conflicts, c)
					continue
				}
			} else {
				ptab.writeGoTo(state.num, sym, nextState.num)
			}
		}

		for prodID := range state.reducible {
			reducibleProd, ok := b.prods.findByID(prodID)
			if !ok {
				return nil, fmt.Errorf("reducible production not found: %v", prodID)
			}

			var reducibleItem *lrItem
			for _, item := range state.items {
				if item.prod != reducibleProd.id {
					continue
				}

				reducibleItem = item
				break
			}
			if reducibleItem == nil {
				for _, item := range state.emptyProdItems {
					if item.prod != reducibleProd.id {
						continue
					}

					reducibleItem = item
					break
				}
				if reducibleItem == nil {
					return nil, fmt.Errorf("reducible item not found; state: %v, production: %v", state.num, reducibleProd.num)
				}
			}

			for a := range reducibleItem.lookAhead.symbols {
				eTerms = append(eTerms, a.num().Int())

				c := ptab.writeReduceAction(state.num, a, reducibleProd.num)
				if c != nil {
					conflicts = append(conflicts, c)
					continue
				}
			}
		}

		ptab.expectedTerminals[state.num] = eTerms
	}

	b.conflicts = conflicts

	if len(conflicts) > 0 {
		return nil, fmt.Errorf("%v conflicts", len(conflicts))
	}

	return ptab, nil
}

type slrTableBuilder struct {
	automaton    *lr0Automaton
	prods        *productionSet
	follow       *followSet
	termCount    int
	nonTermCount int
	symTab       *symbolTable
	sym2AnonPat  map[symbol]string

	conflicts []conflict
}

func (b *slrTableBuilder) build() (*ParsingTable, error) {
	var ptab *ParsingTable
	{
		initialState := b.automaton.states[b.automaton.initialState]
		ptab = &ParsingTable{
			actionTable:       make([]actionEntry, len(b.automaton.states)*b.termCount),
			goToTable:         make([]goToEntry, len(b.automaton.states)*b.nonTermCount),
			stateCount:        len(b.automaton.states),
			terminalCount:     b.termCount,
			nonTerminalCount:  b.nonTermCount,
			expectedTerminals: make([][]int, len(b.automaton.states)),
			InitialState:      initialState.num,
		}
	}

	var conflicts []conflict
	for _, state := range b.automaton.states {
		var eTerms []int

		for sym, kID := range state.next {
			nextState := b.automaton.states[kID]
			if sym.isTerminal() {
				eTerms = append(eTerms, sym.num().Int())

				c := ptab.writeShiftAction(state.num, sym, nextState.num)
				if c != nil {
					conflicts = append(conflicts, c)
					continue
				}
			} else {
				ptab.writeGoTo(state.num, sym, nextState.num)
			}
		}

		for prodID := range state.reducible {
			prod, _ := b.prods.findByID(prodID)
			flw, err := b.follow.find(prod.lhs)
			if err != nil {
				return nil, err
			}
			for sym := range flw.symbols {
				eTerms = append(eTerms, sym.num().Int())

				c := ptab.writeReduceAction(state.num, sym, prod.num)
				if c != nil {
					conflicts = append(conflicts, c)
					continue
				}
			}
			if flw.eof {
				eTerms = append(eTerms, symbolEOF.num().Int())

				c := ptab.writeReduceAction(state.num, symbolEOF, prod.num)
				if c != nil {
					conflicts = append(conflicts, c)
					continue
				}
			}
		}

		ptab.expectedTerminals[state.num] = eTerms
	}

	b.conflicts = conflicts

	if len(conflicts) > 0 {
		return nil, fmt.Errorf("%v conflicts", len(conflicts))
	}

	return ptab, nil
}

type descriptionWriter struct {
	automaton    *lr0Automaton
	prods        *productionSet
	follow       *followSet
	termCount    int
	nonTermCount int
	symTab       *symbolTable
	sym2AnonPat  map[symbol]string
	conflicts    []conflict
}

func (dw *descriptionWriter) write(w io.Writer) {
	conflicts := map[stateNum][]conflict{}
	for _, con := range dw.conflicts {
		switch c := con.(type) {
		case *shiftReduceConflict:
			conflicts[c.state] = append(conflicts[c.state], c)
		case *reduceReduceConflict:
			conflicts[c.state] = append(conflicts[c.state], c)
		}
	}

	fmt.Fprintf(w, "# Conflicts\n\n")

	if len(dw.conflicts) > 0 {
		fmt.Fprintf(w, "%v conflics:\n\n", len(dw.conflicts))

		for _, conflict := range dw.conflicts {
			switch c := conflict.(type) {
			case *shiftReduceConflict:
				fmt.Fprintf(w, "%v: shift/reduce conflict (shift %v, reduce %v) on %v\n", c.state, c.nextState, c.prodNum, dw.symbolToText(c.sym))
			case *reduceReduceConflict:
				fmt.Fprintf(w, "%v: reduce/reduce conflict (reduce %v and %v) on %v\n", c.state, c.prodNum1, c.prodNum2, dw.symbolToText(c.sym))
			}
		}
		fmt.Fprintf(w, "\n")
	} else {
		fmt.Fprintf(w, "no conflicts\n\n")
	}

	fmt.Fprintf(w, "# Terminals\n\n")

	termSyms := dw.symTab.terminalSymbols()

	fmt.Fprintf(w, "%v symbols:\n\n", len(termSyms))

	for _, sym := range termSyms {
		text, ok := dw.symTab.toText(sym)
		if !ok {
			text = fmt.Sprintf("<symbol not found: %v>", sym)
		}
		if strings.HasPrefix(text, "_") {
			fmt.Fprintf(w, "%4v %v: \"%v\"\n", sym.num(), text, dw.sym2AnonPat[sym])
		} else {
			fmt.Fprintf(w, "%4v %v\n", sym.num(), text)
		}
	}

	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# Productions\n\n")

	fmt.Fprintf(w, "%v productions:\n\n", len(dw.prods.getAllProductions()))

	for _, prod := range dw.prods.getAllProductions() {
		fmt.Fprintf(w, "%4v %v\n", prod.num, dw.productionToString(prod, -1))
	}

	fmt.Fprintf(w, "\n# States\n\n")

	fmt.Fprintf(w, "%v states:\n\n", len(dw.automaton.states))

	for _, state := range dw.automaton.states {
		fmt.Fprintf(w, "state %v\n", state.num)
		for _, item := range state.items {
			prod, ok := dw.prods.findByID(item.prod)
			if !ok {
				fmt.Fprintf(w, "<production not found>\n")
				continue
			}
			fmt.Fprintf(w, "    %v\n", dw.productionToString(prod, item.dot))
		}

		fmt.Fprintf(w, "\n")

		var shiftRecs []string
		var reduceRecs []string
		var gotoRecs []string
		var accRec string
		{
			for sym, kID := range state.next {
				nextState := dw.automaton.states[kID]
				if sym.isTerminal() {
					shiftRecs = append(shiftRecs, fmt.Sprintf("shift  %4v on %v", nextState.num, dw.symbolToText(sym)))
				} else {
					gotoRecs = append(gotoRecs, fmt.Sprintf("goto   %4v on %v", nextState.num, dw.symbolToText(sym)))
				}
			}

			for prodID := range state.reducible {
				prod, ok := dw.prods.findByID(prodID)
				if !ok {
					reduceRecs = append(reduceRecs, "<production not found>")
					continue
				}
				if prod.lhs.isStart() {
					accRec = "accept on <EOF>"
					continue
				}

				if dw.follow != nil {
					flw, err := dw.follow.find(prod.lhs)
					if err != nil {
						reduceRecs = append(reduceRecs, fmt.Sprintf("%v", err))
						continue
					}
					for sym := range flw.symbols {
						reduceRecs = append(reduceRecs, fmt.Sprintf("reduce %4v on %v", prod.num, dw.symbolToText(sym)))
					}
					if flw.eof {
						reduceRecs = append(reduceRecs, fmt.Sprintf("reduce %4v on <EOF>", prod.num))
					}
				} else {
					var reducibleItem *lrItem
					for _, item := range state.items {
						if item.prod != prodID {
							continue
						}

						reducibleItem = item
						break
					}
					if reducibleItem == nil {
						for _, item := range state.emptyProdItems {
							if item.prod != prodID {
								continue
							}

							reducibleItem = item
							break
						}
						if reducibleItem == nil {
							reduceRecs = append(reduceRecs, "<item not found>")
							continue
						}
					}
					for a := range reducibleItem.lookAhead.symbols {
						reduceRecs = append(reduceRecs, fmt.Sprintf("reduce %4v on %v", prod.num, dw.symbolToText(a)))
					}
				}
			}
		}

		if len(shiftRecs) > 0 || len(reduceRecs) > 0 {
			for _, rec := range shiftRecs {
				fmt.Fprintf(w, "    %v\n", rec)
			}
			for _, rec := range reduceRecs {
				fmt.Fprintf(w, "    %v\n", rec)
			}
			fmt.Fprintf(w, "\n")
		}
		if len(gotoRecs) > 0 {
			for _, rec := range gotoRecs {
				fmt.Fprintf(w, "    %v\n", rec)
			}
			fmt.Fprintf(w, "\n")
		}
		if accRec != "" {
			fmt.Fprintf(w, "    %v\n\n", accRec)
		}

		cons, ok := conflicts[state.num]
		if ok {
			for _, con := range cons {
				switch c := con.(type) {
				case *shiftReduceConflict:
					fmt.Fprintf(w, "    shift/reduce conflict (shift %v, reduce %v) on %v\n", c.nextState, c.prodNum, dw.symbolToText(c.sym))
				case *reduceReduceConflict:
					fmt.Fprintf(w, "    reduce/reduce conflict (reduce %v and %v) on %v\n", c.prodNum1, c.prodNum2, dw.symbolToText(c.sym))
				}
			}
			fmt.Fprintf(w, "\n")
		}
	}
}

func (dw *descriptionWriter) productionToString(prod *production, dot int) string {
	var w strings.Builder
	fmt.Fprintf(&w, "%v →", dw.symbolToText(prod.lhs))
	for n, rhs := range prod.rhs {
		if n == dot {
			fmt.Fprintf(&w, " ・")
		}
		fmt.Fprintf(&w, " %v", dw.symbolToText(rhs))
	}
	if dot == len(prod.rhs) {
		fmt.Fprintf(&w, " ・")
	}

	return w.String()
}

func (dw *descriptionWriter) symbolToText(sym symbol) string {
	if sym.isNil() {
		return "<NULL>"
	}
	if sym.isEOF() {
		return "<EOF>"
	}

	text, ok := dw.symTab.toText(sym)
	if !ok {
		return fmt.Sprintf("<symbol not found: %v>", sym)
	}

	if strings.HasPrefix(text, "_") {
		pat, ok := dw.sym2AnonPat[sym]
		if !ok {
			return fmt.Sprintf("<pattern not found: %v>", text)
		}

		return fmt.Sprintf(`"%v"`, pat)
	}

	return text
}
