package spec

import (
	"strings"
	"testing"

	verr "github.com/nihei9/vartan/error"
)

func TestLexer_Run(t *testing.T) {
	idTok := func(text string) *token {
		return newIDToken(text, newPosition(1))
	}

	termPatTok := func(text string) *token {
		return newTerminalPatternToken(text, newPosition(1))
	}

	symTok := func(kind tokenKind) *token {
		return newSymbolToken(kind, newPosition(1))
	}

	posTok := func(num int) *token {
		return newPositionToken(num, newPosition(1))
	}

	tests := []struct {
		caption string
		src     string
		tokens  []*token
		err     error
	}{
		{
			caption: "the lexer can recognize all kinds of tokens",
			src:     `id"terminal"'.*+?|()[\':|;#()$1...#`,
			tokens: []*token{
				idTok("id"),
				termPatTok("terminal"),
				termPatTok(`\.\*\+\?\|\(\)\[\\`),
				symTok(tokenKindColon),
				symTok(tokenKindOr),
				symTok(tokenKindSemicolon),
				symTok(tokenKindTreeNodeOpen),
				symTok(tokenKindTreeNodeClose),
				posTok(1),
				symTok(tokenKindExpantion),
				symTok(tokenKindDirectiveMarker),
				newEOFToken(),
			},
		},
		{
			caption: "the lexer can recognize keywords",
			src:     `fragment`,
			tokens: []*token{
				symTok(tokenKindKWFragment),
				newEOFToken(),
			},
		},
		{
			caption: "the lexer can recognize character sequences and escape sequences in terminal",
			src:     `"abc\"\\"`,
			tokens: []*token{
				termPatTok(`abc"\\`),
				newEOFToken(),
			},
		},
		{
			caption: "a pattern must include at least one character",
			src:     `""`,
			err:     synErrEmptyPattern,
		},
		{
			caption: "a literal pattern must include at least one character",
			src:     `''`,
			err:     synErrEmptyPattern,
		},
		{
			caption: "the lexer can recognize newlines and combine consecutive newlines into one",
			src:     "\u000A | \u000D | \u000D\u000A | \u000A\u000A \u000D\u000D \u000D\u000A\u000D\u000A",
			tokens: []*token{
				symTok(tokenKindNewline),
				symTok(tokenKindOr),
				symTok(tokenKindNewline),
				symTok(tokenKindOr),
				symTok(tokenKindNewline),
				symTok(tokenKindOr),
				symTok(tokenKindNewline),
				newEOFToken(),
			},
		},
		{
			caption: "the lexer ignores line comments",
			src: `
// This is the first comment.
foo
// This is the second comment.
// This is the third comment.
bar // This is the fourth comment.
`,
			tokens: []*token{
				symTok(tokenKindNewline),
				idTok("foo"),
				symTok(tokenKindNewline),
				idTok("bar"),
				symTok(tokenKindNewline),
				newEOFToken(),
			},
		},
		{
			caption: "identifiers beginning with an underscore are not allowed because they are used only auto-generated identifiers",
			src:     `_abc`,
			err:     synErrAutoGenID,
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
			caption: "a position must be greater than or equal to 1",
			src:     `$0`,
			err:     synErrZeroPos,
		},
		{
			caption: "the lexer can recognize valid tokens following an invalid token",
			src:     `abc!!!def`,
			tokens: []*token{
				idTok("abc"),
				newInvalidToken("!!!"),
				idTok("def"),
				newEOFToken(),
			},
		},
		{
			caption: "the lexer skips white spaces",
			// \u0009: HT
			// \u0020: SP
			src: "a\u0009b\u0020c",
			tokens: []*token{
				idTok("a"),
				idTok("b"),
				idTok("c"),
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
			if tt.err != nil {
				synErr, ok := err.(*verr.SpecError)
				if !ok {
					t.Fatalf("unexpected error; want: %v, got: %v", tt.err, err)
				}
				if tt.err != synErr.Cause {
					t.Fatalf("unexpected error; want: %v, got: %v", tt.err, synErr.Cause)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error; want: %v, got: %v", tt.err, err)
				}
			}
		})
	}
}

func testToken(t *testing.T, tok, expected *token) {
	t.Helper()
	if tok.kind != expected.kind || tok.text != expected.text || tok.num != expected.num {
		t.Fatalf("unexpected token; want: %+v, got: %+v", expected, tok)
	}
}
