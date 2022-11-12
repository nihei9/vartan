package lexer

import (
	"fmt"
	"io"
)

type ModeID int

func (id ModeID) Int() int {
	return int(id)
}

type StateID int

func (id StateID) Int() int {
	return int(id)
}

type KindID int

func (id KindID) Int() int {
	return int(id)
}

type ModeKindID int

func (id ModeKindID) Int() int {
	return int(id)
}

type LexSpec interface {
	InitialMode() ModeID
	Pop(mode ModeID, modeKind ModeKindID) bool
	Push(mode ModeID, modeKind ModeKindID) (ModeID, bool)
	ModeName(mode ModeID) string
	InitialState(mode ModeID) StateID
	NextState(mode ModeID, state StateID, v int) (StateID, bool)
	Accept(mode ModeID, state StateID) (ModeKindID, bool)
	KindIDAndName(mode ModeID, modeKind ModeKindID) (KindID, string)
}

// Token representes a token.
type Token struct {
	// ModeID is an ID of a lex mode.
	ModeID ModeID

	// KindID is an ID of a kind. This is unique among all modes.
	KindID KindID

	// ModeKindID is an ID of a lexical kind. This is unique only within a mode.
	// Note that you need to use KindID field if you want to identify a kind across all modes.
	ModeKindID ModeKindID

	// Row is a row number where a lexeme appears.
	Row int

	// Col is a column number where a lexeme appears.
	// Note that Col is counted in code points, not bytes.
	Col int

	// Lexeme is a byte sequence matched a pattern of a lexical specification.
	Lexeme []byte

	// When this field is true, it means the token is the EOF token.
	EOF bool

	// When this field is true, it means the token is an error token.
	Invalid bool
}

type LexerOption func(l *Lexer) error

// DisableModeTransition disables the active mode transition. Thus, even if the lexical specification has the push and pop
// operations, the lexer doesn't perform these operations. When the lexical specification has multiple modes, and this option is
// enabled, you need to call the Lexer.Push and Lexer.Pop methods to perform the mode transition. You can use the Lexer.Mode method
// to know the current lex mode.
func DisableModeTransition() LexerOption {
	return func(l *Lexer) error {
		l.passiveModeTran = true
		return nil
	}
}

type lexerState struct {
	srcPtr int
	row    int
	col    int
}

type Lexer struct {
	spec              LexSpec
	src               []byte
	state             lexerState
	lastAcceptedState lexerState
	tokBuf            []*Token
	modeStack         []ModeID
	passiveModeTran   bool
}

// NewLexer returns a new lexer.
func NewLexer(spec LexSpec, src io.Reader, opts ...LexerOption) (*Lexer, error) {
	b, err := io.ReadAll(src)
	if err != nil {
		return nil, err
	}
	l := &Lexer{
		spec: spec,
		src:  b,
		state: lexerState{
			srcPtr: 0,
			row:    0,
			col:    0,
		},
		lastAcceptedState: lexerState{
			srcPtr: 0,
			row:    0,
			col:    0,
		},
		modeStack: []ModeID{
			spec.InitialMode(),
		},
		passiveModeTran: false,
	}
	for _, opt := range opts {
		err := opt(l)
		if err != nil {
			return nil, err
		}
	}

	return l, nil
}

// Next returns a next token.
func (l *Lexer) Next() (*Token, error) {
	if len(l.tokBuf) > 0 {
		tok := l.tokBuf[0]
		l.tokBuf = l.tokBuf[1:]
		return tok, nil
	}

	tok, err := l.nextAndTransition()
	if err != nil {
		return nil, err
	}
	if !tok.Invalid {
		return tok, nil
	}
	errTok := tok
	for {
		tok, err = l.nextAndTransition()
		if err != nil {
			return nil, err
		}
		if !tok.Invalid {
			break
		}
		errTok.Lexeme = append(errTok.Lexeme, tok.Lexeme...)
	}
	l.tokBuf = append(l.tokBuf, tok)

	return errTok, nil
}

func (l *Lexer) nextAndTransition() (*Token, error) {
	tok, err := l.next()
	if err != nil {
		return nil, err
	}
	if tok.EOF || tok.Invalid {
		return tok, nil
	}
	if l.passiveModeTran {
		return tok, nil
	}
	mode := l.Mode()
	if l.spec.Pop(mode, tok.ModeKindID) {
		err := l.PopMode()
		if err != nil {
			return nil, err
		}
	}
	if mode, ok := l.spec.Push(mode, tok.ModeKindID); ok {
		l.PushMode(mode)
	}
	// The checking length of the mode stack must be at after pop and push operations because those operations can be performed
	// at the same time. When the mode stack has just one element and popped it, the mode stack will be temporarily emptied.
	// However, since a push operation may be performed immediately after it, the lexer allows the stack to be temporarily empty.
	if len(l.modeStack) == 0 {
		return nil, fmt.Errorf("a mode stack must have at least one element")
	}
	return tok, nil
}

func (l *Lexer) next() (*Token, error) {
	mode := l.Mode()
	state := l.spec.InitialState(mode)
	buf := []byte{}
	row := l.state.row
	col := l.state.col
	var tok *Token
	for {
		v, eof := l.read()
		if eof {
			if tok != nil {
				l.revert()
				return tok, nil
			}
			// When `buf` has unaccepted data and reads the EOF, the lexer treats the buffered data as an invalid token.
			if len(buf) > 0 {
				return &Token{
					ModeID:     mode,
					ModeKindID: 0,
					Lexeme:     buf,
					Row:        row,
					Col:        col,
					Invalid:    true,
				}, nil
			}
			return &Token{
				ModeID:     mode,
				ModeKindID: 0,
				Row:        row,
				Col:        col,
				EOF:        true,
			}, nil
		}
		buf = append(buf, v)
		nextState, ok := l.spec.NextState(mode, state, int(v))
		if !ok {
			if tok != nil {
				l.revert()
				return tok, nil
			}
			return &Token{
				ModeID:     mode,
				ModeKindID: 0,
				Lexeme:     buf,
				Row:        row,
				Col:        col,
				Invalid:    true,
			}, nil
		}
		state = nextState
		if modeKindID, ok := l.spec.Accept(mode, state); ok {
			kindID, _ := l.spec.KindIDAndName(mode, modeKindID)
			tok = &Token{
				ModeID:     mode,
				KindID:     kindID,
				ModeKindID: modeKindID,
				Lexeme:     buf,
				Row:        row,
				Col:        col,
			}
			l.accept()
		}
	}
}

// Mode returns the current lex mode.
func (l *Lexer) Mode() ModeID {
	return l.modeStack[len(l.modeStack)-1]
}

// PushMode adds a lex mode onto the mode stack.
func (l *Lexer) PushMode(mode ModeID) {
	l.modeStack = append(l.modeStack, mode)
}

// PopMode removes a lex mode from the top of the mode stack.
func (l *Lexer) PopMode() error {
	sLen := len(l.modeStack)
	if sLen == 0 {
		return fmt.Errorf("cannot pop a lex mode from a lex mode stack any more")
	}
	l.modeStack = l.modeStack[:sLen-1]
	return nil
}

func (l *Lexer) read() (byte, bool) {
	if l.state.srcPtr >= len(l.src) {
		return 0, true
	}

	b := l.src[l.state.srcPtr]
	l.state.srcPtr++

	// Count the token positions.
	// The driver treats LF as the end of lines and counts columns in code points, not bytes.
	// To count in code points, we refer to the First Byte column in the Table 3-6.
	//
	// Reference:
	// - [Table 3-6] https://www.unicode.org/versions/Unicode13.0.0/ch03.pdf > Table 3-6.  UTF-8 Bit Distribution
	if b < 128 {
		// 0x0A is LF.
		if b == 0x0A {
			l.state.row++
			l.state.col = 0
		} else {
			l.state.col++
		}
	} else if b>>5 == 6 || b>>4 == 14 || b>>3 == 30 {
		l.state.col++
	}

	return b, false
}

// accept saves the current state.
func (l *Lexer) accept() {
	l.lastAcceptedState = l.state
}

// revert reverts the lexer state to the last accepted state.
//
// We must not call this function consecutively.
func (l *Lexer) revert() {
	l.state = l.lastAcceptedState
}
