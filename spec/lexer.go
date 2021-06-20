//go:generate maleeni compile -l lexspec.json -o clexspec.json

package spec

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	mldriver "github.com/nihei9/maleeni/driver"
	mlspec "github.com/nihei9/maleeni/spec"
)

type tokenKind string

const (
	tokenKindID              = tokenKind("id")
	tokenKindTerminalPattern = tokenKind("terminal pattern")
	tokenKindColon           = tokenKind(":")
	tokenKindOr              = tokenKind("|")
	tokenKindSemicolon       = tokenKind(";")
	tokenKindModifierMarker  = tokenKind("@")
	tokenKindActionLeader    = tokenKind("#")
	tokenKindEOF             = tokenKind("eof")
	tokenKindInvalid         = tokenKind("invalid")
)

type token struct {
	kind tokenKind
	text string
}

func newSymbolToken(kind tokenKind) *token {
	return &token{
		kind: kind,
	}
}

func newIDToken(text string) *token {
	return &token{
		kind: tokenKindID,
		text: text,
	}
}

func newTerminalPatternToken(text string) *token {
	return &token{
		kind: tokenKindTerminalPattern,
		text: text,
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
	s      *mlspec.CompiledLexSpec
	d      *mldriver.Lexer
	dufTok *token
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
	for {
		tok, err := l.d.Next()
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
		case "newline":
			continue
		case "line_comment":
			continue
		case "identifier":
			return newIDToken(tok.Text()), nil
		case "terminal_open":
			var b strings.Builder
			for {
				tok, err := l.d.Next()
				if err != nil {
					return nil, err
				}
				if tok.EOF {
					return nil, synErrUnclosedTerminal
				}
				switch tok.KindName {
				case "pattern":
					// Remove '\' character.
					fmt.Fprintf(&b, strings.ReplaceAll(tok.Text(), `\"`, `"`))
				case "escape_symbol":
					return nil, synErrIncompletedEscSeq
				case "terminal_close":
					return newTerminalPatternToken(b.String()), nil
				}
			}
		case "colon":
			return newSymbolToken(tokenKindColon), nil
		case "or":
			return newSymbolToken(tokenKindOr), nil
		case "semicolon":
			return newSymbolToken(tokenKindSemicolon), nil
		case "modifier_marker":
			return newSymbolToken(tokenKindModifierMarker), nil
		case "action_leader":
			return newSymbolToken(tokenKindActionLeader), nil
		default:
			return newInvalidToken(tok.Text()), nil
		}
	}
}
