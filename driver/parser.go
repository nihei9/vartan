package driver

import (
	"fmt"
	"io"

	mldriver "github.com/nihei9/maleeni/driver"
	"github.com/nihei9/vartan/spec"
)

type SyntaxError struct {
	Row               int
	Col               int
	Message           string
	Token             *mldriver.Token
	ExpectedTerminals []string
}

type ParserOption func(p *Parser) error

// DisableLAC disables LAC (lookahead correction). When the grammar has the LALR class, LAC is enabled by default.
func DisableLAC() ParserOption {
	return func(p *Parser) error {
		p.disableLAC = true
		return nil
	}
}

func SemanticAction(semAct SemanticActionSet) ParserOption {
	return func(p *Parser) error {
		p.semAct = semAct
		return nil
	}
}

type Parser struct {
	gram       *spec.CompiledGrammar
	lex        *mldriver.Lexer
	stateStack *stateStack
	semAct     SemanticActionSet
	disableLAC bool
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
		stateStack: &stateStack{},
	}

	if p.gram.ParsingTable.Class != "lalr" {
		p.disableLAC = true
	}

	for _, opt := range opts {
		err := opt(p)
		if err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (p *Parser) Parse() error {
	p.stateStack.push(p.gram.ParsingTable.InitialState)
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

			if p.semAct != nil {
				p.semAct.Shift(tok)
			}

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
				if p.semAct != nil {
					p.semAct.Accept()
				}

				return nil
			}

			if p.semAct != nil {
				p.semAct.Reduce(prodNum)
			}
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
				ExpectedTerminals: p.searchLookahead(p.stateStack.top()),
			})

			count, ok := p.trapError()
			if !ok {
				if p.semAct != nil {
					p.semAct.MissError()
				}

				return nil
			}
			if p.semAct != nil {
				p.semAct.TrapError(count)
			}

			p.onError = true
			p.shiftCount = 0

			act, err := p.lookupActionOnError()
			if err != nil {
				return err
			}

			p.shift(act * -1)

			if p.semAct != nil {
				p.semAct.ShiftError()
			}
		}
	}
}

// validateLookahead validates whether `term` is a valid lookahead in the current context. When `term` is valid,
// this method returns `true`.
func (p *Parser) validateLookahead(term int) bool {
	p.stateStack.enableExploratoryMode()
	defer p.stateStack.disableExploratoryMode()

	tab := p.gram.ParsingTable

	for {
		act := tab.Action[p.stateStack.topExploratorily()*tab.TerminalCount+term]

		switch {
		case act < 0: // Shift
			return true
		case act > 0: // Reduce
			prodNum := act

			lhs := tab.LHSSymbols[prodNum]
			if lhs == tab.LHSSymbols[tab.StartProduction] {
				return true
			}
			n := tab.AlternativeSymbolCounts[prodNum]
			p.stateStack.popExploratorily(n)
			state := tab.GoTo[p.stateStack.topExploratorily()*tab.NonTerminalCount+lhs]
			p.stateStack.pushExploratorily(state)
		default: // Error
			return false
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
	if !p.disableLAC {
		term := p.tokenToTerminal(tok)
		if !p.validateLookahead(term) {
			return 0
		}
	}

	termCount := p.gram.ParsingTable.TerminalCount
	term := p.tokenToTerminal(tok)
	return p.gram.ParsingTable.Action[p.stateStack.top()*termCount+term]
}

func (p *Parser) lookupActionOnError() (int, error) {
	termCount := p.gram.ParsingTable.TerminalCount
	errSym := p.gram.ParsingTable.ErrorSymbol
	act := p.gram.ParsingTable.Action[p.stateStack.top()*termCount+errSym]
	if act >= 0 {
		return 0, fmt.Errorf("an entry must be a shift action by the error symbol; entry: %v, state: %v, symbol: %v", act, p.stateStack.top(), p.gram.ParsingTable.Terminals[errSym])
	}

	return act, nil
}

func (p *Parser) shift(nextState int) {
	p.stateStack.push(nextState)
}

func (p *Parser) reduce(prodNum int) bool {
	tab := p.gram.ParsingTable
	lhs := tab.LHSSymbols[prodNum]
	if lhs == tab.LHSSymbols[tab.StartProduction] {
		return true
	}
	n := tab.AlternativeSymbolCounts[prodNum]
	p.stateStack.pop(n)
	nextState := tab.GoTo[p.stateStack.top()*tab.NonTerminalCount+lhs]
	p.stateStack.push(nextState)
	return false
}

func (p *Parser) trapError() (int, bool) {
	count := 0
	for {
		if p.gram.ParsingTable.ErrorTrapperStates[p.stateStack.top()] != 0 {
			return count, true
		}

		if p.stateStack.top() != p.gram.ParsingTable.InitialState {
			p.stateStack.pop(1)
			count++
		} else {
			return 0, false
		}
	}
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
	base := p.stateStack.top() * termCount
	for term := 0; term < termCount; term++ {
		if p.disableLAC {
			if p.gram.ParsingTable.Action[base+term] == 0 {
				continue
			}
		} else {
			if !p.validateLookahead(term) {
				continue
			}
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

type stateStack struct {
	items    []int
	itemsExp []int
}

func (s *stateStack) enableExploratoryMode() {
	s.itemsExp = make([]int, len(s.items))
	for i, v := range s.items {
		s.itemsExp[i] = v
	}
}

func (s *stateStack) disableExploratoryMode() {
	s.itemsExp = nil
}

func (s *stateStack) top() int {
	return s.items[len(s.items)-1]
}

func (s *stateStack) topExploratorily() int {
	return s.itemsExp[len(s.itemsExp)-1]
}

func (s *stateStack) push(state int) {
	s.items = append(s.items, state)
}

func (s *stateStack) pushExploratorily(state int) {
	s.itemsExp = append(s.itemsExp, state)
}

func (s *stateStack) pop(n int) {
	s.items = s.items[:len(s.items)-n]
}

func (s *stateStack) popExploratorily(n int) {
	s.itemsExp = s.itemsExp[:len(s.itemsExp)-n]
}
