package lexer

import spec "github.com/nihei9/vartan/spec/grammar"

type lexSpec struct {
	spec *spec.LexicalSpec
}

func NewLexSpec(spec *spec.LexicalSpec) *lexSpec {
	return &lexSpec{
		spec: spec,
	}
}

func (s *lexSpec) InitialMode() ModeID {
	return ModeID(s.spec.InitialModeID.Int())
}

func (s *lexSpec) Pop(mode ModeID, modeKind ModeKindID) bool {
	return s.spec.Specs[mode].Pop[modeKind] == 1
}

func (s *lexSpec) Push(mode ModeID, modeKind ModeKindID) (ModeID, bool) {
	modeID := s.spec.Specs[mode].Push[modeKind]
	return ModeID(modeID.Int()), !modeID.IsNil()
}

func (s *lexSpec) ModeName(mode ModeID) string {
	return s.spec.ModeNames[mode].String()
}

func (s *lexSpec) InitialState(mode ModeID) StateID {
	return StateID(s.spec.Specs[mode].DFA.InitialStateID.Int())
}

func (s *lexSpec) NextState(mode ModeID, state StateID, v int) (StateID, bool) {
	switch s.spec.CompressionLevel {
	case 2:
		tran := s.spec.Specs[mode].DFA.Transition
		rowNum := tran.RowNums[state]
		d := tran.UniqueEntries.RowDisplacement[rowNum]
		if tran.UniqueEntries.Bounds[d+v] != rowNum {
			return StateID(tran.UniqueEntries.EmptyValue.Int()), false
		}
		return StateID(tran.UniqueEntries.Entries[d+v].Int()), true
	case 1:
		tran := s.spec.Specs[mode].DFA.Transition
		next := tran.UncompressedUniqueEntries[tran.RowNums[state]*tran.OriginalColCount+v]
		if next == spec.StateIDNil {
			return StateID(spec.StateIDNil.Int()), false
		}
		return StateID(next.Int()), true
	}

	modeSpec := s.spec.Specs[mode]
	next := modeSpec.DFA.UncompressedTransition[state.Int()*modeSpec.DFA.ColCount+v]
	if next == spec.StateIDNil {
		return StateID(spec.StateIDNil), false
	}
	return StateID(next.Int()), true
}

func (s *lexSpec) Accept(mode ModeID, state StateID) (ModeKindID, bool) {
	modeKindID := s.spec.Specs[mode].DFA.AcceptingStates[state]
	return ModeKindID(modeKindID.Int()), modeKindID != spec.LexModeKindIDNil
}

func (s *lexSpec) KindIDAndName(mode ModeID, modeKind ModeKindID) (KindID, string) {
	kindID := s.spec.KindIDs[mode][modeKind]
	return KindID(kindID.Int()), s.spec.KindNames[kindID].String()
}
