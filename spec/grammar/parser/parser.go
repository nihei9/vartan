package parser

import (
	"fmt"
	"io"

	verr "github.com/nihei9/vartan/error"
	spec "github.com/nihei9/vartan/spec/grammar"
)

type RootNode struct {
	Directives     []*DirectiveNode
	Productions    []*ProductionNode
	LexProductions []*ProductionNode
	Fragments      []*FragmentNode
}

type ProductionNode struct {
	Directives []*DirectiveNode
	LHS        string
	RHS        []*AlternativeNode
	Pos        Position
}

func (n *ProductionNode) isLexical() bool {
	if len(n.RHS) == 1 && len(n.RHS[0].Elements) == 1 && n.RHS[0].Elements[0].Pattern != "" {
		return true
	}
	return false
}

type AlternativeNode struct {
	Elements   []*ElementNode
	Directives []*DirectiveNode
	Pos        Position
}

type ElementNode struct {
	ID        string
	Pattern   string
	Label     *LabelNode
	Literally bool
	Pos       Position
}

type LabelNode struct {
	Name string
	Pos  Position
}

type DirectiveNode struct {
	Name       string
	Parameters []*ParameterNode
	Pos        Position
}

type ParameterNode struct {
	ID            string
	Pattern       string
	String        string
	OrderedSymbol string
	Group         []*DirectiveNode
	Expansion     bool
	Pos           Position
}

type FragmentNode struct {
	LHS string
	RHS string
	Pos Position
}

func raiseSyntaxError(row int, synErr *SyntaxError) {
	panic(&verr.SpecError{
		Cause: synErr,
		Row:   row,
	})
}

func raiseSyntaxErrorWithDetail(row int, synErr *SyntaxError, detail string) {
	panic(&verr.SpecError{
		Cause:  synErr,
		Detail: detail,
		Row:    row,
	})
}

func Parse(src io.Reader) (*RootNode, error) {
	p, err := newParser(src)
	if err != nil {
		return nil, err
	}

	return p.parse()
}

type parser struct {
	lex       *lexer
	peekedTok *token
	lastTok   *token
	errs      verr.SpecErrors

	// A token position that the parser read at last.
	// It is used as additional information in error messages.
	pos Position
}

func newParser(src io.Reader) (*parser, error) {
	lex, err := newLexer(src)
	if err != nil {
		return nil, err
	}
	return &parser{
		lex: lex,
	}, nil
}

func (p *parser) parse() (root *RootNode, retErr error) {
	root = p.parseRoot()
	if len(p.errs) > 0 {
		return nil, p.errs
	}

	return root, nil
}

func (p *parser) parseRoot() *RootNode {
	defer func() {
		err := recover()
		if err != nil {
			specErr, ok := err.(*verr.SpecError)
			if !ok {
				panic(fmt.Errorf("an unexpected error occurred: %v", err))
			}
			p.errs = append(p.errs, specErr)
		}
	}()

	var dirs []*DirectiveNode
	var prods []*ProductionNode
	var lexProds []*ProductionNode
	var fragments []*FragmentNode
	for {
		dir := p.parseTopLevelDirective()
		if dir != nil {
			dirs = append(dirs, dir)
			continue
		}

		fragment := p.parseFragment()
		if fragment != nil {
			fragments = append(fragments, fragment)
			continue
		}

		prod := p.parseProduction()
		if prod != nil {
			if prod.isLexical() {
				lexProds = append(lexProds, prod)
			} else {
				prods = append(prods, prod)
			}
			continue
		}

		if p.consume(tokenKindEOF) {
			break
		}
	}

	return &RootNode{
		Directives:     dirs,
		Productions:    prods,
		LexProductions: lexProds,
		Fragments:      fragments,
	}
}

func (p *parser) parseTopLevelDirective() *DirectiveNode {
	defer func() {
		err := recover()
		if err == nil {
			return
		}

		specErr, ok := err.(*verr.SpecError)
		if !ok {
			panic(err)
		}

		p.errs = append(p.errs, specErr)
		p.skipOverTo(tokenKindSemicolon)
	}()

	dir := p.parseDirective()
	if dir == nil {
		return nil
	}

	p.consume(tokenKindNewline)

	if !p.consume(tokenKindSemicolon) {
		raiseSyntaxError(p.pos.Row, synErrTopLevelDirNoSemicolon)
	}

	return dir
}

func (p *parser) parseFragment() *FragmentNode {
	defer func() {
		err := recover()
		if err == nil {
			return
		}

		specErr, ok := err.(*verr.SpecError)
		if !ok {
			panic(err)
		}

		p.errs = append(p.errs, specErr)
		p.skipOverTo(tokenKindSemicolon)
	}()

	p.consume(tokenKindNewline)

	if !p.consume(tokenKindKWFragment) {
		return nil
	}

	p.consume(tokenKindNewline)

	if !p.consume(tokenKindID) {
		raiseSyntaxError(p.pos.Row, synErrNoProductionName)
	}
	lhs := p.lastTok.text
	lhsPos := p.lastTok.pos

	p.consume(tokenKindNewline)

	if !p.consume(tokenKindColon) {
		raiseSyntaxError(p.pos.Row, synErrNoColon)
	}

	var rhs string
	switch {
	case p.consume(tokenKindTerminalPattern):
		rhs = p.lastTok.text
	case p.consume(tokenKindStringLiteral):
		rhs = spec.EscapePattern(p.lastTok.text)
	default:
		raiseSyntaxError(p.pos.Row, synErrFragmentNoPattern)
	}

	p.consume(tokenKindNewline)

	if !p.consume(tokenKindSemicolon) {
		raiseSyntaxError(p.pos.Row, synErrNoSemicolon)
	}

	if !p.consume(tokenKindNewline) {
		if !p.consume(tokenKindEOF) {
			raiseSyntaxError(p.pos.Row, synErrSemicolonNoNewline)
		}
	}

	return &FragmentNode{
		LHS: lhs,
		RHS: rhs,
		Pos: lhsPos,
	}
}

func (p *parser) parseProduction() *ProductionNode {
	defer func() {
		err := recover()
		if err == nil {
			return
		}

		specErr, ok := err.(*verr.SpecError)
		if !ok {
			panic(err)
		}

		p.errs = append(p.errs, specErr)
		p.skipOverTo(tokenKindSemicolon)
	}()

	p.consume(tokenKindNewline)

	if p.consume(tokenKindEOF) {
		return nil
	}

	if !p.consume(tokenKindID) {
		raiseSyntaxError(p.pos.Row, synErrNoProductionName)
	}
	lhs := p.lastTok.text
	lhsPos := p.lastTok.pos

	var dirs []*DirectiveNode
	for {
		dir := p.parseDirective()
		if dir == nil {
			break
		}
		dirs = append(dirs, dir)
	}

	p.consume(tokenKindNewline)

	if !p.consume(tokenKindColon) {
		raiseSyntaxError(p.pos.Row, synErrNoColon)
	}

	alt := p.parseAlternative()
	rhs := []*AlternativeNode{alt}
	for {
		p.consume(tokenKindNewline)

		if !p.consume(tokenKindOr) {
			break
		}
		alt := p.parseAlternative()
		rhs = append(rhs, alt)
	}

	p.consume(tokenKindNewline)

	if !p.consume(tokenKindSemicolon) {
		raiseSyntaxError(p.pos.Row, synErrNoSemicolon)
	}

	if !p.consume(tokenKindNewline) {
		if !p.consume(tokenKindEOF) {
			raiseSyntaxError(p.pos.Row, synErrSemicolonNoNewline)
		}
	}

	prod := &ProductionNode{
		Directives: dirs,
		LHS:        lhs,
		RHS:        rhs,
		Pos:        lhsPos,
	}

	// Vartan's driver must provide a user with the names of expected tokens when a syntax error occurs.
	// However, if a pattern appears directly in an alternative, Vartan's compiler cannot assign an appropriate
	// name to the pattern. Therefore, this code prohibits alternatives from containing patterns.
	if !prod.isLexical() {
		for _, alt := range prod.RHS {
			for _, elem := range alt.Elements {
				if elem.Pattern != "" {
					raiseSyntaxError(elem.Pos.Row, synErrPatternInAlt)
				}
			}
		}
	}

	return prod
}

func (p *parser) parseAlternative() *AlternativeNode {
	elems := []*ElementNode{}
	for {
		elem := p.parseElement()
		if elem == nil {
			break
		}
		elems = append(elems, elem)
	}

	// When a length of an alternative is zero, we cannot set a position.
	var firstElemPos Position
	if len(elems) > 0 {
		firstElemPos = elems[0].Pos
	}

	var dirs []*DirectiveNode
	for {
		dir := p.parseDirective()
		if dir == nil {
			break
		}
		dirs = append(dirs, dir)
	}

	return &AlternativeNode{
		Elements:   elems,
		Directives: dirs,
		Pos:        firstElemPos,
	}
}

func (p *parser) parseElement() *ElementNode {
	var elem *ElementNode
	switch {
	case p.consume(tokenKindID):
		elem = &ElementNode{
			ID:  p.lastTok.text,
			Pos: p.lastTok.pos,
		}
	case p.consume(tokenKindTerminalPattern):
		elem = &ElementNode{
			Pattern: p.lastTok.text,
			Pos:     p.lastTok.pos,
		}
	case p.consume(tokenKindStringLiteral):
		elem = &ElementNode{
			Pattern:   p.lastTok.text,
			Literally: true,
			Pos:       p.lastTok.pos,
		}
	default:
		if p.consume(tokenKindLabelMarker) {
			raiseSyntaxError(p.pos.Row, synErrLabelWithNoSymbol)
		}
		return nil
	}
	if p.consume(tokenKindLabelMarker) {
		if !p.consume(tokenKindID) {
			raiseSyntaxError(p.pos.Row, synErrNoLabel)
		}
		elem.Label = &LabelNode{
			Name: p.lastTok.text,
			Pos:  p.lastTok.pos,
		}
	}
	return elem
}

func (p *parser) parseDirective() *DirectiveNode {
	p.consume(tokenKindNewline)

	if !p.consume(tokenKindDirectiveMarker) {
		return nil
	}
	dirPos := p.lastTok.pos

	if !p.consume(tokenKindID) {
		raiseSyntaxError(p.pos.Row, synErrNoDirectiveName)
	}
	name := p.lastTok.text

	var params []*ParameterNode
	for {
		param := p.parseParameter()
		if param == nil {
			break
		}
		params = append(params, param)
	}

	return &DirectiveNode{
		Name:       name,
		Parameters: params,
		Pos:        dirPos,
	}
}

func (p *parser) parseParameter() *ParameterNode {
	var param *ParameterNode
	switch {
	case p.consume(tokenKindID):
		param = &ParameterNode{
			ID:  p.lastTok.text,
			Pos: p.lastTok.pos,
		}
	case p.consume(tokenKindTerminalPattern):
		param = &ParameterNode{
			Pattern: p.lastTok.text,
			Pos:     p.lastTok.pos,
		}
	case p.consume(tokenKindStringLiteral):
		param = &ParameterNode{
			String: p.lastTok.text,
			Pos:    p.lastTok.pos,
		}
	case p.consume(tokenKindOrderedSymbolMarker):
		if !p.consume(tokenKindID) {
			raiseSyntaxError(p.pos.Row, synErrNoOrderedSymbolName)
		}
		param = &ParameterNode{
			OrderedSymbol: p.lastTok.text,
			Pos:           p.lastTok.pos,
		}
	case p.consume(tokenKindLParen):
		pos := p.lastTok.pos
		var g []*DirectiveNode
		for {
			dir := p.parseDirective()
			if dir == nil {
				break
			}
			g = append(g, dir)
		}
		if !p.consume(tokenKindRParen) {
			raiseSyntaxError(p.pos.Row, synErrUnclosedDirGroup)
		}
		if len(g) == 0 {
			// Set an empty slice representing an empty directive group to distinguish between the following two cases.
			//
			// - #prec (); // vartan allows this case.
			// - #prec;    // This case will raise an error.
			g = []*DirectiveNode{}
		}
		param = &ParameterNode{
			Group: g,
			Pos:   pos,
		}
	}
	if p.consume(tokenKindExpantion) {
		switch {
		case param == nil:
			raiseSyntaxError(p.pos.Row, synErrStrayExpOp)
		case param.ID == "":
			raiseSyntaxError(p.pos.Row, synErrInvalidExpOperand)
		}
		param.Expansion = true
	}
	return param
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
			panic(err)
		}
	}
	p.pos = tok.pos
	if tok.kind == tokenKindInvalid {
		raiseSyntaxErrorWithDetail(p.pos.Row, synErrInvalidToken, tok.text)
	}
	if tok.kind == expected {
		p.lastTok = tok
		return true
	}
	p.peekedTok = tok

	return false
}

func (p *parser) skip() {
	var tok *token
	var err error
	for {
		if p.peekedTok != nil {
			tok = p.peekedTok
			p.peekedTok = nil
		} else {
			tok, err = p.lex.next()
			if err != nil {
				p.errs = append(p.errs, &verr.SpecError{
					Cause: err,
					Row:   p.pos.Row,
				})
				continue
			}
		}

		break
	}

	p.lastTok = tok
	p.pos = tok.pos
}

func (p *parser) skipOverTo(kind tokenKind) {
	for {
		if p.consume(kind) || p.consume(tokenKindEOF) {
			return
		}
		p.skip()
	}
}
