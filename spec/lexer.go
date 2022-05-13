//go:generate maleeni compile lexspec.json -o clexspec.json
//go:generate maleeni-go clexspec.json --package spec

package spec

import (
	_ "embed"
	"fmt"
	"io"
	"regexp"
	"strings"

	verr "github.com/nihei9/vartan/error"
)

type tokenKind string

const (
	tokenKindKWFragment          = tokenKind("fragment")
	tokenKindID                  = tokenKind("id")
	tokenKindTerminalPattern     = tokenKind("terminal pattern")
	tokenKindStringLiteral       = tokenKind("string")
	tokenKindColon               = tokenKind(":")
	tokenKindOr                  = tokenKind("|")
	tokenKindSemicolon           = tokenKind(";")
	tokenKindLabelMarker         = tokenKind("@")
	tokenKindDirectiveMarker     = tokenKind("#")
	tokenKindExpantion           = tokenKind("...")
	tokenKindOrderedSymbolMarker = tokenKind("$")
	tokenKindLParen              = tokenKind("(")
	tokenKindRParen              = tokenKind(")")
	tokenKindNewline             = tokenKind("newline")
	tokenKindEOF                 = tokenKind("eof")
	tokenKindInvalid             = tokenKind("invalid")
)

var (
	reIDChar             = regexp.MustCompile(`^[0-9a-z_]+$`)
	reIDInvalidDigitsPos = regexp.MustCompile(`^[0-9]`)
)

type Position struct {
	Row int
	Col int
}

func newPosition(row, col int) Position {
	return Position{
		Row: row,
		Col: col,
	}
}

type token struct {
	kind tokenKind
	text string
	pos  Position
}

func newSymbolToken(kind tokenKind, pos Position) *token {
	return &token{
		kind: kind,
		pos:  pos,
	}
}

func newIDToken(text string, pos Position) *token {
	return &token{
		kind: tokenKindID,
		text: text,
		pos:  pos,
	}
}

func newTerminalPatternToken(text string, pos Position) *token {
	return &token{
		kind: tokenKindTerminalPattern,
		text: text,
		pos:  pos,
	}
}

func newStringLiteralToken(text string, pos Position) *token {
	return &token{
		kind: tokenKindStringLiteral,
		text: text,
		pos:  pos,
	}
}

func newEOFToken() *token {
	return &token{
		kind: tokenKindEOF,
	}
}

func newInvalidToken(text string, pos Position) *token {
	return &token{
		kind: tokenKindInvalid,
		text: text,
		pos:  pos,
	}
}

type lexer struct {
	d   *Lexer
	buf *token
}

func newLexer(src io.Reader) (*lexer, error) {
	d, err := NewLexer(NewLexSpec(), src)
	if err != nil {
		return nil, err
	}
	return &lexer{
		d: d,
	}, nil
}

func (l *lexer) next() (*token, error) {
	if l.buf != nil {
		tok := l.buf
		l.buf = nil
		return tok, nil
	}

	var newline *token
	for {
		tok, err := l.lexAndSkipWSs()
		if err != nil {
			return nil, err
		}
		if tok.kind == tokenKindNewline {
			newline = tok
			continue
		}

		if newline != nil {
			l.buf = tok
			return newline, nil
		}
		return tok, nil
	}
}

func (l *lexer) lexAndSkipWSs() (*token, error) {
	var tok *Token
	for {
		var err error
		tok, err = l.d.Next()
		if err != nil {
			return nil, err
		}
		if tok.Invalid {
			return newInvalidToken(string(tok.Lexeme), newPosition(tok.Row+1, tok.Col+1)), nil
		}
		if tok.EOF {
			return newEOFToken(), nil
		}
		switch tok.KindID {
		case KindIDWhiteSpace:
			continue
		case KindIDLineComment:
			continue
		}

		break
	}

	switch tok.KindID {
	case KindIDNewline:
		return newSymbolToken(tokenKindNewline, newPosition(tok.Row+1, tok.Col+1)), nil
	case KindIDKwFragment:
		return newSymbolToken(tokenKindKWFragment, newPosition(tok.Row+1, tok.Col+1)), nil
	case KindIDIdentifier:
		if !reIDChar.Match(tok.Lexeme) {
			return nil, &verr.SpecError{
				Cause:  synErrIDInvalidChar,
				Detail: string(tok.Lexeme),
				Row:    tok.Row + 1,
				Col:    tok.Col + 1,
			}
		}
		if strings.HasPrefix(string(tok.Lexeme), "_") || strings.HasSuffix(string(tok.Lexeme), "_") {
			return nil, &verr.SpecError{
				Cause:  synErrIDInvalidUnderscorePos,
				Detail: string(tok.Lexeme),
				Row:    tok.Row + 1,
				Col:    tok.Col + 1,
			}
		}
		if strings.Contains(string(tok.Lexeme), "__") {
			return nil, &verr.SpecError{
				Cause:  synErrIDConsecutiveUnderscores,
				Detail: string(tok.Lexeme),
				Row:    tok.Row + 1,
				Col:    tok.Col + 1,
			}
		}
		if reIDInvalidDigitsPos.Match(tok.Lexeme) {
			return nil, &verr.SpecError{
				Cause:  synErrIDInvalidDigitsPos,
				Detail: string(tok.Lexeme),
				Row:    tok.Row + 1,
				Col:    tok.Col + 1,
			}
		}
		return newIDToken(string(tok.Lexeme), newPosition(tok.Row+1, tok.Col+1)), nil
	case KindIDTerminalOpen:
		var b strings.Builder
		for {
			tok, err := l.d.Next()
			if err != nil {
				return nil, err
			}
			if tok.EOF {
				return nil, &verr.SpecError{
					Cause: synErrUnclosedTerminal,
					Row:   tok.Row + 1,
					Col:   tok.Col + 1,
				}
			}
			switch tok.KindID {
			case KindIDPattern:
				// The escape sequences in a pattern string are interpreted by the lexer, except for the \".
				// We must interpret the \" before passing them to the lexer because they are delimiters for
				// the pattern strings.
				fmt.Fprint(&b, strings.ReplaceAll(string(tok.Lexeme), `\"`, `"`))
			case KindIDEscapeSymbol:
				return nil, &verr.SpecError{
					Cause: synErrIncompletedEscSeq,
					Row:   tok.Row + 1,
					Col:   tok.Col + 1,
				}
			case KindIDTerminalClose:
				pat := b.String()
				if pat == "" {
					return nil, &verr.SpecError{
						Cause: synErrEmptyPattern,
						Row:   tok.Row + 1,
						Col:   tok.Col + 1,
					}
				}
				return newTerminalPatternToken(pat, newPosition(tok.Row+1, tok.Col+1)), nil
			}
		}
	case KindIDStringLiteralOpen:
		var b strings.Builder
		for {
			tok, err := l.d.Next()
			if err != nil {
				return nil, err
			}
			if tok.EOF {
				return nil, &verr.SpecError{
					Cause: synErrUnclosedString,
					Row:   tok.Row + 1,
					Col:   tok.Col + 1,
				}
			}
			switch tok.KindID {
			case KindIDCharSeq:
				fmt.Fprint(&b, string(tok.Lexeme))
			case KindIDStringLiteralClose:
				str := b.String()
				if str == "" {
					return nil, &verr.SpecError{
						Cause: synErrEmptyString,
						Row:   tok.Row + 1,
						Col:   tok.Col + 1,
					}
				}
				return newStringLiteralToken(str, newPosition(tok.Row+1, tok.Col+1)), nil
			}
		}
	case KindIDColon:
		return newSymbolToken(tokenKindColon, newPosition(tok.Row+1, tok.Col+1)), nil
	case KindIDOr:
		return newSymbolToken(tokenKindOr, newPosition(tok.Row+1, tok.Col+1)), nil
	case KindIDSemicolon:
		return newSymbolToken(tokenKindSemicolon, newPosition(tok.Row+1, tok.Col+1)), nil
	case KindIDLabelMarker:
		return newSymbolToken(tokenKindLabelMarker, newPosition(tok.Row+1, tok.Col+1)), nil
	case KindIDDirectiveMarker:
		return newSymbolToken(tokenKindDirectiveMarker, newPosition(tok.Row+1, tok.Col+1)), nil
	case KindIDExpansion:
		return newSymbolToken(tokenKindExpantion, newPosition(tok.Row+1, tok.Col+1)), nil
	case KindIDOrderedSymbolMarker:
		return newSymbolToken(tokenKindOrderedSymbolMarker, newPosition(tok.Row+1, tok.Col+1)), nil
	case KindIDLParen:
		return newSymbolToken(tokenKindLParen, newPosition(tok.Row+1, tok.Col+1)), nil
	case KindIDRParen:
		return newSymbolToken(tokenKindRParen, newPosition(tok.Row+1, tok.Col+1)), nil
	default:
		return newInvalidToken(string(tok.Lexeme), newPosition(tok.Row+1, tok.Col+1)), nil
	}
}
