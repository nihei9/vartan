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
	p.push(p.gram.ParsingTable.InitialState)
	tok, err := p.nextToken()
	if err != nil {
		return err
	}

ACTION_LOOP:
	for {
		act := p.lookupAction(tok)
		switch {
		case act < 0: // Shift
			nextState := act * -1

			if p.onError {
				// When the parser performs shift three times, the parser recovers from the error state.
				if p.shiftCount < 3 {
					p.shiftCount++
				} else {
					p.onError = false
					p.shiftCount = 0
				}
			}

			p.shift(nextState)

			p.actOnShift(tok)

			tok, err = p.nextToken()
			if err != nil {
				return err
			}
		case act > 0: // Reduce
			prodNum := act

			if p.onError && p.gram.ParsingTable.RecoverProductions[prodNum] != 0 {
				p.onError = false
				p.shiftCount = 0
			}

			accepted := p.reduce(prodNum)
			if accepted {
				p.actOnAccepting()

				return nil
			}

			p.actOnReduction(prodNum)
		default: // Error
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
				ExpectedTerminals: p.searchLookahead(p.top()),
			})

			ok := p.trapError()
			if !ok {
				return nil
			}

			p.onError = true
			p.shiftCount = 0

			act, err := p.lookupActionOnError()
			if err != nil {
				return err
			}

			p.shift(act * -1)

			p.actOnError()
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

func (p *Parser) tokenToTerminal(tok *mldriver.Token) int {
	if tok.EOF {
		return p.gram.ParsingTable.EOFSymbol
	}

	return p.gram.LexicalSpecification.Maleeni.KindToTerminal[tok.KindID]
}

func (p *Parser) lookupAction(tok *mldriver.Token) int {
	termCount := p.gram.ParsingTable.TerminalCount
	term := p.tokenToTerminal(tok)
	return p.gram.ParsingTable.Action[p.top()*termCount+term]
}

func (p *Parser) lookupActionOnError() (int, error) {
	termCount := p.gram.ParsingTable.TerminalCount
	errSym := p.gram.ParsingTable.ErrorSymbol
	act := p.gram.ParsingTable.Action[p.top()*termCount+errSym]
	if act >= 0 {
		return 0, fmt.Errorf("an entry must be a shift action by the error symbol; entry: %v, state: %v, symbol: %v", act, p.top(), p.gram.ParsingTable.Terminals[errSym])
	}

	return act, nil
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

func (p *Parser) trapError() bool {
	for {
		if p.gram.ParsingTable.ErrorTrapperStates[p.top()] != 0 {
			return true
		}

		if p.top() != p.gram.ParsingTable.InitialState {
			p.pop(1)
			p.semStack = p.semStack[:len(p.semStack)-1]
		} else {
			return false
		}
	}
}

func (p *Parser) actOnShift(tok *mldriver.Token) {
	if !p.needSemAct {
		return
	}

	term := p.tokenToTerminal(tok)

	var ast *Node
	var cst *Node
	if p.makeAST {
		ast = &Node{
			KindName: p.gram.ParsingTable.Terminals[term],
			Text:     tok.Text(),
			Row:      tok.Row,
			Col:      tok.Col,
		}
	}
	if p.makeCST {
		cst = &Node{
			KindName: p.gram.ParsingTable.Terminals[term],
			Text:     tok.Text(),
			Row:      tok.Row,
			Col:      tok.Col,
		}
	}

	p.semStack = append(p.semStack, &semanticFrame{
		cst: cst,
		ast: ast,
	})
}

func (p *Parser) actOnReduction(prodNum int) {
	if !p.needSemAct {
		return
	}

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

func (p *Parser) actOnAccepting() {
	if !p.needSemAct {
		return
	}

	top := p.semStack[len(p.semStack)-1]
	p.cst = top.cst
	p.ast = top.ast
}

func (p *Parser) actOnError() {
	if !p.needSemAct {
		return
	}

	errSym := p.gram.ParsingTable.ErrorSymbol

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

func (p *Parser) searchLookahead(state int) []string {
	kinds := []string{}
	term2Kind := p.gram.LexicalSpecification.Maleeni.TerminalToKind
	kindNames := p.gram.LexicalSpecification.Maleeni.Spec.KindNames
	aliases := p.gram.LexicalSpecification.Maleeni.KindAliases
	termCount := p.gram.ParsingTable.TerminalCount
	base := p.top() * termCount
	for term := 0; term < termCount; term++ {
		if p.gram.ParsingTable.Action[base+term] == 0 {
			continue
		}

		// We don't add the error symbol to the look-ahead symbols because users cannot input the error symbol
		// intentionally.
		if term == p.gram.ParsingTable.ErrorSymbol {
			continue
		}

		if term == p.gram.ParsingTable.EOFSymbol {
			kinds = append(kinds, "<eof>")
			continue
		}

		if alias := aliases[term]; alias != "" {
			kinds = append(kinds, alias)
		} else {
			kinds = append(kinds, kindNames[term2Kind[term]].String())
		}
	}

	return kinds
}
