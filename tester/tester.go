package tester

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/nihei9/vartan/driver"
	gspec "github.com/nihei9/vartan/spec/grammar"
	tspec "github.com/nihei9/vartan/spec/test"
)

type TestResult struct {
	TestCasePath string
	Error        error
	Diffs        []*tspec.TreeDiff
}

func (r *TestResult) String() string {
	if r.Error != nil {
		const indent1 = "    "
		const indent2 = indent1 + indent1

		msgLines := strings.Split(r.Error.Error(), "\n")
		msg := fmt.Sprintf("Failed %v:\n%v%v", r.TestCasePath, indent1, strings.Join(msgLines, "\n"+indent1))
		if len(r.Diffs) == 0 {
			return msg
		}
		var diffLines []string
		for _, diff := range r.Diffs {
			diffLines = append(diffLines, diff.Message)
			diffLines = append(diffLines, fmt.Sprintf("%vexpected path: %v", indent1, diff.ExpectedPath))
			diffLines = append(diffLines, fmt.Sprintf("%vactual path:   %v", indent1, diff.ActualPath))
		}
		return fmt.Sprintf("%v\n%v%v", msg, indent2, strings.Join(diffLines, "\n"+indent2))
	}
	return fmt.Sprintf("Passed %v", r.TestCasePath)
}

type TestCaseWithMetadata struct {
	TestCase *tspec.TestCase
	FilePath string
	Error    error
}

func ListTestCases(testPath string) []*TestCaseWithMetadata {
	fi, err := os.Stat(testPath)
	if err != nil {
		return []*TestCaseWithMetadata{
			{
				FilePath: testPath,
				Error:    err,
			},
		}
	}
	if !fi.IsDir() {
		c, err := parseTestCase(testPath)
		return []*TestCaseWithMetadata{
			{
				TestCase: c,
				FilePath: testPath,
				Error:    err,
			},
		}
	}

	es, err := os.ReadDir(testPath)
	if err != nil {
		return []*TestCaseWithMetadata{
			{
				FilePath: testPath,
				Error:    err,
			},
		}
	}
	var cases []*TestCaseWithMetadata
	for _, e := range es {
		cs := ListTestCases(filepath.Join(testPath, e.Name()))
		cases = append(cases, cs...)
	}
	return cases
}

func parseTestCase(testCasePath string) (*tspec.TestCase, error) {
	f, err := os.Open(testCasePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return tspec.ParseTestCase(f)
}

type Tester struct {
	Grammar *gspec.CompiledGrammar
	Cases   []*TestCaseWithMetadata
}

func (t *Tester) Run() []*TestResult {
	var rs []*TestResult
	for _, c := range t.Cases {
		rs = append(rs, runTest(t.Grammar, c))
	}
	return rs
}

func runTest(g *gspec.CompiledGrammar, c *TestCaseWithMetadata) *TestResult {
	var p *driver.Parser
	var tb *driver.DefaulSyntaxTreeBuilder
	{
		gram := driver.NewGrammar(g)
		toks, err := driver.NewTokenStream(g, bytes.NewReader(c.TestCase.Source))
		if err != nil {
			return &TestResult{
				TestCasePath: c.FilePath,
				Error:        err,
			}
		}
		tb = driver.NewDefaultSyntaxTreeBuilder()
		p, err = driver.NewParser(toks, gram, driver.SemanticAction(driver.NewASTActionSet(gram, tb)))
		if err != nil {
			return &TestResult{
				TestCasePath: c.FilePath,
				Error:        err,
			}
		}
	}

	err := p.Parse()
	if err != nil {
		return &TestResult{
			TestCasePath: c.FilePath,
			Error:        err,
		}
	}

	if tb.Tree() == nil {
		var err error
		if len(p.SyntaxErrors()) > 0 {
			err = fmt.Errorf("parse tree was not generated: syntax error occurred")
		} else {
			// The parser should always generate a parse tree in the vartan-test command, so if there is no parse
			// tree, it is a bug. We also include a stack trace in the error message to be sure.
			err = fmt.Errorf("parse tree was not generated: no syntax error:\n%v", string(debug.Stack()))
		}
		return &TestResult{
			TestCasePath: c.FilePath,
			Error:        err,
		}
	}

	// When a parse tree exists, the test continues regardless of whether or not syntax errors occurred.
	diffs := tspec.DiffTree(genTree(tb.Tree()).Fill(), c.TestCase.Output)
	if len(diffs) > 0 {
		return &TestResult{
			TestCasePath: c.FilePath,
			Error:        fmt.Errorf("output mismatch"),
			Diffs:        diffs,
		}
	}
	return &TestResult{
		TestCasePath: c.FilePath,
	}
}

func genTree(dTree *driver.Node) *tspec.Tree {
	var children []*tspec.Tree
	if len(dTree.Children) > 0 {
		children = make([]*tspec.Tree, len(dTree.Children))
		for i, c := range dTree.Children {
			children[i] = genTree(c)
		}
	}
	return tspec.NewTree(dTree.KindName, children...)
}
