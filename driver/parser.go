package driver

import (
	"fmt"
	"io"

	mldriver "github.com/nihei9/maleeni/driver"
	"github.com/nihei9/vartan/spec"
)

type CST struct {
	KindName string
	Text     string
	Children []*CST
}

func printCST(cst *CST, depth int) {
	for i := 0; i < depth; i++ {
		fmt.Printf("    ")
	}
	fmt.Printf("%v", cst.KindName)
	if cst.Text != "" {
		fmt.Printf(` "%v"`, cst.Text)
	}
	fmt.Printf("\n")
	for _, c := range cst.Children {
		printCST(c, depth+1)
	}
}

type semanticFrame struct {
	cst *CST
}

type Parser struct {
	gram       *spec.CompiledGrammar
	lex        *mldriver.Lexer
	stateStack []int
	semStack   []*semanticFrame
	cst        *CST
}

func NewParser(gram *spec.CompiledGrammar, src io.Reader) (*Parser, error) {
	lex, err := mldriver.NewLexer(gram.LexicalSpecification.Maleeni.Spec, src)
	if err != nil {
		return nil, err
	}

	return &Parser{
		gram:       gram,
		lex:        lex,
		stateStack: []int{},
		semStack:   []*semanticFrame{},
	}, nil
}

func (p *Parser) Parse() error {
	termCount := p.gram.ParsingTable.TerminalCount
	p.push(p.gram.ParsingTable.InitialState)
	tok, err := p.lex.Next()
	if err != nil {
		return err
	}
	if tok.Invalid {
		return fmt.Errorf("invalid token: %+v", tok)
	}
	for {
		var tsym int
		if tok.EOF {
			tsym = p.gram.ParsingTable.EOFSymbol
		} else {
			tsym = p.gram.LexicalSpecification.Maleeni.KindToTerminal[tok.Mode.Int()][tok.Kind]
		}
		act := p.gram.ParsingTable.Action[p.top()*termCount+tsym]
		switch {
		case act < 0: // Shift
			tokText := tok.Text()
			tok, err = p.shift(act * -1)
			if err != nil {
				return err
			}
			// semantic action
			p.semStack = append(p.semStack, &semanticFrame{
				cst: &CST{
					KindName: p.gram.ParsingTable.Terminals[tsym],
					Text:     tokText,
				},
			})
		case act > 0: // Reduce
			accepted := p.reduce(act)
			if accepted {
				p.cst = p.semStack[len(p.semStack)-1].cst
				return nil
			}
			// semantic action
			prodNum := act
			lhs := p.gram.ParsingTable.LHSSymbols[prodNum]
			n := p.gram.ParsingTable.AlternativeSymbolCounts[prodNum]
			children := []*CST{}
			for _, f := range p.semStack[len(p.semStack)-n:] {
				children = append(children, f.cst)
			}
			p.semStack = p.semStack[:len(p.semStack)-n]
			p.semStack = append(p.semStack, &semanticFrame{
				cst: &CST{
					KindName: p.gram.ParsingTable.NonTerminals[lhs],
					Children: children,
				},
			})
		default:
			return fmt.Errorf("unexpected token: %v", tok)
		}
	}
}

func (p *Parser) shift(nextState int) (*mldriver.Token, error) {
	p.push(nextState)
	tok, err := p.lex.Next()
	if err != nil {
		return nil, err
	}
	if tok.Invalid {
		return nil, fmt.Errorf("invalid token: %+v", tok)
	}
	return tok, nil
}

func (p *Parser) reduce(prodNum int) bool {
	tab := p.gram.ParsingTable
	lhs := tab.LHSSymbols[prodNum]
	if lhs == tab.LHSSymbols[tab.StartProduction] {
		return true
	}
	n := tab.AlternativeSymbolCounts[prodNum]
	p.pop(n)
	nextState := tab.GoTo[p.top()*tab.NonTerminalCount+lhs]
	p.push(nextState)
	return false
}

func (p *Parser) top() int {
	return p.stateStack[len(p.stateStack)-1]
}

func (p *Parser) push(state int) {
	p.stateStack = append(p.stateStack, state)
}

func (p *Parser) pop(n int) {
	p.stateStack = p.stateStack[:len(p.stateStack)-n]
}

func (p *Parser) GetCST() *CST {
	return p.cst
}
