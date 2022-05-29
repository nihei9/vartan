//go:generate vartan compile tree.vartan -o tree.json
//go:generate vartan-go tree.json --package test

package test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type TreeDiff struct {
	ExpectedPath string
	ActualPath   string
	Message      string
}

func newTreeDiff(expected, actual *Tree, message string) *TreeDiff {
	return &TreeDiff{
		ExpectedPath: expected.path(),
		ActualPath:   actual.path(),
		Message:      message,
	}
}

type Tree struct {
	Parent   *Tree
	Offset   int
	Kind     string
	Children []*Tree
}

func NewTree(kind string, children ...*Tree) *Tree {
	return &Tree{
		Kind:     kind,
		Children: children,
	}
}

func (t *Tree) Fill() *Tree {
	for i, c := range t.Children {
		c.Parent = t
		c.Offset = i
		c.Fill()
	}
	return t
}

func (t *Tree) path() string {
	if t.Parent == nil {
		return t.Kind
	}
	return fmt.Sprintf("%v.[%v]%v", t.Parent.path(), t.Offset, t.Kind)
}

func (t *Tree) Format() []byte {
	var b bytes.Buffer
	t.format(&b, 0)
	return b.Bytes()
}

func (t *Tree) format(buf *bytes.Buffer, depth int) {
	for i := 0; i < depth; i++ {
		buf.WriteString("    ")
	}
	buf.WriteString("(")
	buf.WriteString(t.Kind)
	if len(t.Children) > 0 {
		buf.WriteString("\n")
		for i, c := range t.Children {
			c.format(buf, depth+1)
			if i < len(t.Children)-1 {
				buf.WriteString("\n")
			}
		}
	}
	buf.WriteString(")")
}

func DiffTree(expected, actual *Tree) []*TreeDiff {
	if expected == nil && actual == nil {
		return nil
	}
	if actual.Kind != expected.Kind {
		msg := fmt.Sprintf("unexpected kind: expected '%v' but got '%v'", expected.Kind, actual.Kind)
		return []*TreeDiff{
			newTreeDiff(expected, actual, msg),
		}
	}
	if len(actual.Children) != len(expected.Children) {
		msg := fmt.Sprintf("unexpected node count: expected %v but got %v", len(expected.Children), len(actual.Children))
		return []*TreeDiff{
			newTreeDiff(expected, actual, msg),
		}
	}
	var diffs []*TreeDiff
	for i, exp := range expected.Children {
		if ds := DiffTree(actual.Children[i], exp); len(ds) > 0 {
			diffs = append(diffs, ds...)
		}
	}
	return diffs
}

type TestCase struct {
	Description string
	Source      []byte
	Output      *Tree
}

func ParseTestCase(r io.Reader) (*TestCase, error) {
	bufs, err := splitIntoParts(r)
	if err != nil {
		return nil, err
	}
	if len(bufs) != 3 {
		return nil, fmt.Errorf("too many or too few part delimiters: a test case consists of just tree parts: %v parts found", len(bufs))
	}

	tree, err := parseTree(bytes.NewReader(bufs[2]))
	if err != nil {
		return nil, err
	}

	return &TestCase{
		Description: string(bufs[0]),
		Source:      bufs[1],
		Output:      tree,
	}, nil
}

func splitIntoParts(r io.Reader) ([][]byte, error) {
	var bufs [][]byte
	s := bufio.NewScanner(r)
	for {
		buf, err := readPart(s)
		if err != nil {
			return nil, err
		}
		if buf == nil {
			break
		}
		bufs = append(bufs, buf)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return bufs, nil
}

var reDelim = regexp.MustCompile(`^\s*---+\s*$`)

func readPart(s *bufio.Scanner) ([]byte, error) {
	if !s.Scan() {
		return nil, s.Err()
	}
	buf := &bytes.Buffer{}
	line := s.Bytes()
	if reDelim.Match(line) {
		// Return an empty slice because (*bytes.Buffer).Bytes() returns nil if we have never written data.
		return []byte{}, nil
	}
	_, err := buf.Write(line)
	if err != nil {
		return nil, err
	}
	for s.Scan() {
		line := s.Bytes()
		if reDelim.Match(line) {
			return buf.Bytes(), nil
		}
		_, err := buf.Write([]byte("\n"))
		if err != nil {
			return nil, err
		}
		_, err = buf.Write(line)
		if err != nil {
			return nil, err
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func parseTree(src io.Reader) (*Tree, error) {
	toks, err := NewTokenStream(src)
	if err != nil {
		return nil, err
	}
	gram := NewGrammar()
	tb := NewDefaultSyntaxTreeBuilder()
	p, err := NewParser(toks, gram, SemanticAction(NewASTActionSet(gram, tb)))
	if err != nil {
		return nil, err
	}
	err = p.Parse()
	if err != nil {
		return nil, err
	}
	synErrs := p.SyntaxErrors()
	if len(synErrs) > 0 {
		var b strings.Builder
		b.WriteString("syntax error:")
		for _, synErr := range synErrs {
			b.WriteRune('\n')
			b.Write(formatSyntaxError(synErr, gram))
		}
		return nil, errors.New(b.String())
	}
	return genTree(tb.Tree()).Fill(), nil
}

func formatSyntaxError(synErr *SyntaxError, gram Grammar) []byte {
	var b bytes.Buffer

	b.WriteString(fmt.Sprintf("%v:%v: %v: ", synErr.Row+1, synErr.Col+1, synErr.Message))

	tok := synErr.Token
	switch {
	case tok.EOF():
		b.WriteString("<eof>")
	case tok.Invalid():
		b.WriteString(fmt.Sprintf("'%v' (<invalid>)", string(tok.Lexeme())))
	default:
		b.WriteString(fmt.Sprintf("'%v' (%v)", string(tok.Lexeme()), gram.Terminal(tok.TerminalID())))
	}
	b.WriteString(fmt.Sprintf("; expected: %v", synErr.ExpectedTerminals[0]))
	for _, t := range synErr.ExpectedTerminals[1:] {
		b.WriteString(fmt.Sprintf(", %v", t))
	}

	return b.Bytes()
}

func genTree(node *Node) *Tree {
	var children []*Tree
	if len(node.Children) > 1 {
		children = make([]*Tree, len(node.Children)-1)
		for i, c := range node.Children[1:] {
			children[i] = genTree(c)
		}
	}
	return NewTree(node.Children[0].Text, children...)
}
