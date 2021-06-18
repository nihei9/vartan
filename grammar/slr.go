package grammar

import (
	"fmt"
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

func (t *ParsingTable) writeShiftAction(state stateNum, sym symbol, nextState stateNum) error {
	pos := state.Int()*t.terminalCount + sym.num().Int()
	act := t.actionTable[pos]
	if !act.isEmpty() {
		ty, _, _ := act.describe()
		if ty == ActionTypeReduce {
			return fmt.Errorf("shift/reduce conflict")
		}
	}
	t.actionTable[pos] = newShiftActionEntry(nextState)

	return nil
}

func (t *ParsingTable) writeReduceAction(state stateNum, sym symbol, prod productionNum) error {
	pos := state.Int()*t.terminalCount + sym.num().Int()
	act := t.actionTable[pos]
	if !act.isEmpty() {
		ty, _, p := act.describe()
		if ty == ActionTypeReduce && p != prod {
			return fmt.Errorf("reduce/reduce conflict")
		}
		return fmt.Errorf("shift/reduce conflict")
	}
	t.actionTable[pos] = newReduceActionEntry(prod)

	return nil
}

func (t *ParsingTable) writeGoTo(state stateNum, sym symbol, nextState stateNum) {
	pos := state.Int()*t.nonTerminalCount + sym.num().Int()
	t.goToTable[pos] = newGoToEntry(nextState)
}

func genSLRParsingTable(automaton *lr0Automaton, prods *productionSet, follow *followSet, termCount, nonTermCount int) (*ParsingTable, error) {
	var ptab *ParsingTable
	{
		initialState := automaton.states[automaton.initialState]
		ptab = &ParsingTable{
			actionTable:      make([]actionEntry, len(automaton.states)*termCount),
			goToTable:        make([]goToEntry, len(automaton.states)*nonTermCount),
			stateCount:       len(automaton.states),
			terminalCount:    termCount,
			nonTerminalCount: nonTermCount,
			InitialState:     initialState.num,
		}
	}

	for _, state := range automaton.states {
		for sym, kID := range state.next {
			nextState := automaton.states[kID]
			if sym.isTerminal() {
				err := ptab.writeShiftAction(state.num, sym, nextState.num)
				if err != nil {
					return nil, err
				}
			} else {
				ptab.writeGoTo(state.num, sym, nextState.num)
			}
		}

		for prodID := range state.reducible {
			prod, _ := prods.findByID(prodID)
			flw, err := follow.find(prod.lhs)
			if err != nil {
				return nil, err
			}
			for sym := range flw.symbols {
				err := ptab.writeReduceAction(state.num, sym, prod.num)
				if err != nil {
					return nil, err
				}
			}
			if flw.eof {
				err := ptab.writeReduceAction(state.num, symbolEOF, prod.num)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return ptab, nil
}
