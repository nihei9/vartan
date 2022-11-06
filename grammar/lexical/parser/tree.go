package parser

import (
	"fmt"
	"io"
	"sort"

	spec "github.com/nihei9/vartan/spec/grammar"
)

type CPRange struct {
	From rune
	To   rune
}

type CPTree interface {
	fmt.Stringer
	Range() (rune, rune, bool)
	Optional() (CPTree, bool)
	Repeatable() (CPTree, bool)
	Concatenation() (CPTree, CPTree, bool)
	Alternatives() (CPTree, CPTree, bool)
	Describe() (spec.LexKindName, []spec.LexKindName, error)

	children() (CPTree, CPTree)
	clone() CPTree
}

var (
	_ CPTree = &rootNode{}
	_ CPTree = &symbolNode{}
	_ CPTree = &concatNode{}
	_ CPTree = &altNode{}
	_ CPTree = &quantifierNode{}
	_ CPTree = &fragmentNode{}
)

type rootNode struct {
	kind      spec.LexKindName
	tree      CPTree
	fragments map[spec.LexKindName][]*fragmentNode
}

func newRootNode(kind spec.LexKindName, t CPTree) *rootNode {
	fragments := map[spec.LexKindName][]*fragmentNode{}
	collectFragments(t, fragments)

	return &rootNode{
		kind:      kind,
		tree:      t,
		fragments: fragments,
	}
}

func collectFragments(n CPTree, fragments map[spec.LexKindName][]*fragmentNode) {
	if n == nil {
		return
	}

	if f, ok := n.(*fragmentNode); ok {
		fragments[f.kind] = append(fragments[f.kind], f)
		return
	}

	l, r := n.children()
	collectFragments(l, fragments)
	collectFragments(r, fragments)
}

func (n *rootNode) String() string {
	return fmt.Sprintf("root: %v: %v fragments", n.kind, len(n.fragments))
}

func (n *rootNode) Range() (rune, rune, bool) {
	return n.tree.Range()
}

func (n *rootNode) Optional() (CPTree, bool) {
	return n.tree.Optional()
}

func (n *rootNode) Repeatable() (CPTree, bool) {
	return n.tree.Repeatable()
}

func (n *rootNode) Concatenation() (CPTree, CPTree, bool) {
	return n.tree.Concatenation()
}

func (n *rootNode) Alternatives() (CPTree, CPTree, bool) {
	return n.tree.Alternatives()
}

func (n *rootNode) Describe() (spec.LexKindName, []spec.LexKindName, error) {
	var frags []spec.LexKindName
	for f := range n.fragments {
		frags = append(frags, spec.LexKindName(f))
	}
	sort.Slice(frags, func(i, j int) bool {
		return frags[i] < frags[j]
	})

	return n.kind, frags, nil
}

func (n *rootNode) children() (CPTree, CPTree) {
	return n.tree.children()
}

func (n *rootNode) clone() CPTree {
	return n.tree.clone()
}

func (n *rootNode) incomplete() bool {
	return len(n.fragments) > 0
}

func (n *rootNode) applyFragment(kind spec.LexKindName, fragment CPTree) error {
	root, ok := fragment.(*rootNode)
	if !ok {
		return fmt.Errorf("applyFragment can take only *rootNode: %T", fragment)
	}
	if root.incomplete() {
		return fmt.Errorf("fragment is incomplete")
	}

	fs, ok := n.fragments[kind]
	if !ok {
		return nil
	}
	for _, f := range fs {
		f.tree = root.clone()
	}
	delete(n.fragments, kind)

	return nil
}

type symbolNode struct {
	CPRange
}

func newSymbolNode(cp rune) *symbolNode {
	return &symbolNode{
		CPRange: CPRange{
			From: cp,
			To:   cp,
		},
	}
}

func newRangeSymbolNode(from, to rune) *symbolNode {
	return &symbolNode{
		CPRange: CPRange{
			From: from,
			To:   to,
		},
	}
}

func (n *symbolNode) String() string {
	return fmt.Sprintf("symbol: %X..%X", n.From, n.To)
}

func (n *symbolNode) Range() (rune, rune, bool) {
	return n.From, n.To, true
}

func (n *symbolNode) Optional() (CPTree, bool) {
	return nil, false
}

func (n *symbolNode) Repeatable() (CPTree, bool) {
	return nil, false
}

func (n *symbolNode) Concatenation() (CPTree, CPTree, bool) {
	return nil, nil, false
}

func (n *symbolNode) Alternatives() (CPTree, CPTree, bool) {
	return nil, nil, false
}

func (n *symbolNode) Describe() (spec.LexKindName, []spec.LexKindName, error) {
	return spec.LexKindNameNil, nil, fmt.Errorf("%T cannot describe", n)
}

func (n *symbolNode) children() (CPTree, CPTree) {
	return nil, nil
}

func (n *symbolNode) clone() CPTree {
	return newRangeSymbolNode(n.From, n.To)
}

type concatNode struct {
	left  CPTree
	right CPTree
}

func newConcatNode(left, right CPTree) *concatNode {
	return &concatNode{
		left:  left,
		right: right,
	}
}

func (n *concatNode) String() string {
	return "concat"
}

func (n *concatNode) Range() (rune, rune, bool) {
	return 0, 0, false
}

func (n *concatNode) Optional() (CPTree, bool) {
	return nil, false
}

func (n *concatNode) Repeatable() (CPTree, bool) {
	return nil, false
}

func (n *concatNode) Concatenation() (CPTree, CPTree, bool) {
	return n.left, n.right, true
}

func (n *concatNode) Alternatives() (CPTree, CPTree, bool) {
	return nil, nil, false
}

func (n *concatNode) Describe() (spec.LexKindName, []spec.LexKindName, error) {
	return spec.LexKindNameNil, nil, fmt.Errorf("%T cannot describe", n)
}

func (n *concatNode) children() (CPTree, CPTree) {
	return n.left, n.right
}

func (n *concatNode) clone() CPTree {
	if n == nil {
		return nil
	}
	return newConcatNode(n.left.clone(), n.right.clone())
}

type altNode struct {
	left  CPTree
	right CPTree
}

func newAltNode(left, right CPTree) *altNode {
	return &altNode{
		left:  left,
		right: right,
	}
}

func (n *altNode) String() string {
	return "alt"
}

func (n *altNode) Range() (rune, rune, bool) {
	return 0, 0, false
}

func (n *altNode) Optional() (CPTree, bool) {
	return nil, false
}

func (n *altNode) Repeatable() (CPTree, bool) {
	return nil, false
}

func (n *altNode) Concatenation() (CPTree, CPTree, bool) {
	return nil, nil, false
}

func (n *altNode) Alternatives() (CPTree, CPTree, bool) {
	return n.left, n.right, true
}

func (n *altNode) Describe() (spec.LexKindName, []spec.LexKindName, error) {
	return spec.LexKindNameNil, nil, fmt.Errorf("%T cannot describe", n)
}

func (n *altNode) children() (CPTree, CPTree) {
	return n.left, n.right
}

func (n *altNode) clone() CPTree {
	return newAltNode(n.left.clone(), n.right.clone())
}

type quantifierNode struct {
	optional   bool
	repeatable bool
	tree       CPTree
}

func (n *quantifierNode) String() string {
	switch {
	case n.repeatable:
		return "repeatable (>= 0 times)"
	case n.optional:
		return "optional (0 or 1 times)"
	default:
		return "invalid quantifier"
	}
}

func newRepeatNode(t CPTree) *quantifierNode {
	return &quantifierNode{
		repeatable: true,
		tree:       t,
	}
}

func newRepeatOneOrMoreNode(t CPTree) *concatNode {
	return newConcatNode(
		t,
		&quantifierNode{
			repeatable: true,
			tree:       t.clone(),
		})
}

func newOptionNode(t CPTree) *quantifierNode {
	return &quantifierNode{
		optional: true,
		tree:     t,
	}
}

func (n *quantifierNode) Range() (rune, rune, bool) {
	return 0, 0, false
}

func (n *quantifierNode) Optional() (CPTree, bool) {
	return n.tree, n.optional
}

func (n *quantifierNode) Repeatable() (CPTree, bool) {
	return n.tree, n.repeatable
}

func (n *quantifierNode) Concatenation() (CPTree, CPTree, bool) {
	return nil, nil, false
}

func (n *quantifierNode) Alternatives() (CPTree, CPTree, bool) {
	return nil, nil, false
}

func (n *quantifierNode) Describe() (spec.LexKindName, []spec.LexKindName, error) {
	return spec.LexKindNameNil, nil, fmt.Errorf("%T cannot describe", n)
}

func (n *quantifierNode) children() (CPTree, CPTree) {
	return n.tree, nil
}

func (n *quantifierNode) clone() CPTree {
	if n.repeatable {
		return newRepeatNode(n.tree.clone())
	}
	return newOptionNode(n.tree.clone())
}

type fragmentNode struct {
	kind spec.LexKindName
	tree CPTree
}

func newFragmentNode(kind spec.LexKindName, t CPTree) *fragmentNode {
	return &fragmentNode{
		kind: kind,
		tree: t,
	}
}

func (n *fragmentNode) String() string {
	return fmt.Sprintf("fragment: %v", n.kind)
}

func (n *fragmentNode) Range() (rune, rune, bool) {
	return n.tree.Range()
}

func (n *fragmentNode) Optional() (CPTree, bool) {
	return n.tree.Optional()
}

func (n *fragmentNode) Repeatable() (CPTree, bool) {
	return n.tree.Repeatable()
}

func (n *fragmentNode) Concatenation() (CPTree, CPTree, bool) {
	return n.tree.Concatenation()
}

func (n *fragmentNode) Alternatives() (CPTree, CPTree, bool) {
	return n.tree.Alternatives()
}

func (n *fragmentNode) Describe() (spec.LexKindName, []spec.LexKindName, error) {
	return spec.LexKindNameNil, nil, fmt.Errorf("%T cannot describe", n)
}

func (n *fragmentNode) children() (CPTree, CPTree) {
	return n.tree.children()
}

func (n *fragmentNode) clone() CPTree {
	if n.tree == nil {
		return newFragmentNode(n.kind, nil)
	}
	return newFragmentNode(n.kind, n.tree.clone())
}

//nolint:unused
func printCPTree(w io.Writer, t CPTree, ruledLine string, childRuledLinePrefix string) {
	if t == nil {
		return
	}
	fmt.Fprintf(w, "%v%v\n", ruledLine, t)
	children := []CPTree{}
	switch n := t.(type) {
	case *rootNode:
		children = append(children, n.tree)
	case *fragmentNode:
		children = append(children, n.tree)
	default:
		left, right := t.children()
		if left != nil {
			children = append(children, left)
		}
		if right != nil {
			children = append(children, right)
		}
	}
	num := len(children)
	for i, child := range children {
		line := "└─ "
		if num > 1 {
			if i == 0 {
				line = "├─ "
			} else if i < num-1 {
				line = "│  "
			}
		}
		prefix := "│  "
		if i >= num-1 {
			prefix = "    "
		}
		printCPTree(w, child, childRuledLinePrefix+line, childRuledLinePrefix+prefix)
	}
}
