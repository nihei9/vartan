package test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestTree_Format(t *testing.T) {
	expected := `(a
    (b
        (c))
    (d)
    (e))`
	tree := NewNonTerminalTree("a",
		NewNonTerminalTree("b",
			NewNonTerminalTree("c"),
		),
		NewNonTerminalTree("d"),
		NewNonTerminalTree("e"),
	)
	actual := string(tree.Format())
	if actual != expected {
		t.Fatalf("unexpected format:\n%v", actual)
	}
}

func TestDiffTree(t *testing.T) {
	tests := []struct {
		t1        *Tree
		t2        *Tree
		different bool
	}{
		{
			t1: NewNonTerminalTree("a"),
			t2: NewNonTerminalTree("a"),
		},
		{
			t1: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
			),
			t2: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
			),
		},
		{
			t1: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
				NewNonTerminalTree("c"),
				NewNonTerminalTree("d"),
			),
			t2: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
				NewNonTerminalTree("c"),
				NewNonTerminalTree("d"),
			),
		},
		{
			t1: NewNonTerminalTree("a",
				NewNonTerminalTree("b",
					NewNonTerminalTree("c"),
				),
				NewNonTerminalTree("d",
					NewNonTerminalTree("d"),
				),
			),
			t2: NewNonTerminalTree("a",
				NewNonTerminalTree("b",
					NewNonTerminalTree("c"),
				),
				NewNonTerminalTree("d",
					NewNonTerminalTree("d"),
				),
			),
		},
		{
			t1: NewNonTerminalTree("_"),
			t2: NewNonTerminalTree("a"),
		},
		{
			t1: NewNonTerminalTree("a",
				NewNonTerminalTree("_"),
			),
			t2: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
			),
		},
		{
			t1: NewNonTerminalTree("_",
				NewNonTerminalTree("b"),
			),
			t2: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
			),
		},
		{
			t1:        NewNonTerminalTree("a"),
			t2:        NewNonTerminalTree("b"),
			different: true,
		},
		{
			t1: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
			),
			t2:        NewNonTerminalTree("a"),
			different: true,
		},
		{
			t1: NewNonTerminalTree("a"),
			t2: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
			),
			different: true,
		},
		{
			t1: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
			),
			t2: NewNonTerminalTree("a",
				NewNonTerminalTree("c"),
			),
			different: true,
		},
		{
			t1: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
				NewNonTerminalTree("c"),
				NewNonTerminalTree("d"),
			),
			t2: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
				NewNonTerminalTree("c"),
			),
			different: true,
		},
		{
			t1: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
				NewNonTerminalTree("c"),
			),
			t2: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
				NewNonTerminalTree("c"),
				NewNonTerminalTree("d"),
			),
			different: true,
		},
		{
			t1: NewNonTerminalTree("a",
				NewNonTerminalTree("b",
					NewNonTerminalTree("c"),
				),
			),
			t2: NewNonTerminalTree("a",
				NewNonTerminalTree("b",
					NewNonTerminalTree("d"),
				),
			),
			different: true,
		},
		{
			t1: NewNonTerminalTree("a",
				NewNonTerminalTree("_"),
				NewNonTerminalTree("c"),
			),
			t2: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
				NewNonTerminalTree("x"),
			),
			different: true,
		},
		{
			t1: NewNonTerminalTree("_"),
			t2: NewNonTerminalTree("a",
				NewNonTerminalTree("b"),
			),
			different: true,
		},
		{
			t1: NewNonTerminalTree("a",
				NewNonTerminalTree("_"),
			),
			t2: NewNonTerminalTree("a",
				NewNonTerminalTree("b",
					NewNonTerminalTree("c"),
				),
			),
			different: true,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			diffs := DiffTree(tt.t1, tt.t2)
			if tt.different && len(diffs) == 0 {
				t.Fatalf("unexpected result")
			} else if !tt.different && len(diffs) > 0 {
				t.Fatalf("unexpected result")
			}
		})
	}
}

func TestParseTestCase(t *testing.T) {
	tests := []struct {
		src      string
		tc       *TestCase
		parseErr bool
	}{
		{
			src: `test
---
foo
---
(foo)
`,
			tc: &TestCase{
				Description: "test",
				Source:      []byte("foo"),
				Output:      NewNonTerminalTree("foo").Fill(),
			},
		},
		{
			src: `
test

---

foo

---

(foo)

`,
			tc: &TestCase{
				Description: "\ntest\n",
				Source:      []byte("\nfoo\n"),
				Output:      NewNonTerminalTree("foo").Fill(),
			},
		},
		// The length of a part delimiter may be greater than 3.
		{
			src: `
test
----
foo
----
(foo)
`,
			tc: &TestCase{
				Description: "\ntest",
				Source:      []byte("foo"),
				Output:      NewNonTerminalTree("foo").Fill(),
			},
		},
		// The description part may be empty.
		{
			src: `----
foo
----
(foo)
`,
			tc: &TestCase{
				Description: "",
				Source:      []byte("foo"),
				Output:      NewNonTerminalTree("foo").Fill(),
			},
		},
		// The source part may be empty.
		{
			src: `test
---
---
(foo)
`,
			tc: &TestCase{
				Description: "test",
				Source:      []byte{},
				Output:      NewNonTerminalTree("foo").Fill(),
			},
		},
		// NOTE: If there is a delimiter at the end of a test case, we really want to make it a syntax error,
		// but we allow it to simplify the implementation of the parser.
		{
			src: `test
----
foo
----
(foo)
---
`,
			tc: &TestCase{
				Description: "test",
				Source:      []byte("foo"),
				Output:      NewNonTerminalTree("foo").Fill(),
			},
		},
		{
			src:      ``,
			parseErr: true,
		},
		{
			src: `test
---
`,
			parseErr: true,
		},
		{
			src: `test
---
foo
`,
			parseErr: true,
		},
		{
			src: `test
---
foo
---
`,
			parseErr: true,
		},
		{
			src: `test
--
foo
--
(foo)
`,
			parseErr: true,
		},
		{
			src: `test
---
foo
---
?
`,
			parseErr: true,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			tc, err := ParseTestCase(strings.NewReader(tt.src))
			if tt.parseErr {
				if err == nil {
					t.Fatalf("an expected error didn't occur")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				testTestCase(t, tt.tc, tc)
			}
		})
	}
}

func testTestCase(t *testing.T, expected, actual *TestCase) {
	t.Helper()

	if expected.Description != actual.Description ||
		!reflect.DeepEqual(expected.Source, actual.Source) ||
		len(DiffTree(expected.Output, actual.Output)) > 0 {
		t.Fatalf("unexpected test case: want: %#v, got: %#v", expected, actual)
	}
}
