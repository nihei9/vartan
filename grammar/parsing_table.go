package grammar

import (
	"fmt"
	"sort"

	"github.com/nihei9/vartan/spec"
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

	// errorTrapperStates's index means a state number, and when `errorTrapperStates[stateNum]` is `1`,
	// the state has an item having the following form. The `α` and `β` can be empty.
	//
	// A → α・error β
	errorTrapperStates []int

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

func (t *ParsingTable) readAction(row int, col int) actionEntry {
	return t.actionTable[row*t.terminalCount+col]
}

func (t *ParsingTable) writeAction(row int, col int, act actionEntry) {
	t.actionTable[row*t.terminalCount+col] = act
}

func (t *ParsingTable) writeGoTo(state stateNum, sym symbol, nextState stateNum) {
	pos := state.Int()*t.nonTerminalCount + sym.num().Int()
	t.goToTable[pos] = newGoToEntry(nextState)
}

type lrTableBuilder struct {
	class        Class
	automaton    *lr0Automaton
	prods        *productionSet
	termCount    int
	nonTermCount int
	symTab       *symbolTable
	sym2AnonPat  map[symbol]string
	precAndAssoc *precAndAssoc

	conflicts []conflict
}

func (b *lrTableBuilder) build() (*ParsingTable, error) {
	var ptab *ParsingTable
	{
		initialState := b.automaton.states[b.automaton.initialState]
		ptab = &ParsingTable{
			actionTable:        make([]actionEntry, len(b.automaton.states)*b.termCount),
			goToTable:          make([]goToEntry, len(b.automaton.states)*b.nonTermCount),
			stateCount:         len(b.automaton.states),
			terminalCount:      b.termCount,
			nonTerminalCount:   b.nonTermCount,
			errorTrapperStates: make([]int, len(b.automaton.states)),
			InitialState:       initialState.num,
		}
	}

	for _, state := range b.automaton.states {
		if state.isErrorTrapper {
			ptab.errorTrapperStates[state.num] = 1
		}

		for sym, kID := range state.next {
			nextState := b.automaton.states[kID]
			if sym.isTerminal() {
				b.writeShiftAction(ptab, state.num, sym, nextState.num)
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
				b.writeReduceAction(ptab, state.num, a, reducibleProd.num)
			}
		}
	}

	return ptab, nil
}

// writeShiftAction writes a shift action to the parsing table. When a shift/reduce conflict occurred,
// we prioritize the shift action.
func (b *lrTableBuilder) writeShiftAction(tab *ParsingTable, state stateNum, sym symbol, nextState stateNum) {
	act := tab.readAction(state.Int(), sym.num().Int())
	if !act.isEmpty() {
		ty, _, p := act.describe()
		if ty == ActionTypeReduce {
			b.conflicts = append(b.conflicts, &shiftReduceConflict{
				state:     state,
				sym:       sym,
				nextState: nextState,
				prodNum:   p,
			})

			if b.resolveConflict(sym.num(), p) == ActionTypeShift {
				tab.writeAction(state.Int(), sym.num().Int(), newShiftActionEntry(nextState))
			}

			return
		}
	}

	tab.writeAction(state.Int(), sym.num().Int(), newShiftActionEntry(nextState))
}

// writeReduceAction writes a reduce action to the parsing table. When a shift/reduce conflict occurred,
// we prioritize the shift action, and when a reduce/reduce conflict we prioritize the action that reduces
// the production with higher priority. Productions defined earlier in the grammar file have a higher priority.
func (b *lrTableBuilder) writeReduceAction(tab *ParsingTable, state stateNum, sym symbol, prod productionNum) {
	act := tab.readAction(state.Int(), sym.num().Int())
	if !act.isEmpty() {
		ty, s, p := act.describe()
		switch ty {
		case ActionTypeReduce:
			if p == prod {
				return
			}

			b.conflicts = append(b.conflicts, &reduceReduceConflict{
				state:    state,
				sym:      sym,
				prodNum1: p,
				prodNum2: prod,
			})

			if p < prod {
				tab.writeAction(state.Int(), sym.num().Int(), newReduceActionEntry(p))
			} else {
				tab.writeAction(state.Int(), sym.num().Int(), newReduceActionEntry(prod))
			}
		case ActionTypeShift:
			b.conflicts = append(b.conflicts, &shiftReduceConflict{
				state:     state,
				sym:       sym,
				nextState: s,
				prodNum:   prod,
			})

			if b.resolveConflict(sym.num(), prod) == ActionTypeReduce {
				tab.writeAction(state.Int(), sym.num().Int(), newReduceActionEntry(prod))
			}
		}

		return
	}

	tab.writeAction(state.Int(), sym.num().Int(), newReduceActionEntry(prod))
}

func (b *lrTableBuilder) resolveConflict(sym symbolNum, prod productionNum) ActionType {
	symPrec := b.precAndAssoc.terminalPrecedence(sym)
	prodPrec := b.precAndAssoc.productionPredence(prod)
	if symPrec < prodPrec {
		return ActionTypeShift
	}
	if symPrec == prodPrec {
		assoc := b.precAndAssoc.productionAssociativity(prod)
		if assoc != assocTypeLeft {
			return ActionTypeShift
		}
	}

	return ActionTypeReduce
}

func (b *lrTableBuilder) genDescription(tab *ParsingTable, gram *Grammar) (*spec.Description, error) {
	var terms []*spec.Terminal
	{
		termSyms := b.symTab.terminalSymbols()
		terms = make([]*spec.Terminal, len(termSyms)+2)

		terms[symbolEOF.num()] = &spec.Terminal{
			Number: symbolEOF.num().Int(),
			Name:   "<eof>",
		}

		for _, sym := range termSyms {
			name, ok := b.symTab.toText(sym)
			if !ok {
				return nil, fmt.Errorf("failed to generate terminals: symbol not found: %v", sym)
			}

			term := &spec.Terminal{
				Number: sym.num().Int(),
				Name:   name,
				Alias:  gram.kindAliases[sym],
			}

			pat, ok := b.sym2AnonPat[sym]
			if ok {
				term.Anonymous = true
				term.Pattern = pat
			}

			prec := b.precAndAssoc.terminalPrecedence(sym.num())
			if prec != precNil {
				term.Precedence = prec
			}

			assoc := b.precAndAssoc.terminalAssociativity(sym.num())
			switch assoc {
			case assocTypeLeft:
				term.Associativity = "l"
			case assocTypeRight:
				term.Associativity = "r"
			}

			terms[sym.num()] = term
		}
	}

	var nonTerms []*spec.NonTerminal
	{
		nonTermSyms := b.symTab.nonTerminalSymbols()
		nonTerms = make([]*spec.NonTerminal, len(nonTermSyms)+1)
		for _, sym := range nonTermSyms {
			name, ok := b.symTab.toText(sym)
			if !ok {
				return nil, fmt.Errorf("failed to generate non-terminals: symbol not found: %v", sym)
			}

			nonTerms[sym.num()] = &spec.NonTerminal{
				Number: sym.num().Int(),
				Name:   name,
			}
		}
	}

	var prods []*spec.Production
	{
		ps := gram.productionSet.getAllProductions()
		prods = make([]*spec.Production, len(ps)+1)
		for _, p := range ps {
			rhs := make([]int, len(p.rhs))
			for i, e := range p.rhs {
				if e.isTerminal() {
					rhs[i] = e.num().Int()
				} else {
					rhs[i] = e.num().Int() * -1
				}
			}

			prod := &spec.Production{
				Number: p.num.Int(),
				LHS:    p.lhs.num().Int(),
				RHS:    rhs,
			}

			prec := b.precAndAssoc.productionPredence(p.num)
			if prec != precNil {
				prod.Precedence = prec
			}

			assoc := b.precAndAssoc.productionAssociativity(p.num)
			switch assoc {
			case assocTypeLeft:
				prod.Associativity = "l"
			case assocTypeRight:
				prod.Associativity = "r"
			}

			prods[p.num.Int()] = prod
		}
	}

	var states []*spec.State
	{
		srConflicts := map[stateNum][]*shiftReduceConflict{}
		rrConflicts := map[stateNum][]*reduceReduceConflict{}
		for _, con := range b.conflicts {
			switch c := con.(type) {
			case *shiftReduceConflict:
				srConflicts[c.state] = append(srConflicts[c.state], c)
			case *reduceReduceConflict:
				rrConflicts[c.state] = append(rrConflicts[c.state], c)
			}
		}

		states = make([]*spec.State, len(b.automaton.states))
		for _, s := range b.automaton.states {
			kernel := make([]*spec.Item, len(s.items))
			for i, item := range s.items {
				p, ok := b.prods.findByID(item.prod)
				if !ok {
					return nil, fmt.Errorf("failed to generate states: production of kernel item not found: %v", item.prod)
				}

				kernel[i] = &spec.Item{
					Production: p.num.Int(),
					Dot:        item.dot,
				}
			}

			sort.Slice(kernel, func(i, j int) bool {
				if kernel[i].Production < kernel[j].Production {
					return true
				}
				if kernel[i].Production > kernel[j].Production {
					return false
				}
				return kernel[i].Dot < kernel[j].Dot
			})

			var shift []*spec.Transition
			var goTo []*spec.Transition
			for sym, kID := range s.next {
				nextState := b.automaton.states[kID]
				if sym.isTerminal() {
					shift = append(shift, &spec.Transition{
						Symbol: sym.num().Int(),
						State:  nextState.num.Int(),
					})
				} else {
					goTo = append(goTo, &spec.Transition{
						Symbol: sym.num().Int(),
						State:  nextState.num.Int(),
					})
				}
			}

			sort.Slice(shift, func(i, j int) bool {
				return shift[i].State < shift[j].State
			})

			sort.Slice(goTo, func(i, j int) bool {
				return goTo[i].State < goTo[j].State
			})

			var reduce []*spec.Reduce
			for _, item := range s.items {
				if !item.reducible {
					continue
				}

				syms := make([]int, len(item.lookAhead.symbols))
				i := 0
				for a := range item.lookAhead.symbols {
					syms[i] = a.num().Int()
					i++
				}

				sort.Slice(syms, func(i, j int) bool {
					return syms[i] < syms[j]
				})

				prod, ok := gram.productionSet.findByID(item.prod)
				if !ok {
					return nil, fmt.Errorf("failed to generate states: reducible production not found: %v", item.prod)
				}

				reduce = append(reduce, &spec.Reduce{
					LookAhead:  syms,
					Production: prod.num.Int(),
				})

				sort.Slice(reduce, func(i, j int) bool {
					return reduce[i].Production < reduce[j].Production
				})
			}

			sr := []*spec.SRConflict{}
			rr := []*spec.RRConflict{}
			{
				for _, c := range srConflicts[s.num] {
					conflict := &spec.SRConflict{
						Symbol:     c.sym.num().Int(),
						State:      c.nextState.Int(),
						Production: c.prodNum.Int(),
					}

					ty, s, p := tab.getAction(s.num, c.sym.num())
					switch ty {
					case ActionTypeShift:
						n := s.Int()
						conflict.AdoptedState = &n
					case ActionTypeReduce:
						n := p.Int()
						conflict.AdoptedProduction = &n
					}

					sr = append(sr, conflict)
				}

				sort.Slice(sr, func(i, j int) bool {
					return sr[i].Symbol < sr[j].Symbol
				})

				for _, c := range rrConflicts[s.num] {
					conflict := &spec.RRConflict{
						Symbol:      c.sym.num().Int(),
						Production1: c.prodNum1.Int(),
						Production2: c.prodNum2.Int(),
					}

					_, _, p := tab.getAction(s.num, c.sym.num())
					conflict.AdoptedProduction = p.Int()

					rr = append(rr, conflict)
				}

				sort.Slice(rr, func(i, j int) bool {
					return rr[i].Symbol < rr[j].Symbol
				})
			}

			states[s.num.Int()] = &spec.State{
				Number:     s.num.Int(),
				Kernel:     kernel,
				Shift:      shift,
				Reduce:     reduce,
				GoTo:       goTo,
				SRConflict: sr,
				RRConflict: rr,
			}
		}
	}

	return &spec.Description{
		Class:        string(b.class),
		Terminals:    terms,
		NonTerminals: nonTerms,
		Productions:  prods,
		States:       states,
	}, nil
}
