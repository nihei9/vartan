package spec

import (
	"io"
)

type RootNode struct {
	Productions []*ProductionNode
}

type ProductionNode struct {
	LHS string
	RHS []*AlternativeNode
}

type AlternativeNode struct {
	Elements []*ElementNode
}

type ElementNode struct {
	ID      string
	Pattern string
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
	prod := p.parseProduction()
	if prod == nil {
		raiseSyntaxError(synErrNoProduction)
	}
	root := &RootNode{
		Productions: []*ProductionNode{prod},
	}
	for {
		prod := p.parseProduction()
		if prod == nil {
			break
		}
		root.Productions = append(root.Productions, prod)
	}
	return root
}

func (p *parser) parseProduction() *ProductionNode {
	if p.consume(tokenKindEOF) {
		return nil
	}
	if !p.consume(tokenKindID) {
		raiseSyntaxError(synErrNoProductionName)
	}
	lhs := p.lastTok.text
	if !p.consume(tokenKindColon) {
		raiseSyntaxError(synErrNoColon)
	}
	alt := p.parseAlternative()
	rhs := []*AlternativeNode{alt}
	for {
		if !p.consume(tokenKindOr) {
			break
		}
		alt := p.parseAlternative()
		rhs = append(rhs, alt)
	}
	if !p.consume(tokenKindSemicolon) {
		raiseSyntaxError(synErrNoSemicolon)
	}
	return &ProductionNode{
		LHS: lhs,
		RHS: rhs,
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
	return &AlternativeNode{
		Elements: elems,
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
