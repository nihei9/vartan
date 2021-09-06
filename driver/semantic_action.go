package driver

import (
	"fmt"
	"io"

	mldriver "github.com/nihei9/maleeni/driver"
	"github.com/nihei9/vartan/spec"
)

type SemanticActionSet interface {
	// Shift runs when the driver shifts a symbol onto the state stack. `tok` is a token corresponding to
	// the symbol. When the driver recovered from an error state by shifting the token, `recovered` is true.
	Shift(tok *mldriver.Token, recovered bool)

	// Reduce runs when the driver reduces an RHS of a production to its LHS. `prodNum` is a number of
	// the production. When the driver recovered from an error state by reducing the production,
	// `recovered` is true.
	Reduce(prodNum int, recovered bool)

	// Accept runs when the driver accepts an input.
	Accept()

	// TrapAndShiftError runs when the driver traps a syntax error and shifts a error symbol onto the state stack.
	// `n` is the number of frames that the driver discards from the state stack.
	// Unlike `Shift` function, this function doesn't take a token as an argument because a token corresponding to
	// the error symbol doesn't exist.
	TrapAndShiftError(n int)

	// MissError runs when the driver fails to trap a syntax error.
	MissError()
}

var _ SemanticActionSet = &SyntaxTreeActionSet{}

type Node struct {
	KindName string
	Text     string
	Row      int
	Col      int
	Children []*Node
}

func PrintTree(w io.Writer, node *Node) {
	printTree(w, node, "", "")
}

func printTree(w io.Writer, node *Node, ruledLine string, childRuledLinePrefix string) {
	if node == nil {
		return
	}

	if node.Text != "" {
		fmt.Fprintf(w, "%v%v %#v\n", ruledLine, node.KindName, node.Text)
	} else {
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
	gram     *spec.CompiledGrammar
	makeAST  bool
	makeCST  bool
	ast      *Node
	cst      *Node
	semStack *semanticStack
}

func NewSyntaxTreeActionSet(gram *spec.CompiledGrammar, makeAST bool, makeCST bool) *SyntaxTreeActionSet {
	return &SyntaxTreeActionSet{
		gram:     gram,
		makeAST:  makeAST,
		makeCST:  makeCST,
		semStack: newSemanticStack(),
	}
}

func (a *SyntaxTreeActionSet) Shift(tok *mldriver.Token, recovered bool) {
	term := a.tokenToTerminal(tok)

	var ast *Node
	var cst *Node
	if a.makeAST {
		ast = &Node{
			KindName: a.gram.ParsingTable.Terminals[term],
			Text:     tok.Text(),
			Row:      tok.Row,
			Col:      tok.Col,
		}
	}
	if a.makeCST {
		cst = &Node{
			KindName: a.gram.ParsingTable.Terminals[term],
			Text:     tok.Text(),
			Row:      tok.Row,
			Col:      tok.Col,
		}
	}

	a.semStack.push(&semanticFrame{
		cst: cst,
		ast: ast,
	})
}

func (a *SyntaxTreeActionSet) Reduce(prodNum int, recovered bool) {
	lhs := a.gram.ParsingTable.LHSSymbols[prodNum]

	// When an alternative is empty, `n` will be 0, and `handle` will be empty slice.
	n := a.gram.ParsingTable.AlternativeSymbolCounts[prodNum]
	handle := a.semStack.pop(n)

	var ast *Node
	var cst *Node
	if a.makeAST {
		act := a.gram.ASTAction.Entries[prodNum]
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
			KindName: a.gram.ParsingTable.NonTerminals[lhs],
			Children: children,
		}
	}
	if a.makeCST {
		children := make([]*Node, len(handle))
		for i, f := range handle {
			children[i] = f.cst
		}

		cst = &Node{
			KindName: a.gram.ParsingTable.NonTerminals[lhs],
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

func (a *SyntaxTreeActionSet) TrapAndShiftError(n int) {
	a.semStack.pop(n)

	errSym := a.gram.ParsingTable.ErrorSymbol

	var ast *Node
	var cst *Node
	if a.makeAST {
		ast = &Node{
			KindName: a.gram.ParsingTable.Terminals[errSym],
		}
	}
	if a.makeCST {
		cst = &Node{
			KindName: a.gram.ParsingTable.Terminals[errSym],
		}
	}

	a.semStack.push(&semanticFrame{
		cst: cst,
		ast: ast,
	})
}

func (a *SyntaxTreeActionSet) MissError() {
}

func (a *SyntaxTreeActionSet) CST() *Node {
	return a.cst
}

func (a *SyntaxTreeActionSet) AST() *Node {
	return a.ast
}

func (a *SyntaxTreeActionSet) tokenToTerminal(tok *mldriver.Token) int {
	if tok.EOF {
		return a.gram.ParsingTable.EOFSymbol
	}

	return a.gram.LexicalSpecification.Maleeni.KindToTerminal[tok.KindID]
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
