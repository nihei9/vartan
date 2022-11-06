package parser

import (
	"strings"
	"testing"
)

func TestLexer(t *testing.T) {
	tests := []struct {
		caption string
		src     string
		tokens  []*token
		err     error
	}{
		{
			caption: "lexer can recognize ordinaly characters",
			src:     "123abcいろは",
			tokens: []*token{
				newToken(tokenKindChar, '1'),
				newToken(tokenKindChar, '2'),
				newToken(tokenKindChar, '3'),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindChar, 'b'),
				newToken(tokenKindChar, 'c'),
				newToken(tokenKindChar, 'い'),
				newToken(tokenKindChar, 'ろ'),
				newToken(tokenKindChar, 'は'),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "lexer can recognize the special characters in default mode",
			src:     ".*+?|()[\\u",
			tokens: []*token{
				newToken(tokenKindAnyChar, nullChar),
				newToken(tokenKindRepeat, nullChar),
				newToken(tokenKindRepeatOneOrMore, nullChar),
				newToken(tokenKindOption, nullChar),
				newToken(tokenKindAlt, nullChar),
				newToken(tokenKindGroupOpen, nullChar),
				newToken(tokenKindGroupClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "lexer can recognize the escape sequences in default mode",
			src:     "\\\\\\.\\*\\+\\?\\|\\(\\)\\[",
			tokens: []*token{
				newToken(tokenKindChar, '\\'),
				newToken(tokenKindChar, '.'),
				newToken(tokenKindChar, '*'),
				newToken(tokenKindChar, '+'),
				newToken(tokenKindChar, '?'),
				newToken(tokenKindChar, '|'),
				newToken(tokenKindChar, '('),
				newToken(tokenKindChar, ')'),
				newToken(tokenKindChar, '['),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "], {, and } are treated as an ordinary character in default mode",
			src:     "]{}",
			tokens: []*token{
				newToken(tokenKindChar, ']'),
				newToken(tokenKindChar, '{'),
				newToken(tokenKindChar, '}'),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "lexer can recognize the special characters in bracket expression mode",
			src:     "[a-z\\u{09AF}][^a-z\\u{09abcf}]",
			tokens: []*token{
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindCharRange, nullChar),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("09AF"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindCharRange, nullChar),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("09abcf"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "lexer can recognize the escape sequences in bracket expression mode",
			src:     "[\\^a\\-z]",
			tokens: []*token{
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '^'),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "in a bracket expression, the special characters are also handled as normal characters",
			src:     "[\\\\.*+?|()[",
			tokens: []*token{
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '\\'),
				newToken(tokenKindChar, '.'),
				newToken(tokenKindChar, '*'),
				newToken(tokenKindChar, '+'),
				newToken(tokenKindChar, '?'),
				newToken(tokenKindChar, '|'),
				newToken(tokenKindChar, '('),
				newToken(tokenKindChar, ')'),
				newToken(tokenKindChar, '['),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "hyphen symbols that appear in bracket expressions are handled as the character range symbol or ordinary characters",
			// [...-...][...-][-...][-]
			//  ~~~~~~~     ~  ~     ~
			//     ^        ^  ^     ^
			//     |        |  |     `-- Ordinary Character (b)
			//     |        |  `-- Ordinary Character (b)
			//     |        `-- Ordinary Character (b)
			//     `-- Character Range (a)
			//
			// a. *-* is handled as a character-range expression.
			// b. *-, -*, or - are handled as ordinary characters.
			src: "[a-z][a-][-z][-][--][---][^a-z][^a-][^-z][^-][^--][^---]",
			tokens: []*token{
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindCharRange, nullChar),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindCharRange, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),

				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindCharRange, nullChar),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, 'a'),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindChar, 'z'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindCharRange, nullChar),
				newToken(tokenKindChar, '-'),
				newToken(tokenKindBExpClose, nullChar),

				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "caret symbols that appear in bracket expressions are handled as the logical inverse symbol or ordinary characters",
			// [^...^...][^]
			// ~~   ~    ~~
			// ^    ^    ^^
			// |    |    |`-- Ordinary Character (c)
			// |    |    `-- Bracket Expression
			// |    `-- Ordinary Character (b)
			// `-- Inverse Bracket Expression (a)
			//
			// a. Bracket expressions that have a caret symbol at the beginning are handled as logical inverse expressions.
			// b. caret symbols that appear as the second and the subsequent symbols are handled as ordinary symbols.
			// c. When a bracket expression has just one symbol, a caret symbol at the beginning is handled as an ordinary character.
			src: "[^^][^]",
			tokens: []*token{
				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindChar, '^'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindChar, '^'),
				newToken(tokenKindBExpClose, nullChar),
				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "lexer raises an error when an invalid escape sequence appears",
			src:     "\\@",
			err:     synErrInvalidEscSeq,
		},
		{
			caption: "lexer raises an error when the incomplete escape sequence (EOF following \\) appears",
			src:     "\\",
			err:     synErrIncompletedEscSeq,
		},
		{
			caption: "lexer raises an error when an invalid escape sequence appears",
			src:     "[\\@",
			tokens: []*token{
				newToken(tokenKindBExpOpen, nullChar),
			},
			err: synErrInvalidEscSeq,
		},
		{
			caption: "lexer raises an error when the incomplete escape sequence (EOF following \\) appears",
			src:     "[\\",
			tokens: []*token{
				newToken(tokenKindBExpOpen, nullChar),
			},
			err: synErrIncompletedEscSeq,
		},
		{
			caption: "lexer can recognize the special characters and code points in code point expression mode",
			src:     "\\u{0123}\\u{4567}\\u{89abcd}\\u{efAB}\\u{CDEF01}[\\u{0123}\\u{4567}\\u{89abcd}\\u{efAB}\\u{CDEF01}][^\\u{0123}\\u{4567}\\u{89abcd}\\u{efAB}\\u{CDEF01}]",
			tokens: []*token{
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("0123"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("4567"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("89abcd"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("efAB"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("CDEF01"),
				newToken(tokenKindRBrace, nullChar),

				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("0123"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("4567"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("89abcd"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("efAB"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("CDEF01"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindBExpClose, nullChar),

				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("0123"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("4567"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("89abcd"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("efAB"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("CDEF01"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindBExpClose, nullChar),

				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "a one digit hex string isn't a valid code point",
			src:     "\\u{0",
			tokens: []*token{
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
			},
			err: synErrInvalidCodePoint,
		},
		{
			caption: "a two digits hex string isn't a valid code point",
			src:     "\\u{01",
			tokens: []*token{
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
			},
			err: synErrInvalidCodePoint,
		},
		{
			caption: "a three digits hex string isn't a valid code point",
			src:     "\\u{012",
			tokens: []*token{
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
			},
			err: synErrInvalidCodePoint,
		},
		{
			caption: "a four digits hex string is a valid code point",
			src:     "\\u{0123}",
			tokens: []*token{
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("0123"),
				newToken(tokenKindRBrace, nullChar),
			},
		},
		{
			caption: "a five digits hex string isn't a valid code point",
			src:     "\\u{01234",
			tokens: []*token{
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
			},
			err: synErrInvalidCodePoint,
		},
		{
			caption: "a six digits hex string is a valid code point",
			src:     "\\u{012345}",
			tokens: []*token{
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCodePointToken("012345"),
				newToken(tokenKindRBrace, nullChar),
			},
		},
		{
			caption: "a seven digits hex string isn't a valid code point",
			src:     "\\u{0123456",
			tokens: []*token{
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
			},
			err: synErrInvalidCodePoint,
		},
		{
			caption: "a code point must be hex digits",
			src:     "\\u{g",
			tokens: []*token{
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
			},
			err: synErrInvalidCodePoint,
		},
		{
			caption: "a code point must be hex digits",
			src:     "\\u{G",
			tokens: []*token{
				newToken(tokenKindCodePointLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
			},
			err: synErrInvalidCodePoint,
		},
		{
			caption: "lexer can recognize the special characters and symbols in character property expression mode",
			src:     "\\p{Letter}\\p{General_Category=Letter}[\\p{Letter}\\p{General_Category=Letter}][^\\p{Letter}\\p{General_Category=Letter}]",
			tokens: []*token{
				newToken(tokenKindCharPropLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCharPropSymbolToken("Letter"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCharPropLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCharPropSymbolToken("General_Category"),
				newToken(tokenKindEqual, nullChar),
				newCharPropSymbolToken("Letter"),
				newToken(tokenKindRBrace, nullChar),

				newToken(tokenKindBExpOpen, nullChar),
				newToken(tokenKindCharPropLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCharPropSymbolToken("Letter"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCharPropLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCharPropSymbolToken("General_Category"),
				newToken(tokenKindEqual, nullChar),
				newCharPropSymbolToken("Letter"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindBExpClose, nullChar),

				newToken(tokenKindInverseBExpOpen, nullChar),
				newToken(tokenKindCharPropLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCharPropSymbolToken("Letter"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindCharPropLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newCharPropSymbolToken("General_Category"),
				newToken(tokenKindEqual, nullChar),
				newCharPropSymbolToken("Letter"),
				newToken(tokenKindRBrace, nullChar),
				newToken(tokenKindBExpClose, nullChar),

				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "lexer can recognize the special characters and symbols in fragment expression mode",
			src:     "\\f{integer}",
			tokens: []*token{
				newToken(tokenKindFragmentLeader, nullChar),
				newToken(tokenKindLBrace, nullChar),
				newFragmentSymbolToken("integer"),
				newToken(tokenKindRBrace, nullChar),

				newToken(tokenKindEOF, nullChar),
			},
		},
		{
			caption: "a fragment expression is not supported in a bracket expression",
			src:     "[\\f",
			tokens: []*token{
				newToken(tokenKindBExpOpen, nullChar),
			},
			err: synErrInvalidEscSeq,
		},
		{
			caption: "a fragment expression is not supported in an inverse bracket expression",
			src:     "[^\\f",
			tokens: []*token{
				newToken(tokenKindInverseBExpOpen, nullChar),
			},
			err: synErrInvalidEscSeq,
		},
	}
	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			lex := newLexer(strings.NewReader(tt.src))
			var err error
			var tok *token
			i := 0
			for {
				tok, err = lex.next()
				if err != nil {
					break
				}
				if i >= len(tt.tokens) {
					break
				}
				eTok := tt.tokens[i]
				i++
				testToken(t, tok, eTok)

				if tok.kind == tokenKindEOF {
					break
				}
			}
			if tt.err != nil {
				if err != ParseErr {
					t.Fatalf("unexpected error: want: %v, got: %v", ParseErr, err)
				}
				detail, cause := lex.error()
				if cause != tt.err {
					t.Fatalf("unexpected error: want: %v, got: %v (%v)", tt.err, cause, detail)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
			if i < len(tt.tokens) {
				t.Fatalf("expecte more tokens")
			}
		})
	}
}

func testToken(t *testing.T, a, e *token) {
	t.Helper()
	if e.kind != a.kind || e.char != a.char || e.codePoint != a.codePoint {
		t.Fatalf("unexpected token: want: %+v, got: %+v", e, a)
	}
}
