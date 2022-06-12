package tester

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar"
	gspec "github.com/nihei9/vartan/spec/grammar"
	tspec "github.com/nihei9/vartan/spec/test"
)

func TestTester_Run(t *testing.T) {
	grammarSrc1 := `
#name test;

s
    : foo bar baz
    | foo error baz #recover
    ;

ws #skip
    : "[\u{0009}\u{0020}]+";
foo
    : 'foo';
bar
    : 'bar';
baz
    : 'baz';
`

	grammarSrc2 := `
#name test;

s
    : foos
    ;
foos
    : foos foo #ast foos... foo
    | foo
    ;

ws #skip
    : "[\u{0009}\u{0020}]+";
foo
    : 'foo';
`

	tests := []struct {
		grammarSrc string
		testSrc    string
		error      bool
	}{
		{
			grammarSrc: grammarSrc1,
			testSrc: `
Test
---
foo bar baz
---
(s
    (foo 'foo') (bar 'bar') (baz 'baz'))
`,
		},
		{
			grammarSrc: grammarSrc1,
			testSrc: `
Test
---
foo ? baz
---
(s
    (foo 'foo') (error) (baz 'baz'))
`,
		},
		{
			grammarSrc: grammarSrc1,
			testSrc: `
Test
---
foo bar baz
---
(s)
`,
			error: true,
		},
		{
			grammarSrc: grammarSrc1,
			testSrc: `
Test
---
foo bar baz
---
(s
    (foo) (bar))
`,
			error: true,
		},
		{
			grammarSrc: grammarSrc1,
			testSrc: `
Test
---
foo bar baz
---
(s
    (foo) (bar) (xxx))
`,
			error: true,
		},
		{
			grammarSrc: grammarSrc2,
			testSrc: `
Test
---
foo foo foo
---
(s
    (foos
        (foo 'foo') (foo 'foo') (foo 'foo')))
`,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			ast, err := gspec.Parse(strings.NewReader(tt.grammarSrc))
			if err != nil {
				t.Fatal(err)
			}
			b := grammar.GrammarBuilder{
				AST: ast,
			}
			g, err := b.Build()
			if err != nil {
				t.Fatal(err)
			}
			cg, _, err := grammar.Compile(g)
			if err != nil {
				t.Fatal(err)
			}
			c, err := tspec.ParseTestCase(strings.NewReader(tt.testSrc))
			if err != nil {
				t.Fatal(err)
			}
			tester := &Tester{
				Grammar: cg,
				Cases: []*TestCaseWithMetadata{
					{
						TestCase: c,
					},
				},
			}
			rs := tester.Run()
			if tt.error {
				errOccurred := false
				for _, r := range rs {
					if r.Error != nil {
						errOccurred = true
					}
				}
				if !errOccurred {
					t.Fatal("this test must fail, but it passed")
				}
			} else {
				for _, r := range rs {
					if r.Error != nil {
						t.Fatalf("unexpected error occurred: %v", r.Error)
					}
				}
			}
		})
	}
}
