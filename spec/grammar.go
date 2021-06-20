package spec

import mlspec "github.com/nihei9/maleeni/spec"

type CompiledGrammar struct {
	LexicalSpecification *LexicalSpecification `json:"lexical_specification"`
	ParsingTable         *ParsingTable         `json:"parsing_table"`
}

type LexicalSpecification struct {
	Lexer   string   `json:"lexer"`
	Maleeni *Maleeni `json:"maleeni"`
}

type Maleeni struct {
	Spec           *mlspec.CompiledLexSpec `json:"spec"`
	KindToTerminal [][]int                 `json:"kind_to_terminal"`
	Skip           [][]int                 `json:"skip"`
}

type ParsingTable struct {
	Action                  []int    `json:"action"`
	GoTo                    []int    `json:"goto"`
	StateCount              int      `json:"state_count"`
	InitialState            int      `json:"initial_state"`
	StartProduction         int      `json:"start_production"`
	LHSSymbols              []int    `json:"lhs_symbols"`
	AlternativeSymbolCounts []int    `json:"alternative_symbol_counts"`
	Terminals               []string `json:"terminals"`
	TerminalCount           int      `json:"terminal_count"`
	NonTerminals            []string `json:"non_terminals"`
	NonTerminalCount        int      `json:"non_terminal_count"`
	EOFSymbol               int      `json:"eof_symbol"`
}
