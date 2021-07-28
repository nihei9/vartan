package spec

import (
	"fmt"
	"io"

	verr "github.com/nihei9/vartan/error"
)

type RootNode struct {
	Productions    []*ProductionNode
	LexProductions []*ProductionNode
	Fragments      []*FragmentNode
}

type ProductionNode struct {
	Directive *DirectiveNode
	LHS       string
	RHS       []*AlternativeNode
	Pos       Position
}

func (n *ProductionNode) isLexical() bool {
	if len(n.RHS) == 1 && len(n.RHS[0].Elements) == 1 && n.RHS[0].Elements[0].Pattern != "" {
		return true
	}
	return false
}

type AlternativeNode struct {
	Elements  []*ElementNode
	Directive *DirectiveNode
	Pos       Position
}

type ElementNode struct {
	ID      string
	Pattern string
	Pos     Position
}

type DirectiveNode struct {
	Name       string
	Parameters []*ParameterNode
	Pos        Position
}

type ParameterNode struct {
	ID   string
	Tree *TreeStructNode
	Pos  Position
}

type TreeStructNode struct {
	Name     string
	Children []*TreeChildNode
	Pos      Position
}

type TreeChildNode struct {
	Position  int
	Expansion bool
	Pos       Position
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

	var prods []*ProductionNode
	var lexProds []*ProductionNode
	var fragments []*FragmentNode
	for {
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
		Productions:    prods,
		LexProductions: lexProds,
		Fragments:      fragments,
	}
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

		return
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

	if !p.consume(tokenKindTerminalPattern) {
		raiseSyntaxError(p.pos.Row, synErrFragmentNoPattern)
	}
	rhs := p.lastTok.text

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

		return
	}()

	p.consume(tokenKindNewline)

	if p.consume(tokenKindEOF) {
		return nil
	}

	dir := p.parseDirective()
	if dir != nil {
		if !p.consume(tokenKindNewline) {
			raiseSyntaxError(p.pos.Row, synErrProdDirNoNewline)
		}
	}

	if !p.consume(tokenKindID) {
		raiseSyntaxError(p.pos.Row, synErrNoProductionName)
	}
	lhs := p.lastTok.text
	lhsPos := p.lastTok.pos

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

	return &ProductionNode{
		Directive: dir,
		LHS:       lhs,
		RHS:       rhs,
		Pos:       lhsPos,
	}
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

	dir := p.parseDirective()

	return &AlternativeNode{
		Elements:  elems,
		Directive: dir,
		Pos:       firstElemPos,
	}
}

func (p *parser) parseElement() *ElementNode {
	switch {
	case p.consume(tokenKindID):
		return &ElementNode{
			ID:  p.lastTok.text,
			Pos: p.lastTok.pos,
		}
	case p.consume(tokenKindTerminalPattern):
		return &ElementNode{
			Pattern: p.lastTok.text,
			Pos:     p.lastTok.pos,
		}
	}
	return nil
}

func (p *parser) parseDirective() *DirectiveNode {
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
	switch {
	case p.consume(tokenKindID):
		return &ParameterNode{
			ID:  p.lastTok.text,
			Pos: p.lastTok.pos,
		}
	case p.consume(tokenKindTreeNodeOpen):
		if !p.consume(tokenKindID) {
			raiseSyntaxError(p.pos.Row, synErrTreeInvalidFirstElem)
		}
		name := p.lastTok.text
		namePos := p.lastTok.pos

		var children []*TreeChildNode
		for {
			if !p.consume(tokenKindPosition) {
				break
			}

			child := &TreeChildNode{
				Position: p.lastTok.num,
				Pos:      p.lastTok.pos,
			}
			if p.consume(tokenKindExpantion) {
				child.Expansion = true
			}

			children = append(children, child)
		}

		if !p.consume(tokenKindTreeNodeClose) {
			raiseSyntaxError(p.pos.Row, synErrTreeUnclosed)
		}

		return &ParameterNode{
			Tree: &TreeStructNode{
				Name:     name,
				Children: children,
				Pos:      namePos,
			},
			Pos: namePos,
		}
	}

	return nil
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
