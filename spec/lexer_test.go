package spec

import (
	"strings"
	"testing"
)

func TestLexer_Run(t *testing.T) {
	tests := []struct {
		caption string
		src     string
		tokens  []*token
		err     error
	}{
		{
			caption: "the lexer can recognize all kinds of tokens",
			src:     `id"terminal":|;`,
			tokens: []*token{
				newIDToken("id"),
				newTerminalPatternToken("terminal"),
				newSymbolToken(tokenKindColon),
				newSymbolToken(tokenKindOr),
				newSymbolToken(tokenKindSemicolon),
				newEOFToken(),
			},
		},
		{
			caption: "the lexer can recognize character sequences and escape sequences in terminal",
			src:     `"abc\"\\"`,
			tokens: []*token{
				newTerminalPatternToken(`abc"\\`),
				newEOFToken(),
			},
		},
		{
			caption: "an unclosed terminal is not a valid token",
			src:     `"abc`,
			err:     synErrUnclosedTerminal,
		},
		{
			caption: "an incompleted terminal is not a valid token",
			src:     `"\`,
			err:     synErrIncompletedEscSeq,
		},
		{
			caption: "the lexer can recognize valid tokens following an invalid token",
			src:     `abc!!!def`,
			tokens: []*token{
				newIDToken("abc"),
				newInvalidToken("!!!"),
				newIDToken("def"),
				newEOFToken(),
			},
		},
		{
			caption: "the lexer skips white spaces",
			// \u0009: HT
			// \u000A: LF
			// \u000D: CR
			// \u0020: SP
			src: "a\u0020b\u000Ac\u000Dd\u000D\u000Ae\u0009f",
			tokens: []*token{
				newIDToken("a"),
				newIDToken("b"),
				newIDToken("c"),
				newIDToken("d"),
				newIDToken("e"),
				newIDToken("f"),
				newEOFToken(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			l, err := newLexer(strings.NewReader(tt.src))
			if err != nil {
				t.Fatal(err)
			}
			n := 0
			for {
				var tok *token
				tok, err = l.next()
				if err != nil {
					break
				}
				testToken(t, tok, tt.tokens[n])
				n++
				if tok.kind == tokenKindEOF {
					break
				}
			}
			if err != tt.err {
				t.Fatalf("unexpected error; want: %v, got: %v", tt.err, err)
			}
		})
	}
}

func testToken(t *testing.T, tok, expected *token) {
	t.Helper()
	if tok.kind != expected.kind || tok.text != expected.text {
		t.Fatalf("unexpected token; want: %+v, got: %+v", expected, tok)
	}
}
