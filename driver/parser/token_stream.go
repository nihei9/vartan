package parser

import (
	"io"

	"github.com/nihei9/vartan/driver/lexer"
	spec "github.com/nihei9/vartan/spec/grammar"
)

type vToken struct {
	terminalID int
	tok        *lexer.Token
}

func (t *vToken) TerminalID() int {
	return t.terminalID
}

func (t *vToken) Lexeme() []byte {
	return t.tok.Lexeme
}

func (t *vToken) EOF() bool {
	return t.tok.EOF
}

func (t *vToken) Invalid() bool {
	return t.tok.Invalid
}

func (t *vToken) BytePosition() (int, int) {
	return t.tok.BytePos, t.tok.ByteLen
}

func (t *vToken) Position() (int, int) {
	return t.tok.Row, t.tok.Col
}

type tokenStream struct {
	lex            *lexer.Lexer
	kindToTerminal []int
}

func NewTokenStream(g *spec.CompiledGrammar, src io.Reader) (TokenStream, error) {
	lex, err := lexer.NewLexer(lexer.NewLexSpec(g.Lexical), src)
	if err != nil {
		return nil, err
	}

	return &tokenStream{
		lex:            lex,
		kindToTerminal: g.Syntactic.KindToTerminal,
	}, nil
}

func (l *tokenStream) Next() (VToken, error) {
	tok, err := l.lex.Next()
	if err != nil {
		return nil, err
	}
	return &vToken{
		terminalID: l.kindToTerminal[tok.KindID],
		tok:        tok,
	}, nil
}
