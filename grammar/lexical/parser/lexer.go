package parser

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type tokenKind string

const (
	tokenKindChar            tokenKind = "char"
	tokenKindAnyChar         tokenKind = "."
	tokenKindRepeat          tokenKind = "*"
	tokenKindRepeatOneOrMore tokenKind = "+"
	tokenKindOption          tokenKind = "?"
	tokenKindAlt             tokenKind = "|"
	tokenKindGroupOpen       tokenKind = "("
	tokenKindGroupClose      tokenKind = ")"
	tokenKindBExpOpen        tokenKind = "["
	tokenKindInverseBExpOpen tokenKind = "[^"
	tokenKindBExpClose       tokenKind = "]"
	tokenKindCharRange       tokenKind = "-"
	tokenKindCodePointLeader tokenKind = "\\u"
	tokenKindCharPropLeader  tokenKind = "\\p"
	tokenKindFragmentLeader  tokenKind = "\\f"
	tokenKindLBrace          tokenKind = "{"
	tokenKindRBrace          tokenKind = "}"
	tokenKindEqual           tokenKind = "="
	tokenKindCodePoint       tokenKind = "code point"
	tokenKindCharPropSymbol  tokenKind = "character property symbol"
	tokenKindFragmentSymbol  tokenKind = "fragment symbol"
	tokenKindEOF             tokenKind = "eof"
)

type token struct {
	kind           tokenKind
	char           rune
	propSymbol     string
	codePoint      string
	fragmentSymbol string
}

const nullChar = '\u0000'

func newToken(kind tokenKind, char rune) *token {
	return &token{
		kind: kind,
		char: char,
	}
}

func newCodePointToken(codePoint string) *token {
	return &token{
		kind:      tokenKindCodePoint,
		codePoint: codePoint,
	}
}

func newCharPropSymbolToken(propSymbol string) *token {
	return &token{
		kind:       tokenKindCharPropSymbol,
		propSymbol: propSymbol,
	}
}

func newFragmentSymbolToken(fragmentSymbol string) *token {
	return &token{
		kind:           tokenKindFragmentSymbol,
		fragmentSymbol: fragmentSymbol,
	}
}

type lexerMode string

const (
	lexerModeDefault     lexerMode = "default"
	lexerModeBExp        lexerMode = "bracket expression"
	lexerModeCPExp       lexerMode = "code point expression"
	lexerModeCharPropExp lexerMode = "character property expression"
	lexerModeFragmentExp lexerMode = "fragment expression"
)

type lexerModeStack struct {
	stack []lexerMode
}

func newLexerModeStack() *lexerModeStack {
	return &lexerModeStack{
		stack: []lexerMode{
			lexerModeDefault,
		},
	}
}

func (s *lexerModeStack) top() lexerMode {
	return s.stack[len(s.stack)-1]
}

func (s *lexerModeStack) push(m lexerMode) {
	s.stack = append(s.stack, m)
}

func (s *lexerModeStack) pop() {
	s.stack = s.stack[:len(s.stack)-1]
}

type rangeState string

// [a-z]
// ^^^^
// |||`-- ready
// ||`-- expect range terminator
// |`-- read range initiator
// `-- ready
const (
	rangeStateReady                 rangeState = "ready"
	rangeStateReadRangeInitiator    rangeState = "read range initiator"
	rangeStateExpectRangeTerminator rangeState = "expect range terminator"
)

type lexer struct {
	src        *bufio.Reader
	peekChar2  rune
	peekEOF2   bool
	peekChar1  rune
	peekEOF1   bool
	lastChar   rune
	reachedEOF bool
	prevChar1  rune
	prevEOF1   bool
	prevChar2  rune
	pervEOF2   bool
	modeStack  *lexerModeStack
	rangeState rangeState

	errCause  error
	errDetail string
}

func newLexer(src io.Reader) *lexer {
	return &lexer{
		src:        bufio.NewReader(src),
		peekChar2:  nullChar,
		peekEOF2:   false,
		peekChar1:  nullChar,
		peekEOF1:   false,
		lastChar:   nullChar,
		reachedEOF: false,
		prevChar1:  nullChar,
		prevEOF1:   false,
		prevChar2:  nullChar,
		pervEOF2:   false,
		modeStack:  newLexerModeStack(),
		rangeState: rangeStateReady,
	}
}

func (l *lexer) error() (string, error) {
	return l.errDetail, l.errCause
}

func (l *lexer) next() (*token, error) {
	c, eof, err := l.read()
	if err != nil {
		return nil, err
	}
	if eof {
		return newToken(tokenKindEOF, nullChar), nil
	}

	switch l.modeStack.top() {
	case lexerModeBExp:
		tok, err := l.nextInBExp(c)
		if err != nil {
			return nil, err
		}
		if tok.kind == tokenKindChar || tok.kind == tokenKindCodePointLeader || tok.kind == tokenKindCharPropLeader {
			switch l.rangeState {
			case rangeStateReady:
				l.rangeState = rangeStateReadRangeInitiator
			case rangeStateExpectRangeTerminator:
				l.rangeState = rangeStateReady
			}
		}
		switch tok.kind {
		case tokenKindBExpClose:
			l.modeStack.pop()
		case tokenKindCharRange:
			l.rangeState = rangeStateExpectRangeTerminator
		case tokenKindCodePointLeader:
			l.modeStack.push(lexerModeCPExp)
		case tokenKindCharPropLeader:
			l.modeStack.push(lexerModeCharPropExp)
		}
		return tok, nil
	case lexerModeCPExp:
		tok, err := l.nextInCodePoint(c)
		if err != nil {
			return nil, err
		}
		switch tok.kind {
		case tokenKindRBrace:
			l.modeStack.pop()
		}
		return tok, nil
	case lexerModeCharPropExp:
		tok, err := l.nextInCharProp(c)
		if err != nil {
			return nil, err
		}
		switch tok.kind {
		case tokenKindRBrace:
			l.modeStack.pop()
		}
		return tok, nil
	case lexerModeFragmentExp:
		tok, err := l.nextInFragment(c)
		if err != nil {
			return nil, err
		}
		switch tok.kind {
		case tokenKindRBrace:
			l.modeStack.pop()
		}
		return tok, nil
	default:
		tok, err := l.nextInDefault(c)
		if err != nil {
			return nil, err
		}
		switch tok.kind {
		case tokenKindBExpOpen:
			l.modeStack.push(lexerModeBExp)
			l.rangeState = rangeStateReady
		case tokenKindInverseBExpOpen:
			l.modeStack.push(lexerModeBExp)
			l.rangeState = rangeStateReady
		case tokenKindCodePointLeader:
			l.modeStack.push(lexerModeCPExp)
		case tokenKindCharPropLeader:
			l.modeStack.push(lexerModeCharPropExp)
		case tokenKindFragmentLeader:
			l.modeStack.push(lexerModeFragmentExp)
		}
		return tok, nil
	}
}

func (l *lexer) nextInDefault(c rune) (*token, error) {
	switch c {
	case '*':
		return newToken(tokenKindRepeat, nullChar), nil
	case '+':
		return newToken(tokenKindRepeatOneOrMore, nullChar), nil
	case '?':
		return newToken(tokenKindOption, nullChar), nil
	case '.':
		return newToken(tokenKindAnyChar, nullChar), nil
	case '|':
		return newToken(tokenKindAlt, nullChar), nil
	case '(':
		return newToken(tokenKindGroupOpen, nullChar), nil
	case ')':
		return newToken(tokenKindGroupClose, nullChar), nil
	case '[':
		c1, eof, err := l.read()
		if err != nil {
			return nil, err
		}
		if eof {
			err := l.restore()
			if err != nil {
				return nil, err
			}
			return newToken(tokenKindBExpOpen, nullChar), nil
		}
		if c1 != '^' {
			err := l.restore()
			if err != nil {
				return nil, err
			}
			return newToken(tokenKindBExpOpen, nullChar), nil
		}
		c2, eof, err := l.read()
		if err != nil {
			return nil, err
		}
		if eof {
			err := l.restore()
			if err != nil {
				return nil, err
			}
			return newToken(tokenKindInverseBExpOpen, nullChar), nil
		}
		if c2 != ']' {
			err := l.restore()
			if err != nil {
				return nil, err
			}
			return newToken(tokenKindInverseBExpOpen, nullChar), nil
		}
		err = l.restore()
		if err != nil {
			return nil, err
		}
		err = l.restore()
		if err != nil {
			return nil, err
		}
		return newToken(tokenKindBExpOpen, nullChar), nil
	case '\\':
		c, eof, err := l.read()
		if err != nil {
			return nil, err
		}
		if eof {
			l.errCause = synErrIncompletedEscSeq
			return nil, ParseErr
		}
		if c == 'u' {
			return newToken(tokenKindCodePointLeader, nullChar), nil
		}
		if c == 'p' {
			return newToken(tokenKindCharPropLeader, nullChar), nil
		}
		if c == 'f' {
			return newToken(tokenKindFragmentLeader, nullChar), nil
		}
		if c == '\\' || c == '.' || c == '*' || c == '+' || c == '?' || c == '|' || c == '(' || c == ')' || c == '[' || c == ']' {
			return newToken(tokenKindChar, c), nil
		}
		l.errCause = synErrInvalidEscSeq
		l.errDetail = fmt.Sprintf("\\%v is not supported", string(c))
		return nil, ParseErr
	default:
		return newToken(tokenKindChar, c), nil
	}
}

func (l *lexer) nextInBExp(c rune) (*token, error) {
	switch c {
	case '-':
		if l.rangeState != rangeStateReadRangeInitiator {
			return newToken(tokenKindChar, c), nil
		}
		c1, eof, err := l.read()
		if err != nil {
			return nil, err
		}
		if eof {
			err := l.restore()
			if err != nil {
				return nil, err
			}
			return newToken(tokenKindChar, c), nil
		}
		if c1 != ']' {
			err := l.restore()
			if err != nil {
				return nil, err
			}
			return newToken(tokenKindCharRange, nullChar), nil
		}
		err = l.restore()
		if err != nil {
			return nil, err
		}
		return newToken(tokenKindChar, c), nil
	case ']':
		return newToken(tokenKindBExpClose, nullChar), nil
	case '\\':
		c, eof, err := l.read()
		if err != nil {
			return nil, err
		}
		if eof {
			l.errCause = synErrIncompletedEscSeq
			return nil, ParseErr
		}
		if c == 'u' {
			return newToken(tokenKindCodePointLeader, nullChar), nil
		}
		if c == 'p' {
			return newToken(tokenKindCharPropLeader, nullChar), nil
		}
		if c == '\\' || c == '^' || c == '-' || c == ']' {
			return newToken(tokenKindChar, c), nil
		}
		l.errCause = synErrInvalidEscSeq
		l.errDetail = fmt.Sprintf("\\%v is not supported in a bracket expression", string(c))
		return nil, ParseErr
	default:
		return newToken(tokenKindChar, c), nil
	}
}

func (l *lexer) nextInCodePoint(c rune) (*token, error) {
	switch c {
	case '{':
		return newToken(tokenKindLBrace, nullChar), nil
	case '}':
		return newToken(tokenKindRBrace, nullChar), nil
	default:
		if !isHexDigit(c) {
			l.errCause = synErrInvalidCodePoint
			return nil, ParseErr
		}
		var b strings.Builder
		fmt.Fprint(&b, string(c))
		n := 1
		for {
			c, eof, err := l.read()
			if err != nil {
				return nil, err
			}
			if eof {
				err := l.restore()
				if err != nil {
					return nil, err
				}
				break
			}
			if c == '}' {
				err := l.restore()
				if err != nil {
					return nil, err
				}
				break
			}
			if !isHexDigit(c) || n >= 6 {
				l.errCause = synErrInvalidCodePoint
				return nil, ParseErr
			}
			fmt.Fprint(&b, string(c))
			n++
		}
		cp := b.String()
		cpLen := len(cp)
		if !(cpLen == 4 || cpLen == 6) {
			l.errCause = synErrInvalidCodePoint
			return nil, ParseErr
		}
		return newCodePointToken(b.String()), nil
	}
}

func isHexDigit(c rune) bool {
	if c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
		return true
	}
	return false
}

func (l *lexer) nextInCharProp(c rune) (*token, error) {
	switch c {
	case '{':
		return newToken(tokenKindLBrace, nullChar), nil
	case '}':
		return newToken(tokenKindRBrace, nullChar), nil
	case '=':
		return newToken(tokenKindEqual, nullChar), nil
	default:
		var b strings.Builder
		fmt.Fprint(&b, string(c))
		n := 1
		for {
			c, eof, err := l.read()
			if err != nil {
				return nil, err
			}
			if eof {
				err := l.restore()
				if err != nil {
					return nil, err
				}
				break
			}
			if c == '}' || c == '=' {
				err := l.restore()
				if err != nil {
					return nil, err
				}
				break
			}
			fmt.Fprint(&b, string(c))
			n++
		}
		sym := strings.TrimSpace(b.String())
		if len(sym) == 0 {
			l.errCause = synErrCharPropInvalidSymbol
			return nil, ParseErr
		}
		return newCharPropSymbolToken(sym), nil
	}
}

func (l *lexer) nextInFragment(c rune) (*token, error) {
	switch c {
	case '{':
		return newToken(tokenKindLBrace, nullChar), nil
	case '}':
		return newToken(tokenKindRBrace, nullChar), nil
	default:
		var b strings.Builder
		fmt.Fprint(&b, string(c))
		n := 1
		for {
			c, eof, err := l.read()
			if err != nil {
				return nil, err
			}
			if eof {
				err := l.restore()
				if err != nil {
					return nil, err
				}
				break
			}
			if c == '}' {
				err := l.restore()
				if err != nil {
					return nil, err
				}
				break
			}
			fmt.Fprint(&b, string(c))
			n++
		}
		sym := strings.TrimSpace(b.String())
		if len(sym) == 0 {
			l.errCause = SynErrFragmentInvalidSymbol
			return nil, ParseErr
		}
		return newFragmentSymbolToken(sym), nil
	}
}

func (l *lexer) read() (rune, bool, error) {
	if l.reachedEOF {
		return l.lastChar, l.reachedEOF, nil
	}
	if l.peekChar1 != nullChar || l.peekEOF1 {
		l.prevChar2 = l.prevChar1
		l.pervEOF2 = l.prevEOF1
		l.prevChar1 = l.lastChar
		l.prevEOF1 = l.reachedEOF
		l.lastChar = l.peekChar1
		l.reachedEOF = l.peekEOF1
		l.peekChar1 = l.peekChar2
		l.peekEOF1 = l.peekEOF2
		l.peekChar2 = nullChar
		l.peekEOF2 = false
		return l.lastChar, l.reachedEOF, nil
	}
	c, _, err := l.src.ReadRune()
	if err != nil {
		if err == io.EOF {
			l.prevChar2 = l.prevChar1
			l.pervEOF2 = l.prevEOF1
			l.prevChar1 = l.lastChar
			l.prevEOF1 = l.reachedEOF
			l.lastChar = nullChar
			l.reachedEOF = true
			return l.lastChar, l.reachedEOF, nil
		}
		return nullChar, false, err
	}
	l.prevChar2 = l.prevChar1
	l.pervEOF2 = l.prevEOF1
	l.prevChar1 = l.lastChar
	l.prevEOF1 = l.reachedEOF
	l.lastChar = c
	l.reachedEOF = false
	return l.lastChar, l.reachedEOF, nil
}

func (l *lexer) restore() error {
	if l.lastChar == nullChar && !l.reachedEOF {
		return fmt.Errorf("failed to call restore() because the last character is null")
	}
	l.peekChar2 = l.peekChar1
	l.peekEOF2 = l.peekEOF1
	l.peekChar1 = l.lastChar
	l.peekEOF1 = l.reachedEOF
	l.lastChar = l.prevChar1
	l.reachedEOF = l.prevEOF1
	l.prevChar1 = l.prevChar2
	l.prevEOF1 = l.pervEOF2
	l.prevChar2 = nullChar
	l.pervEOF2 = false
	return nil
}
