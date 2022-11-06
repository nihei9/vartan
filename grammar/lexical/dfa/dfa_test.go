package dfa

import (
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar/lexical/parser"
	spec "github.com/nihei9/vartan/spec/grammar"
)

func TestGenDFA(t *testing.T) {
	p := parser.NewParser(spec.LexKindName("test"), strings.NewReader("(a|b)*abb"))
	cpt, err := p.Parse()
	if err != nil {
		t.Fatal(err)
	}
	bt, symTab, err := ConvertCPTreeToByteTree(map[spec.LexModeKindID]parser.CPTree{
		spec.LexModeKindIDMin: cpt,
	})
	if err != nil {
		t.Fatal(err)
	}
	dfa := GenDFA(bt, symTab)
	if dfa == nil {
		t.Fatalf("DFA is nil")
	}

	symPos := func(n uint16) symbolPosition {
		pos, err := newSymbolPosition(n, false)
		if err != nil {
			panic(err)
		}
		return pos
	}

	endPos := func(n uint16) symbolPosition {
		pos, err := newSymbolPosition(n, true)
		if err != nil {
			panic(err)
		}
		return pos
	}

	s0 := newSymbolPositionSet().add(symPos(1)).add(symPos(2)).add(symPos(3))
	s1 := newSymbolPositionSet().add(symPos(1)).add(symPos(2)).add(symPos(3)).add(symPos(4))
	s2 := newSymbolPositionSet().add(symPos(1)).add(symPos(2)).add(symPos(3)).add(symPos(5))
	s3 := newSymbolPositionSet().add(symPos(1)).add(symPos(2)).add(symPos(3)).add(endPos(6))

	rune2Int := func(char rune, index int) uint8 {
		return uint8([]byte(string(char))[index])
	}

	tranS0 := [256]string{}
	tranS0[rune2Int('a', 0)] = s1.hash()
	tranS0[rune2Int('b', 0)] = s0.hash()

	tranS1 := [256]string{}
	tranS1[rune2Int('a', 0)] = s1.hash()
	tranS1[rune2Int('b', 0)] = s2.hash()

	tranS2 := [256]string{}
	tranS2[rune2Int('a', 0)] = s1.hash()
	tranS2[rune2Int('b', 0)] = s3.hash()

	tranS3 := [256]string{}
	tranS3[rune2Int('a', 0)] = s1.hash()
	tranS3[rune2Int('b', 0)] = s0.hash()

	expectedTranTab := map[string][256]string{
		s0.hash(): tranS0,
		s1.hash(): tranS1,
		s2.hash(): tranS2,
		s3.hash(): tranS3,
	}
	if len(dfa.TransitionTable) != len(expectedTranTab) {
		t.Errorf("transition table is mismatched: want: %v entries, got: %v entries", len(expectedTranTab), len(dfa.TransitionTable))
	}
	for h, eTranTab := range expectedTranTab {
		tranTab, ok := dfa.TransitionTable[h]
		if !ok {
			t.Errorf("no entry; hash: %v", h)
			continue
		}
		if len(tranTab) != len(eTranTab) {
			t.Errorf("transition table is mismatched: hash: %v, want: %v entries, got: %v entries", h, len(eTranTab), len(tranTab))
		}
		for c, eNext := range eTranTab {
			if eNext == "" {
				continue
			}

			next := tranTab[c]
			if next == "" {
				t.Errorf("no enatry: hash: %v, char: %v", h, c)
			}
			if next != eNext {
				t.Errorf("next state is mismatched: want: %v, got: %v", eNext, next)
			}
		}
	}

	if dfa.InitialState != s0.hash() {
		t.Errorf("initial state is mismatched: want: %v, got: %v", s0.hash(), dfa.InitialState)
	}

	accTab := map[string]spec.LexModeKindID{
		s3.hash(): 1,
	}
	if len(dfa.AcceptingStatesTable) != len(accTab) {
		t.Errorf("accepting states are mismatched: want: %v entries, got: %v entries", len(accTab), len(dfa.AcceptingStatesTable))
	}
	for eState, eID := range accTab {
		id, ok := dfa.AcceptingStatesTable[eState]
		if !ok {
			t.Errorf("accepting state is not found: state: %v", eState)
		}
		if id != eID {
			t.Errorf("ID is mismatched: state: %v, want: %v, got: %v", eState, eID, id)
		}
	}
}
