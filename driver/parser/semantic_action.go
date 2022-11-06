package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// SemanticActionSet is a set of semantic actions a parser calls.
type SemanticActionSet interface {
	// Shift runs when the parser shifts a symbol onto a state stack. `tok` is a token corresponding to the symbol.
	// When the parser recovered from an error state by shifting the token, `recovered` is true.
	Shift(tok VToken, recovered bool)

	// Reduce runs when the parser reduces an RHS of a production to its LHS. `prodNum` is a number of the production.
	// When the parser recovered from an error state by reducing the production, `recovered` is true.
	Reduce(prodNum int, recovered bool)

	// Accept runs when the parser accepts an input.
	Accept()

	// TrapAndShiftError runs when the parser traps a syntax error and shifts a error symbol onto the state stack.
	// `cause` is a token that caused a syntax error. `popped` is the number of frames that the parser discards
	// from the state stack.
	// Unlike `Shift` function, this function doesn't take a token to be shifted as an argument because a token
	// corresponding to the error symbol doesn't exist.
	TrapAndShiftError(cause VToken, popped int)

	// MissError runs when the parser fails to trap a syntax error. `cause` is a token that caused a syntax error.
	MissError(cause VToken)
}

var _ SemanticActionSet = &SyntaxTreeActionSet{}

// SyntaxTreeNode is a node of a syntax tree. A node type used in SyntaxTreeActionSet must implement SyntaxTreeNode interface.
type SyntaxTreeNode interface {
	// ChildCount returns a child count of a node. A parser calls this method to know the child count to be expanded by an `#ast`
	// directive with `...` operator.
	ChildCount() int

	// ExpandChildren returns children of a node. A parser calls this method to fetch the children to be expanded by an `#ast`
	// directive with `...` operator.
	ExpandChildren() []SyntaxTreeNode
}

var _ SyntaxTreeNode = &Node{}

// SyntaxTreeBuilder allows you to construct a syntax tree containing arbitrary user-defined node types.
// The parser uses SyntaxTreeBuilder interface as a part of semantic actions via SyntaxTreeActionSet interface.
type SyntaxTreeBuilder interface {
	Shift(kindName string, text string, row, col int) SyntaxTreeNode
	ShiftError(kindName string) SyntaxTreeNode
	Reduce(kindName string, children []SyntaxTreeNode) SyntaxTreeNode
	Accept(f SyntaxTreeNode)
}

var _ SyntaxTreeBuilder = &DefaulSyntaxTreeBuilder{}

// DefaulSyntaxTreeBuilder is a implementation of SyntaxTreeBuilder.
type DefaulSyntaxTreeBuilder struct {
	tree *Node
}

// NewDefaultSyntaxTreeBuilder returns a new DefaultSyntaxTreeBuilder.
func NewDefaultSyntaxTreeBuilder() *DefaulSyntaxTreeBuilder {
	return &DefaulSyntaxTreeBuilder{}
}

// Shift is a implementation of SyntaxTreeBuilder.Shift.
func (b *DefaulSyntaxTreeBuilder) Shift(kindName string, text string, row, col int) SyntaxTreeNode {
	return &Node{
		Type:     NodeTypeTerminal,
		KindName: kindName,
		Text:     text,
		Row:      row,
		Col:      col,
	}
}

// ShiftError is a implementation of SyntaxTreeBuilder.ShiftError.
func (b *DefaulSyntaxTreeBuilder) ShiftError(kindName string) SyntaxTreeNode {
	return &Node{
		Type:     NodeTypeError,
		KindName: kindName,
	}
}

// Reduce is a implementation of SyntaxTreeBuilder.Reduce.
func (b *DefaulSyntaxTreeBuilder) Reduce(kindName string, children []SyntaxTreeNode) SyntaxTreeNode {
	cNodes := make([]*Node, len(children))
	for i, c := range children {
		cNodes[i] = c.(*Node)
	}
	return &Node{
		Type:     NodeTypeNonTerminal,
		KindName: kindName,
		Children: cNodes,
	}
}

// Accept is a implementation of SyntaxTreeBuilder.Accept.
func (b *DefaulSyntaxTreeBuilder) Accept(f SyntaxTreeNode) {
	b.tree = f.(*Node)
}

// Tree returns a syntax tree when the parser has accepted an input. If a syntax error occurs, the return value is nil.
func (b *DefaulSyntaxTreeBuilder) Tree() *Node {
	return b.tree
}

// SyntaxTreeActionSet is a implementation of SemanticActionSet interface and constructs a syntax tree.
type SyntaxTreeActionSet struct {
	gram             Grammar
	builder          SyntaxTreeBuilder
	semStack         *semanticStack
	disableASTAction bool
}

// NewASTActionSet returns a new SyntaxTreeActionSet that constructs an AST (Abstract Syntax Tree).
// When grammar `gram` contains `#ast` directives, the new SyntaxTreeActionSet this function returns interprets them.
func NewASTActionSet(gram Grammar, builder SyntaxTreeBuilder) *SyntaxTreeActionSet {
	return &SyntaxTreeActionSet{
		gram:     gram,
		builder:  builder,
		semStack: newSemanticStack(),
	}
}

// NewCSTTActionSet returns a new SyntaxTreeActionSet that constructs a CST (Concrete Syntax Tree).
// Even if grammar `gram` contains `#ast` directives, the new SyntaxTreeActionSet this function returns ignores them.
func NewCSTActionSet(gram Grammar, builder SyntaxTreeBuilder) *SyntaxTreeActionSet {
	return &SyntaxTreeActionSet{
		gram:             gram,
		builder:          builder,
		semStack:         newSemanticStack(),
		disableASTAction: true,
	}
}

// Shift is a implementation of SemanticActionSet.Shift method.
func (a *SyntaxTreeActionSet) Shift(tok VToken, recovered bool) {
	term := a.tokenToTerminal(tok)
	row, col := tok.Position()
	a.semStack.push(a.builder.Shift(a.gram.Terminal(term), string(tok.Lexeme()), row, col))
}

// Reduce is a implementation of SemanticActionSet.Reduce method.
func (a *SyntaxTreeActionSet) Reduce(prodNum int, recovered bool) {
	lhs := a.gram.LHS(prodNum)

	// When an alternative is empty, `n` will be 0, and `handle` will be empty slice.
	n := a.gram.AlternativeSymbolCount(prodNum)
	handle := a.semStack.pop(n)

	var astAct []int
	if !a.disableASTAction {
		astAct = a.gram.ASTAction(prodNum)
	}
	var children []SyntaxTreeNode
	if astAct != nil {
		// Count the number of children in advance to avoid frequent growth in a slice for children.
		{
			l := 0
			for _, e := range astAct {
				if e > 0 {
					l++
				} else {
					offset := e*-1 - 1
					l += handle[offset].ChildCount()
				}
			}

			children = make([]SyntaxTreeNode, l)
		}

		p := 0
		for _, e := range astAct {
			if e > 0 {
				offset := e - 1
				children[p] = handle[offset]
				p++
			} else {
				offset := e*-1 - 1
				for _, c := range handle[offset].ExpandChildren() {
					children[p] = c
					p++
				}
			}
		}
	} else {
		// If an alternative has no AST action, a driver generates
		// a node with the same structure as a CST.
		children = handle
	}

	a.semStack.push(a.builder.Reduce(a.gram.NonTerminal(lhs), children))
}

// Accept is a implementation of SemanticActionSet.Accept method.
func (a *SyntaxTreeActionSet) Accept() {
	top := a.semStack.pop(1)
	a.builder.Accept(top[0])
}

// TrapAndShiftError is a implementation of SemanticActionSet.TrapAndShiftError method.
func (a *SyntaxTreeActionSet) TrapAndShiftError(cause VToken, popped int) {
	a.semStack.pop(popped)
	a.semStack.push(a.builder.ShiftError(a.gram.Terminal(a.gram.Error())))
}

// MissError is a implementation of SemanticActionSet.MissError method.
func (a *SyntaxTreeActionSet) MissError(cause VToken) {
}

func (a *SyntaxTreeActionSet) tokenToTerminal(tok VToken) int {
	if tok.EOF() {
		return a.gram.EOF()
	}

	return tok.TerminalID()
}

type semanticStack struct {
	frames []SyntaxTreeNode
}

func newSemanticStack() *semanticStack {
	return &semanticStack{
		frames: make([]SyntaxTreeNode, 0, 100),
	}
}

func (s *semanticStack) push(f SyntaxTreeNode) {
	s.frames = append(s.frames, f)
}

func (s *semanticStack) pop(n int) []SyntaxTreeNode {
	fs := s.frames[len(s.frames)-n:]
	s.frames = s.frames[:len(s.frames)-n]

	return fs
}

type NodeType int

const (
	NodeTypeError       = 0
	NodeTypeTerminal    = 1
	NodeTypeNonTerminal = 2
)

// Node is a implementation of SyntaxTreeNode interface.
type Node struct {
	Type     NodeType
	KindName string
	Text     string
	Row      int
	Col      int
	Children []*Node
}

func (n *Node) MarshalJSON() ([]byte, error) {
	switch n.Type {
	case NodeTypeError:
		return json.Marshal(struct {
			Type     NodeType `json:"type"`
			KindName string   `json:"kind_name"`
		}{
			Type:     n.Type,
			KindName: n.KindName,
		})
	case NodeTypeTerminal:
		if n.KindName == "" {
			return json.Marshal(struct {
				Type NodeType `json:"type"`
				Text string   `json:"text"`
				Row  int      `json:"row"`
				Col  int      `json:"col"`
			}{
				Type: n.Type,
				Text: n.Text,
				Row:  n.Row,
				Col:  n.Col,
			})
		}
		return json.Marshal(struct {
			Type     NodeType `json:"type"`
			KindName string   `json:"kind_name"`
			Text     string   `json:"text"`
			Row      int      `json:"row"`
			Col      int      `json:"col"`
		}{
			Type:     n.Type,
			KindName: n.KindName,
			Text:     n.Text,
			Row:      n.Row,
			Col:      n.Col,
		})
	case NodeTypeNonTerminal:
		return json.Marshal(struct {
			Type     NodeType `json:"type"`
			KindName string   `json:"kind_name"`
			Children []*Node  `json:"children"`
		}{
			Type:     n.Type,
			KindName: n.KindName,
			Children: n.Children,
		})
	default:
		return nil, fmt.Errorf("invalid node type: %v", n.Type)
	}
}

// ChildCount is a implementation of SyntaxTreeNode.ChildCount.
func (n *Node) ChildCount() int {
	return len(n.Children)
}

// ExpandChildren is a implementation of SyntaxTreeNode.ExpandChildren.
func (n *Node) ExpandChildren() []SyntaxTreeNode {
	fs := make([]SyntaxTreeNode, len(n.Children))
	for i, n := range n.Children {
		fs[i] = n
	}
	return fs
}

// PrintTree prints a syntax tree whose root is `node`.
func PrintTree(w io.Writer, node *Node) {
	printTree(w, node, "", "")
}

func printTree(w io.Writer, node *Node, ruledLine string, childRuledLinePrefix string) {
	if node == nil {
		return
	}

	switch node.Type {
	case NodeTypeError:
		fmt.Fprintf(w, "%v%v\n", ruledLine, node.KindName)
	case NodeTypeTerminal:
		fmt.Fprintf(w, "%v%v %v\n", ruledLine, node.KindName, strconv.Quote(node.Text))
	case NodeTypeNonTerminal:
		fmt.Fprintf(w, "%v%v\n", ruledLine, node.KindName)

		num := len(node.Children)
		for i, child := range node.Children {
			var line string
			if num > 1 && i < num-1 {
				line = "├─ "
			} else {
				line = "└─ "
			}

			var prefix string
			if i >= num-1 {
				prefix = "   "
			} else {
				prefix = "│  "
			}

			printTree(w, child, childRuledLinePrefix+line, childRuledLinePrefix+prefix)
		}
	}
}
