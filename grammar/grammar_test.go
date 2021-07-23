package grammar

import (
	"fmt"
	"testing"
)

func TestResolveQuantifier(t *testing.T) {
	symTab := newSymbolTable()
	symTab.registerTerminalSymbol("a")
	symTab.registerTerminalSymbol("b")
	symTab.registerTerminalSymbol("c")
	genSym := newTestSymbolGenerator(t, symTab)

	tests := []struct {
		caption  string
		syms     []symbol
		opts     []bool
		symsList [][]symbol
	}{
		{
			caption: "a?",
			syms: []symbol{
				genSym("a"),
			},
			opts: []bool{
				true,
			},
			symsList: [][]symbol{
				{genSym("a")},
				{},
			},
		},
		{
			caption: "a? b?",
			syms: []symbol{
				genSym("a"),
				genSym("b"),
			},
			opts: []bool{
				true,
				true,
			},
			symsList: [][]symbol{
				{genSym("a"), genSym("b")},
				{genSym("a")},
				{genSym("b")},
				{},
			},
		},
		{
			caption: "a b? c",
			syms: []symbol{
				genSym("a"),
				genSym("b"),
				genSym("c"),
			},
			opts: []bool{
				false,
				true,
				false,
			},
			symsList: [][]symbol{
				{genSym("a"), genSym("b"), genSym("c")},
				{genSym("a"), genSym("c")},
			},
		},
		{
			caption: "a? b c?",
			syms: []symbol{
				genSym("a"),
				genSym("b"),
				genSym("c"),
			},
			opts: []bool{
				true,
				false,
				true,
			},
			symsList: [][]symbol{
				{genSym("a"), genSym("b"), genSym("c")},
				{genSym("a"), genSym("b")},
				{genSym("b"), genSym("c")},
				{genSym("b")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.caption), func(t *testing.T) {
			l := resolveQuantifiers(tt.syms, tt.opts)
			if len(l) != len(tt.symsList) {
				t.Fatalf("unexpected symbols list; want: %+v, got: %+v", tt.symsList, l)
			}
			for i, eSyms := range tt.symsList {
				syms := l[i]
				if len(syms) != len(eSyms) {
					t.Fatalf("unexpected symbols; want: %+v, got: %+v", eSyms, syms)
				}

				for j, eSym := range eSyms {
					sym := syms[j]
					if sym != eSym {
						t.Fatalf("unexpected symbols; want: %+v, got: %+v", eSyms, syms)
					}
				}
			}
		})
	}
}
