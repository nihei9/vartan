package driver

import (
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec"
)

func TestParserWithLAC(t *testing.T) {
	specSrc := `
#name test;

S
    : C C
	;
C
    : c C
	| d
	;

c: 'c';
d: 'd';
`

	src := `ccd`

	actLogWithLAC := []string{
		"shift/c",
		"shift/c",
		"shift/d",
		"miss",
	}

	actLogWithoutLAC := []string{
		"shift/c",
		"shift/c",
		"shift/d",
		"reduce/C",
		"reduce/C",
		"reduce/C",
		"miss",
	}

	ast, err := spec.Parse(strings.NewReader(specSrc))
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

	gram, err := grammar.Compile(g, grammar.SpecifyClass(grammar.ClassLALR))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("LAC is enabled", func(t *testing.T) {
		semAct := &testSemAct{
			gram: gram,
		}

		toks, err := NewTokenStream(gram, strings.NewReader(src))
		if err != nil {
			t.Fatal(err)
		}

		p, err := NewParser(toks, NewGrammar(gram), SemanticAction(semAct))
		if err != nil {
			t.Fatal(err)
		}

		err = p.Parse()
		if err != nil {
			t.Fatal(err)
		}

		if len(semAct.actLog) != len(actLogWithLAC) {
			t.Fatalf("unexpected action log; want: %+v, got: %+v", actLogWithLAC, semAct.actLog)
		}

		for i, e := range actLogWithLAC {
			if semAct.actLog[i] != e {
				t.Fatalf("unexpected action log; want: %+v, got: %+v", actLogWithLAC, semAct.actLog)
			}
		}
	})

	t.Run("LAC is disabled", func(t *testing.T) {
		semAct := &testSemAct{
			gram: gram,
		}

		toks, err := NewTokenStream(gram, strings.NewReader(src))
		if err != nil {
			t.Fatal(err)
		}

		p, err := NewParser(toks, NewGrammar(gram), SemanticAction(semAct), DisableLAC())
		if err != nil {
			t.Fatal(err)
		}

		err = p.Parse()
		if err != nil {
			t.Fatal(err)
		}

		if len(semAct.actLog) != len(actLogWithoutLAC) {
			t.Fatalf("unexpected action log; want: %+v, got: %+v", actLogWithoutLAC, semAct.actLog)
		}

		for i, e := range actLogWithoutLAC {
			if semAct.actLog[i] != e {
				t.Fatalf("unexpected action log; want: %+v, got: %+v", actLogWithoutLAC, semAct.actLog)
			}
		}
	})
}
