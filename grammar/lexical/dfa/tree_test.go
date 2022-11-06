package dfa

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nihei9/vartan/grammar/lexical/parser"
	spec "github.com/nihei9/vartan/spec/grammar"
)

func TestByteTree(t *testing.T) {
	tests := []struct {
		root     byteTree
		nullable bool
		first    *symbolPositionSet
		last     *symbolPositionSet
	}{
		{
			root:     newSymbolNodeWithPos(0, 1),
			nullable: false,
			first:    newSymbolPositionSet().add(1),
			last:     newSymbolPositionSet().add(1),
		},
		{
			root:     newEndMarkerNodeWithPos(1, 1),
			nullable: false,
			first:    newSymbolPositionSet().add(1),
			last:     newSymbolPositionSet().add(1),
		},
		{
			root: newConcatNode(
				newSymbolNodeWithPos(0, 1),
				newSymbolNodeWithPos(0, 2),
			),
			nullable: false,
			first:    newSymbolPositionSet().add(1),
			last:     newSymbolPositionSet().add(2),
		},
		{
			root: newConcatNode(
				newRepeatNode(newSymbolNodeWithPos(0, 1)),
				newSymbolNodeWithPos(0, 2),
			),
			nullable: false,
			first:    newSymbolPositionSet().add(1).add(2),
			last:     newSymbolPositionSet().add(2),
		},
		{
			root: newConcatNode(
				newSymbolNodeWithPos(0, 1),
				newRepeatNode(newSymbolNodeWithPos(0, 2)),
			),
			nullable: false,
			first:    newSymbolPositionSet().add(1),
			last:     newSymbolPositionSet().add(1).add(2),
		},
		{
			root: newConcatNode(
				newRepeatNode(newSymbolNodeWithPos(0, 1)),
				newRepeatNode(newSymbolNodeWithPos(0, 2)),
			),
			nullable: true,
			first:    newSymbolPositionSet().add(1).add(2),
			last:     newSymbolPositionSet().add(1).add(2),
		},
		{
			root: newAltNode(
				newSymbolNodeWithPos(0, 1),
				newSymbolNodeWithPos(0, 2),
			),
			nullable: false,
			first:    newSymbolPositionSet().add(1).add(2),
			last:     newSymbolPositionSet().add(1).add(2),
		},
		{
			root: newAltNode(
				newRepeatNode(newSymbolNodeWithPos(0, 1)),
				newSymbolNodeWithPos(0, 2),
			),
			nullable: true,
			first:    newSymbolPositionSet().add(1).add(2),
			last:     newSymbolPositionSet().add(1).add(2),
		},
		{
			root: newAltNode(
				newSymbolNodeWithPos(0, 1),
				newRepeatNode(newSymbolNodeWithPos(0, 2)),
			),
			nullable: true,
			first:    newSymbolPositionSet().add(1).add(2),
			last:     newSymbolPositionSet().add(1).add(2),
		},
		{
			root: newAltNode(
				newRepeatNode(newSymbolNodeWithPos(0, 1)),
				newRepeatNode(newSymbolNodeWithPos(0, 2)),
			),
			nullable: true,
			first:    newSymbolPositionSet().add(1).add(2),
			last:     newSymbolPositionSet().add(1).add(2),
		},
		{
			root:     newRepeatNode(newSymbolNodeWithPos(0, 1)),
			nullable: true,
			first:    newSymbolPositionSet().add(1),
			last:     newSymbolPositionSet().add(1),
		},
		{
			root:     newOptionNode(newSymbolNodeWithPos(0, 1)),
			nullable: true,
			first:    newSymbolPositionSet().add(1),
			last:     newSymbolPositionSet().add(1),
		},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("#%v", i), func(t *testing.T) {
			if tt.root.nullable() != tt.nullable {
				t.Errorf("unexpected nullable attribute; want: %v, got: %v", tt.nullable, tt.root.nullable())
			}
			if tt.first.hash() != tt.root.first().hash() {
				t.Errorf("unexpected first positions attribute; want: %v, got: %v", tt.first, tt.root.first())
			}
			if tt.last.hash() != tt.root.last().hash() {
				t.Errorf("unexpected last positions attribute; want: %v, got: %v", tt.last, tt.root.last())
			}
		})
	}
}

func newSymbolNodeWithPos(v byte, pos symbolPosition) *symbolNode {
	n := newSymbolNode(v)
	n.pos = pos
	return n
}

func newEndMarkerNodeWithPos(id int, pos symbolPosition) *endMarkerNode {
	n := newEndMarkerNode(spec.LexModeKindID(id))
	n.pos = pos
	return n
}

func TestFollowAndSymbolTable(t *testing.T) {
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

	{
		followTab := genFollowTable(bt)
		if followTab == nil {
			t.Fatal("follow table is nil")
		}
		expectedFollowTab := followTable{
			1: newSymbolPositionSet().add(symPos(1)).add(symPos(2)).add(symPos(3)),
			2: newSymbolPositionSet().add(symPos(1)).add(symPos(2)).add(symPos(3)),
			3: newSymbolPositionSet().add(symPos(4)),
			4: newSymbolPositionSet().add(symPos(5)),
			5: newSymbolPositionSet().add(endPos(6)),
		}
		testFollowTable(t, expectedFollowTab, followTab)
	}

	{
		entry := func(v byte) byteRange {
			return byteRange{
				from: v,
				to:   v,
			}
		}

		expectedSymTab := &symbolTable{
			symPos2Byte: map[symbolPosition]byteRange{
				symPos(1): entry(byte('a')),
				symPos(2): entry(byte('b')),
				symPos(3): entry(byte('a')),
				symPos(4): entry(byte('b')),
				symPos(5): entry(byte('b')),
			},
			endPos2ID: map[symbolPosition]spec.LexModeKindID{
				endPos(6): 1,
			},
		}
		testSymbolTable(t, expectedSymTab, symTab)
	}
}

func testFollowTable(t *testing.T, expected, actual followTable) {
	if len(actual) != len(expected) {
		t.Errorf("unexpected number of the follow table entries; want: %v, got: %v", len(expected), len(actual))
	}
	for ePos, eSet := range expected {
		aSet, ok := actual[ePos]
		if !ok {
			t.Fatalf("follow entry is not found: position: %v, follow: %v", ePos, eSet)
		}
		if aSet.hash() != eSet.hash() {
			t.Fatalf("follow entry of position %v is mismatched: want: %v, got: %v", ePos, aSet, eSet)
		}
	}
}

func testSymbolTable(t *testing.T, expected, actual *symbolTable) {
	t.Helper()

	if len(actual.symPos2Byte) != len(expected.symPos2Byte) {
		t.Errorf("unexpected symPos2Byte entries: want: %v entries, got: %v entries", len(expected.symPos2Byte), len(actual.symPos2Byte))
	}
	for ePos, eByte := range expected.symPos2Byte {
		byte, ok := actual.symPos2Byte[ePos]
		if !ok {
			t.Errorf("a symbol position entry is not found: %v -> %v", ePos, eByte)
			continue
		}
		if byte.from != eByte.from || byte.to != eByte.to {
			t.Errorf("unexpected symbol position entry: want: %v -> %v, got: %v -> %v", ePos, eByte, ePos, byte)
		}
	}

	if len(actual.endPos2ID) != len(expected.endPos2ID) {
		t.Errorf("unexpected endPos2ID entries: want: %v entries, got: %v entries", len(expected.endPos2ID), len(actual.endPos2ID))
	}
	for ePos, eID := range expected.endPos2ID {
		id, ok := actual.endPos2ID[ePos]
		if !ok {
			t.Errorf("an end position entry is not found: %v -> %v", ePos, eID)
			continue
		}
		if id != eID {
			t.Errorf("unexpected end position entry: want: %v -> %v, got: %v -> %v", ePos, eID, ePos, id)
		}
	}
}
