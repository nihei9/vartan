package utf8

import (
	"fmt"
	"strings"
)

type CharBlock struct {
	From []byte
	To   []byte
}

func (b *CharBlock) String() string {
	var s strings.Builder
	fmt.Fprint(&s, "<")
	fmt.Fprintf(&s, "%X", b.From[0])
	for i := 1; i < len(b.From); i++ {
		fmt.Fprintf(&s, " %X", b.From[i])
	}
	fmt.Fprint(&s, "..")
	fmt.Fprintf(&s, "%X", b.To[0])
	for i := 1; i < len(b.To); i++ {
		fmt.Fprintf(&s, " %X", b.To[i])
	}
	fmt.Fprint(&s, ">")
	return s.String()
}

func GenCharBlocks(from, to rune) ([]*CharBlock, error) {
	rs, err := splitCodePoint(from, to)
	if err != nil {
		return nil, err
	}

	blks := make([]*CharBlock, len(rs))
	for i, r := range rs {
		blks[i] = &CharBlock{
			From: []byte(string(r.from)),
			To:   []byte(string(r.to)),
		}
	}

	return blks, nil
}

type cpRange struct {
	from rune
	to   rune
}

// splitCodePoint splits a code point range represented by <from..to> into some blocks. The code points that
// the block contains will be a continuous byte sequence when encoded into UTF-8. For instance, this function
// splits <U+0000..U+07FF> into <U+0000..U+007F> and <U+0080..U+07FF> because <U+0000..U+07FF> is continuous on
// the code point but non-continuous in the UTF-8 byte sequence (In UTF-8, <U+0000..U+007F> is encoded <00..7F>,
// and <U+0080..U+07FF> is encoded <C2 80..DF BF>).
//
// The blocks don't contain surrogate code points <U+D800..U+DFFF> because byte sequences encoding them are
// ill-formed in UTF-8. For instance, <U+D000..U+FFFF> is split into <U+D000..U+D7FF> and <U+E000..U+FFFF>.
// However, when `from` or `to` itself is the surrogate code point, this function returns an error.
func splitCodePoint(from, to rune) ([]*cpRange, error) {
	if from > to {
		return nil, fmt.Errorf("code point range must be from <= to: U+%X..U+%X", from, to)
	}
	if from < 0x0000 || from > 0x10ffff || to < 0x0000 || to > 0x10ffff {
		return nil, fmt.Errorf("code point must be >=U+0000 and <=U+10FFFF: U+%X..U+%X", from, to)
	}
	// https://www.unicode.org/versions/Unicode13.0.0/ch03.pdf > 3.9 Unicode Encoding Forms > UTF-8 D92
	// > Because surrogate code points are not Unicode scalar values, any UTF-8 byte sequence that would otherwise
	// > map to code points U+D800..U+DFFF is ill-formed.
	if from >= 0xd800 && from <= 0xdfff || to >= 0xd800 && to <= 0xdfff {
		return nil, fmt.Errorf("surrogate code points U+D800..U+DFFF are not allowed in UTF-8: U+%X..U+%X", from, to)
	}

	in := &cpRange{
		from: from,
		to:   to,
	}
	var rs []*cpRange
	for in.from <= in.to {
		r := &cpRange{
			from: in.from,
			to:   in.to,
		}
		// https://www.unicode.org/versions/Unicode13.0.0/ch03.pdf > 3.9 Unicode Encoding Forms > UTF-8 Table 3-7.  Well-Formed UTF-8 Byte Sequences
		switch {
		case in.from <= 0x007f && in.to > 0x007f:
			r.to = 0x007f
		case in.from <= 0x07ff && in.to > 0x07ff:
			r.to = 0x07ff
		case in.from <= 0x0fff && in.to > 0x0fff:
			r.to = 0x0fff
		case in.from <= 0xcfff && in.to > 0xcfff:
			r.to = 0xcfff
		case in.from <= 0xd7ff && in.to > 0xd7ff:
			r.to = 0xd7ff
		case in.from <= 0xffff && in.to > 0xffff:
			r.to = 0xffff
		case in.from <= 0x3ffff && in.to > 0x3ffff:
			r.to = 0x3ffff
		case in.from <= 0xfffff && in.to > 0xfffff:
			r.to = 0xfffff
		}
		rs = append(rs, r)
		in.from = r.to + 1

		// Skip surrogate code points U+D800..U+DFFF.
		if in.from >= 0xd800 && in.from <= 0xdfff {
			in.from = 0xe000
		}
	}
	return rs, nil
}
