package grammar

import (
	"fmt"
	"sort"

	"github.com/nihei9/vartan/grammar/symbol"
	spec "github.com/nihei9/vartan/spec/grammar"
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

type conflictResolutionMethod int

func (m conflictResolutionMethod) Int() int {
	return int(m)
}

const (
	ResolvedByPrec      conflictResolutionMethod = 1
	ResolvedByAssoc     conflictResolutionMethod = 2
	ResolvedByShift     conflictResolutionMethod = 3
	ResolvedByProdOrder conflictResolutionMethod = 4
)

type conflict interface {
	conflict()
}

type shiftReduceConflict struct {
	state      stateNum
	sym        symbol.Symbol
	nextState  stateNum
	prodNum    productionNum
	resolvedBy conflictResolutionMethod
}

func (c *shiftReduceConflict) conflict() {
}

type reduceReduceConflict struct {
	state      stateNum
	sym        symbol.Symbol
	prodNum1   productionNum
	prodNum2   productionNum
	resolvedBy conflictResolutionMethod
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

func (t *ParsingTable) getAction(state stateNum, sym symbol.SymbolNum) (ActionType, stateNum, productionNum) {
	pos := state.Int()*t.terminalCount + sym.Int()
	return t.actionTable[pos].describe()
}

func (t *ParsingTable) getGoTo(state stateNum, sym symbol.SymbolNum) (GoToType, stateNum) {
	pos := state.Int()*t.nonTerminalCount + sym.Int()
	return t.goToTable[pos].describe()
}

func (t *ParsingTable) readAction(row int, col int) actionEntry {
	return t.actionTable[row*t.terminalCount+col]
}

func (t *ParsingTable) writeAction(row int, col int, act actionEntry) {
	t.actionTable[row*t.terminalCount+col] = act
}

func (t *ParsingTable) writeGoTo(state stateNum, sym symbol.Symbol, nextState stateNum) {
	pos := state.Int()*t.nonTerminalCount + sym.Num().Int()
	t.goToTable[pos] = newGoToEntry(nextState)
}

type lrTableBuilder struct {
	automaton    *lr0Automaton
	prods        *productionSet
	termCount    int
	nonTermCount int
	symTab       *symbol.SymbolTableReader
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
			if sym.IsTerminal() {
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
func (b *lrTableBuilder) writeShiftAction(tab *ParsingTable, state stateNum, sym symbol.Symbol, nextState stateNum) {
	act := tab.readAction(state.Int(), sym.Num().Int())
	if !act.isEmpty() {
		ty, _, p := act.describe()
		if ty == ActionTypeReduce {
			act, method := b.resolveSRConflict(sym.Num(), p)
			b.conflicts = append(b.conflicts, &shiftReduceConflict{
				state:      state,
				sym:        sym,
				nextState:  nextState,
				prodNum:    p,
				resolvedBy: method,
			})
			if act == ActionTypeShift {
				tab.writeAction(state.Int(), sym.Num().Int(), newShiftActionEntry(nextState))
			}
			return
		}
	}
	tab.writeAction(state.Int(), sym.Num().Int(), newShiftActionEntry(nextState))
}

// writeReduceAction writes a reduce action to the parsing table. When a shift/reduce conflict occurred,
// we prioritize the shift action, and when a reduce/reduce conflict we prioritize the action that reduces
// the production with higher priority. Productions defined earlier in the grammar file have a higher priority.
func (b *lrTableBuilder) writeReduceAction(tab *ParsingTable, state stateNum, sym symbol.Symbol, prod productionNum) {
	act := tab.readAction(state.Int(), sym.Num().Int())
	if !act.isEmpty() {
		ty, s, p := act.describe()
		switch ty {
		case ActionTypeReduce:
			if p == prod {
				return
			}

			b.conflicts = append(b.conflicts, &reduceReduceConflict{
				state:      state,
				sym:        sym,
				prodNum1:   p,
				prodNum2:   prod,
				resolvedBy: ResolvedByProdOrder,
			})
			if p < prod {
				tab.writeAction(state.Int(), sym.Num().Int(), newReduceActionEntry(p))
			} else {
				tab.writeAction(state.Int(), sym.Num().Int(), newReduceActionEntry(prod))
			}
		case ActionTypeShift:
			act, method := b.resolveSRConflict(sym.Num(), prod)
			b.conflicts = append(b.conflicts, &shiftReduceConflict{
				state:      state,
				sym:        sym,
				nextState:  s,
				prodNum:    prod,
				resolvedBy: method,
			})
			if act == ActionTypeReduce {
				tab.writeAction(state.Int(), sym.Num().Int(), newReduceActionEntry(prod))
			}
		}
		return
	}
	tab.writeAction(state.Int(), sym.Num().Int(), newReduceActionEntry(prod))
}

func (b *lrTableBuilder) resolveSRConflict(sym symbol.SymbolNum, prod productionNum) (ActionType, conflictResolutionMethod) {
	symPrec := b.precAndAssoc.terminalPrecedence(sym)
	prodPrec := b.precAndAssoc.productionPredence(prod)
	if symPrec == 0 || prodPrec == 0 {
		return ActionTypeShift, ResolvedByShift
	}
	if symPrec == prodPrec {
		assoc := b.precAndAssoc.productionAssociativity(prod)
		if assoc != assocTypeLeft {
			return ActionTypeShift, ResolvedByAssoc
		}
		return ActionTypeReduce, ResolvedByAssoc
	}
	if symPrec < prodPrec {
		return ActionTypeShift, ResolvedByPrec
	}
	return ActionTypeReduce, ResolvedByPrec
}

func (b *lrTableBuilder) genReport(tab *ParsingTable, gram *Grammar) (*spec.Report, error) {
	var terms []*spec.Terminal
	{
		termSyms := b.symTab.TerminalSymbols()
		terms = make([]*spec.Terminal, len(termSyms)+1)

		for _, sym := range termSyms {
			name, ok := b.symTab.ToText(sym)
			if !ok {
				return nil, fmt.Errorf("failed to generate terminals: symbol not found: %v", sym)
			}

			term := &spec.Terminal{
				Number: sym.Num().Int(),
				Name:   name,
			}

			prec := b.precAndAssoc.terminalPrecedence(sym.Num())
			if prec != precNil {
				term.Precedence = prec
			}

			assoc := b.precAndAssoc.terminalAssociativity(sym.Num())
			switch assoc {
			case assocTypeLeft:
				term.Associativity = "l"
			case assocTypeRight:
				term.Associativity = "r"
			}

			terms[sym.Num()] = term
		}
	}

	var nonTerms []*spec.NonTerminal
	{
		nonTermSyms := b.symTab.NonTerminalSymbols()
		nonTerms = make([]*spec.NonTerminal, len(nonTermSyms)+1)
		for _, sym := range nonTermSyms {
			name, ok := b.symTab.ToText(sym)
			if !ok {
				return nil, fmt.Errorf("failed to generate non-terminals: symbol not found: %v", sym)
			}

			nonTerms[sym.Num()] = &spec.NonTerminal{
				Number: sym.Num().Int(),
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
				if e.IsTerminal() {
					rhs[i] = e.Num().Int()
				} else {
					rhs[i] = e.Num().Int() * -1
				}
			}

			prod := &spec.Production{
				Number: p.num.Int(),
				LHS:    p.lhs.Num().Int(),
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
			var reduce []*spec.Reduce
			var goTo []*spec.Transition
			{
			TERMINALS_LOOP:
				for _, t := range b.symTab.TerminalSymbols() {
					act, next, prod := tab.getAction(s.num, t.Num())
					switch act {
					case ActionTypeShift:
						shift = append(shift, &spec.Transition{
							Symbol: t.Num().Int(),
							State:  next.Int(),
						})
					case ActionTypeReduce:
						for _, r := range reduce {
							if r.Production == prod.Int() {
								r.LookAhead = append(r.LookAhead, t.Num().Int())
								continue TERMINALS_LOOP
							}
						}
						reduce = append(reduce, &spec.Reduce{
							LookAhead:  []int{t.Num().Int()},
							Production: prod.Int(),
						})
					}
				}

				for _, n := range b.symTab.NonTerminalSymbols() {
					ty, next := tab.getGoTo(s.num, n.Num())
					if ty == GoToTypeRegistered {
						goTo = append(goTo, &spec.Transition{
							Symbol: n.Num().Int(),
							State:  next.Int(),
						})
					}
				}

				sort.Slice(shift, func(i, j int) bool {
					return shift[i].State < shift[j].State
				})
				sort.Slice(reduce, func(i, j int) bool {
					return reduce[i].Production < reduce[j].Production
				})
				sort.Slice(goTo, func(i, j int) bool {
					return goTo[i].State < goTo[j].State
				})
			}

			sr := []*spec.SRConflict{}
			rr := []*spec.RRConflict{}
			{
				for _, c := range srConflicts[s.num] {
					conflict := &spec.SRConflict{
						Symbol:     c.sym.Num().Int(),
						State:      c.nextState.Int(),
						Production: c.prodNum.Int(),
						ResolvedBy: c.resolvedBy.Int(),
					}

					ty, s, p := tab.getAction(s.num, c.sym.Num())
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
						Symbol:      c.sym.Num().Int(),
						Production1: c.prodNum1.Int(),
						Production2: c.prodNum2.Int(),
						ResolvedBy:  c.resolvedBy.Int(),
					}

					_, _, p := tab.getAction(s.num, c.sym.Num())
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

	return &spec.Report{
		Terminals:    terms,
		NonTerminals: nonTerms,
		Productions:  prods,
		States:       states,
	}, nil
}
