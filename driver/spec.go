package driver

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
	return g.g.ParsingTable.InitialState
}

func (g *grammarImpl) StartProduction() int {
	return g.g.ParsingTable.StartProduction
}

func (g *grammarImpl) RecoverProduction(prod int) bool {
	return g.g.ParsingTable.RecoverProductions[prod] != 0
}

func (g *grammarImpl) Action(state int, terminal int) int {
	return g.g.ParsingTable.Action[state*g.g.ParsingTable.TerminalCount+terminal]
}

func (g *grammarImpl) GoTo(state int, lhs int) int {
	return g.g.ParsingTable.GoTo[state*g.g.ParsingTable.NonTerminalCount+lhs]
}

func (g *grammarImpl) AlternativeSymbolCount(prod int) int {
	return g.g.ParsingTable.AlternativeSymbolCounts[prod]
}

func (g *grammarImpl) TerminalCount() int {
	return g.g.ParsingTable.TerminalCount
}

func (g *grammarImpl) SkipTerminal(terminal int) bool {
	return g.g.ParsingTable.TerminalSkip[terminal] == 1
}

func (g *grammarImpl) ErrorTrapperState(state int) bool {
	return g.g.ParsingTable.ErrorTrapperStates[state] != 0
}

func (g *grammarImpl) NonTerminal(nonTerminal int) string {
	return g.g.ParsingTable.NonTerminals[nonTerminal]
}

func (g *grammarImpl) LHS(prod int) int {
	return g.g.ParsingTable.LHSSymbols[prod]
}

func (g *grammarImpl) EOF() int {
	return g.g.ParsingTable.EOFSymbol
}

func (g *grammarImpl) Error() int {
	return g.g.ParsingTable.ErrorSymbol
}

func (g *grammarImpl) Terminal(terminal int) string {
	return g.g.ParsingTable.Terminals[terminal]
}

func (g *grammarImpl) ASTAction(prod int) []int {
	return g.g.ASTAction.Entries[prod]
}
