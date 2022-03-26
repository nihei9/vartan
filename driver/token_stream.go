package driver

import (
	"io"

	mldriver "github.com/nihei9/maleeni/driver"
	"github.com/nihei9/vartan/spec"
)

type vToken struct {
	terminalID int
	skip       bool
	tok        *mldriver.Token
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

func (t *vToken) Skip() bool {
	return t.skip
}

func (t *vToken) Position() (int, int) {
	return t.tok.Row, t.tok.Col
}

type tokenStream struct {
	lex            *mldriver.Lexer
	kindToTerminal []int
	skip           []int
}

func NewTokenStream(g *spec.CompiledGrammar, src io.Reader) (TokenStream, error) {
	lex, err := mldriver.NewLexer(mldriver.NewLexSpec(g.LexicalSpecification.Maleeni.Spec), src)
	if err != nil {
		return nil, err
	}

	return &tokenStream{
		lex:            lex,
		kindToTerminal: g.LexicalSpecification.Maleeni.KindToTerminal,
		skip:           g.LexicalSpecification.Maleeni.Skip,
	}, nil
}

func (l *tokenStream) Next() (VToken, error) {
	tok, err := l.lex.Next()
	if err != nil {
		return nil, err
	}
	return &vToken{
		terminalID: l.kindToTerminal[tok.KindID],
		skip:       l.skip[tok.KindID] > 0,
		tok:        tok,
	}, nil
}