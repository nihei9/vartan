package driver

import (
	"fmt"
	"io"

	mldriver "github.com/nihei9/maleeni/driver"
)

type Grammar interface {
	// Class returns a class of grammar.
	Class() string

	// InitialState returns the initial state of a parser.
	InitialState() int

	// StartProduction returns the start production of grammar.
	StartProduction() int

	// Action returns an ACTION entry corresponding to a (state, terminal symbol) pair.
	Action(state int, terminal int) int

	// GoTo returns a GOTO entry corresponding to a (state, non-terminal symbol) pair.
	GoTo(state int, lhs int) int

	// ErrorTrapperState returns true when a state can shift the error symbol.
	ErrorTrapperState(state int) bool

	// LHS returns a LHS symbol of a production.
	LHS(prod int) int

	// AlternativeSymbolCount returns a symbol count of p production.
	AlternativeSymbolCount(prod int) int

	// RecoverProduction returns true when a production has the recover directive.
	RecoverProduction(prod int) bool

	// LexicalSpecification returns a lexical specification.
	LexicalSpecification() mldriver.LexSpec

	// TerminalCount returns a terminal symbol count of grammar.
	TerminalCount() int

	// EOF returns the EOF symbol.
	EOF() int

	// Error returns the error symbol.
	Error() int

	// Terminal retuns a string representaion of a terminal symbol.
	Terminal(terminal int) string

	// TerminalAlias returns an alias for a terminal.
	TerminalAlias(terminal int) string

	// Skip returns true when a terminal symbol must be skipped.
	Skip(kind mldriver.KindID) bool

	// LexicalKindToTerminal maps a lexical kind to a terminal symbol.
	LexicalKindToTerminal(kind mldriver.KindID) int
}

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
	gram       Grammar
	lex        *mldriver.Lexer
	stateStack *stateStack
	semAct     SemanticActionSet
	disableLAC bool
	onError    bool
	shiftCount int
	synErrs    []*SyntaxError
}

func NewParser(gram Grammar, src io.Reader, opts ...ParserOption) (*Parser, error) {
	lex, err := mldriver.NewLexer(gram.LexicalSpecification(), src)
	if err != nil {
		return nil, err
	}

	p := &Parser{
		gram:       gram,
		lex:        lex,
		stateStack: &stateStack{},
	}

	if p.gram.Class() != "lalr" {
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
	p.stateStack.push(p.gram.InitialState())
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

			recovered := false
			if p.onError {
				p.shiftCount++

				// When the parser performs shift three times, the parser recovers from the error state.
				if p.shiftCount >= 3 {
					p.onError = false
					p.shiftCount = 0
					recovered = true
				}
			}

			p.shift(nextState)

			if p.semAct != nil {
				p.semAct.Shift(tok, recovered)
			}

			tok, err = p.nextToken()
			if err != nil {
				return err
			}
		case act > 0: // Reduce
			prodNum := act

			recovered := false
			if p.onError && p.gram.RecoverProduction(prodNum) {
				p.onError = false
				p.shiftCount = 0
				recovered = true
			}

			accepted := p.reduce(prodNum)
			if accepted {
				if p.semAct != nil {
					p.semAct.Accept()
				}

				return nil
			}

			if p.semAct != nil {
				p.semAct.Reduce(prodNum, recovered)
			}
		default: // Error
			if p.onError {
				tok, err = p.nextToken()
				if err != nil {
					return err
				}
				if tok.EOF {
					if p.semAct != nil {
						p.semAct.MissError(tok)
					}

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
					p.semAct.MissError(tok)
				}

				return nil
			}

			p.onError = true
			p.shiftCount = 0

			act, err := p.lookupActionOnError()
			if err != nil {
				return err
			}

			p.shift(act * -1)

			if p.semAct != nil {
				p.semAct.TrapAndShiftError(tok, count)
			}
		}
	}
}

// validateLookahead validates whether `term` is a valid lookahead in the current context. When `term` is valid,
// this method returns `true`.
func (p *Parser) validateLookahead(term int) bool {
	p.stateStack.enableExploratoryMode()
	defer p.stateStack.disableExploratoryMode()

	for {
		act := p.gram.Action(p.stateStack.topExploratorily(), term)

		switch {
		case act < 0: // Shift
			return true
		case act > 0: // Reduce
			prodNum := act

			lhs := p.gram.LHS(prodNum)
			if lhs == p.gram.LHS(p.gram.StartProduction()) {
				return true
			}
			n := p.gram.AlternativeSymbolCount(prodNum)
			p.stateStack.popExploratorily(n)
			state := p.gram.GoTo(p.stateStack.topExploratorily(), lhs)
			p.stateStack.pushExploratorily(state)
		default: // Error
			return false
		}
	}
}

func (p *Parser) nextToken() (*mldriver.Token, error) {
	for {
		// We don't have to check whether the token is invalid because the kind ID of the invalid token is 0,
		// and the parsing table doesn't have an entry corresponding to the kind ID 0. Thus we can detect
		// a syntax error because the parser cannot find an entry corresponding to the invalid token.
		tok, err := p.lex.Next()
		if err != nil {
			return nil, err
		}

		if p.gram.Skip(tok.KindID) {
			continue
		}

		return tok, nil
	}
}

func (p *Parser) tokenToTerminal(tok *mldriver.Token) int {
	if tok.EOF {
		return p.gram.EOF()
	}

	return p.gram.LexicalKindToTerminal(tok.KindID)
}

func (p *Parser) lookupAction(tok *mldriver.Token) int {
	if !p.disableLAC {
		term := p.tokenToTerminal(tok)
		if !p.validateLookahead(term) {
			return 0
		}
	}

	return p.gram.Action(p.stateStack.top(), p.tokenToTerminal(tok))
}

func (p *Parser) lookupActionOnError() (int, error) {
	act := p.gram.Action(p.stateStack.top(), p.gram.Error())
	if act >= 0 {
		return 0, fmt.Errorf("an entry must be a shift action by the error symbol; entry: %v, state: %v, symbol: %v", act, p.stateStack.top(), p.gram.Terminal(p.gram.Error()))
	}

	return act, nil
}

func (p *Parser) shift(nextState int) {
	p.stateStack.push(nextState)
}

func (p *Parser) reduce(prodNum int) bool {
	lhs := p.gram.LHS(prodNum)
	if lhs == p.gram.LHS(p.gram.StartProduction()) {
		return true
	}
	n := p.gram.AlternativeSymbolCount(prodNum)
	p.stateStack.pop(n)
	nextState := p.gram.GoTo(p.stateStack.top(), lhs)
	p.stateStack.push(nextState)
	return false
}

func (p *Parser) trapError() (int, bool) {
	count := 0
	for {
		if p.gram.ErrorTrapperState(p.stateStack.top()) {
			return count, true
		}

		if p.stateStack.top() != p.gram.InitialState() {
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
	termCount := p.gram.TerminalCount()
	for term := 0; term < termCount; term++ {
		if p.disableLAC {
			if p.gram.Action(p.stateStack.top(), term) == 0 {
				continue
			}
		} else {
			if !p.validateLookahead(term) {
				continue
			}
		}

		// We don't add the error symbol to the look-ahead symbols because users cannot input the error symbol
		// intentionally.
		if term == p.gram.Error() {
			continue
		}

		if term == p.gram.EOF() {
			kinds = append(kinds, "<eof>")
			continue
		}

		if alias := p.gram.TerminalAlias(term); alias != "" {
			kinds = append(kinds, alias)
		} else {
			kinds = append(kinds, p.gram.Terminal(term))
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
