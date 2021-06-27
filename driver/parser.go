package driver

import (
	"fmt"
	"io"

	mldriver "github.com/nihei9/maleeni/driver"
	"github.com/nihei9/vartan/spec"
)

type Node struct {
	KindName string
	Text     string
	Children []*Node
}

func PrintTree(node *Node, depth int) {
	for i := 0; i < depth; i++ {
		fmt.Printf("    ")
	}
	fmt.Printf("%v", node.KindName)
	if node.Text != "" {
		fmt.Printf(` "%v"`, node.Text)
	}
	fmt.Printf("\n")
	for _, c := range node.Children {
		PrintTree(c, depth+1)
	}
}

type semanticFrame struct {
	cst *Node
	ast *Node
}

type Parser struct {
	gram       *spec.CompiledGrammar
	lex        *mldriver.Lexer
	stateStack []int
	semStack   []*semanticFrame
	cst        *Node
	ast        *Node
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
	tok, err := p.nextToken()
	if err != nil {
		return err
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
				cst: &Node{
					KindName: p.gram.ParsingTable.Terminals[tsym],
					Text:     tokText,
				},
				ast: &Node{
					KindName: p.gram.ParsingTable.Terminals[tsym],
					Text:     tokText,
				},
			})
		case act > 0: // Reduce
			accepted := p.reduce(act)
			if accepted {
				top := p.semStack[len(p.semStack)-1]
				p.cst = top.cst
				p.ast = top.ast
				return nil
			}

			// semantic action
			prodNum := act
			lhs := p.gram.ParsingTable.LHSSymbols[prodNum]
			n := p.gram.ParsingTable.AlternativeSymbolCounts[prodNum]
			handle := p.semStack[len(p.semStack)-n:]

			var cst *Node
			{
				children := make([]*Node, len(handle))
				for i, f := range handle {
					children[i] = f.cst
				}
				cst = &Node{
					KindName: p.gram.ParsingTable.NonTerminals[lhs],
					Children: children,
				}
			}

			var ast *Node
			{
				act := p.gram.ASTAction.Entries[prodNum]
				children := []*Node{}
				if act != nil {
					for _, e := range act {
						if e > 0 {
							offset := e - 1
							children = append(children, handle[offset].ast)
						} else {
							offset := e*-1 - 1
							for _, c := range handle[offset].ast.Children {
								children = append(children, c)
							}
						}
					}
				} else {
					// If an alternative has no AST action, a driver generates
					// a node with the same structure as a CST.
					for _, f := range handle {
						children = append(children, f.ast)
					}
				}
				ast = &Node{
					KindName: p.gram.ParsingTable.NonTerminals[lhs],
					Children: children,
				}
			}

			p.semStack = p.semStack[:len(p.semStack)-n]
			p.semStack = append(p.semStack, &semanticFrame{
				cst: cst,
				ast: ast,
			})
		default:
			return fmt.Errorf("unexpected token: %v", tok)
		}
	}
}

func (p *Parser) nextToken() (*mldriver.Token, error) {
	skip := p.gram.LexicalSpecification.Maleeni.Skip
	for {
		tok, err := p.lex.Next()
		if err != nil {
			return nil, err
		}
		if tok.Invalid {
			return nil, fmt.Errorf("invalid token: %+v", tok)
		}

		if skip[tok.Mode.Int()][tok.Kind] > 0 {
			continue
		}

		return tok, nil
	}
}

func (p *Parser) shift(nextState int) (*mldriver.Token, error) {
	p.push(nextState)
	return p.nextToken()
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

func (p *Parser) CST() *Node {
	return p.cst
}

func (p *Parser) AST() *Node {
	return p.ast
}
