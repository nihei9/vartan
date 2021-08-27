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

type SyntaxError struct {
	Row               int
	Col               int
	Message           string
	Token             *mldriver.Token
	ExpectedTerminals []string
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
	onError    bool
	shiftCount int
	synErrs    []*SyntaxError
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

ACTION_LOOP:
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
			p.shift(act * -1)
			tok, err = p.nextToken()
			if err != nil {
				return err
			}

			if p.onError {
				// When the parser performs shift three times, the parser recovers from the error state.
				if p.shiftCount < 3 {
					p.shiftCount++
				} else {
					p.onError = false
					p.shiftCount = 0
				}
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

			prodNum := act

			if p.onError && p.gram.ParsingTable.RecoverProductions[prodNum] != 0 {
				p.onError = false
				p.shiftCount = 0
			}

			// semantic action
			if p.needSemAct {
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
			if p.onError {
				tok, err = p.nextToken()
				if err != nil {
					return err
				}
				if tok.EOF {
					return nil
				}

				continue ACTION_LOOP
			}

			p.synErrs = append(p.synErrs, &SyntaxError{
				Row:               tok.Row,
				Col:               tok.Col,
				Message:           "unexpected token",
				Token:             tok,
				ExpectedTerminals: p.expectedTerms(p.top()),
			})

			for {
				if p.gram.ParsingTable.ErrorTrapperStates[p.top()] != 0 {
					p.onError = true
					p.shiftCount = 0

					errSym := p.gram.ParsingTable.ErrorSymbol
					act := p.gram.ParsingTable.Action[p.top()*termCount+errSym]
					if act >= 0 {
						return fmt.Errorf("an entry must be a shift action by the error symbol; entry: %v, state: %v, symbol: %v", act, p.top(), p.gram.ParsingTable.Terminals[errSym])
					}
					p.shift(act * -1)

					// semantic action
					if p.needSemAct {
						var ast *Node
						var cst *Node
						if p.makeAST {
							ast = &Node{
								KindName: p.gram.ParsingTable.Terminals[errSym],
							}
						}
						if p.makeCST {
							cst = &Node{
								KindName: p.gram.ParsingTable.Terminals[errSym],
							}
						}

						p.semStack = append(p.semStack, &semanticFrame{
							cst: cst,
							ast: ast,
						})
					}

					continue ACTION_LOOP
				}

				if p.top() != p.gram.ParsingTable.InitialState {
					p.pop(1)
					p.semStack = p.semStack[:len(p.semStack)-1]
				} else {
					return nil
				}
			}
		}
	}
}

func (p *Parser) nextToken() (*mldriver.Token, error) {
	skip := p.gram.LexicalSpecification.Maleeni.Skip
	for {
		// We don't have to check whether the token is invalid because the kind ID of the invalid token is 0,
		// and the parsing table doesn't have an entry corresponding to the kind ID 0. Thus we can detect
		// a syntax error because the parser cannot find an entry corresponding to the invalid token.
		tok, err := p.lex.Next()
		if err != nil {
			return nil, err
		}

		if skip[tok.KindID] > 0 {
			continue
		}

		return tok, nil
	}
}

func (p *Parser) shift(nextState int) {
	p.push(nextState)
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

func (p *Parser) SyntaxErrors() []*SyntaxError {
	return p.synErrs
}

func (p *Parser) expectedTerms(state int) []string {
	kinds := []string{}
	terms := p.gram.ParsingTable.ExpectedTerminals[state]
	aliases := p.gram.LexicalSpecification.Maleeni.KindAliases
	for _, tsym := range terms {
		// We don't add the error symbol to the look-ahead symbols because users cannot input the error symbol
		// intentionally.
		if tsym == p.gram.ParsingTable.ErrorSymbol {
			continue
		}

		if tsym == p.gram.ParsingTable.EOFSymbol {
			kinds = append(kinds, "<eof>")
			continue
		}

		if alias := aliases[tsym]; alias != "" {
			kinds = append(kinds, alias)
		} else {
			term2Kind := p.gram.LexicalSpecification.Maleeni.TerminalToKind
			kindNames := p.gram.LexicalSpecification.Maleeni.Spec.KindNames
			kinds = append(kinds, kindNames[term2Kind[tsym]].String())
		}
	}

	return kinds
}
