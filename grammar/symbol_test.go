package grammar

import "testing"

func TestSymbol(t *testing.T) {
	tab := newSymbolTable()
	w := tab.writer()
	_, _ = w.registerStartSymbol("expr'")
	_, _ = w.registerNonTerminalSymbol("expr")
	_, _ = w.registerNonTerminalSymbol("term")
	_, _ = w.registerNonTerminalSymbol("factor")
	_, _ = w.registerTerminalSymbol("id")
	_, _ = w.registerTerminalSymbol("add")
	_, _ = w.registerTerminalSymbol("mul")
	_, _ = w.registerTerminalSymbol("l_paren")
	_, _ = w.registerTerminalSymbol("r_paren")

	nonTermTexts := []string{
		"", // Nil
		"expr'",
		"expr",
		"term",
		"factor",
	}

	termTexts := []string{
		"",            // Nil
		symbolNameEOF, // EOF
		"id",
		"add",
		"mul",
		"l_paren",
		"r_paren",
	}

	tests := []struct {
		text          string
		isNil         bool
		isStart       bool
		isEOF         bool
		isNonTerminal bool
		isTerminal    bool
	}{
		{
			text:          "expr'",
			isStart:       true,
			isNonTerminal: true,
		},
		{
			text:          "expr",
			isNonTerminal: true,
		},
		{
			text:          "term",
			isNonTerminal: true,
		},
		{
			text:          "factor",
			isNonTerminal: true,
		},
		{
			text:       "id",
			isTerminal: true,
		},
		{
			text:       "add",
			isTerminal: true,
		},
		{
			text:       "mul",
			isTerminal: true,
		},
		{
			text:       "l_paren",
			isTerminal: true,
		},
		{
			text:       "r_paren",
			isTerminal: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			r := tab.reader()
			sym, ok := r.toSymbol(tt.text)
			if !ok {
				t.Fatalf("symbol was not found")
			}
			testSymbolProperty(t, sym, tt.isNil, tt.isStart, tt.isEOF, tt.isNonTerminal, tt.isTerminal)
			text, ok := r.toText(sym)
			if !ok {
				t.Fatalf("text was not found")
			}
			if text != tt.text {
				t.Fatalf("unexpected text representation; want: %v, got: %v", tt.text, text)
			}
		})
	}

	t.Run("EOF", func(t *testing.T) {
		testSymbolProperty(t, symbolEOF, false, false, true, false, true)
	})

	t.Run("Nil", func(t *testing.T) {
		testSymbolProperty(t, symbolNil, true, false, false, false, false)
	})

	t.Run("texts of non-terminals", func(t *testing.T) {
		r := tab.reader()
		ts, err := r.nonTerminalTexts()
		if err != nil {
			t.Fatal(err)
		}
		if len(ts) != len(nonTermTexts) {
			t.Fatalf("unexpected non-terminal count; want: %v (%#v), got: %v (%#v)", len(nonTermTexts), nonTermTexts, len(ts), ts)
		}
		for i, text := range ts {
			if text != nonTermTexts[i] {
				t.Fatalf("unexpected non-terminal; want: %v, got: %v", nonTermTexts[i], text)
			}
		}
	})

	t.Run("texts of terminals", func(t *testing.T) {
		r := tab.reader()
		ts, err := r.terminalTexts()
		if err != nil {
			t.Fatal(err)
		}
		if len(ts) != len(termTexts) {
			t.Fatalf("unexpected terminal count; want: %v (%#v), got: %v (%#v)", len(termTexts), termTexts, len(ts), ts)
		}
		for i, text := range ts {
			if text != termTexts[i] {
				t.Fatalf("unexpected terminal; want: %v, got: %v", termTexts[i], text)
			}
		}
	})
}

func testSymbolProperty(t *testing.T, sym symbol, isNil, isStart, isEOF, isNonTerminal, isTerminal bool) {
	t.Helper()

	if v := sym.isNil(); v != isNil {
		t.Fatalf("isNil property is mismatched; want: %v, got: %v", isNil, v)
	}
	if v := sym.isStart(); v != isStart {
		t.Fatalf("isStart property is mismatched; want: %v, got: %v", isStart, v)
	}
	if v := sym.isEOF(); v != isEOF {
		t.Fatalf("isEOF property is mismatched; want: %v, got: %v", isEOF, v)
	}
	if v := sym.isNonTerminal(); v != isNonTerminal {
		t.Fatalf("isNonTerminal property is mismatched; want: %v, got: %v", isNonTerminal, v)
	}
	if v := sym.isTerminal(); v != isTerminal {
		t.Fatalf("isTerminal property is mismatched; want: %v, got: %v", isTerminal, v)
	}
}
