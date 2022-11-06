package lexical

import (
	"encoding/json"
	"fmt"
	"testing"

	spec "github.com/nihei9/vartan/spec/grammar"
)

func TestLexSpec_Validate(t *testing.T) {
	// We expect that the spelling inconsistency error will occur.
	spec := &LexSpec{
		Entries: []*LexEntry{
			{
				Modes: []spec.LexModeName{
					// 'Default' is the spelling inconsistency because 'default' is predefined.
					"Default",
				},
				Kind:    "foo",
				Pattern: "foo",
			},
		},
	}
	err := spec.Validate()
	if err == nil {
		t.Fatalf("expected error didn't occur")
	}
}

func TestSnakeCaseToUpperCamelCase(t *testing.T) {
	tests := []struct {
		snake string
		camel string
	}{
		{
			snake: "foo",
			camel: "Foo",
		},
		{
			snake: "foo_bar",
			camel: "FooBar",
		},
		{
			snake: "foo_bar_baz",
			camel: "FooBarBaz",
		},
		{
			snake: "Foo",
			camel: "Foo",
		},
		{
			snake: "fooBar",
			camel: "FooBar",
		},
		{
			snake: "FOO",
			camel: "FOO",
		},
		{
			snake: "FOO_BAR",
			camel: "FOOBAR",
		},
		{
			snake: "_foo_bar_",
			camel: "FooBar",
		},
		{
			snake: "___foo___bar___",
			camel: "FooBar",
		},
	}
	for _, tt := range tests {
		c := SnakeCaseToUpperCamelCase(tt.snake)
		if c != tt.camel {
			t.Errorf("unexpected string; want: %v, got: %v", tt.camel, c)
		}
	}
}

func TestFindSpellingInconsistencies(t *testing.T) {
	tests := []struct {
		ids        []string
		duplicated [][]string
	}{
		{
			ids:        []string{"foo", "foo"},
			duplicated: nil,
		},
		{
			ids:        []string{"foo", "Foo"},
			duplicated: [][]string{{"Foo", "foo"}},
		},
		{
			ids:        []string{"foo", "foo", "Foo"},
			duplicated: [][]string{{"Foo", "foo"}},
		},
		{
			ids:        []string{"foo_bar_baz", "FooBarBaz"},
			duplicated: [][]string{{"FooBarBaz", "foo_bar_baz"}},
		},
		{
			ids:        []string{"foo", "Foo", "bar", "Bar"},
			duplicated: [][]string{{"Bar", "bar"}, {"Foo", "foo"}},
		},
		{
			ids:        []string{"foo", "Foo", "bar", "Bar", "baz", "bra"},
			duplicated: [][]string{{"Bar", "bar"}, {"Foo", "foo"}},
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			duplicated := FindSpellingInconsistencies(tt.ids)
			if len(duplicated) != len(tt.duplicated) {
				t.Fatalf("unexpected IDs; want: %#v, got: %#v", tt.duplicated, duplicated)
			}
			for i, dupIDs := range duplicated {
				if len(dupIDs) != len(tt.duplicated[i]) {
					t.Fatalf("unexpected IDs; want: %#v, got: %#v", tt.duplicated[i], dupIDs)
				}
				for j, id := range dupIDs {
					if id != tt.duplicated[i][j] {
						t.Fatalf("unexpected IDs; want: %#v, got: %#v", tt.duplicated[i], dupIDs)
					}
				}
			}
		})
	}
}

func TestCompile(t *testing.T) {
	tests := []struct {
		Caption string
		Spec    string
		Err     bool
	}{
		{
			Caption: "allow duplicates names between fragments and non-fragments",
			Spec: `
{
    "name": "test",
    "entries": [
        {
            "kind": "a2z",
            "pattern": "\\f{a2z}"
        },
        {
            "fragment": true,
            "kind": "a2z",
            "pattern": "[a-z]"
        }
    ]
}
`,
		},
		{
			Caption: "don't allow duplicates names in non-fragments",
			Spec: `
{
    "name": "test",
    "entries": [
        {
            "kind": "a2z",
            "pattern": "a|b|c|d|e|f|g|h|i|j|k|l|m|n|o|p|q|r|s|t|u|v|w|x|y|z"
        },
        {
            "kind": "a2z",
            "pattern": "[a-z]"
        }
    ]
}
`,
			Err: true,
		},
		{
			Caption: "don't allow duplicates names in fragments",
			Spec: `
{
    "name": "test",
    "entries": [
        {
            "kind": "a2z",
            "pattern": "\\f{a2z}"
        },
        {
            "fragments": true,
            "kind": "a2z",
            "pattern": "a|b|c|d|e|f|g|h|i|j|k|l|m|n|o|p|q|r|s|t|u|v|w|x|y|z"
        },
        {
            "fragments": true,
            "kind": "a2z",
            "pattern": "[a-z]"
        }
    ]
}
`,
			Err: true,
		},
		{
			Caption: "don't allow kind names in the same mode to contain spelling inconsistencies",
			Spec: `
{
    "name": "test",
    "entries": [
        {
            "kind": "foo_1",
            "pattern": "foo_1"
        },
        {
            "kind": "foo1",
            "pattern": "foo1"
        }
    ]
}
`,
			Err: true,
		},
		{
			Caption: "don't allow kind names across modes to contain spelling inconsistencies",
			Spec: `
{
    "name": "test",
    "entries": [
        {
            "modes": ["default"],
            "kind": "foo_1",
            "pattern": "foo_1"
        },
        {
            "modes": ["other_mode"],
            "kind": "foo1",
            "pattern": "foo1"
        }
    ]
}
`,
			Err: true,
		},
		{
			Caption: "don't allow mode names to contain spelling inconsistencies",
			Spec: `
{
    "name": "test",
    "entries": [
        {
            "modes": ["foo_1"],
            "kind": "a",
            "pattern": "a"
        },
        {
            "modes": ["foo1"],
            "kind": "b",
            "pattern": "b"
        }
    ]
}
`,
			Err: true,
		},
		{
			Caption: "allow fragment names in the same mode to contain spelling inconsistencies because fragments will not appear in output files",
			Spec: `
{
    "name": "test",
    "entries": [
        {
            "kind": "a",
            "pattern": "a"
        },
        {
            "fragment": true,
            "kind": "foo_1",
            "pattern": "foo_1"
        },
        {
            "fragment": true,
            "kind": "foo1",
            "pattern": "foo1"
        }
    ]
}
`,
		},
		{
			Caption: "allow fragment names across modes to contain spelling inconsistencies because fragments will not appear in output files",
			Spec: `
{
    "name": "test",
    "entries": [
        {
            "modes": ["default"],
            "kind": "a",
            "pattern": "a"
        },
        {
            "modes": ["default"],
            "fragment": true,
            "kind": "foo_1",
            "pattern": "foo_1"
        },
        {
            "modes": ["other_mode"],
            "fragment": true,
            "kind": "foo1",
            "pattern": "foo1"
        }
    ]
}
`,
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v %s", i, tt.Caption), func(t *testing.T) {
			lspec := &LexSpec{}
			err := json.Unmarshal([]byte(tt.Spec), lspec)
			if err != nil {
				t.Fatalf("%v", err)
			}
			clspec, err, _ := Compile(lspec, CompressionLevelMin)
			if tt.Err {
				if err == nil {
					t.Fatalf("expected an error")
				}
				if clspec != nil {
					t.Fatalf("Compile function mustn't return a compiled specification")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if clspec == nil {
					t.Fatalf("Compile function must return a compiled specification")
				}
			}
		})
	}
}
