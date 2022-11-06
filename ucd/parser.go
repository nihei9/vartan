package ucd

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"io"
	"regexp"
	"strings"
)

type CodePointRange struct {
	From rune
	To   rune
}

var codePointRangeNil = &CodePointRange{
	From: 0,
	To:   0,
}

type field string

func (f field) codePointRange() (*CodePointRange, error) {
	var from, to rune
	var err error
	cp := reCodePointRange.FindStringSubmatch(string(f))
	from, err = decodeHexToRune(cp[1])
	if err != nil {
		return codePointRangeNil, err
	}
	if cp[2] != "" {
		to, err = decodeHexToRune(cp[2])
		if err != nil {
			return codePointRangeNil, err
		}
	} else {
		to = from
	}
	return &CodePointRange{
		From: from,
		To:   to,
	}, nil
}

func decodeHexToRune(hexCodePoint string) (rune, error) {
	h := hexCodePoint
	if len(h)%2 != 0 {
		h = "0" + h
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return 0, err
	}
	l := len(b)
	for i := 0; i < 4-l; i++ {
		b = append([]byte{0}, b...)
	}
	n := binary.BigEndian.Uint32(b)
	return rune(n), nil
}

func (f field) symbol() string {
	return string(f)
}

func (f field) normalizedSymbol() string {
	return normalizeSymbolicValue(string(f))
}

var symValReplacer = strings.NewReplacer("_", "", "-", "", "\x20", "")

// normalizeSymbolicValue normalizes a symbolic value. The normalized value meets UAX44-LM3.
//
// https://www.unicode.org/reports/tr44/#UAX44-LM3
func normalizeSymbolicValue(s string) string {
	v := strings.ToLower(symValReplacer.Replace(s))
	if strings.HasPrefix(v, "is") && v != "is" {
		return v[2:]
	}
	return v
}

var (
	reLine           = regexp.MustCompile(`^\s*(.*?)\s*(#.*)?$`)
	reCodePointRange = regexp.MustCompile(`^([[:xdigit:]]+)(?:..([[:xdigit:]]+))?$`)

	specialCommentPrefix = "# @missing:"
)

// This parser can parse data files of Unicode Character Database (UCD).
// Specifically, it has the following two functions:
// - Converts each line of the data files into a slice of fields.
// - Recognizes specially-formatted comments starting `@missing` and generates a slice of fields.
//
// However, for practical purposes, each field needs to be analyzed more specifically.
// For instance, in UnicodeData.txt, the first field represents a range of code points,
// so it needs to be recognized as a hexadecimal string.
// You can perform more specific parsing for each file by implementing a dedicated parser that wraps this parser.
//
// https://www.unicode.org/reports/tr44/#Format_Conventions
type parser struct {
	scanner       *bufio.Scanner
	fields        []field
	defaultFields []field
	err           error

	fieldBuf        []field
	defaultFieldBuf []field
}

func newParser(r io.Reader) *parser {
	return &parser{
		scanner:         bufio.NewScanner(r),
		fieldBuf:        make([]field, 50),
		defaultFieldBuf: make([]field, 50),
	}
}

func (p *parser) parse() bool {
	for p.scanner.Scan() {
		p.parseRecord(p.scanner.Text())
		if p.fields != nil || p.defaultFields != nil {
			return true
		}
	}
	p.err = p.scanner.Err()
	return false
}

func (p *parser) parseRecord(src string) {
	ms := reLine.FindStringSubmatch(src)
	mFields := ms[1]
	mComment := ms[2]
	if mFields != "" {
		p.fields = parseFields(p.fieldBuf, mFields)
	} else {
		p.fields = nil
	}
	if strings.HasPrefix(mComment, specialCommentPrefix) {
		p.defaultFields = parseFields(p.defaultFieldBuf, strings.Replace(mComment, specialCommentPrefix, "", -1))
	} else {
		p.defaultFields = nil
	}
}

func parseFields(buf []field, src string) []field {
	n := 0
	for _, f := range strings.Split(src, ";") {
		buf[n] = field(strings.TrimSpace(f))
		n++
	}

	return buf[:n]
}
