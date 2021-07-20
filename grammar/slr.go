package grammar

import (
	"fmt"
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
	if len(conflicts) > 0 {
		var msg strings.Builder
		for _, conflict := range conflicts {
			fmt.Fprintf(&msg, "\n")
			switch c := conflict.(type) {
			case *shiftReduceConflict:
				sym, ok := b.symTab.toText(c.sym)
				if !ok {
					sym = fmt.Sprintf("<not found: %v>", c.sym)
				}
				fmt.Fprintf(&msg, "%v: shift/reduce conflict (shift %v, reduce %v) on %v", c.state, c.nextState, c.prodNum, sym)
			case *reduceReduceConflict:
				sym, ok := b.symTab.toText(c.sym)
				if !ok {
					sym = fmt.Sprintf("<not found: %v>", c.sym)
				}
				fmt.Fprintf(&msg, "%v: reduce/reduce conflict (reduce %v and %v) on %v", c.state, c.prodNum1, c.prodNum2, sym)
			}
		}
		return nil, fmt.Errorf("%v conflicts:%v", len(conflicts), msg.String())
	}

	return ptab, nil
}
