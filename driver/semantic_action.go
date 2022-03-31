package driver

import (
	"fmt"
	"io"
)

type SemanticActionSet interface {
	// Shift runs when the driver shifts a symbol onto the state stack. `tok` is a token corresponding to
	// the symbol. When the driver recovered from an error state by shifting the token, `recovered` is true.
	Shift(tok VToken, recovered bool)

	// Reduce runs when the driver reduces an RHS of a production to its LHS. `prodNum` is a number of
	// the production. When the driver recovered from an error state by reducing the production,
	// `recovered` is true.
	Reduce(prodNum int, recovered bool)

	// Accept runs when the driver accepts an input.
	Accept()

	// TrapAndShiftError runs when the driver traps a syntax error and shifts a error symbol onto the state stack.
	// `cause` is a token that caused a syntax error. `popped` is the number of frames that the driver discards
	// from the state stack.
	// Unlike `Shift` function, this function doesn't take a token to be shifted as an argument because a token
	// corresponding to the error symbol doesn't exist.
	TrapAndShiftError(cause VToken, popped int)

	// MissError runs when the driver fails to trap a syntax error. `cause` is a token that caused a syntax error.
	MissError(cause VToken)
}

var _ SemanticActionSet = &SyntaxTreeActionSet{}

type Node struct {
	KindName string
	Text     string
	Row      int
	Col      int
	Children []*Node
	Error    bool
}

func PrintTree(w io.Writer, node *Node) {
	printTree(w, node, "", "")
}

func printTree(w io.Writer, node *Node, ruledLine string, childRuledLinePrefix string) {
	if node == nil {
		return
	}

	switch {
	case node.Error:
		fmt.Fprintf(w, "%v!%v\n", ruledLine, node.KindName)
	case node.Text != "":
		fmt.Fprintf(w, "%v%v %#v\n", ruledLine, node.KindName, node.Text)
	default:
		fmt.Fprintf(w, "%v%v\n", ruledLine, node.KindName)
	}

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

type SyntaxTreeActionSet struct {
	gram     Grammar
	makeAST  bool
	makeCST  bool
	ast      *Node
	cst      *Node
	semStack *semanticStack
}

func NewSyntaxTreeActionSet(gram Grammar, makeAST bool, makeCST bool) *SyntaxTreeActionSet {
	return &SyntaxTreeActionSet{
		gram:     gram,
		makeAST:  makeAST,
		makeCST:  makeCST,
		semStack: newSemanticStack(),
	}
}

func (a *SyntaxTreeActionSet) Shift(tok VToken, recovered bool) {
	term := a.tokenToTerminal(tok)

	var ast *Node
	var cst *Node
	if a.makeAST {
		row, col := tok.Position()
		ast = &Node{
			KindName: a.gram.Terminal(term),
			Text:     string(tok.Lexeme()),
			Row:      row,
			Col:      col,
		}
	}
	if a.makeCST {
		row, col := tok.Position()
		cst = &Node{
			KindName: a.gram.Terminal(term),
			Text:     string(tok.Lexeme()),
			Row:      row,
			Col:      col,
		}
	}

	a.semStack.push(&semanticFrame{
		cst: cst,
		ast: ast,
	})
}

func (a *SyntaxTreeActionSet) Reduce(prodNum int, recovered bool) {
	lhs := a.gram.LHS(prodNum)

	// When an alternative is empty, `n` will be 0, and `handle` will be empty slice.
	n := a.gram.AlternativeSymbolCount(prodNum)
	handle := a.semStack.pop(n)

	var ast *Node
	var cst *Node
	if a.makeAST {
		act := a.gram.ASTAction(prodNum)
		var children []*Node
		if act != nil {
			// Count the number of children in advance to avoid frequent growth in a slice for children.
			{
				l := 0
				for _, e := range act {
					if e > 0 {
						l++
					} else {
						offset := e*-1 - 1
						l += len(handle[offset].ast.Children)
					}
				}

				children = make([]*Node, l)
			}

			p := 0
			for _, e := range act {
				if e > 0 {
					offset := e - 1
					children[p] = handle[offset].ast
					p++
				} else {
					offset := e*-1 - 1
					for _, c := range handle[offset].ast.Children {
						children[p] = c
						p++
					}
				}
			}
		} else {
			// If an alternative has no AST action, a driver generates
			// a node with the same structure as a CST.

			children = make([]*Node, len(handle))
			for i, f := range handle {
				children[i] = f.ast
			}
		}

		ast = &Node{
			KindName: a.gram.NonTerminal(lhs),
			Children: children,
		}
	}
	if a.makeCST {
		children := make([]*Node, len(handle))
		for i, f := range handle {
			children[i] = f.cst
		}

		cst = &Node{
			KindName: a.gram.NonTerminal(lhs),
			Children: children,
		}
	}

	a.semStack.push(&semanticFrame{
		cst: cst,
		ast: ast,
	})

}

func (a *SyntaxTreeActionSet) Accept() {
	top := a.semStack.pop(1)
	a.cst = top[0].cst
	a.ast = top[0].ast
}

func (a *SyntaxTreeActionSet) TrapAndShiftError(cause VToken, popped int) {
	a.semStack.pop(popped)

	var ast *Node
	var cst *Node
	if a.makeAST {
		ast = &Node{
			KindName: a.gram.Terminal(a.gram.Error()),
			Error:    true,
		}
	}
	if a.makeCST {
		cst = &Node{
			KindName: a.gram.Terminal(a.gram.Error()),
			Error:    true,
		}
	}

	a.semStack.push(&semanticFrame{
		cst: cst,
		ast: ast,
	})
}

func (a *SyntaxTreeActionSet) MissError(cause VToken) {
}

func (a *SyntaxTreeActionSet) CST() *Node {
	return a.cst
}

func (a *SyntaxTreeActionSet) AST() *Node {
	return a.ast
}

func (a *SyntaxTreeActionSet) tokenToTerminal(tok VToken) int {
	if tok.EOF() {
		return a.gram.EOF()
	}

	return tok.TerminalID()
}

type semanticFrame struct {
	cst *Node
	ast *Node
}

type semanticStack struct {
	frames []*semanticFrame
}

func newSemanticStack() *semanticStack {
	return &semanticStack{}
}

func (s *semanticStack) push(f *semanticFrame) {
	s.frames = append(s.frames, f)
}

func (s *semanticStack) pop(n int) []*semanticFrame {
	fs := s.frames[len(s.frames)-n:]
	s.frames = s.frames[:len(s.frames)-n]

	return fs
}
