package driver

import (
	"fmt"
	"io"
	"strings"

	mldriver "github.com/nihei9/maleeni/driver"
	mlspec "github.com/nihei9/maleeni/spec"
	"github.com/nihei9/vartan/spec"
)

type Node struct {
	KindName string
	Text     string
	Row      int
	Col      int
	Children []*Node
}

func PrintTree(w io.Writer, node *Node) {
	printTree(w, node, "", "")
}

func printTree(w io.Writer, node *Node, ruledLine string, childRuledLinePrefix string) {
	if node == nil {
		return
	}

	if node.Text != "" {
		fmt.Fprintf(w, "%v%v %#v\n", ruledLine, node.KindName, node.Text)
	} else {
		fmt.Fprintf(w, "%v%v\n", ruledLine, node.KindName)
	}

	num := len(node.Children)
	for i, child := range node.Children {
		var line string
		if num > 1 && i < num-1 {
			line = "├─ "
		} else {
			line = "└─ "
		}

		var prefix string
		if i >= num-1 {
			prefix = "   "
		} else {
			prefix = "│  "
		}

		printTree(w, child, childRuledLinePrefix+line, childRuledLinePrefix+prefix)
	}
}

type ParserOption func(p *Parser) error

func MakeAST() ParserOption {
	return func(p *Parser) error {
		p.makeAST = true
		return nil
	}
}

func MakeCST() ParserOption {
	return func(p *Parser) error {
		p.makeCST = true
		return nil
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
	makeAST    bool
	makeCST    bool
	needSemAct bool
}

func NewParser(gram *spec.CompiledGrammar, src io.Reader, opts ...ParserOption) (*Parser, error) {
	lex, err := mldriver.NewLexer(gram.LexicalSpecification.Maleeni.Spec, src)
	if err != nil {
		return nil, err
	}

	p := &Parser{
		gram:       gram,
		lex:        lex,
		stateStack: []int{},
	}

	for _, opt := range opts {
		err := opt(p)
		if err != nil {
			return nil, err
		}
	}

	p.needSemAct = p.makeAST || p.makeCST

	return p, nil
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
			tsym = p.gram.LexicalSpecification.Maleeni.KindToTerminal[tok.KindID]
		}
		act := p.gram.ParsingTable.Action[p.top()*termCount+tsym]
		switch {
		case act < 0: // Shift
			tokText := tok.Text()
			tokRow := tok.Row
			tokCol := tok.Col
			tok, err = p.shift(act * -1)
			if err != nil {
				return err
			}

			// semantic action
			if p.needSemAct {
				var ast *Node
				var cst *Node
				if p.makeAST {
					ast = &Node{
						KindName: p.gram.ParsingTable.Terminals[tsym],
						Text:     tokText,
						Row:      tokRow,
						Col:      tokCol,
					}
				}
				if p.makeCST {
					cst = &Node{
						KindName: p.gram.ParsingTable.Terminals[tsym],
						Text:     tokText,
						Row:      tokRow,
						Col:      tokCol,
					}
				}

				p.semStack = append(p.semStack, &semanticFrame{
					cst: cst,
					ast: ast,
				})
			}
		case act > 0: // Reduce
			accepted := p.reduce(act)
			if accepted {
				if p.needSemAct {
					top := p.semStack[len(p.semStack)-1]
					p.cst = top.cst
					p.ast = top.ast
				}

				return nil
			}

			// semantic action
			if p.needSemAct {
				prodNum := act
				lhs := p.gram.ParsingTable.LHSSymbols[prodNum]

				// When an alternative is empty, `n` will be 0, and `handle` will be empty slice.
				n := p.gram.ParsingTable.AlternativeSymbolCounts[prodNum]
				handle := p.semStack[len(p.semStack)-n:]

				var ast *Node
				var cst *Node
				if p.makeAST {
					act := p.gram.ASTAction.Entries[prodNum]
					var children []*Node
					if act != nil {
						// Count the number of children in advance to avoid frequent growth in a slice for children.
						{
							l := 0
							for _, e := range act {
								if e > 0 {
									l++
								} else {
									offset := e*-1 - 1
									l += len(handle[offset].ast.Children)
								}
							}

							children = make([]*Node, l)
						}

						p := 0
						for _, e := range act {
							if e > 0 {
								offset := e - 1
								children[p] = handle[offset].ast
								p++
							} else {
								offset := e*-1 - 1
								for _, c := range handle[offset].ast.Children {
									children[p] = c
									p++
								}
							}
						}
					} else {
						// If an alternative has no AST action, a driver generates
						// a node with the same structure as a CST.

						children = make([]*Node, len(handle))
						for i, f := range handle {
							children[i] = f.ast
						}
					}

					ast = &Node{
						KindName: p.gram.ParsingTable.NonTerminals[lhs],
						Children: children,
					}
				}
				if p.makeCST {
					children := make([]*Node, len(handle))
					for i, f := range handle {
						children[i] = f.cst
					}

					cst = &Node{
						KindName: p.gram.ParsingTable.NonTerminals[lhs],
						Children: children,
					}
				}

				p.semStack = p.semStack[:len(p.semStack)-n]
				p.semStack = append(p.semStack, &semanticFrame{
					cst: cst,
					ast: ast,
				})
			}
		default:
			var tokText string
			if tok.EOF {
				tokText = "<EOF>"
			} else {
				tokText = fmt.Sprintf("%v:%v: %v (%v)", tok.Row+1, tok.Col+1, tok.KindName.String(), tok.Text())
			}

			eKinds, eof := p.expectedKinds(p.top())

			var b strings.Builder
			if len(eKinds) > 0 {
				fmt.Fprintf(&b, "%v", eKinds[0])
				for _, k := range eKinds[1:] {
					fmt.Fprintf(&b, ", %v", k)
				}
			}
			if eof {
				if len(eKinds) > 0 {
					fmt.Fprintf(&b, ", <EOF>")
				} else {
					fmt.Fprintf(&b, "<EOF>")
				}
			}

			return fmt.Errorf("unexpected token: %v, expected: %v", tokText, b.String())
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
			return nil, fmt.Errorf("invalid token: %v:%v: '%v'", tok.Row+1, tok.Col+1, tok.Text())
		}

		if skip[tok.KindID] > 0 {
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

func (p *Parser) expectedKinds(state int) ([]mlspec.LexKindName, bool) {
	kinds := []mlspec.LexKindName{}
	eof := false
	terms := p.gram.ParsingTable.ExpectedTerminals[state]
	for _, tsym := range terms {
		if tsym == 1 {
			eof = true
			continue
		}

		kindID := p.gram.LexicalSpecification.Maleeni.TerminalToKind[tsym]
		kindName := p.gram.LexicalSpecification.Maleeni.Spec.KindNames[kindID]
		kinds = append(kinds, kindName)
	}

	return kinds, eof
}
