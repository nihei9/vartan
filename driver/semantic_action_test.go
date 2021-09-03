package driver

import (
	"fmt"
	"strings"
	"testing"

	mldriver "github.com/nihei9/maleeni/driver"
	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec"
)

type testSemAct struct {
	gram   *spec.CompiledGrammar
	actLog []string
}

func (a *testSemAct) Shift(tok *mldriver.Token) {
	a.actLog = append(a.actLog, fmt.Sprintf("shift/%v", tok.KindName))
}

func (a *testSemAct) ShiftError() {
	a.actLog = append(a.actLog, "shift/error")
}

func (a *testSemAct) Reduce(prodNum int) {
	lhsSym := a.gram.ParsingTable.LHSSymbols[prodNum]
	lhsText := a.gram.ParsingTable.NonTerminals[lhsSym]
	a.actLog = append(a.actLog, fmt.Sprintf("reduce/%v", lhsText))
}

func (a *testSemAct) Accept() {
	a.actLog = append(a.actLog, "accept")
}

func (a *testSemAct) TrapError(n int) {
	a.actLog = append(a.actLog, fmt.Sprintf("trap/%v", n))
}

func (a *testSemAct) MissError() {
	a.actLog = append(a.actLog, "miss")
}

func TestParserWithSemanticAction(t *testing.T) {
	specSrcWithErrorProd := `
seq
    : seq elem semicolon
	| elem semicolon
	| error semicolon #recover
	;
elem
    : char char char
	;

ws: "[\u{0009}\u{0020}]+" #skip;
semicolon: ';';
char: "[a-z]";
`

	specSrcWithoutErrorProd := `
seq
    : seq elem semicolon
	| elem semicolon
	;
elem
    : char char char
	;

ws: "[\u{0009}\u{0020}]+" #skip;
semicolon: ';';
char: "[a-z]";
`

	tests := []struct {
		caption string
		specSrc string
		src     string
		actLog  []string
	}{
		{
			caption: "when an input contains no syntax error, the driver calls `Shift`, `Reduce`, and `Accept`.",
			specSrc: specSrcWithErrorProd,
			src:     `a b c; d e f;`,
			actLog: []string{
				"shift/char",
				"shift/char",
				"shift/char",
				"reduce/elem",
				"shift/semicolon",
				"reduce/seq",

				"shift/char",
				"shift/char",
				"shift/char",
				"reduce/elem",
				"shift/semicolon",
				"reduce/seq",

				"accept",
			},
		},
		{
			caption: "when a grammar has `error` symbol, the driver calls `TrapError` and `ShiftError`.",
			specSrc: specSrcWithErrorProd,
			src:     `a; b !; c d !; e f g;`,
			actLog: []string{
				"shift/char",
				"trap/1",
				"shift/error",
				"shift/semicolon",
				"reduce/seq",

				"shift/char",
				"trap/2",
				"shift/error",
				"shift/semicolon",
				"reduce/seq",

				"shift/char",
				"shift/char",
				"trap/3",
				"shift/error",
				"shift/semicolon",
				"reduce/seq",

				"shift/char",
				"shift/char",
				"shift/char",
				"reduce/elem",
				"shift/semicolon",
				"reduce/seq",

				// Even if the input contains syntax errors, the driver calls `Accept` when the input is accepted
				// according to the error production.
				"accept",
			},
		},
		{
			caption: "the driver doesn't call `Accept` when a syntax error is trapped, but the input doesn't meet the error production",
			specSrc: specSrcWithErrorProd,
			src:     `a !`,
			actLog: []string{
				"shift/char",
				"trap/1",
				"shift/error",
			},
		},
		{
			caption: "when a syntax error isn't trapped, the driver calls `MissError`.",
			specSrc: specSrcWithoutErrorProd,
			src:     `a !`,
			actLog: []string{
				"shift/char",

				"miss",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.caption, func(t *testing.T) {
			ast, err := spec.Parse(strings.NewReader(tt.specSrc))
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

			semAct := &testSemAct{
				gram: gram,
			}
			p, err := NewParser(gram, strings.NewReader(tt.src), SemanticAction(semAct))
			if err != nil {
				t.Fatal(err)
			}

			err = p.Parse()
			if err != nil {
				t.Fatal(err)
			}

			if len(semAct.actLog) != len(tt.actLog) {
				t.Fatalf("unexpected action log; want: %+v, got: %+v", tt.actLog, semAct.actLog)
			}

			for i, e := range tt.actLog {
				if semAct.actLog[i] != e {
					t.Fatalf("unexpected action log; want: %+v, got: %+v", tt.actLog, semAct.actLog)
				}
			}
		})
	}
}