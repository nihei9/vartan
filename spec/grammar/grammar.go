package grammar

import "strconv"

type CompiledGrammar struct {
	Name      string         `json:"name"`
	Lexical   *LexicalSpec   `json:"lexical"`
	Syntactic *SyntacticSpec `json:"syntactic"`
	ASTAction *ASTAction     `json:"ast_action"`
}

// StateID represents an ID of a state of a transition table.
type StateID int

const (
	// StateIDNil represents an empty entry of a transition table.
	// When the driver reads this value, it raises an error meaning lexical analysis failed.
	StateIDNil = StateID(0)

	// StateIDMin is the minimum value of the state ID. All valid state IDs are represented as
	// sequential numbers starting from this value.
	StateIDMin = StateID(1)
)

func (id StateID) Int() int {
	return int(id)
}

// LexModeID represents an ID of a lex mode.
type LexModeID int

const (
	LexModeIDNil     = LexModeID(0)
	LexModeIDDefault = LexModeID(1)
)

func (n LexModeID) String() string {
	return strconv.Itoa(int(n))
}

func (n LexModeID) Int() int {
	return int(n)
}

func (n LexModeID) IsNil() bool {
	return n == LexModeIDNil
}

// LexModeName represents a name of a lex mode.
type LexModeName string

const (
	LexModeNameNil     = LexModeName("")
	LexModeNameDefault = LexModeName("default")
)

func (m LexModeName) String() string {
	return string(m)
}

// LexKindID represents an ID of a lexical kind and is unique across all modes.
type LexKindID int

const (
	LexKindIDNil = LexKindID(0)
	LexKindIDMin = LexKindID(1)
)

func (id LexKindID) Int() int {
	return int(id)
}

// LexModeKindID represents an ID of a lexical kind and is unique within a mode.
// Use LexKindID to identify a kind across all modes uniquely.
type LexModeKindID int

const (
	LexModeKindIDNil = LexModeKindID(0)
	LexModeKindIDMin = LexModeKindID(1)
)

func (id LexModeKindID) Int() int {
	return int(id)
}

// LexKindName represents a name of a lexical kind.
type LexKindName string

const LexKindNameNil = LexKindName("")

func (k LexKindName) String() string {
	return string(k)
}

type RowDisplacementTable struct {
	OriginalRowCount int       `json:"original_row_count"`
	OriginalColCount int       `json:"original_col_count"`
	EmptyValue       StateID   `json:"empty_value"`
	Entries          []StateID `json:"entries"`
	Bounds           []int     `json:"bounds"`
	RowDisplacement  []int     `json:"row_displacement"`
}

type UniqueEntriesTable struct {
	UniqueEntries             *RowDisplacementTable `json:"unique_entries,omitempty"`
	UncompressedUniqueEntries []StateID             `json:"uncompressed_unique_entries,omitempty"`
	RowNums                   []int                 `json:"row_nums"`
	OriginalRowCount          int                   `json:"original_row_count"`
	OriginalColCount          int                   `json:"original_col_count"`
	EmptyValue                int                   `json:"empty_value"`
}

type TransitionTable struct {
	InitialStateID         StateID             `json:"initial_state_id"`
	AcceptingStates        []LexModeKindID     `json:"accepting_states"`
	RowCount               int                 `json:"row_count"`
	ColCount               int                 `json:"col_count"`
	Transition             *UniqueEntriesTable `json:"transition,omitempty"`
	UncompressedTransition []StateID           `json:"uncompressed_transition,omitempty"`
}

type CompiledLexModeSpec struct {
	KindNames []LexKindName    `json:"kind_names"`
	Push      []LexModeID      `json:"push"`
	Pop       []int            `json:"pop"`
	DFA       *TransitionTable `json:"dfa"`
}

type LexicalSpec struct {
	InitialModeID    LexModeID              `json:"initial_mode_id"`
	ModeNames        []LexModeName          `json:"mode_names"`
	KindNames        []LexKindName          `json:"kind_names"`
	KindIDs          [][]LexKindID          `json:"kind_ids"`
	CompressionLevel int                    `json:"compression_level"`
	Specs            []*CompiledLexModeSpec `json:"specs"`
}

type SyntacticSpec struct {
	Action                  []int    `json:"action"`
	GoTo                    []int    `json:"goto"`
	StateCount              int      `json:"state_count"`
	InitialState            int      `json:"initial_state"`
	StartProduction         int      `json:"start_production"`
	LHSSymbols              []int    `json:"lhs_symbols"`
	AlternativeSymbolCounts []int    `json:"alternative_symbol_counts"`
	Terminals               []string `json:"terminals"`
	TerminalCount           int      `json:"terminal_count"`
	TerminalSkip            []int    `json:"terminal_skip"`
	KindToTerminal          []int    `json:"kind_to_terminal"`
	NonTerminals            []string `json:"non_terminals"`
	NonTerminalCount        int      `json:"non_terminal_count"`
	EOFSymbol               int      `json:"eof_symbol"`
	ErrorSymbol             int      `json:"error_symbol"`
	ErrorTrapperStates      []int    `json:"error_trapper_states"`
	RecoverProductions      []int    `json:"recover_productions"`
}

type ASTAction struct {
	Entries [][]int `json:"entries"`
}
