package parser

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	spec "github.com/nihei9/vartan/spec/grammar"
	"github.com/nihei9/vartan/ucd"
)

type PatternEntry struct {
	ID      spec.LexModeKindID
	Pattern []byte
}

type parser struct {
	kind      spec.LexKindName
	lex       *lexer
	peekedTok *token
	lastTok   *token

	// If and only if isContributoryPropertyExposed is true, the parser interprets contributory properties that
	// appear in property expressions.
	//
	// The contributory properties are not exposed, and users cannot use those properties because the parser
	// follows [UAX #44 5.13 Property APIs]. For instance, \p{Other_Alphabetic} is invalid.
	//
	// isContributoryPropertyExposed is set to true when the parser is generated recursively. The parser needs to
	// interpret derived properties internally because the derived properties consist of other properties that
	// may contain the contributory properties.
	//
	// [UAX #44 5.13 Property APIs] says:
	// > The following subtypes of Unicode character properties should generally not be exposed in APIs,
	// > except in limited circumstances. They may not be useful, particularly in public API collections,
	// > and may instead prove misleading to the users of such API collections.
	// >   * Contributory properties are not recommended for public APIs.
	// > ...
	// https://unicode.org/reports/tr44/#Property_APIs
	isContributoryPropertyExposed bool

	errCause  error
	errDetail string
}

func NewParser(kind spec.LexKindName, src io.Reader) *parser {
	return &parser{
		kind:                          kind,
		lex:                           newLexer(src),
		isContributoryPropertyExposed: false,
	}
}

func (p *parser) exposeContributoryProperty() {
	p.isContributoryPropertyExposed = true
}

func (p *parser) Error() (string, error) {
	return p.errDetail, p.errCause
}

func (p *parser) Parse() (root CPTree, retErr error) {
	defer func() {
		err := recover()
		if err != nil {
			var ok bool
			retErr, ok = err.(error)
			if !ok {
				panic(err)
			}
			return
		}
	}()

	return newRootNode(p.kind, p.parseRegexp()), nil
}

func (p *parser) parseRegexp() CPTree {
	alt := p.parseAlt()
	if alt == nil {
		if p.consume(tokenKindGroupClose) {
			p.raiseParseError(synErrGroupNoInitiator, "")
		}
		p.raiseParseError(synErrNullPattern, "")
	}
	if p.consume(tokenKindGroupClose) {
		p.raiseParseError(synErrGroupNoInitiator, "")
	}
	p.expect(tokenKindEOF)
	return alt
}

func (p *parser) parseAlt() CPTree {
	left := p.parseConcat()
	if left == nil {
		if p.consume(tokenKindAlt) {
			p.raiseParseError(synErrAltLackOfOperand, "")
		}
		return nil
	}
	for {
		if !p.consume(tokenKindAlt) {
			break
		}
		right := p.parseConcat()
		if right == nil {
			p.raiseParseError(synErrAltLackOfOperand, "")
		}
		left = newAltNode(left, right)
	}
	return left
}

func (p *parser) parseConcat() CPTree {
	left := p.parseRepeat()
	for {
		right := p.parseRepeat()
		if right == nil {
			break
		}
		left = newConcatNode(left, right)
	}
	return left
}

func (p *parser) parseRepeat() CPTree {
	group := p.parseGroup()
	if group == nil {
		if p.consume(tokenKindRepeat) {
			p.raiseParseError(synErrRepNoTarget, "* needs an operand")
		}
		if p.consume(tokenKindRepeatOneOrMore) {
			p.raiseParseError(synErrRepNoTarget, "+ needs an operand")
		}
		if p.consume(tokenKindOption) {
			p.raiseParseError(synErrRepNoTarget, "? needs an operand")
		}
		return nil
	}
	if p.consume(tokenKindRepeat) {
		return newRepeatNode(group)
	}
	if p.consume(tokenKindRepeatOneOrMore) {
		return newRepeatOneOrMoreNode(group)
	}
	if p.consume(tokenKindOption) {
		return newOptionNode(group)
	}
	return group
}

func (p *parser) parseGroup() CPTree {
	if p.consume(tokenKindGroupOpen) {
		alt := p.parseAlt()
		if alt == nil {
			if p.consume(tokenKindEOF) {
				p.raiseParseError(synErrGroupUnclosed, "")
			}
			p.raiseParseError(synErrGroupNoElem, "")
		}
		if p.consume(tokenKindEOF) {
			p.raiseParseError(synErrGroupUnclosed, "")
		}
		if !p.consume(tokenKindGroupClose) {
			p.raiseParseError(synErrGroupInvalidForm, "")
		}
		return alt
	}
	return p.parseSingleChar()
}

func (p *parser) parseSingleChar() CPTree {
	if p.consume(tokenKindAnyChar) {
		return genAnyCharAST()
	}
	if p.consume(tokenKindBExpOpen) {
		left := p.parseBExpElem()
		if left == nil {
			if p.consume(tokenKindEOF) {
				p.raiseParseError(synErrBExpUnclosed, "")
			}
			p.raiseParseError(synErrBExpNoElem, "")
		}
		for {
			right := p.parseBExpElem()
			if right == nil {
				break
			}
			left = newAltNode(left, right)
		}
		if p.consume(tokenKindEOF) {
			p.raiseParseError(synErrBExpUnclosed, "")
		}
		p.expect(tokenKindBExpClose)
		return left
	}
	if p.consume(tokenKindInverseBExpOpen) {
		elem := p.parseBExpElem()
		if elem == nil {
			if p.consume(tokenKindEOF) {
				p.raiseParseError(synErrBExpUnclosed, "")
			}
			p.raiseParseError(synErrBExpNoElem, "")
		}
		inverse := exclude(elem, genAnyCharAST())
		if inverse == nil {
			p.raiseParseError(synErrUnmatchablePattern, "")
		}
		for {
			elem := p.parseBExpElem()
			if elem == nil {
				break
			}
			inverse = exclude(elem, inverse)
			if inverse == nil {
				p.raiseParseError(synErrUnmatchablePattern, "")
			}
		}
		if p.consume(tokenKindEOF) {
			p.raiseParseError(synErrBExpUnclosed, "")
		}
		p.expect(tokenKindBExpClose)
		return inverse
	}
	if p.consume(tokenKindCodePointLeader) {
		return p.parseCodePoint()
	}
	if p.consume(tokenKindCharPropLeader) {
		return p.parseCharProp()
	}
	if p.consume(tokenKindFragmentLeader) {
		return p.parseFragment()
	}
	c := p.parseNormalChar()
	if c == nil {
		if p.consume(tokenKindBExpClose) {
			p.raiseParseError(synErrBExpInvalidForm, "")
		}
		return nil
	}
	return c
}

func (p *parser) parseBExpElem() CPTree {
	var left CPTree
	switch {
	case p.consume(tokenKindCodePointLeader):
		left = p.parseCodePoint()
	case p.consume(tokenKindCharPropLeader):
		left = p.parseCharProp()
		if p.consume(tokenKindCharRange) {
			p.raiseParseError(synErrRangePropIsUnavailable, "")
		}
	default:
		left = p.parseNormalChar()
	}
	if left == nil {
		return nil
	}
	if !p.consume(tokenKindCharRange) {
		return left
	}
	var right CPTree
	switch {
	case p.consume(tokenKindCodePointLeader):
		right = p.parseCodePoint()
	case p.consume(tokenKindCharPropLeader):
		p.raiseParseError(synErrRangePropIsUnavailable, "")
	default:
		right = p.parseNormalChar()
	}
	if right == nil {
		p.raiseParseError(synErrRangeInvalidForm, "")
	}
	from, _, _ := left.Range()
	_, to, _ := right.Range()
	if !isValidOrder(from, to) {
		p.raiseParseError(synErrRangeInvalidOrder, fmt.Sprintf("%X..%X", from, to))
	}
	return newRangeSymbolNode(from, to)
}

func (p *parser) parseCodePoint() CPTree {
	if !p.consume(tokenKindLBrace) {
		p.raiseParseError(synErrCPExpInvalidForm, "")
	}
	if !p.consume(tokenKindCodePoint) {
		p.raiseParseError(synErrCPExpInvalidForm, "")
	}

	n, err := strconv.ParseInt(p.lastTok.codePoint, 16, 64)
	if err != nil {
		panic(fmt.Errorf("failed to decode a code point (%v) into a int: %v", p.lastTok.codePoint, err))
	}
	if n < 0x0000 || n > 0x10FFFF {
		p.raiseParseError(synErrCPExpOutOfRange, "")
	}

	sym := newSymbolNode(rune(n))

	if !p.consume(tokenKindRBrace) {
		p.raiseParseError(synErrCPExpInvalidForm, "")
	}

	return sym
}

func (p *parser) parseCharProp() CPTree {
	if !p.consume(tokenKindLBrace) {
		p.raiseParseError(synErrCharPropExpInvalidForm, "")
	}
	var sym1, sym2 string
	if !p.consume(tokenKindCharPropSymbol) {
		p.raiseParseError(synErrCharPropExpInvalidForm, "")
	}
	sym1 = p.lastTok.propSymbol
	if p.consume(tokenKindEqual) {
		if !p.consume(tokenKindCharPropSymbol) {
			p.raiseParseError(synErrCharPropExpInvalidForm, "")
		}
		sym2 = p.lastTok.propSymbol
	}

	var alt CPTree
	var propName, propVal string
	if sym2 != "" {
		propName = sym1
		propVal = sym2
	} else {
		propName = ""
		propVal = sym1
	}
	if !p.isContributoryPropertyExposed && ucd.IsContributoryProperty(propName) {
		p.raiseParseError(synErrCharPropUnsupported, propName)
	}
	pat, err := ucd.NormalizeCharacterProperty(propName, propVal)
	if err != nil {
		p.raiseParseError(synErrCharPropUnsupported, err.Error())
	}
	if pat != "" {
		p := NewParser(p.kind, bytes.NewReader([]byte(pat)))
		p.exposeContributoryProperty()
		ast, err := p.Parse()
		if err != nil {
			panic(err)
		}
		alt = ast
	} else {
		cpRanges, inverse, err := ucd.FindCodePointRanges(propName, propVal)
		if err != nil {
			p.raiseParseError(synErrCharPropUnsupported, err.Error())
		}
		if inverse {
			r := cpRanges[0]
			alt = exclude(newRangeSymbolNode(r.From, r.To), genAnyCharAST())
			if alt == nil {
				p.raiseParseError(synErrUnmatchablePattern, "")
			}
			for _, r := range cpRanges[1:] {
				alt = exclude(newRangeSymbolNode(r.From, r.To), alt)
				if alt == nil {
					p.raiseParseError(synErrUnmatchablePattern, "")
				}
			}
		} else {
			for _, r := range cpRanges {
				alt = genAltNode(
					alt,
					newRangeSymbolNode(r.From, r.To),
				)
			}
		}
	}

	if !p.consume(tokenKindRBrace) {
		p.raiseParseError(synErrCharPropExpInvalidForm, "")
	}

	return alt
}

func (p *parser) parseFragment() CPTree {
	if !p.consume(tokenKindLBrace) {
		p.raiseParseError(synErrFragmentExpInvalidForm, "")
	}
	if !p.consume(tokenKindFragmentSymbol) {
		p.raiseParseError(synErrFragmentExpInvalidForm, "")
	}
	sym := p.lastTok.fragmentSymbol

	if !p.consume(tokenKindRBrace) {
		p.raiseParseError(synErrFragmentExpInvalidForm, "")
	}

	return newFragmentNode(spec.LexKindName(sym), nil)
}

func (p *parser) parseNormalChar() CPTree {
	if !p.consume(tokenKindChar) {
		return nil
	}
	return newSymbolNode(p.lastTok.char)
}

func exclude(symbol, base CPTree) CPTree {
	if left, right, ok := symbol.Alternatives(); ok {
		return exclude(right, exclude(left, base))
	}

	if left, right, ok := base.Alternatives(); ok {
		return genAltNode(
			exclude(symbol, left),
			exclude(symbol, right),
		)
	}

	if bFrom, bTo, ok := base.Range(); ok {
		sFrom, sTo, ok := symbol.Range()
		if !ok {
			panic(fmt.Errorf("invalid symbol tree: %T", symbol))
		}

		switch {
		case sFrom > bFrom && sTo < bTo:
			return genAltNode(
				newRangeSymbolNode(bFrom, sFrom-1),
				newRangeSymbolNode(sTo+1, bTo),
			)
		case sFrom <= bFrom && sTo >= bFrom && sTo < bTo:
			return newRangeSymbolNode(sTo+1, bTo)
		case sFrom > bFrom && sFrom <= bTo && sTo >= bTo:
			return newRangeSymbolNode(bFrom, sFrom-1)
		case sFrom <= bFrom && sTo >= bTo:
			return nil
		default:
			return base
		}
	}

	panic(fmt.Errorf("invalid base tree: %T", base))
}

func genAnyCharAST() CPTree {
	return newRangeSymbolNode(0x0, 0x10FFFF)
}

func isValidOrder(from, to rune) bool {
	return from <= to
}

func genConcatNode(cs ...CPTree) CPTree {
	nonNilNodes := []CPTree{}
	for _, c := range cs {
		if c == nil {
			continue
		}
		nonNilNodes = append(nonNilNodes, c)
	}
	if len(nonNilNodes) <= 0 {
		return nil
	}
	if len(nonNilNodes) == 1 {
		return nonNilNodes[0]
	}
	concat := newConcatNode(nonNilNodes[0], nonNilNodes[1])
	for _, c := range nonNilNodes[2:] {
		concat = newConcatNode(concat, c)
	}
	return concat
}

func genAltNode(cs ...CPTree) CPTree {
	nonNilNodes := []CPTree{}
	for _, c := range cs {
		if c == nil {
			continue
		}
		nonNilNodes = append(nonNilNodes, c)
	}
	if len(nonNilNodes) <= 0 {
		return nil
	}
	if len(nonNilNodes) == 1 {
		return nonNilNodes[0]
	}
	alt := newAltNode(nonNilNodes[0], nonNilNodes[1])
	for _, c := range nonNilNodes[2:] {
		alt = newAltNode(alt, c)
	}
	return alt
}

func (p *parser) expect(expected tokenKind) {
	if !p.consume(expected) {
		tok := p.peekedTok
		p.raiseParseError(synErrUnexpectedToken, fmt.Sprintf("expected: %v, actual: %v", expected, tok.kind))
	}
}

func (p *parser) consume(expected tokenKind) bool {
	var tok *token
	var err error
	if p.peekedTok != nil {
		tok = p.peekedTok
		p.peekedTok = nil
	} else {
		tok, err = p.lex.next()
		if err != nil {
			if err == ParseErr {
				detail, cause := p.lex.error()
				p.raiseParseError(cause, detail)
			}
			panic(err)
		}
	}
	p.lastTok = tok
	if tok.kind == expected {
		return true
	}
	p.peekedTok = tok
	p.lastTok = nil

	return false
}

func (p *parser) raiseParseError(err error, detail string) {
	p.errCause = err
	p.errDetail = detail
	panic(ParseErr)
}
