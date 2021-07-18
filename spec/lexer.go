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
	tokenKindColon           = tokenKind(":")
	tokenKindOr              = tokenKind("|")
	tokenKindSemicolon       = tokenKind(";")
	tokenKindDirectiveMarker = tokenKind("#")
	tokenKindTreeNodeOpen    = tokenKind("'(")
	tokenKindTreeNodeClose   = tokenKind(")")
	tokenKindPosition        = tokenKind("$")
	tokenKindExpantion       = tokenKind("...")
	tokenKindNewline         = tokenKind("newline")
	tokenKindEOF             = tokenKind("eof")
	tokenKindInvalid         = tokenKind("invalid")
)

type Position struct {
	row int
}

func newPosition(row int) Position {
	return Position{
		row: row,
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

func newInvalidToken(text string) *token {
	return &token{
		kind: tokenKindInvalid,
		text: text,
	}
}

type lexer struct {
	s   *mlspec.CompiledLexSpec
	d   *mldriver.Lexer
	buf *token
	row int
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
		s:   s,
		d:   d,
		row: 1,
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
			newInvalidToken(tok.Text())
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
		row := l.row
		l.row++
		return newSymbolToken(tokenKindNewline, newPosition(row)), nil
	case "kw_fragment":
		return newSymbolToken(tokenKindKWFragment, newPosition(l.row)), nil
	case "identifier":
		if strings.HasPrefix(tok.Text(), "_") {
			return nil, &verr.SpecError{
				Cause: synErrAutoGenID,
				Row:   l.row,
			}
		}
		return newIDToken(tok.Text(), newPosition(l.row)), nil
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
					Row:   l.row,
				}
			}
			switch tok.KindName {
			case "pattern":
				// Remove '\' character.
				fmt.Fprintf(&b, strings.ReplaceAll(tok.Text(), `\"`, `"`))
			case "escape_symbol":
				return nil, &verr.SpecError{
					Cause: synErrIncompletedEscSeq,
					Row:   l.row,
				}
			case "terminal_close":
				return newTerminalPatternToken(b.String(), newPosition(l.row)), nil
			}
		}
	case "colon":
		return newSymbolToken(tokenKindColon, newPosition(l.row)), nil
	case "or":
		return newSymbolToken(tokenKindOr, newPosition(l.row)), nil
	case "semicolon":
		return newSymbolToken(tokenKindSemicolon, newPosition(l.row)), nil
	case "directive_marker":
		return newSymbolToken(tokenKindDirectiveMarker, newPosition(l.row)), nil
	case "tree_node_open":
		return newSymbolToken(tokenKindTreeNodeOpen, newPosition(l.row)), nil
	case "tree_node_close":
		return newSymbolToken(tokenKindTreeNodeClose, newPosition(l.row)), nil
	case "position":
		// Remove '$' character and convert to an integer.
		num, err := strconv.Atoi(tok.Text()[1:])
		if err != nil {
			return nil, err
		}
		if num == 0 {
			return nil, &verr.SpecError{
				Cause: synErrZeroPos,
				Row:   l.row,
			}
		}
		return newPositionToken(num, newPosition(l.row)), nil
	case "expansion":
		return newSymbolToken(tokenKindExpantion, newPosition(l.row)), nil
	default:
		return newInvalidToken(tok.Text()), nil
	}
}
