package lexer

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar/lexical"
	spec "github.com/nihei9/vartan/spec/grammar"
)

func newLexEntry(modes []string, kind string, pattern string, push string, pop bool) *lexical.LexEntry {
	ms := []spec.LexModeName{}
	for _, m := range modes {
		ms = append(ms, spec.LexModeName(m))
	}
	return &lexical.LexEntry{
		Kind:    spec.LexKindName(kind),
		Pattern: pattern,
		Modes:   ms,
		Push:    spec.LexModeName(push),
		Pop:     pop,
	}
}

func newLexEntryDefaultNOP(kind string, pattern string) *lexical.LexEntry {
	return &lexical.LexEntry{
		Kind:    spec.LexKindName(kind),
		Pattern: pattern,
		Modes: []spec.LexModeName{
			spec.LexModeNameDefault,
		},
	}
}

func newLexEntryFragment(kind string, pattern string) *lexical.LexEntry {
	return &lexical.LexEntry{
		Kind:     spec.LexKindName(kind),
		Pattern:  pattern,
		Fragment: true,
	}
}

func newToken(modeID ModeID, kindID KindID, modeKindID ModeKindID, lexeme []byte) *Token {
	return &Token{
		ModeID:     modeID,
		KindID:     kindID,
		ModeKindID: modeKindID,
		Lexeme:     lexeme,
	}
}

func newTokenDefault(kindID int, modeKindID int, lexeme []byte) *Token {
	return newToken(
		ModeID(spec.LexModeIDDefault.Int()),
		KindID(spec.LexKindID(kindID).Int()),
		ModeKindID(spec.LexModeKindID(modeKindID).Int()),
		lexeme,
	)
}

func newEOFToken(modeID ModeID, modeName string) *Token {
	return &Token{
		ModeID:     modeID,
		ModeKindID: 0,
		EOF:        true,
	}
}

func newEOFTokenDefault() *Token {
	return newEOFToken(ModeID(spec.LexModeIDDefault.Int()), spec.LexModeNameDefault.String())
}

func newInvalidTokenDefault(lexeme []byte) *Token {
	return &Token{
		ModeID:     ModeID(spec.LexModeIDDefault.Int()),
		ModeKindID: 0,
		Lexeme:     lexeme,
		Invalid:    true,
	}
}

func withPos(tok *Token, row, col int) *Token {
	tok.Row = row
	tok.Col = col
	return tok
}

func TestLexer_Next(t *testing.T) {
	test := []struct {
		lspec           *lexical.LexSpec
		src             string
		tokens          []*Token
		passiveModeTran bool
		tran            func(l *Lexer, tok *Token) error
	}{
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("t1", "(a|b)*abb"),
					newLexEntryDefaultNOP("t2", " +"),
				},
			},
			src: "abb aabb aaabb babb bbabb abbbabb",
			tokens: []*Token{
				newTokenDefault(1, 1, []byte("abb")),
				newTokenDefault(2, 2, []byte(" ")),
				newTokenDefault(1, 1, []byte("aabb")),
				newTokenDefault(2, 2, []byte(" ")),
				newTokenDefault(1, 1, []byte("aaabb")),
				newTokenDefault(2, 2, []byte(" ")),
				newTokenDefault(1, 1, []byte("babb")),
				newTokenDefault(2, 2, []byte(" ")),
				newTokenDefault(1, 1, []byte("bbabb")),
				newTokenDefault(2, 2, []byte(" ")),
				newTokenDefault(1, 1, []byte("abbbabb")),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("t1", "b?a+"),
					newLexEntryDefaultNOP("t2", "(ab)?(cd)+"),
					newLexEntryDefaultNOP("t3", " +"),
				},
			},
			src: "ba baaa a aaa abcd abcdcdcd cd cdcdcd",
			tokens: []*Token{
				newTokenDefault(1, 1, []byte("ba")),
				newTokenDefault(3, 3, []byte(" ")),
				newTokenDefault(1, 1, []byte("baaa")),
				newTokenDefault(3, 3, []byte(" ")),
				newTokenDefault(1, 1, []byte("a")),
				newTokenDefault(3, 3, []byte(" ")),
				newTokenDefault(1, 1, []byte("aaa")),
				newTokenDefault(3, 3, []byte(" ")),
				newTokenDefault(2, 2, []byte("abcd")),
				newTokenDefault(3, 3, []byte(" ")),
				newTokenDefault(2, 2, []byte("abcdcdcd")),
				newTokenDefault(3, 3, []byte(" ")),
				newTokenDefault(2, 2, []byte("cd")),
				newTokenDefault(3, 3, []byte(" ")),
				newTokenDefault(2, 2, []byte("cdcdcd")),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("t1", "."),
				},
			},
			src: string([]byte{
				0x00,
				0x7f,
				0xc2, 0x80,
				0xdf, 0xbf,
				0xe1, 0x80, 0x80,
				0xec, 0xbf, 0xbf,
				0xed, 0x80, 0x80,
				0xed, 0x9f, 0xbf,
				0xee, 0x80, 0x80,
				0xef, 0xbf, 0xbf,
				0xf0, 0x90, 0x80, 0x80,
				0xf0, 0xbf, 0xbf, 0xbf,
				0xf1, 0x80, 0x80, 0x80,
				0xf3, 0xbf, 0xbf, 0xbf,
				0xf4, 0x80, 0x80, 0x80,
				0xf4, 0x8f, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0x00}),
				newTokenDefault(1, 1, []byte{0x7f}),
				newTokenDefault(1, 1, []byte{0xc2, 0x80}),
				newTokenDefault(1, 1, []byte{0xdf, 0xbf}),
				newTokenDefault(1, 1, []byte{0xe1, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xec, 0xbf, 0xbf}),
				newTokenDefault(1, 1, []byte{0xed, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xed, 0x9f, 0xbf}),
				newTokenDefault(1, 1, []byte{0xee, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xef, 0xbf, 0xbf}),
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xf0, 0xbf, 0xbf, 0xbf}),
				newTokenDefault(1, 1, []byte{0xf1, 0x80, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xf3, 0xbf, 0xbf, 0xbf}),
				newTokenDefault(1, 1, []byte{0xf4, 0x80, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xf4, 0x8f, 0xbf, 0xbf}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("t1", "[ab.*+?|()[\\]]"),
				},
			},
			src: "ab.*+?|()[]",
			tokens: []*Token{
				newTokenDefault(1, 1, []byte("a")),
				newTokenDefault(1, 1, []byte("b")),
				newTokenDefault(1, 1, []byte(".")),
				newTokenDefault(1, 1, []byte("*")),
				newTokenDefault(1, 1, []byte("+")),
				newTokenDefault(1, 1, []byte("?")),
				newTokenDefault(1, 1, []byte("|")),
				newTokenDefault(1, 1, []byte("(")),
				newTokenDefault(1, 1, []byte(")")),
				newTokenDefault(1, 1, []byte("[")),
				newTokenDefault(1, 1, []byte("]")),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					// all 1 byte characters except null character (U+0000)
					//
					// NOTE:
					// vartan cannot handle the null character in patterns because lexical.lexer,
					// specifically read() and restore(), recognizes the null characters as that a symbol doesn't exist.
					// If a pattern needs a null character, use code point expression \u{0000}.
					newLexEntryDefaultNOP("char_1_byte", "[\x01-\x7f]"),
				},
			},
			src: string([]byte{
				0x01,
				0x02,
				0x7e,
				0x7f,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0x01}),
				newTokenDefault(1, 1, []byte{0x02}),
				newTokenDefault(1, 1, []byte{0x7e}),
				newTokenDefault(1, 1, []byte{0x7f}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					// all 2 byte characters
					newLexEntryDefaultNOP("char_2_byte", "[\xc2\x80-\xdf\xbf]"),
				},
			},
			src: string([]byte{
				0xc2, 0x80,
				0xc2, 0x81,
				0xdf, 0xbe,
				0xdf, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0xc2, 0x80}),
				newTokenDefault(1, 1, []byte{0xc2, 0x81}),
				newTokenDefault(1, 1, []byte{0xdf, 0xbe}),
				newTokenDefault(1, 1, []byte{0xdf, 0xbf}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					// All bytes are the same.
					newLexEntryDefaultNOP("char_3_byte", "[\xe0\xa0\x80-\xe0\xa0\x80]"),
				},
			},
			src: string([]byte{
				0xe0, 0xa0, 0x80,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0xe0, 0xa0, 0x80}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					// The first two bytes are the same.
					newLexEntryDefaultNOP("char_3_byte", "[\xe0\xa0\x80-\xe0\xa0\xbf]"),
				},
			},
			src: string([]byte{
				0xe0, 0xa0, 0x80,
				0xe0, 0xa0, 0x81,
				0xe0, 0xa0, 0xbe,
				0xe0, 0xa0, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0xe0, 0xa0, 0x80}),
				newTokenDefault(1, 1, []byte{0xe0, 0xa0, 0x81}),
				newTokenDefault(1, 1, []byte{0xe0, 0xa0, 0xbe}),
				newTokenDefault(1, 1, []byte{0xe0, 0xa0, 0xbf}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					// The first byte are the same.
					newLexEntryDefaultNOP("char_3_byte", "[\xe0\xa0\x80-\xe0\xbf\xbf]"),
				},
			},
			src: string([]byte{
				0xe0, 0xa0, 0x80,
				0xe0, 0xa0, 0x81,
				0xe0, 0xbf, 0xbe,
				0xe0, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0xe0, 0xa0, 0x80}),
				newTokenDefault(1, 1, []byte{0xe0, 0xa0, 0x81}),
				newTokenDefault(1, 1, []byte{0xe0, 0xbf, 0xbe}),
				newTokenDefault(1, 1, []byte{0xe0, 0xbf, 0xbf}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					// all 3 byte characters
					newLexEntryDefaultNOP("char_3_byte", "[\xe0\xa0\x80-\xef\xbf\xbf]"),
				},
			},
			src: string([]byte{
				0xe0, 0xa0, 0x80,
				0xe0, 0xa0, 0x81,
				0xe0, 0xbf, 0xbe,
				0xe0, 0xbf, 0xbf,
				0xe1, 0x80, 0x80,
				0xe1, 0x80, 0x81,
				0xec, 0xbf, 0xbe,
				0xec, 0xbf, 0xbf,
				0xed, 0x80, 0x80,
				0xed, 0x80, 0x81,
				0xed, 0x9f, 0xbe,
				0xed, 0x9f, 0xbf,
				0xee, 0x80, 0x80,
				0xee, 0x80, 0x81,
				0xef, 0xbf, 0xbe,
				0xef, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0xe0, 0xa0, 0x80}),
				newTokenDefault(1, 1, []byte{0xe0, 0xa0, 0x81}),
				newTokenDefault(1, 1, []byte{0xe0, 0xbf, 0xbe}),
				newTokenDefault(1, 1, []byte{0xe0, 0xbf, 0xbf}),
				newTokenDefault(1, 1, []byte{0xe1, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xe1, 0x80, 0x81}),
				newTokenDefault(1, 1, []byte{0xec, 0xbf, 0xbe}),
				newTokenDefault(1, 1, []byte{0xec, 0xbf, 0xbf}),
				newTokenDefault(1, 1, []byte{0xed, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xed, 0x80, 0x81}),
				newTokenDefault(1, 1, []byte{0xed, 0x9f, 0xbe}),
				newTokenDefault(1, 1, []byte{0xed, 0x9f, 0xbf}),
				newTokenDefault(1, 1, []byte{0xee, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xee, 0x80, 0x81}),
				newTokenDefault(1, 1, []byte{0xef, 0xbf, 0xbe}),
				newTokenDefault(1, 1, []byte{0xef, 0xbf, 0xbf}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					// All bytes are the same.
					newLexEntryDefaultNOP("char_4_byte", "[\xf0\x90\x80\x80-\xf0\x90\x80\x80]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0x80, 0x80}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					// The first 3 bytes are the same.
					newLexEntryDefaultNOP("char_4_byte", "[\xf0\x90\x80\x80-\xf0\x90\x80\xbf]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
				0xf0, 0x90, 0x80, 0x81,
				0xf0, 0x90, 0x80, 0xbe,
				0xf0, 0x90, 0x80, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0x80, 0x81}),
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0x80, 0xbe}),
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0x80, 0xbf}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					// The first 2 bytes are the same.
					newLexEntryDefaultNOP("char_4_byte", "[\xf0\x90\x80\x80-\xf0\x90\xbf\xbf]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
				0xf0, 0x90, 0x80, 0x81,
				0xf0, 0x90, 0xbf, 0xbe,
				0xf0, 0x90, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0x80, 0x81}),
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0xbf, 0xbe}),
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0xbf, 0xbf}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					// The first byte are the same.
					newLexEntryDefaultNOP("char_4_byte", "[\xf0\x90\x80\x80-\xf0\xbf\xbf\xbf]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
				0xf0, 0x90, 0x80, 0x81,
				0xf0, 0xbf, 0xbf, 0xbe,
				0xf0, 0xbf, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0x80, 0x81}),
				newTokenDefault(1, 1, []byte{0xf0, 0xbf, 0xbf, 0xbe}),
				newTokenDefault(1, 1, []byte{0xf0, 0xbf, 0xbf, 0xbf}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					// all 4 byte characters
					newLexEntryDefaultNOP("char_4_byte", "[\xf0\x90\x80\x80-\xf4\x8f\xbf\xbf]"),
				},
			},
			src: string([]byte{
				0xf0, 0x90, 0x80, 0x80,
				0xf0, 0x90, 0x80, 0x81,
				0xf0, 0xbf, 0xbf, 0xbe,
				0xf0, 0xbf, 0xbf, 0xbf,
				0xf1, 0x80, 0x80, 0x80,
				0xf1, 0x80, 0x80, 0x81,
				0xf3, 0xbf, 0xbf, 0xbe,
				0xf3, 0xbf, 0xbf, 0xbf,
				0xf4, 0x80, 0x80, 0x80,
				0xf4, 0x80, 0x80, 0x81,
				0xf4, 0x8f, 0xbf, 0xbe,
				0xf4, 0x8f, 0xbf, 0xbf,
			}),
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xf0, 0x90, 0x80, 0x81}),
				newTokenDefault(1, 1, []byte{0xf0, 0xbf, 0xbf, 0xbe}),
				newTokenDefault(1, 1, []byte{0xf0, 0xbf, 0xbf, 0xbf}),
				newTokenDefault(1, 1, []byte{0xf1, 0x80, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xf1, 0x80, 0x80, 0x81}),
				newTokenDefault(1, 1, []byte{0xf3, 0xbf, 0xbf, 0xbe}),
				newTokenDefault(1, 1, []byte{0xf3, 0xbf, 0xbf, 0xbf}),
				newTokenDefault(1, 1, []byte{0xf4, 0x80, 0x80, 0x80}),
				newTokenDefault(1, 1, []byte{0xf4, 0x80, 0x80, 0x81}),
				newTokenDefault(1, 1, []byte{0xf4, 0x8f, 0xbf, 0xbe}),
				newTokenDefault(1, 1, []byte{0xf4, 0x8f, 0xbf, 0xbf}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("non_number", "[^0-9]+[0-9]"),
				},
			},
			src: "foo9",
			tokens: []*Token{
				newTokenDefault(1, 1, []byte("foo9")),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("char_1_byte", "\\u{006E}"),
					newLexEntryDefaultNOP("char_2_byte", "\\u{03BD}"),
					newLexEntryDefaultNOP("char_3_byte", "\\u{306B}"),
					newLexEntryDefaultNOP("char_4_byte", "\\u{01F638}"),
				},
			},
			src: "nνに😸",
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0x6E}),
				newTokenDefault(2, 2, []byte{0xCE, 0xBD}),
				newTokenDefault(3, 3, []byte{0xE3, 0x81, 0xAB}),
				newTokenDefault(4, 4, []byte{0xF0, 0x9F, 0x98, 0xB8}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("code_points_alt", "[\\u{006E}\\u{03BD}\\u{306B}\\u{01F638}]"),
				},
			},
			src: "nνに😸",
			tokens: []*Token{
				newTokenDefault(1, 1, []byte{0x6E}),
				newTokenDefault(1, 1, []byte{0xCE, 0xBD}),
				newTokenDefault(1, 1, []byte{0xE3, 0x81, 0xAB}),
				newTokenDefault(1, 1, []byte{0xF0, 0x9F, 0x98, 0xB8}),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("t1", "\\f{a2c}\\f{d2f}+"),
					newLexEntryFragment("a2c", "abc"),
					newLexEntryFragment("d2f", "def"),
				},
			},
			src: "abcdefdefabcdef",
			tokens: []*Token{
				newTokenDefault(1, 1, []byte("abcdefdef")),
				newTokenDefault(1, 1, []byte("abcdef")),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("t1", "(\\f{a2c}|\\f{d2f})+"),
					newLexEntryFragment("a2c", "abc"),
					newLexEntryFragment("d2f", "def"),
				},
			},
			src: "abcdefdefabc",
			tokens: []*Token{
				newTokenDefault(1, 1, []byte("abcdefdefabc")),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("t1", "\\f{a2c_or_d2f}+"),
					newLexEntryFragment("a2c_or_d2f", "\\f{a2c}|\\f{d2f}"),
					newLexEntryFragment("a2c", "abc"),
					newLexEntryFragment("d2f", "def"),
				},
			},
			src: "abcdefdefabc",
			tokens: []*Token{
				newTokenDefault(1, 1, []byte("abcdefdefabc")),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("white_space", ` *`),
					newLexEntry([]string{"default"}, "string_open", `"`, "string", false),
					newLexEntry([]string{"string"}, "escape_sequence", `\\[n"\\]`, "", false),
					newLexEntry([]string{"string"}, "char_sequence", `[^"\\]*`, "", false),
					newLexEntry([]string{"string"}, "string_close", `"`, "", true),
				},
			},
			src: `"" "Hello world.\n\"Hello world.\""`,
			tokens: []*Token{
				newToken(1, 2, 2, []byte(`"`)),
				newToken(2, 5, 3, []byte(`"`)),
				newToken(1, 1, 1, []byte(` `)),
				newToken(1, 2, 2, []byte(`"`)),
				newToken(2, 4, 2, []byte(`Hello world.`)),
				newToken(2, 3, 1, []byte(`\n`)),
				newToken(2, 3, 1, []byte(`\"`)),
				newToken(2, 4, 2, []byte(`Hello world.`)),
				newToken(2, 3, 1, []byte(`\"`)),
				newToken(2, 5, 3, []byte(`"`)),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					// `white_space` is enabled in multiple modes.
					newLexEntry([]string{"default", "state_a", "state_b"}, "white_space", ` *`, "", false),
					newLexEntry([]string{"default"}, "char_a", `a`, "state_a", false),
					newLexEntry([]string{"state_a"}, "char_b", `b`, "state_b", false),
					newLexEntry([]string{"state_a"}, "back_from_a", `<`, "", true),
					newLexEntry([]string{"state_b"}, "back_from_b", `<`, "", true),
				},
			},
			src: ` a b < < `,
			tokens: []*Token{
				newToken(1, 1, 1, []byte(` `)),
				newToken(1, 2, 2, []byte(`a`)),
				newToken(2, 1, 1, []byte(` `)),
				newToken(2, 3, 2, []byte(`b`)),
				newToken(3, 1, 1, []byte(` `)),
				newToken(3, 5, 2, []byte(`<`)),
				newToken(2, 1, 1, []byte(` `)),
				newToken(2, 4, 3, []byte(`<`)),
				newToken(1, 1, 1, []byte(` `)),
				newEOFTokenDefault(),
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntry([]string{"default", "mode_1", "mode_2"}, "white_space", ` *`, "", false),
					newLexEntry([]string{"default"}, "char", `.`, "", false),
					newLexEntry([]string{"default"}, "push_1", `-> 1`, "", false),
					newLexEntry([]string{"mode_1"}, "push_2", `-> 2`, "", false),
					newLexEntry([]string{"mode_1"}, "pop_1", `<-`, "", false),
					newLexEntry([]string{"mode_2"}, "pop_2", `<-`, "", false),
				},
			},
			src: `-> 1 -> 2 <- <- a`,
			tokens: []*Token{
				newToken(1, 3, 3, []byte(`-> 1`)),
				newToken(2, 1, 1, []byte(` `)),
				newToken(2, 4, 2, []byte(`-> 2`)),
				newToken(3, 1, 1, []byte(` `)),
				newToken(3, 6, 2, []byte(`<-`)),
				newToken(2, 1, 1, []byte(` `)),
				newToken(2, 5, 3, []byte(`<-`)),
				newToken(1, 1, 1, []byte(` `)),
				newToken(1, 2, 2, []byte(`a`)),
				newEOFTokenDefault(),
			},
			passiveModeTran: true,
			tran: func(l *Lexer, tok *Token) error {
				switch l.spec.ModeName(l.Mode()) {
				case "default":
					switch tok.KindID {
					case 3: // push_1
						l.PushMode(2)
					}
				case "mode_1":
					switch tok.KindID {
					case 4: // push_2
						l.PushMode(3)
					case 5: // pop_1
						return l.PopMode()
					}
				case "mode_2":
					switch tok.KindID {
					case 6: // pop_2
						return l.PopMode()
					}
				}
				return nil
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntry([]string{"default", "mode_1", "mode_2"}, "white_space", ` *`, "", false),
					newLexEntry([]string{"default"}, "char", `.`, "", false),
					newLexEntry([]string{"default"}, "push_1", `-> 1`, "mode_1", false),
					newLexEntry([]string{"mode_1"}, "push_2", `-> 2`, "", false),
					newLexEntry([]string{"mode_1"}, "pop_1", `<-`, "", false),
					newLexEntry([]string{"mode_2"}, "pop_2", `<-`, "", true),
				},
			},
			src: `-> 1 -> 2 <- <- a`,
			tokens: []*Token{
				newToken(1, 3, 3, []byte(`-> 1`)),
				newToken(2, 1, 1, []byte(` `)),
				newToken(2, 4, 2, []byte(`-> 2`)),
				newToken(3, 1, 1, []byte(` `)),
				newToken(3, 6, 2, []byte(`<-`)),
				newToken(2, 1, 1, []byte(` `)),
				newToken(2, 5, 3, []byte(`<-`)),
				newToken(1, 1, 1, []byte(` `)),
				newToken(1, 2, 2, []byte(`a`)),
				newEOFTokenDefault(),
			},
			// Active mode transition and an external transition function can be used together.
			passiveModeTran: false,
			tran: func(l *Lexer, tok *Token) error {
				switch l.spec.ModeName(l.Mode()) {
				case "mode_1":
					switch tok.KindID {
					case 4: // push_2
						l.PushMode(3)
					case 5: // pop_1
						return l.PopMode()
					}
				}
				return nil
			},
		},
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("dot", spec.EscapePattern(`.`)),
					newLexEntryDefaultNOP("star", spec.EscapePattern(`*`)),
					newLexEntryDefaultNOP("plus", spec.EscapePattern(`+`)),
					newLexEntryDefaultNOP("question", spec.EscapePattern(`?`)),
					newLexEntryDefaultNOP("vbar", spec.EscapePattern(`|`)),
					newLexEntryDefaultNOP("lparen", spec.EscapePattern(`(`)),
					newLexEntryDefaultNOP("rparen", spec.EscapePattern(`)`)),
					newLexEntryDefaultNOP("lbrace", spec.EscapePattern(`[`)),
					newLexEntryDefaultNOP("backslash", spec.EscapePattern(`\`)),
				},
			},
			src: `.*+?|()[\`,
			tokens: []*Token{
				newTokenDefault(1, 1, []byte(`.`)),
				newTokenDefault(2, 2, []byte(`*`)),
				newTokenDefault(3, 3, []byte(`+`)),
				newTokenDefault(4, 4, []byte(`?`)),
				newTokenDefault(5, 5, []byte(`|`)),
				newTokenDefault(6, 6, []byte(`(`)),
				newTokenDefault(7, 7, []byte(`)`)),
				newTokenDefault(8, 8, []byte(`[`)),
				newTokenDefault(9, 9, []byte(`\`)),
				newEOFTokenDefault(),
			},
		},
		// Character properties are available in a bracket expression.
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("letter", `[\p{Letter}]+`),
					newLexEntryDefaultNOP("non_letter", `[^\p{Letter}]+`),
				},
			},
			src: `foo123`,
			tokens: []*Token{
				newTokenDefault(1, 1, []byte(`foo`)),
				newTokenDefault(2, 2, []byte(`123`)),
				newEOFTokenDefault(),
			},
		},
		// The driver can continue lexical analysis even after it detects an invalid token.
		{
			lspec: &lexical.LexSpec{
				Entries: []*lexical.LexEntry{
					newLexEntryDefaultNOP("lower", `[a-z]+`),
				},
			},
			src: `foo123bar`,
			tokens: []*Token{
				newTokenDefault(1, 1, []byte(`foo`)),
				newInvalidTokenDefault([]byte(`123`)),
				newTokenDefault(1, 1, []byte(`bar`)),
				newEOFTokenDefault(),
			},
		},
	}
	for i, tt := range test {
		for compLv := lexical.CompressionLevelMin; compLv <= lexical.CompressionLevelMax; compLv++ {
			t.Run(fmt.Sprintf("#%v-%v", i, compLv), func(t *testing.T) {
				clspec, err, cerrs := lexical.Compile(tt.lspec, compLv)
				if err != nil {
					for _, cerr := range cerrs {
						t.Logf("%#v", cerr)
					}
					t.Fatalf("unexpected error: %v", err)
				}
				opts := []LexerOption{}
				if tt.passiveModeTran {
					opts = append(opts, DisableModeTransition())
				}
				lexer, err := NewLexer(NewLexSpec(clspec), strings.NewReader(tt.src), opts...)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				for _, eTok := range tt.tokens {
					tok, err := lexer.Next()
					if err != nil {
						t.Log(err)
						break
					}
					testToken(t, eTok, tok, false)

					if tok.EOF {
						break
					}

					if tt.tran != nil {
						err := tt.tran(lexer, tok)
						if err != nil {
							t.Fatalf("unexpected error: %v", err)
						}
					}
				}
			})
		}
	}
}

func TestLexer_Next_WithPosition(t *testing.T) {
	lspec := &lexical.LexSpec{
		Entries: []*lexical.LexEntry{
			newLexEntryDefaultNOP("newline", `\u{000A}+`),
			newLexEntryDefaultNOP("any", `.`),
		},
	}

	clspec, err, _ := lexical.Compile(lspec, lexical.CompressionLevelMax)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	src := string([]byte{
		0x00,
		0x7F,
		0x0A,

		0xC2, 0x80,
		0xDF, 0xBF,
		0x0A,

		0xE0, 0xA0, 0x80,
		0xE0, 0xBF, 0xBF,
		0xE1, 0x80, 0x80,
		0xEC, 0xBF, 0xBF,
		0xED, 0x80, 0x80,
		0xED, 0x9F, 0xBF,
		0xEE, 0x80, 0x80,
		0xEF, 0xBF, 0xBF,
		0x0A,

		0xF0, 0x90, 0x80, 0x80,
		0xF0, 0xBF, 0xBF, 0xBF,
		0xF1, 0x80, 0x80, 0x80,
		0xF3, 0xBF, 0xBF, 0xBF,
		0xF4, 0x80, 0x80, 0x80,
		0xF4, 0x8F, 0xBF, 0xBF,
		0x0A,
		0x0A,
		0x0A,
	})

	expected := []*Token{
		withPos(newTokenDefault(2, 2, []byte{0x00}), 0, 0),
		withPos(newTokenDefault(2, 2, []byte{0x7F}), 0, 1),
		withPos(newTokenDefault(1, 1, []byte{0x0A}), 0, 2),

		withPos(newTokenDefault(2, 2, []byte{0xC2, 0x80}), 1, 0),
		withPos(newTokenDefault(2, 2, []byte{0xDF, 0xBF}), 1, 1),
		withPos(newTokenDefault(1, 1, []byte{0x0A}), 1, 2),

		withPos(newTokenDefault(2, 2, []byte{0xE0, 0xA0, 0x80}), 2, 0),
		withPos(newTokenDefault(2, 2, []byte{0xE0, 0xBF, 0xBF}), 2, 1),
		withPos(newTokenDefault(2, 2, []byte{0xE1, 0x80, 0x80}), 2, 2),
		withPos(newTokenDefault(2, 2, []byte{0xEC, 0xBF, 0xBF}), 2, 3),
		withPos(newTokenDefault(2, 2, []byte{0xED, 0x80, 0x80}), 2, 4),
		withPos(newTokenDefault(2, 2, []byte{0xED, 0x9F, 0xBF}), 2, 5),
		withPos(newTokenDefault(2, 2, []byte{0xEE, 0x80, 0x80}), 2, 6),
		withPos(newTokenDefault(2, 2, []byte{0xEF, 0xBF, 0xBF}), 2, 7),
		withPos(newTokenDefault(1, 1, []byte{0x0A}), 2, 8),

		withPos(newTokenDefault(2, 2, []byte{0xF0, 0x90, 0x80, 0x80}), 3, 0),
		withPos(newTokenDefault(2, 2, []byte{0xF0, 0xBF, 0xBF, 0xBF}), 3, 1),
		withPos(newTokenDefault(2, 2, []byte{0xF1, 0x80, 0x80, 0x80}), 3, 2),
		withPos(newTokenDefault(2, 2, []byte{0xF3, 0xBF, 0xBF, 0xBF}), 3, 3),
		withPos(newTokenDefault(2, 2, []byte{0xF4, 0x80, 0x80, 0x80}), 3, 4),
		withPos(newTokenDefault(2, 2, []byte{0xF4, 0x8F, 0xBF, 0xBF}), 3, 5),

		// When a token contains multiple line breaks, the driver sets the token position to
		// the line number where a lexeme first appears.
		withPos(newTokenDefault(1, 1, []byte{0x0A, 0x0A, 0x0A}), 3, 6),

		withPos(newEOFTokenDefault(), 0, 0),
	}

	lexer, err := NewLexer(NewLexSpec(clspec), strings.NewReader(src))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, eTok := range expected {
		tok, err := lexer.Next()
		if err != nil {
			t.Fatal(err)
		}

		testToken(t, eTok, tok, true)

		if tok.EOF {
			break
		}
	}
}

func testToken(t *testing.T, expected, actual *Token, checkPosition bool) {
	t.Helper()

	if actual.ModeID != expected.ModeID ||
		actual.KindID != expected.KindID ||
		actual.ModeKindID != expected.ModeKindID ||
		!bytes.Equal(actual.Lexeme, expected.Lexeme) ||
		actual.EOF != expected.EOF ||
		actual.Invalid != expected.Invalid {
		t.Fatalf(`unexpected token; want: %v ("%#v"), got: %v ("%#v")`, expected, string(expected.Lexeme), actual, string(actual.Lexeme))
	}

	if checkPosition {
		if actual.Row != expected.Row || actual.Col != expected.Col {
			t.Fatalf(`unexpected token; want: %v ("%#v"), got: %v ("%#v")`, expected, string(expected.Lexeme), actual, string(actual.Lexeme))
		}
	}
}
