package driver

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar"
	spec "github.com/nihei9/vartan/spec/grammar"
)

type testSemAct struct {
	gram   *spec.CompiledGrammar
	actLog []string
}

func (a *testSemAct) Shift(tok VToken, recovered bool) {
	t := a.gram.ParsingTable.Terminals[tok.TerminalID()]
	if recovered {
		a.actLog = append(a.actLog, fmt.Sprintf("shift/%v/recovered", t))
	} else {
		a.actLog = append(a.actLog, fmt.Sprintf("shift/%v", t))
	}
}

func (a *testSemAct) Reduce(prodNum int, recovered bool) {
	lhsSym := a.gram.ParsingTable.LHSSymbols[prodNum]
	lhsText := a.gram.ParsingTable.NonTerminals[lhsSym]
	if recovered {
		a.actLog = append(a.actLog, fmt.Sprintf("reduce/%v/recovered", lhsText))
	} else {
		a.actLog = append(a.actLog, fmt.Sprintf("reduce/%v", lhsText))
	}
}

func (a *testSemAct) Accept() {
	a.actLog = append(a.actLog, "accept")
}

func (a *testSemAct) TrapAndShiftError(cause VToken, popped int) {
	a.actLog = append(a.actLog, fmt.Sprintf("trap/%v/shift/error", popped))
}

func (a *testSemAct) MissError(cause VToken) {
	a.actLog = append(a.actLog, "miss")
}

func TestParserWithSemanticAction(t *testing.T) {
	specSrcWithErrorProd := `
#name test;

seq
    : seq elem semicolon
	| elem semicolon
    | error star star semicolon
	| error semicolon #recover
	;
elem
    : char char char
	;

ws #skip
    : "[\u{0009}\u{0020}]+";
semicolon
    : ';';
star
    : '*';
char
    : "[a-z]";
`

	specSrcWithoutErrorProd := `
#name test;

seq
    : seq elem semicolon
	| elem semicolon
	;
elem
    : char char char
	;

ws #skip
    : "[\u{0009}\u{0020}]+";
semicolon
    : ';';
char
    : "[a-z]";
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
			caption: "when a grammar has `error` symbol, the driver calls `TrapAndShiftError`.",
			specSrc: specSrcWithErrorProd,
			src:     `a; b !; c d !; e ! * *; h i j;`,
			actLog: []string{
				"shift/char",
				"trap/1/shift/error",
				"shift/semicolon",
				"reduce/seq/recovered",

				"shift/char",
				"trap/2/shift/error",
				"shift/semicolon",
				"reduce/seq/recovered",

				"shift/char",
				"shift/char",
				"trap/3/shift/error",
				"shift/semicolon",
				"reduce/seq/recovered",

				"shift/char",
				"trap/2/shift/error",
				"shift/star",
				"shift/star",
				// When the driver shifts three times, it recovers from an error.
				"shift/semicolon/recovered",
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
			caption: "when the input doesn't meet the error production, the driver calls `MissError`.",
			specSrc: specSrcWithErrorProd,
			src:     `a !`,
			actLog: []string{
				"shift/char",
				"trap/1/shift/error",

				"miss",
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

			gram, _, err := grammar.Compile(g)
			if err != nil {
				t.Fatal(err)
			}

			toks, err := NewTokenStream(gram, strings.NewReader(tt.src))
			if err != nil {
				t.Fatal(err)
			}

			semAct := &testSemAct{
				gram: gram,
			}
			p, err := NewParser(toks, NewGrammar(gram), SemanticAction(semAct))
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
