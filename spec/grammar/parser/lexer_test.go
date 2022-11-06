package parser

import (
	"strings"
	"testing"

	verr "github.com/nihei9/vartan/error"
)

func TestLexer_Run(t *testing.T) {
	idTok := func(text string) *token {
		return newIDToken(text, newPosition(1, 0))
	}

	termPatTok := func(text string) *token {
		return newTerminalPatternToken(text, newPosition(1, 0))
	}

	strTok := func(text string) *token {
		return newStringLiteralToken(text, newPosition(1, 0))
	}

	symTok := func(kind tokenKind) *token {
		return newSymbolToken(kind, newPosition(1, 0))
	}

	invalidTok := func(text string) *token {
		return newInvalidToken(text, newPosition(1, 0))
	}

	tests := []struct {
		caption string
		src     string
		tokens  []*token
		err     error
	}{
		{
			caption: "the lexer can recognize all kinds of tokens",
			src:     `id"terminal"'string':|;@...#$()`,
			tokens: []*token{
				idTok("id"),
				termPatTok("terminal"),
				strTok(`string`),
				symTok(tokenKindColon),
				symTok(tokenKindOr),
				symTok(tokenKindSemicolon),
				symTok(tokenKindLabelMarker),
				symTok(tokenKindExpantion),
				symTok(tokenKindDirectiveMarker),
				symTok(tokenKindOrderedSymbolMarker),
				symTok(tokenKindLParen),
				symTok(tokenKindRParen),
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
			caption: "the lexer can recognize character sequences and escape sequences in a terminal",
			src:     `"abc\"\\"`,
			tokens: []*token{
				termPatTok(`abc"\\`),
				newEOFToken(),
			},
		},
		{
			caption: "backslashes are recognized as they are because escape sequences are not allowed in strings",
			src:     `'\\\'`,
			tokens: []*token{
				strTok(`\\\`),
				newEOFToken(),
			},
		},
		{
			caption: "a pattern must include at least one character",
			src:     `""`,
			err:     synErrEmptyPattern,
		},
		{
			caption: "a string must include at least one character",
			src:     `''`,
			err:     synErrEmptyString,
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
			caption: "an identifier cannot contain the capital-case letters",
			src:     `Abc`,
			err:     synErrIDInvalidChar,
		},
		{
			caption: "an identifier cannot contain the capital-case letters",
			src:     `Zyx`,
			err:     synErrIDInvalidChar,
		},
		{
			caption: "the underscore cannot be placed at the beginning of an identifier",
			src:     `_abc`,
			err:     synErrIDInvalidUnderscorePos,
		},
		{
			caption: "the underscore cannot be placed at the end of an identifier",
			src:     `abc_`,
			err:     synErrIDInvalidUnderscorePos,
		},
		{
			caption: "the underscore cannot be placed consecutively",
			src:     `a__b`,
			err:     synErrIDConsecutiveUnderscores,
		},
		{
			caption: "the digits cannot be placed at the biginning of an identifier",
			src:     `0abc`,
			err:     synErrIDInvalidDigitsPos,
		},
		{
			caption: "the digits cannot be placed at the biginning of an identifier",
			src:     `9abc`,
			err:     synErrIDInvalidDigitsPos,
		},
		{
			caption: "an unclosed terminal is not a valid token",
			src:     `"abc`,
			err:     synErrUnclosedTerminal,
		},
		{
			caption: "an incompleted escape sequence in a pattern is not a valid token",
			src:     `"\`,
			err:     synErrIncompletedEscSeq,
		},
		{
			caption: "an unclosed string is not a valid token",
			src:     `'abc`,
			err:     synErrUnclosedString,
		},
		{
			caption: "the lexer can recognize valid tokens following an invalid token",
			src:     `abc!!!def`,
			tokens: []*token{
				idTok("abc"),
				invalidTok("!!!"),
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
	if tok.kind != expected.kind || tok.text != expected.text {
		t.Fatalf("unexpected token; want: %+v, got: %+v", expected, tok)
	}
}
