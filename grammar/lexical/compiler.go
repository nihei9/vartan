package lexical

import (
	"bytes"
	"fmt"

	"github.com/nihei9/vartan/compressor"
	"github.com/nihei9/vartan/grammar/lexical/dfa"
	psr "github.com/nihei9/vartan/grammar/lexical/parser"
	spec "github.com/nihei9/vartan/spec/grammar"
)

type CompileError struct {
	Kind     spec.LexKindName
	Fragment bool
	Cause    error
	Detail   string
}

func Compile(lexspec *LexSpec, compLv int) (*spec.LexicalSpec, error, []*CompileError) {
	err := lexspec.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid lexical specification:\n%w", err), nil
	}

	modeEntries, modeNames, modeName2ID, fragmetns := groupEntriesByLexMode(lexspec.Entries)

	modeSpecs := []*spec.CompiledLexModeSpec{
		nil,
	}
	for i, es := range modeEntries[1:] {
		modeName := modeNames[i+1]
		modeSpec, err, cerrs := compile(es, modeName2ID, fragmetns, compLv)
		if err != nil {
			return nil, fmt.Errorf("failed to compile in %v mode: %w", modeName, err), cerrs
		}
		modeSpecs = append(modeSpecs, modeSpec)
	}

	var kindNames []spec.LexKindName
	var name2ID map[spec.LexKindName]spec.LexKindID
	{
		name2ID = map[spec.LexKindName]spec.LexKindID{}
		id := spec.LexKindIDMin
		for _, modeSpec := range modeSpecs[1:] {
			for _, name := range modeSpec.KindNames[1:] {
				if _, ok := name2ID[name]; ok {
					continue
				}
				name2ID[name] = id
				id++
			}
		}

		kindNames = make([]spec.LexKindName, len(name2ID)+1)
		for name, id := range name2ID {
			kindNames[id] = name
		}
	}

	var kindIDs [][]spec.LexKindID
	{
		kindIDs = make([][]spec.LexKindID, len(modeSpecs))
		for i, modeSpec := range modeSpecs[1:] {
			ids := make([]spec.LexKindID, len(modeSpec.KindNames))
			for modeID, name := range modeSpec.KindNames {
				if modeID == 0 {
					continue
				}
				ids[modeID] = name2ID[name]
			}
			kindIDs[i+1] = ids
		}
	}

	return &spec.LexicalSpec{
		InitialModeID:    spec.LexModeIDDefault,
		ModeNames:        modeNames,
		KindNames:        kindNames,
		KindIDs:          kindIDs,
		CompressionLevel: compLv,
		Specs:            modeSpecs,
	}, nil, nil
}

func groupEntriesByLexMode(entries []*LexEntry) ([][]*LexEntry, []spec.LexModeName, map[spec.LexModeName]spec.LexModeID, map[spec.LexKindName]*LexEntry) {
	modeNames := []spec.LexModeName{
		spec.LexModeNameNil,
		spec.LexModeNameDefault,
	}
	modeName2ID := map[spec.LexModeName]spec.LexModeID{
		spec.LexModeNameNil:     spec.LexModeIDNil,
		spec.LexModeNameDefault: spec.LexModeIDDefault,
	}
	lastModeID := spec.LexModeIDDefault
	modeEntries := [][]*LexEntry{
		nil,
		{},
	}
	fragments := map[spec.LexKindName]*LexEntry{}
	for _, e := range entries {
		if e.Fragment {
			fragments[e.Kind] = e
			continue
		}
		ms := e.Modes
		if len(ms) == 0 {
			ms = []spec.LexModeName{
				spec.LexModeNameDefault,
			}
		}
		for _, modeName := range ms {
			modeID, ok := modeName2ID[modeName]
			if !ok {
				modeID = lastModeID + 1
				lastModeID = modeID
				modeName2ID[modeName] = modeID
				modeNames = append(modeNames, modeName)
				modeEntries = append(modeEntries, []*LexEntry{})
			}
			modeEntries[modeID] = append(modeEntries[modeID], e)
		}
	}
	return modeEntries, modeNames, modeName2ID, fragments
}

func compile(
	entries []*LexEntry,
	modeName2ID map[spec.LexModeName]spec.LexModeID,
	fragments map[spec.LexKindName]*LexEntry,
	compLv int,
) (*spec.CompiledLexModeSpec, error, []*CompileError) {
	var kindNames []spec.LexKindName
	kindIDToName := map[spec.LexModeKindID]spec.LexKindName{}
	var patterns map[spec.LexModeKindID][]byte
	{
		kindNames = append(kindNames, spec.LexKindNameNil)
		patterns = map[spec.LexModeKindID][]byte{}
		for i, e := range entries {
			kindID := spec.LexModeKindID(i + 1)

			kindNames = append(kindNames, e.Kind)
			kindIDToName[kindID] = e.Kind
			patterns[kindID] = []byte(e.Pattern)
		}
	}

	push := []spec.LexModeID{
		spec.LexModeIDNil,
	}
	pop := []int{
		0,
	}
	for _, e := range entries {
		pushV := spec.LexModeIDNil
		if e.Push != "" {
			pushV = modeName2ID[e.Push]
		}
		push = append(push, pushV)
		popV := 0
		if e.Pop {
			popV = 1
		}
		pop = append(pop, popV)
	}

	fragmentPatterns := map[spec.LexKindName][]byte{}
	for k, e := range fragments {
		fragmentPatterns[k] = []byte(e.Pattern)
	}

	fragmentCPTrees := make(map[spec.LexKindName]psr.CPTree, len(fragmentPatterns))
	{
		var cerrs []*CompileError
		for kind, pat := range fragmentPatterns {
			p := psr.NewParser(kind, bytes.NewReader(pat))
			t, err := p.Parse()
			if err != nil {
				if err == psr.ParseErr {
					detail, cause := p.Error()
					cerrs = append(cerrs, &CompileError{
						Kind:     kind,
						Fragment: true,
						Cause:    cause,
						Detail:   detail,
					})
				} else {
					cerrs = append(cerrs, &CompileError{
						Kind:     kind,
						Fragment: true,
						Cause:    err,
					})
				}
				continue
			}
			fragmentCPTrees[kind] = t
		}
		if len(cerrs) > 0 {
			return nil, fmt.Errorf("compile error"), cerrs
		}

		err := psr.CompleteFragments(fragmentCPTrees)
		if err != nil {
			if err == psr.ParseErr {
				for _, frag := range fragmentCPTrees {
					kind, frags, err := frag.Describe()
					if err != nil {
						return nil, err, nil
					}

					cerrs = append(cerrs, &CompileError{
						Kind:     kind,
						Fragment: true,
						Cause:    fmt.Errorf("fragment contains undefined fragments or cycles"),
						Detail:   fmt.Sprintf("%v", frags),
					})
				}

				return nil, fmt.Errorf("compile error"), cerrs
			}

			return nil, err, nil
		}
	}

	cpTrees := map[spec.LexModeKindID]psr.CPTree{}
	{
		pats := make([]*psr.PatternEntry, len(patterns)+1)
		pats[spec.LexModeKindIDNil] = &psr.PatternEntry{
			ID: spec.LexModeKindIDNil,
		}
		for id, pattern := range patterns {
			pats[id] = &psr.PatternEntry{
				ID:      id,
				Pattern: pattern,
			}
		}

		var cerrs []*CompileError
		for _, pat := range pats {
			if pat.ID == spec.LexModeKindIDNil {
				continue
			}

			p := psr.NewParser(kindIDToName[pat.ID], bytes.NewReader(pat.Pattern))
			t, err := p.Parse()
			if err != nil {
				if err == psr.ParseErr {
					detail, cause := p.Error()
					cerrs = append(cerrs, &CompileError{
						Kind:     kindIDToName[pat.ID],
						Fragment: false,
						Cause:    cause,
						Detail:   detail,
					})
				} else {
					cerrs = append(cerrs, &CompileError{
						Kind:     kindIDToName[pat.ID],
						Fragment: false,
						Cause:    err,
					})
				}
				continue
			}

			complete, err := psr.ApplyFragments(t, fragmentCPTrees)
			if err != nil {
				return nil, err, nil
			}
			if !complete {
				_, frags, err := t.Describe()
				if err != nil {
					return nil, err, nil
				}

				cerrs = append(cerrs, &CompileError{
					Kind:     kindIDToName[pat.ID],
					Fragment: false,
					Cause:    fmt.Errorf("pattern contains undefined fragments"),
					Detail:   fmt.Sprintf("%v", frags),
				})
				continue
			}

			cpTrees[pat.ID] = t
		}
		if len(cerrs) > 0 {
			return nil, fmt.Errorf("compile error"), cerrs
		}
	}

	var tranTab *spec.TransitionTable
	{
		root, symTab, err := dfa.ConvertCPTreeToByteTree(cpTrees)
		if err != nil {
			return nil, err, nil
		}
		d := dfa.GenDFA(root, symTab)
		tranTab, err = dfa.GenTransitionTable(d)
		if err != nil {
			return nil, err, nil
		}
	}

	var err error
	switch compLv {
	case 2:
		tranTab, err = compressTransitionTableLv2(tranTab)
		if err != nil {
			return nil, err, nil
		}
	case 1:
		tranTab, err = compressTransitionTableLv1(tranTab)
		if err != nil {
			return nil, err, nil
		}
	}

	return &spec.CompiledLexModeSpec{
		KindNames: kindNames,
		Push:      push,
		Pop:       pop,
		DFA:       tranTab,
	}, nil, nil
}

const (
	CompressionLevelMin = 0
	CompressionLevelMax = 2
)

func compressTransitionTableLv2(tranTab *spec.TransitionTable) (*spec.TransitionTable, error) {
	ueTab := compressor.NewUniqueEntriesTable()
	{
		orig, err := compressor.NewOriginalTable(convertStateIDSliceToIntSlice(tranTab.UncompressedTransition), tranTab.ColCount)
		if err != nil {
			return nil, err
		}
		err = ueTab.Compress(orig)
		if err != nil {
			return nil, err
		}
	}

	rdTab := compressor.NewRowDisplacementTable(0)
	{
		orig, err := compressor.NewOriginalTable(ueTab.UniqueEntries, ueTab.OriginalColCount)
		if err != nil {
			return nil, err
		}
		err = rdTab.Compress(orig)
		if err != nil {
			return nil, err
		}
	}

	tranTab.Transition = &spec.UniqueEntriesTable{
		UniqueEntries: &spec.RowDisplacementTable{
			OriginalRowCount: rdTab.OriginalRowCount,
			OriginalColCount: rdTab.OriginalColCount,
			EmptyValue:       spec.StateIDNil,
			Entries:          convertIntSliceToStateIDSlice(rdTab.Entries),
			Bounds:           rdTab.Bounds,
			RowDisplacement:  rdTab.RowDisplacement,
		},
		RowNums:          ueTab.RowNums,
		OriginalRowCount: ueTab.OriginalRowCount,
		OriginalColCount: ueTab.OriginalColCount,
	}
	tranTab.UncompressedTransition = nil

	return tranTab, nil
}

func compressTransitionTableLv1(tranTab *spec.TransitionTable) (*spec.TransitionTable, error) {
	ueTab := compressor.NewUniqueEntriesTable()
	{
		orig, err := compressor.NewOriginalTable(convertStateIDSliceToIntSlice(tranTab.UncompressedTransition), tranTab.ColCount)
		if err != nil {
			return nil, err
		}
		err = ueTab.Compress(orig)
		if err != nil {
			return nil, err
		}
	}

	tranTab.Transition = &spec.UniqueEntriesTable{
		UncompressedUniqueEntries: convertIntSliceToStateIDSlice(ueTab.UniqueEntries),
		RowNums:                   ueTab.RowNums,
		OriginalRowCount:          ueTab.OriginalRowCount,
		OriginalColCount:          ueTab.OriginalColCount,
	}
	tranTab.UncompressedTransition = nil

	return tranTab, nil
}

func convertStateIDSliceToIntSlice(s []spec.StateID) []int {
	is := make([]int, len(s))
	for i, v := range s {
		is[i] = v.Int()
	}
	return is
}

func convertIntSliceToStateIDSlice(s []int) []spec.StateID {
	ss := make([]spec.StateID, len(s))
	for i, v := range s {
		ss[i] = spec.StateID(v)
	}
	return ss
}
