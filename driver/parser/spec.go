package parser

import spec "github.com/nihei9/vartan/spec/grammar"

type grammarImpl struct {
	g *spec.CompiledGrammar
}

func NewGrammar(g *spec.CompiledGrammar) *grammarImpl {
	return &grammarImpl{
		g: g,
	}
}

func (g *grammarImpl) InitialState() int {
	return g.g.Syntactic.InitialState
}

func (g *grammarImpl) StartProduction() int {
	return g.g.Syntactic.StartProduction
}

func (g *grammarImpl) RecoverProduction(prod int) bool {
	return g.g.Syntactic.RecoverProductions[prod] != 0
}

func (g *grammarImpl) Action(state int, terminal int) int {
	return g.g.Syntactic.Action[state*g.g.Syntactic.TerminalCount+terminal]
}

func (g *grammarImpl) GoTo(state int, lhs int) int {
	return g.g.Syntactic.GoTo[state*g.g.Syntactic.NonTerminalCount+lhs]
}

func (g *grammarImpl) AlternativeSymbolCount(prod int) int {
	return g.g.Syntactic.AlternativeSymbolCounts[prod]
}

func (g *grammarImpl) TerminalCount() int {
	return g.g.Syntactic.TerminalCount
}

func (g *grammarImpl) SkipTerminal(terminal int) bool {
	return g.g.Syntactic.TerminalSkip[terminal] == 1
}

func (g *grammarImpl) ErrorTrapperState(state int) bool {
	return g.g.Syntactic.ErrorTrapperStates[state] != 0
}

func (g *grammarImpl) NonTerminal(nonTerminal int) string {
	return g.g.Syntactic.NonTerminals[nonTerminal]
}

func (g *grammarImpl) LHS(prod int) int {
	return g.g.Syntactic.LHSSymbols[prod]
}

func (g *grammarImpl) EOF() int {
	return g.g.Syntactic.EOFSymbol
}

func (g *grammarImpl) Error() int {
	return g.g.Syntactic.ErrorSymbol
}

func (g *grammarImpl) Terminal(terminal int) string {
	return g.g.Syntactic.Terminals[terminal]
}

func (g *grammarImpl) ASTAction(prod int) []int {
	return g.g.ASTAction.Entries[prod]
}
