package test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestDiffTree(t *testing.T) {
	tests := []struct {
		t1        *Tree
		t2        *Tree
		different bool
	}{
		{
			t1: NewTree("a"),
			t2: NewTree("a"),
		},
		{
			t1: NewTree("a",
				NewTree("b"),
			),
			t2: NewTree("a",
				NewTree("b"),
			),
		},
		{
			t1: NewTree("a",
				NewTree("b"),
				NewTree("c"),
				NewTree("d"),
			),
			t2: NewTree("a",
				NewTree("b"),
				NewTree("c"),
				NewTree("d"),
			),
		},
		{
			t1: NewTree("a",
				NewTree("b",
					NewTree("c"),
				),
				NewTree("d",
					NewTree("d"),
				),
			),
			t2: NewTree("a",
				NewTree("b",
					NewTree("c"),
				),
				NewTree("d",
					NewTree("d"),
				),
			),
		},
		{
			t1:        NewTree("a"),
			t2:        NewTree("b"),
			different: true,
		},
		{
			t1: NewTree("a",
				NewTree("b"),
			),
			t2:        NewTree("a"),
			different: true,
		},
		{
			t1: NewTree("a"),
			t2: NewTree("a",
				NewTree("b"),
			),
			different: true,
		},
		{
			t1: NewTree("a",
				NewTree("b"),
			),
			t2: NewTree("a",
				NewTree("c"),
			),
			different: true,
		},
		{
			t1: NewTree("a",
				NewTree("b"),
				NewTree("c"),
				NewTree("d"),
			),
			t2: NewTree("a",
				NewTree("b"),
				NewTree("c"),
			),
			different: true,
		},
		{
			t1: NewTree("a",
				NewTree("b"),
				NewTree("c"),
			),
			t2: NewTree("a",
				NewTree("b"),
				NewTree("c"),
				NewTree("d"),
			),
			different: true,
		},
		{
			t1: NewTree("a",
				NewTree("b",
					NewTree("c"),
				),
			),
			t2: NewTree("a",
				NewTree("b",
					NewTree("d"),
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
				Output:      NewTree("foo").Fill(),
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
				Output:      NewTree("foo").Fill(),
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
				Output:      NewTree("foo").Fill(),
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
				Output:      NewTree("foo").Fill(),
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
				Output:      NewTree("foo").Fill(),
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
				Output:      NewTree("foo").Fill(),
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
