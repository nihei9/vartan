package driver

import (
	mldriver "github.com/nihei9/maleeni/driver"
	"github.com/nihei9/vartan/spec"
)

type gram struct {
	g *spec.CompiledGrammar
}

func NewGrammar(g *spec.CompiledGrammar) *gram {
	return &gram{
		g: g,
	}
}

func (g *gram) LexicalSpecification() mldriver.LexSpec {
	return mldriver.NewLexSpec(g.g.LexicalSpecification.Maleeni.Spec)
}

func (g *gram) Class() string {
	return g.g.ParsingTable.Class
}

func (g *gram) InitialState() int {
	return g.g.ParsingTable.InitialState
}

func (g *gram) StartProduction() int {
	return g.g.ParsingTable.StartProduction
}

func (g *gram) RecoverProduction(prod int) bool {
	return g.g.ParsingTable.RecoverProductions[prod] != 0
}

func (g *gram) Action(state int, terminal int) int {
	return g.g.ParsingTable.Action[state*g.g.ParsingTable.TerminalCount+terminal]
}

func (g *gram) GoTo(state int, lhs int) int {
	return g.g.ParsingTable.GoTo[state*g.g.ParsingTable.NonTerminalCount+lhs]
}

func (g *gram) AlternativeSymbolCount(prod int) int {
	return g.g.ParsingTable.AlternativeSymbolCounts[prod]
}

func (g *gram) TerminalCount() int {
	return g.g.ParsingTable.TerminalCount
}

func (g *gram) ErrorTrapperState(state int) bool {
	return g.g.ParsingTable.ErrorTrapperStates[state] != 0
}

func (g *gram) LHS(prod int) int {
	return g.g.ParsingTable.LHSSymbols[prod]
}

func (g *gram) EOF() int {
	return g.g.ParsingTable.EOFSymbol
}

func (g *gram) Error() int {
	return g.g.ParsingTable.ErrorSymbol
}

func (g *gram) Terminal(terminal int) string {
	return g.g.ParsingTable.Terminals[terminal]
}

func (g *gram) TerminalAlias(terminal int) string {
	return g.g.LexicalSpecification.Maleeni.KindAliases[terminal]
}

func (g *gram) Skip(kind mldriver.KindID) bool {
	return g.g.LexicalSpecification.Maleeni.Skip[kind] > 0
}

func (g *gram) LexicalKindToTerminal(kind mldriver.KindID) int {
	return g.g.LexicalSpecification.Maleeni.KindToTerminal[kind]
}
