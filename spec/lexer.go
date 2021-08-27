//go:generate maleeni compile -l lexspec.json -o clexspec.json

package spec

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	mldriver "github.com/nihei9/maleeni/driver"
	mlspec "github.com/nihei9/maleeni/spec"
	verr "github.com/nihei9/vartan/error"
)

type tokenKind string

const (
	tokenKindKWFragment      = tokenKind("fragment")
	tokenKindID              = tokenKind("id")
	tokenKindTerminalPattern = tokenKind("terminal pattern")
	tokenKindStringLiteral   = tokenKind("string")
	tokenKindColon           = tokenKind(":")
	tokenKindOr              = tokenKind("|")
	tokenKindSemicolon       = tokenKind(";")
	tokenKindDirectiveMarker = tokenKind("#")
	tokenKindTreeNodeOpen    = tokenKind("#(")
	tokenKindTreeNodeClose   = tokenKind(")")
	tokenKindPosition        = tokenKind("$")
	tokenKindExpantion       = tokenKind("...")
	tokenKindMetaDataMarker  = tokenKind("%")
	tokenKindNewline         = tokenKind("newline")
	tokenKindEOF             = tokenKind("eof")
	tokenKindInvalid         = tokenKind("invalid")
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
	num  int
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

func newPositionToken(num int, pos Position) *token {
	return &token{
		kind: tokenKindPosition,
		num:  num,
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
	s   *mlspec.CompiledLexSpec
	d   *mldriver.Lexer
	buf *token
}

//go:embed clexspec.json
var lexspec []byte

func newLexer(src io.Reader) (*lexer, error) {
	s := &mlspec.CompiledLexSpec{}
	err := json.Unmarshal(lexspec, s)
	if err != nil {
		return nil, err
	}
	d, err := mldriver.NewLexer(s, src)
	if err != nil {
		return nil, err
	}
	return &lexer{
		s: s,
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
	var tok *mldriver.Token
	for {
		var err error
		tok, err = l.d.Next()
		if err != nil {
			return nil, err
		}
		if tok.Invalid {
			return newInvalidToken(tok.Text(), newPosition(tok.Row+1, tok.Col+1)), nil
		}
		if tok.EOF {
			return newEOFToken(), nil
		}
		switch tok.KindName {
		case "white_space":
			continue
		case "line_comment":
			continue
		}

		break
	}

	switch tok.KindName {
	case "newline":
		return newSymbolToken(tokenKindNewline, newPosition(tok.Row+1, tok.Col+1)), nil
	case "kw_fragment":
		return newSymbolToken(tokenKindKWFragment, newPosition(tok.Row+1, tok.Col+1)), nil
	case "identifier":
		if strings.HasPrefix(tok.Text(), "_") {
			return nil, &verr.SpecError{
				Cause:  synErrAutoGenID,
				Detail: tok.Text(),
				Row:    tok.Row + 1,
				Col:    tok.Col + 1,
			}
		}
		return newIDToken(tok.Text(), newPosition(tok.Row+1, tok.Col+1)), nil
	case "terminal_open":
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
			switch tok.KindName {
			case "pattern":
				// The escape sequences in a pattern string are interpreted by the lexer, except for the \".
				// We must interpret the \" before passing them to the lexer because they are delimiters for
				// the pattern strings.
				fmt.Fprintf(&b, strings.ReplaceAll(tok.Text(), `\"`, `"`))
			case "escape_symbol":
				return nil, &verr.SpecError{
					Cause: synErrIncompletedEscSeq,
					Row:   tok.Row + 1,
					Col:   tok.Col + 1,
				}
			case "terminal_close":
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
	case "string_literal_open":
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
			switch tok.KindName {
			case "char_seq":
				fmt.Fprintf(&b, tok.Text())
			case "escaped_quot":
				// Remove '\' character.
				fmt.Fprintf(&b, `'`)
			case "escaped_back_slash":
				// Remove '\' character.
				fmt.Fprintf(&b, `\`)
			case "escape_symbol":
				return nil, &verr.SpecError{
					Cause: synErrIncompletedEscSeq,
					Row:   tok.Row + 1,
					Col:   tok.Col + 1,
				}
			case "string_literal_close":
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
	case "colon":
		return newSymbolToken(tokenKindColon, newPosition(tok.Row+1, tok.Col+1)), nil
	case "or":
		return newSymbolToken(tokenKindOr, newPosition(tok.Row+1, tok.Col+1)), nil
	case "semicolon":
		return newSymbolToken(tokenKindSemicolon, newPosition(tok.Row+1, tok.Col+1)), nil
	case "directive_marker":
		return newSymbolToken(tokenKindDirectiveMarker, newPosition(tok.Row+1, tok.Col+1)), nil
	case "tree_node_open":
		return newSymbolToken(tokenKindTreeNodeOpen, newPosition(tok.Row+1, tok.Col+1)), nil
	case "tree_node_close":
		return newSymbolToken(tokenKindTreeNodeClose, newPosition(tok.Row+1, tok.Col+1)), nil
	case "position":
		// Remove '$' character and convert to an integer.
		num, err := strconv.Atoi(tok.Text()[1:])
		if err != nil {
			return nil, err
		}
		if num == 0 {
			return nil, &verr.SpecError{
				Cause: synErrZeroPos,
				Row:   tok.Row + 1,
				Col:   tok.Col + 1,
			}
		}
		return newPositionToken(num, newPosition(tok.Row+1, tok.Col+1)), nil
	case "expansion":
		return newSymbolToken(tokenKindExpantion, newPosition(tok.Row+1, tok.Col+1)), nil
	case "metadata_marker":
		return newSymbolToken(tokenKindMetaDataMarker, newPosition(tok.Row+1, tok.Col+1)), nil
	default:
		return newInvalidToken(tok.Text(), newPosition(tok.Row+1, tok.Col+1)), nil
	}
}
