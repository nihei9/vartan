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
	"strconv"
	"strings"
	"unicode/utf8"
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
	Lexeme   string
}

func NewNonTerminalTree(kind string, children ...*Tree) *Tree {
	return &Tree{
		Kind:     kind,
		Children: children,
	}
}

func NewTerminalNode(kind string, lexeme string) *Tree {
	return &Tree{
		Kind:   kind,
		Lexeme: lexeme,
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
	if t.Kind == "" {
		buf.WriteString("<anonymous>")
	} else {
		buf.WriteString(t.Kind)
	}
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
	// _ matches any symbols.
	if expected.Kind != "_" && actual.Kind != expected.Kind {
		msg := fmt.Sprintf("unexpected kind: expected '%v' but got '%v'", expected.Kind, actual.Kind)
		return []*TreeDiff{
			newTreeDiff(expected, actual, msg),
		}
	}
	if expected.Lexeme != actual.Lexeme {
		msg := fmt.Sprintf("unexpected lexeme: expected '%v' but got '%v'", expected.Lexeme, actual.Lexeme)
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
		if ds := DiffTree(exp, actual.Children[i]); len(ds) > 0 {
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
	parts, err := splitIntoParts(r)
	if err != nil {
		return nil, err
	}
	if len(parts) != 3 {
		return nil, fmt.Errorf("too many or too few part delimiters: a test case consists of just tree parts: %v parts found", len(parts))
	}

	tp := &treeParser{
		lineOffset: parts[0].lineCount + parts[1].lineCount + 2,
	}
	tree, err := tp.parseTree(bytes.NewReader(parts[2].buf))
	if err != nil {
		return nil, err
	}

	return &TestCase{
		Description: string(parts[0].buf),
		Source:      parts[1].buf,
		Output:      tree,
	}, nil
}

type testCasePart struct {
	buf       []byte
	lineCount int
}

func splitIntoParts(r io.Reader) ([]*testCasePart, error) {
	var bufs []*testCasePart
	s := bufio.NewScanner(r)
	for {
		buf, lineCount, err := readPart(s)
		if err != nil {
			return nil, err
		}
		if buf == nil {
			break
		}
		bufs = append(bufs, &testCasePart{
			buf:       buf,
			lineCount: lineCount,
		})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return bufs, nil
}

var reDelim = regexp.MustCompile(`^\s*---+\s*$`)

func readPart(s *bufio.Scanner) ([]byte, int, error) {
	if !s.Scan() {
		return nil, 0, s.Err()
	}
	buf := &bytes.Buffer{}
	line := s.Bytes()
	if reDelim.Match(line) {
		// Return an empty slice because (*bytes.Buffer).Bytes() returns nil if we have never written data.
		return []byte{}, 0, nil
	}
	_, err := buf.Write(line)
	if err != nil {
		return nil, 0, err
	}
	lineCount := 1
	for s.Scan() {
		line := s.Bytes()
		if reDelim.Match(line) {
			return buf.Bytes(), lineCount, nil
		}
		_, err := buf.Write([]byte("\n"))
		if err != nil {
			return nil, 0, err
		}
		_, err = buf.Write(line)
		if err != nil {
			return nil, 0, err
		}
		lineCount++
	}
	if err := s.Err(); err != nil {
		return nil, 0, err
	}
	return buf.Bytes(), lineCount, nil
}

type treeParser struct {
	lineOffset int
}

func (tp *treeParser) parseTree(src io.Reader) (*Tree, error) {
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
		b.Write(formatSyntaxError(synErrs[0], gram, tp.lineOffset))
		for _, synErr := range synErrs[1:] {
			b.WriteRune('\n')
			b.Write(formatSyntaxError(synErr, gram, tp.lineOffset))
		}
		return nil, errors.New(b.String())
	}
	t, err := tp.genTree(tb.Tree())
	if err != nil {
		return nil, err
	}
	return t.Fill(), nil
}

func formatSyntaxError(synErr *SyntaxError, gram Grammar, lineOffset int) []byte {
	var b bytes.Buffer

	b.WriteString(fmt.Sprintf("%v:%v: %v: ", lineOffset+synErr.Row+1, synErr.Col+1, synErr.Message))

	tok := synErr.Token
	switch {
	case tok.EOF():
		b.WriteString("<eof>")
	case tok.Invalid():
		b.WriteString(fmt.Sprintf("'%v' (<invalid>)", string(tok.Lexeme())))
	default:
		if term := gram.Terminal(tok.TerminalID()); term != "" {
			if alias := gram.TerminalAlias(tok.TerminalID()); alias != "" {
				b.WriteString(fmt.Sprintf("'%v' (%v)", string(tok.Lexeme()), alias))
			} else {
				b.WriteString(fmt.Sprintf("'%v' (%v)", string(tok.Lexeme()), term))
			}
		} else {
			b.WriteString(fmt.Sprintf("'%v'", string(tok.Lexeme())))
		}
	}
	b.WriteString(fmt.Sprintf(": expected: %v", synErr.ExpectedTerminals[0]))
	for _, t := range synErr.ExpectedTerminals[1:] {
		b.WriteString(fmt.Sprintf(", %v", t))
	}

	return b.Bytes()
}

func (tp *treeParser) genTree(node *Node) (*Tree, error) {
	// A node labeled 'error' cannot have children. It always must be (error).
	if sym := node.Children[0]; sym.Text == "error" {
		if len(node.Children) > 1 {
			return nil, fmt.Errorf("%v:%v: error node cannot take children", tp.lineOffset+sym.Row+1, sym.Col+1)
		}
		return NewTerminalNode(sym.Text, ""), nil
	}

	if len(node.Children) == 2 && node.Children[1].KindName == "string" {
		var text string
		str := node.Children[1].Children[0]
		switch str.KindName {
		case "raw_string":
			text = str.Children[0].Text
		case "interpreted_string":
			var b strings.Builder
			for _, c := range str.Children {
				switch c.KindName {
				case "escaped_seq":
					b.WriteString(strings.TrimPrefix(`\`, c.Text))
				case "escape_char":
					return nil, fmt.Errorf("%v:%v: incomplete escape sequence", tp.lineOffset+c.Row+1, c.Col+1)
				case "codepoint_expr":
					cp := c.Children[0]
					n, err := strconv.ParseInt(cp.Text, 16, 64)
					if err != nil {
						return nil, fmt.Errorf("%v:%v: %v", tp.lineOffset+cp.Row+1, cp.Col+1, err)
					}
					if !utf8.ValidRune(rune(n)) {
						return nil, fmt.Errorf("%v:%v: invalid code point: %v", tp.lineOffset+cp.Row+1, cp.Col+1, cp.Text)
					}
					b.WriteRune(rune(n))
				default:
					b.WriteString(c.Text)
				}
			}
			text = b.String()
		}
		return NewTerminalNode(node.Children[0].Text, text), nil
	}

	var children []*Tree
	if len(node.Children) > 1 {
		children = make([]*Tree, len(node.Children)-1)
		for i, c := range node.Children[1:] {
			var err error
			children[i], err = tp.genTree(c)
			if err != nil {
				return nil, err
			}
		}
	}
	return NewNonTerminalTree(node.Children[0].Text, children...), nil
}
