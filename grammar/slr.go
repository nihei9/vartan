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
	actionTable      []actionEntry
	goToTable        []goToEntry
	stateCount       int
	terminalCount    int
	nonTerminalCount int

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
			actionTable:      make([]actionEntry, len(b.automaton.states)*b.termCount),
			goToTable:        make([]goToEntry, len(b.automaton.states)*b.nonTermCount),
			stateCount:       len(b.automaton.states),
			terminalCount:    b.termCount,
			nonTerminalCount: b.nonTermCount,
			InitialState:     initialState.num,
		}
	}

	var conflicts []conflict
	for _, state := range b.automaton.states {
		for sym, kID := range state.next {
			nextState := b.automaton.states[kID]
			if sym.isTerminal() {
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
				c := ptab.writeReduceAction(state.num, sym, prod.num)
				if c != nil {
					conflicts = append(conflicts, c)
					continue
				}
			}
			if flw.eof {
				c := ptab.writeReduceAction(state.num, symbolEOF, prod.num)
				if c != nil {
					conflicts = append(conflicts, c)
					continue
				}
			}
		}
	}

	b.conflicts = conflicts

	if len(conflicts) > 0 {
		return nil, fmt.Errorf("%v conflicts", len(conflicts))
	}

	return ptab, nil
}

func (b *slrTableBuilder) writeDescription(w io.Writer) {
	conflicts := map[stateNum][]conflict{}
	for _, con := range b.conflicts {
		switch c := con.(type) {
		case *shiftReduceConflict:
			conflicts[c.state] = append(conflicts[c.state], c)
		case *reduceReduceConflict:
			conflicts[c.state] = append(conflicts[c.state], c)
		}
	}

	fmt.Fprintf(w, "# Conflicts\n\n")

	if len(b.conflicts) > 0 {
		fmt.Fprintf(w, "%v conflics:\n\n", len(b.conflicts))

		for _, conflict := range b.conflicts {
			switch c := conflict.(type) {
			case *shiftReduceConflict:
				fmt.Fprintf(w, "%v: shift/reduce conflict (shift %v, reduce %v) on %v\n", c.state, c.nextState, c.prodNum, b.symbolToText(c.sym))
			case *reduceReduceConflict:
				fmt.Fprintf(w, "%v: reduce/reduce conflict (reduce %v and %v) on %v\n", c.state, c.prodNum1, c.prodNum2, b.symbolToText(c.sym))
			}
		}
		fmt.Fprintf(w, "\n")
	} else {
		fmt.Fprintf(w, "no conflicts\n\n")
	}

	fmt.Fprintf(w, "# Productions\n\n")

	fmt.Fprintf(w, "%v productions:\n\n", len(b.prods.getAllProductions()))

	for _, prod := range b.prods.getAllProductions() {
		fmt.Fprintf(w, "%4v %v\n", prod.num, b.productionToString(prod, -1))
	}

	fmt.Fprintf(w, "\n# States\n\n")

	fmt.Fprintf(w, "%v states:\n\n", len(b.automaton.states))

	for _, state := range b.automaton.states {
		fmt.Fprintf(w, "state %v\n", state.num)
		for _, item := range state.items {
			prod, ok := b.prods.findByID(item.prod)
			if !ok {
				fmt.Fprintf(w, "<production not found>\n")
				continue
			}
			fmt.Fprintf(w, "    %v\n", b.productionToString(prod, item.dot))
		}

		fmt.Fprintf(w, "\n")

		var shiftRecs []string
		var reduceRecs []string
		var gotoRecs []string
		var accRec string
		{
			for sym, kID := range state.next {
				nextState := b.automaton.states[kID]
				if sym.isTerminal() {
					shiftRecs = append(shiftRecs, fmt.Sprintf("shift  %4v on %v", nextState.num, b.symbolToText(sym)))
				} else {
					gotoRecs = append(gotoRecs, fmt.Sprintf("goto   %4v on %v", nextState.num, b.symbolToText(sym)))
				}
			}

			for prodID := range state.reducible {
				prod, ok := b.prods.findByID(prodID)
				if !ok {
					reduceRecs = append(reduceRecs, "<production not found>")
					continue
				}
				if prod.lhs.isStart() {
					accRec = "accept on <EOF>"
					continue
				}
				flw, err := b.follow.find(prod.lhs)
				if err != nil {
					reduceRecs = append(reduceRecs, fmt.Sprintf("%v", err))
					continue
				}
				for sym := range flw.symbols {
					reduceRecs = append(reduceRecs, fmt.Sprintf("reduce %4v on %v", prod.num, b.symbolToText(sym)))
				}
				if flw.eof {
					reduceRecs = append(reduceRecs, fmt.Sprintf("reduce %4v on <EOF>", prod.num))
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
					fmt.Fprintf(w, "    shift/reduce conflict (shift %v, reduce %v) on %v\n", c.nextState, c.prodNum, b.symbolToText(c.sym))
				case *reduceReduceConflict:
					fmt.Fprintf(w, "    reduce/reduce conflict (reduce %v and %v) on %v\n", c.prodNum1, c.prodNum2, b.symbolToText(c.sym))
				}
			}
			fmt.Fprintf(w, "\n")
		}
	}
}

func (b *slrTableBuilder) productionToString(prod *production, dot int) string {
	var w strings.Builder
	fmt.Fprintf(&w, "%v →", b.symbolToText(prod.lhs))
	for n, rhs := range prod.rhs {
		if n == dot {
			fmt.Fprintf(&w, " ・")
		}
		fmt.Fprintf(&w, " %v", b.symbolToText(rhs))
	}
	if dot == len(prod.rhs) {
		fmt.Fprintf(&w, " ・")
	}

	return w.String()
}

func (b *slrTableBuilder) symbolToText(sym symbol) string {
	text, ok := b.symTab.toText(sym)
	if !ok {
		return fmt.Sprintf("<symbol not found: %v>", sym)
	}

	if strings.HasPrefix(text, "_") {
		pat, ok := b.sym2AnonPat[sym]
		if !ok {
			return fmt.Sprintf("<pattern not found: %v>", text)
		}

		return fmt.Sprintf(`"%v"`, pat)
	}

	return text
}
