package spec

import (
	"io"
)

type RootNode struct {
	Productions []*ProductionNode
	Fragments   []*FragmentNode
}

type ProductionNode struct {
	Directive *DirectiveNode
	LHS       string
	RHS       []*AlternativeNode
}

type AlternativeNode struct {
	Elements  []*ElementNode
	Directive *DirectiveNode
}

type ElementNode struct {
	ID      string
	Pattern string
}

type DirectiveNode struct {
	Name      string
	Parameter *ParameterNode
}

type ParameterNode struct {
	ID   string
	Tree *TreeStructNode
}

type TreeStructNode struct {
	Name     string
	Children []*TreeChildNode
}

type TreeChildNode struct {
	Position  int
	Expansion bool
}

type FragmentNode struct {
	LHS string
	RHS string
}

func raiseSyntaxError(synErr *SyntaxError) {
	panic(synErr)
}

func Parse(src io.Reader) (*RootNode, error) {
	p, err := newParser(src)
	if err != nil {
		return nil, err
	}
	root, err := p.parse()
	if err != nil {
		return nil, err
	}
	return root, nil
}

type parser struct {
	lex       *lexer
	peekedTok *token
	lastTok   *token
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
	defer func() {
		err := recover()
		if err != nil {
			retErr = err.(error)
			return
		}
	}()
	return p.parseRoot(), nil
}

func (p *parser) parseRoot() *RootNode {
	var prods []*ProductionNode
	var fragments []*FragmentNode
	for {
		p.consume(tokenKindNewline)

		fragment := p.parseFragment()
		if fragment != nil {
			fragments = append(fragments, fragment)
			continue
		}

		prod := p.parseProduction()
		if prod != nil {
			prods = append(prods, prod)
			continue
		}

		break
	}
	if len(prods) == 0 {
		raiseSyntaxError(synErrNoProduction)
	}

	return &RootNode{
		Productions: prods,
		Fragments:   fragments,
	}
}

func (p *parser) parseFragment() *FragmentNode {
	if !p.consume(tokenKindKWFragment) {
		return nil
	}

	p.consume(tokenKindNewline)

	if !p.consume(tokenKindID) {
		raiseSyntaxError(synErrNoProductionName)
	}
	lhs := p.lastTok.text

	p.consume(tokenKindNewline)

	if !p.consume(tokenKindColon) {
		raiseSyntaxError(synErrNoColon)
	}

	if !p.consume(tokenKindTerminalPattern) {
		raiseSyntaxError(synErrFragmentNoPattern)
	}
	rhs := p.lastTok.text

	p.consume(tokenKindNewline)

	if !p.consume(tokenKindSemicolon) {
		raiseSyntaxError(synErrNoSemicolon)
	}

	if !p.consume(tokenKindNewline) {
		if !p.consume(tokenKindEOF) {
			raiseSyntaxError(synErrSemicolonNoNewline)
		}
	}

	return &FragmentNode{
		LHS: lhs,
		RHS: rhs,
	}
}

func (p *parser) parseProduction() *ProductionNode {
	if p.consume(tokenKindEOF) {
		return nil
	}

	dir := p.parseDirective()
	if dir != nil {
		if !p.consume(tokenKindNewline) {
			raiseSyntaxError(synErrProdDirNoNewline)
		}
	}

	if !p.consume(tokenKindID) {
		raiseSyntaxError(synErrNoProductionName)
	}
	lhs := p.lastTok.text

	p.consume(tokenKindNewline)

	if !p.consume(tokenKindColon) {
		raiseSyntaxError(synErrNoColon)
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
		raiseSyntaxError(synErrNoSemicolon)
	}

	if !p.consume(tokenKindNewline) {
		if !p.consume(tokenKindEOF) {
			raiseSyntaxError(synErrSemicolonNoNewline)
		}
	}

	return &ProductionNode{
		Directive: dir,
		LHS:       lhs,
		RHS:       rhs,
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

	dir := p.parseDirective()

	return &AlternativeNode{
		Elements:  elems,
		Directive: dir,
	}
}

func (p *parser) parseElement() *ElementNode {
	switch {
	case p.consume(tokenKindID):
		return &ElementNode{
			ID: p.lastTok.text,
		}
	case p.consume(tokenKindTerminalPattern):
		return &ElementNode{
			Pattern: p.lastTok.text,
		}
	}
	return nil
}

func (p *parser) parseDirective() *DirectiveNode {
	if !p.consume(tokenKindDirectiveMarker) {
		return nil
	}

	if !p.consume(tokenKindID) {
		raiseSyntaxError(synErrNoDirectiveName)
	}
	name := p.lastTok.text

	var param *ParameterNode
	switch {
	case p.consume(tokenKindID):
		param = &ParameterNode{
			ID: p.lastTok.text,
		}
	case p.consume(tokenKindTreeNodeOpen):
		if !p.consume(tokenKindID) {
			raiseSyntaxError(synErrTreeInvalidFirstElem)
		}
		name := p.lastTok.text

		var children []*TreeChildNode
		for {
			if !p.consume(tokenKindPosition) {
				break
			}

			child := &TreeChildNode{
				Position: p.lastTok.num,
			}
			if p.consume(tokenKindExpantion) {
				child.Expansion = true
			}

			children = append(children, child)
		}

		if !p.consume(tokenKindTreeNodeClose) {
			raiseSyntaxError(synErrTreeUnclosed)
		}

		param = &ParameterNode{
			Tree: &TreeStructNode{
				Name:     name,
				Children: children,
			},
		}
	}

	return &DirectiveNode{
		Name:      name,
		Parameter: param,
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
			panic(err)
		}
	}
	p.lastTok = tok
	if tok.kind == tokenKindInvalid {
		raiseSyntaxError(synErrInvalidToken)
	}
	if tok.kind == expected {
		return true
	}
	p.peekedTok = tok
	p.lastTok = nil

	return false
}
