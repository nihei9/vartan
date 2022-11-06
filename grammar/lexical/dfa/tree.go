package dfa

import (
	"fmt"
	"io"
	"sort"

	"github.com/nihei9/vartan/grammar/lexical/parser"
	spec "github.com/nihei9/vartan/spec/grammar"
	"github.com/nihei9/vartan/utf8"
)

type byteTree interface {
	fmt.Stringer
	children() (byteTree, byteTree)
	nullable() bool
	first() *symbolPositionSet
	last() *symbolPositionSet
	clone() byteTree
}

var (
	_ byteTree = &symbolNode{}
	_ byteTree = &endMarkerNode{}
	_ byteTree = &concatNode{}
	_ byteTree = &altNode{}
	_ byteTree = &repeatNode{}
	_ byteTree = &optionNode{}
)

type byteRange struct {
	from byte
	to   byte
}

type symbolNode struct {
	byteRange
	pos       symbolPosition
	firstMemo *symbolPositionSet
	lastMemo  *symbolPositionSet
}

func newSymbolNode(value byte) *symbolNode {
	return &symbolNode{
		byteRange: byteRange{
			from: value,
			to:   value,
		},
		pos: symbolPositionNil,
	}
}

func newRangeSymbolNode(from, to byte) *symbolNode {
	return &symbolNode{
		byteRange: byteRange{
			from: from,
			to:   to,
		},
		pos: symbolPositionNil,
	}
}

func (n *symbolNode) String() string {
	return fmt.Sprintf("symbol: value: %v-%v, pos: %v", n.from, n.to, n.pos)
}

func (n *symbolNode) children() (byteTree, byteTree) {
	return nil, nil
}

func (n *symbolNode) nullable() bool {
	return false
}

func (n *symbolNode) first() *symbolPositionSet {
	if n.firstMemo == nil {
		n.firstMemo = newSymbolPositionSet()
		n.firstMemo.add(n.pos)
	}
	return n.firstMemo
}

func (n *symbolNode) last() *symbolPositionSet {
	if n.lastMemo == nil {
		n.lastMemo = newSymbolPositionSet()
		n.lastMemo.add(n.pos)
	}
	return n.lastMemo
}

func (n *symbolNode) clone() byteTree {
	return newRangeSymbolNode(n.from, n.to)
}

type endMarkerNode struct {
	id        spec.LexModeKindID
	pos       symbolPosition
	firstMemo *symbolPositionSet
	lastMemo  *symbolPositionSet
}

func newEndMarkerNode(id spec.LexModeKindID) *endMarkerNode {
	return &endMarkerNode{
		id:  id,
		pos: symbolPositionNil,
	}
}

func (n *endMarkerNode) String() string {
	return fmt.Sprintf("end: pos: %v", n.pos)
}

func (n *endMarkerNode) children() (byteTree, byteTree) {
	return nil, nil
}

func (n *endMarkerNode) nullable() bool {
	return false
}

func (n *endMarkerNode) first() *symbolPositionSet {
	if n.firstMemo == nil {
		n.firstMemo = newSymbolPositionSet()
		n.firstMemo.add(n.pos)
	}
	return n.firstMemo
}

func (n *endMarkerNode) last() *symbolPositionSet {
	if n.lastMemo == nil {
		n.lastMemo = newSymbolPositionSet()
		n.lastMemo.add(n.pos)
	}
	return n.lastMemo
}

func (n *endMarkerNode) clone() byteTree {
	return newEndMarkerNode(n.id)
}

type concatNode struct {
	left      byteTree
	right     byteTree
	firstMemo *symbolPositionSet
	lastMemo  *symbolPositionSet
}

func newConcatNode(left, right byteTree) *concatNode {
	return &concatNode{
		left:  left,
		right: right,
	}
}

func (n *concatNode) String() string {
	return "concat"
}

func (n *concatNode) children() (byteTree, byteTree) {
	return n.left, n.right
}

func (n *concatNode) nullable() bool {
	return n.left.nullable() && n.right.nullable()
}

func (n *concatNode) first() *symbolPositionSet {
	if n.firstMemo == nil {
		n.firstMemo = newSymbolPositionSet()
		n.firstMemo.merge(n.left.first())
		if n.left.nullable() {
			n.firstMemo.merge(n.right.first())
		}
		n.firstMemo.sortAndRemoveDuplicates()
	}
	return n.firstMemo
}

func (n *concatNode) last() *symbolPositionSet {
	if n.lastMemo == nil {
		n.lastMemo = newSymbolPositionSet()
		n.lastMemo.merge(n.right.last())
		if n.right.nullable() {
			n.lastMemo.merge(n.left.last())
		}
		n.lastMemo.sortAndRemoveDuplicates()
	}
	return n.lastMemo
}

func (n *concatNode) clone() byteTree {
	return newConcatNode(n.left.clone(), n.right.clone())
}

type altNode struct {
	left      byteTree
	right     byteTree
	firstMemo *symbolPositionSet
	lastMemo  *symbolPositionSet
}

func newAltNode(left, right byteTree) *altNode {
	return &altNode{
		left:  left,
		right: right,
	}
}

func (n *altNode) String() string {
	return "alt"
}

func (n *altNode) children() (byteTree, byteTree) {
	return n.left, n.right
}

func (n *altNode) nullable() bool {
	return n.left.nullable() || n.right.nullable()
}

func (n *altNode) first() *symbolPositionSet {
	if n.firstMemo == nil {
		n.firstMemo = newSymbolPositionSet()
		n.firstMemo.merge(n.left.first())
		n.firstMemo.merge(n.right.first())
		n.firstMemo.sortAndRemoveDuplicates()
	}
	return n.firstMemo
}

func (n *altNode) last() *symbolPositionSet {
	if n.lastMemo == nil {
		n.lastMemo = newSymbolPositionSet()
		n.lastMemo.merge(n.left.last())
		n.lastMemo.merge(n.right.last())
		n.lastMemo.sortAndRemoveDuplicates()
	}
	return n.lastMemo
}

func (n *altNode) clone() byteTree {
	return newAltNode(n.left.clone(), n.right.clone())
}

type repeatNode struct {
	left      byteTree
	firstMemo *symbolPositionSet
	lastMemo  *symbolPositionSet
}

func newRepeatNode(left byteTree) *repeatNode {
	return &repeatNode{
		left: left,
	}
}

func (n *repeatNode) String() string {
	return "repeat"
}

func (n *repeatNode) children() (byteTree, byteTree) {
	return n.left, nil
}

func (n *repeatNode) nullable() bool {
	return true
}

func (n *repeatNode) first() *symbolPositionSet {
	if n.firstMemo == nil {
		n.firstMemo = newSymbolPositionSet()
		n.firstMemo.merge(n.left.first())
		n.firstMemo.sortAndRemoveDuplicates()
	}
	return n.firstMemo
}

func (n *repeatNode) last() *symbolPositionSet {
	if n.lastMemo == nil {
		n.lastMemo = newSymbolPositionSet()
		n.lastMemo.merge(n.left.last())
		n.lastMemo.sortAndRemoveDuplicates()
	}
	return n.lastMemo
}

func (n *repeatNode) clone() byteTree {
	return newRepeatNode(n.left.clone())
}

type optionNode struct {
	left      byteTree
	firstMemo *symbolPositionSet
	lastMemo  *symbolPositionSet
}

func newOptionNode(left byteTree) *optionNode {
	return &optionNode{
		left: left,
	}
}

func (n *optionNode) String() string {
	return "option"
}

func (n *optionNode) children() (byteTree, byteTree) {
	return n.left, nil
}

func (n *optionNode) nullable() bool {
	return true
}

func (n *optionNode) first() *symbolPositionSet {
	if n.firstMemo == nil {
		n.firstMemo = newSymbolPositionSet()
		n.firstMemo.merge(n.left.first())
		n.firstMemo.sortAndRemoveDuplicates()
	}
	return n.firstMemo
}

func (n *optionNode) last() *symbolPositionSet {
	if n.lastMemo == nil {
		n.lastMemo = newSymbolPositionSet()
		n.lastMemo.merge(n.left.last())
		n.lastMemo.sortAndRemoveDuplicates()
	}
	return n.lastMemo
}

func (n *optionNode) clone() byteTree {
	return newOptionNode(n.left.clone())
}

type followTable map[symbolPosition]*symbolPositionSet

func genFollowTable(root byteTree) followTable {
	follow := followTable{}
	calcFollow(follow, root)
	return follow
}

func calcFollow(follow followTable, ast byteTree) {
	if ast == nil {
		return
	}
	left, right := ast.children()
	calcFollow(follow, left)
	calcFollow(follow, right)
	switch n := ast.(type) {
	case *concatNode:
		l, r := n.children()
		for _, p := range l.last().set() {
			if _, ok := follow[p]; !ok {
				follow[p] = newSymbolPositionSet()
			}
			follow[p].merge(r.first())
		}
	case *repeatNode:
		for _, p := range n.last().set() {
			if _, ok := follow[p]; !ok {
				follow[p] = newSymbolPositionSet()
			}
			follow[p].merge(n.first())
		}
	}
}

func positionSymbols(node byteTree, n uint16) (uint16, error) {
	if node == nil {
		return n, nil
	}

	l, r := node.children()
	p := n
	p, err := positionSymbols(l, p)
	if err != nil {
		return p, err
	}
	p, err = positionSymbols(r, p)
	if err != nil {
		return p, err
	}
	switch n := node.(type) {
	case *symbolNode:
		n.pos, err = newSymbolPosition(p, false)
		if err != nil {
			return p, err
		}
		p++
	case *endMarkerNode:
		n.pos, err = newSymbolPosition(p, true)
		if err != nil {
			return p, err
		}
		p++
	}
	node.first()
	node.last()
	return p, nil
}

func concat(ts ...byteTree) byteTree {
	nonNilNodes := []byteTree{}
	for _, t := range ts {
		if t == nil {
			continue
		}
		nonNilNodes = append(nonNilNodes, t)
	}
	if len(nonNilNodes) <= 0 {
		return nil
	}
	if len(nonNilNodes) == 1 {
		return nonNilNodes[0]
	}
	concat := newConcatNode(nonNilNodes[0], nonNilNodes[1])
	for _, t := range nonNilNodes[2:] {
		concat = newConcatNode(concat, t)
	}
	return concat
}

func oneOf(ts ...byteTree) byteTree {
	nonNilNodes := []byteTree{}
	for _, t := range ts {
		if t == nil {
			continue
		}
		nonNilNodes = append(nonNilNodes, t)
	}
	if len(nonNilNodes) <= 0 {
		return nil
	}
	if len(nonNilNodes) == 1 {
		return nonNilNodes[0]
	}
	alt := newAltNode(nonNilNodes[0], nonNilNodes[1])
	for _, t := range nonNilNodes[2:] {
		alt = newAltNode(alt, t)
	}
	return alt
}

//nolint:unused
func printByteTree(w io.Writer, t byteTree, ruledLine string, childRuledLinePrefix string, withAttrs bool) {
	if t == nil {
		return
	}
	fmt.Fprintf(w, "%v%v", ruledLine, t)
	if withAttrs {
		fmt.Fprintf(w, ", nullable: %v, first: %v, last: %v", t.nullable(), t.first(), t.last())
	}
	fmt.Fprintf(w, "\n")
	left, right := t.children()
	children := []byteTree{}
	if left != nil {
		children = append(children, left)
	}
	if right != nil {
		children = append(children, right)
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
		printByteTree(w, child, childRuledLinePrefix+line, childRuledLinePrefix+prefix, withAttrs)
	}
}

func ConvertCPTreeToByteTree(cpTrees map[spec.LexModeKindID]parser.CPTree) (byteTree, *symbolTable, error) {
	var ids []spec.LexModeKindID
	for id := range cpTrees {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	var bt byteTree
	for _, id := range ids {
		cpTree := cpTrees[id]
		t, err := convCPTreeToByteTree(cpTree)
		if err != nil {
			return nil, nil, err
		}
		bt = oneOf(bt, concat(t, newEndMarkerNode(id)))
	}
	_, err := positionSymbols(bt, symbolPositionMin)
	if err != nil {
		return nil, nil, err
	}

	return bt, genSymbolTable(bt), nil
}

func convCPTreeToByteTree(cpTree parser.CPTree) (byteTree, error) {
	if from, to, ok := cpTree.Range(); ok {
		bs, err := utf8.GenCharBlocks(from, to)
		if err != nil {
			return nil, err
		}
		var a byteTree
		for _, b := range bs {
			var c byteTree
			for i := 0; i < len(b.From); i++ {
				c = concat(c, newRangeSymbolNode(b.From[i], b.To[i]))
			}
			a = oneOf(a, c)
		}
		return a, nil
	}

	if tree, ok := cpTree.Repeatable(); ok {
		t, err := convCPTreeToByteTree(tree)
		if err != nil {
			return nil, err
		}
		return newRepeatNode(t), nil
	}

	if tree, ok := cpTree.Optional(); ok {
		t, err := convCPTreeToByteTree(tree)
		if err != nil {
			return nil, err
		}
		return newOptionNode(t), nil
	}

	if left, right, ok := cpTree.Concatenation(); ok {
		l, err := convCPTreeToByteTree(left)
		if err != nil {
			return nil, err
		}
		r, err := convCPTreeToByteTree(right)
		if err != nil {
			return nil, err
		}
		return newConcatNode(l, r), nil
	}

	if left, right, ok := cpTree.Alternatives(); ok {
		l, err := convCPTreeToByteTree(left)
		if err != nil {
			return nil, err
		}
		r, err := convCPTreeToByteTree(right)
		if err != nil {
			return nil, err
		}
		return newAltNode(l, r), nil
	}

	return nil, fmt.Errorf("invalid tree type: %T", cpTree)
}
