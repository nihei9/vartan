package grammar

import (
	"fmt"
)

type symbolKind string

const (
	symbolKindNonTerminal = symbolKind("non-terminal")
	symbolKindTerminal    = symbolKind("terminal")
)

func (t symbolKind) String() string {
	return string(t)
}

type symbolNum uint16

func (n symbolNum) Int() int {
	return int(n)
}

type symbol uint16

func (s symbol) String() string {
	kind, isStart, isEOF, num := s.describe()
	var prefix string
	switch {
	case isStart:
		prefix = "s"
	case isEOF:
		prefix = "e"
	case kind == symbolKindNonTerminal:
		prefix = "n"
	case kind == symbolKindTerminal:
		prefix = "t"
	default:
		prefix = "?"
	}
	return fmt.Sprintf("%v%v", prefix, num)
}

const (
	maskKindPart    = uint16(0x8000) // 1000 0000 0000 0000
	maskNonTerminal = uint16(0x0000) // 0000 0000 0000 0000
	maskTerminal    = uint16(0x8000) // 1000 0000 0000 0000

	maskSubKindpart    = uint16(0x4000) // 0100 0000 0000 0000
	maskNonStartAndEOF = uint16(0x0000) // 0000 0000 0000 0000
	maskStartOrEOF     = uint16(0x4000) // 0100 0000 0000 0000

	maskNumberPart = uint16(0x3fff) // 0011 1111 1111 1111

	symbolNil   = symbol(0)      // 0000 0000 0000 0000
	symbolStart = symbol(0x4001) // 0100 0000 0000 0001
	symbolEOF   = symbol(0xc001) // 1100 0000 0000 0001: The EOF symbol is treated as a terminal symbol.

	nonTerminalNumMin = symbolNum(2)           // The number 1 is used by a start symbol.
	terminalNumMin    = symbolNum(2)           // The number 1 is used by the EOF symbol.
	symbolNumMax      = symbolNum(0xffff) >> 2 // 0011 1111 1111 1111
)

func newSymbol(kind symbolKind, isStart bool, num symbolNum) (symbol, error) {
	if num > symbolNumMax {
		return symbolNil, fmt.Errorf("a symbol number exceeds the limit; limit: %v, passed: %v", symbolNumMax, num)
	}
	if kind == symbolKindTerminal && isStart {
		return symbolNil, fmt.Errorf("a start symbol must be a non-terminal symbol")
	}

	kindMask := maskNonTerminal
	if kind == symbolKindTerminal {
		kindMask = maskTerminal
	}
	startMask := maskNonStartAndEOF
	if isStart {
		startMask = maskStartOrEOF
	}
	return symbol(kindMask | startMask | uint16(num)), nil
}

func (s symbol) num() symbolNum {
	_, _, _, num := s.describe()
	return num
}

func (s symbol) byte() []byte {
	if s.isNil() {
		return []byte{0, 0}
	}
	return []byte{byte(uint16(s) >> 8), byte(uint16(s) & 0x00ff)}
}

func (s symbol) isNil() bool {
	_, _, _, num := s.describe()
	return num == 0
}

func (s symbol) isStart() bool {
	if s.isNil() {
		return false
	}
	_, isStart, _, _ := s.describe()
	return isStart
}

func (s symbol) isEOF() bool {
	if s.isNil() {
		return false
	}
	_, _, isEOF, _ := s.describe()
	return isEOF
}

func (s symbol) isNonTerminal() bool {
	if s.isNil() {
		return false
	}
	kind, _, _, _ := s.describe()
	if kind == symbolKindNonTerminal {
		return true
	}
	return false
}

func (s symbol) isTerminal() bool {
	if s.isNil() {
		return false
	}
	return !s.isNonTerminal()
}

func (s symbol) describe() (symbolKind, bool, bool, symbolNum) {
	kind := symbolKindNonTerminal
	if uint16(s)&maskKindPart > 0 {
		kind = symbolKindTerminal
	}
	isStart := false
	isEOF := false
	if uint16(s)&maskSubKindpart > 0 {
		if kind == symbolKindNonTerminal {
			isStart = true
		} else {
			isEOF = true
		}
	}
	num := symbolNum(uint16(s) & maskNumberPart)
	return kind, isStart, isEOF, num
}

type symbolTable struct {
	text2Sym     map[string]symbol
	sym2Text     map[symbol]string
	nonTermTexts []string
	termTexts    []string
	nonTermNum   symbolNum
	termNum      symbolNum
}

func newSymbolTable() *symbolTable {
	return &symbolTable{
		text2Sym: map[string]symbol{},
		sym2Text: map[symbol]string{},
		termTexts: []string{
			"", // Nil
			"", // EOF
		},
		nonTermTexts: []string{
			"", // Nil
			"", // Start Symbol
		},
		nonTermNum: nonTerminalNumMin,
		termNum:    terminalNumMin,
	}
}

func (t *symbolTable) registerStartSymbol(text string) (symbol, error) {
	t.text2Sym[text] = symbolStart
	t.sym2Text[symbolStart] = text
	t.nonTermTexts[symbolStart.num().Int()] = text
	return symbolStart, nil
}

func (t *symbolTable) registerNonTerminalSymbol(text string) (symbol, error) {
	if sym, ok := t.text2Sym[text]; ok {
		return sym, nil
	}
	sym, err := newSymbol(symbolKindNonTerminal, false, t.nonTermNum)
	if err != nil {
		return symbolNil, err
	}
	t.nonTermNum++
	t.text2Sym[text] = sym
	t.sym2Text[sym] = text
	t.nonTermTexts = append(t.nonTermTexts, text)
	return sym, nil
}

func (t *symbolTable) registerTerminalSymbol(text string) (symbol, error) {
	if sym, ok := t.text2Sym[text]; ok {
		return sym, nil
	}
	sym, err := newSymbol(symbolKindTerminal, false, t.termNum)
	if err != nil {
		return symbolNil, err
	}
	t.termNum++
	t.text2Sym[text] = sym
	t.sym2Text[sym] = text
	t.termTexts = append(t.termTexts, text)
	return sym, nil
}

func (t *symbolTable) toSymbol(text string) (symbol, bool) {
	if sym, ok := t.text2Sym[text]; ok {
		return sym, true
	}
	return symbolNil, false
}

func (t *symbolTable) toText(sym symbol) (string, bool) {
	text, ok := t.sym2Text[sym]
	return text, ok
}

func (t *symbolTable) getTerminalTexts() ([]string, error) {
	if t.termNum == terminalNumMin {
		return nil, fmt.Errorf("symbol table has no terminals")
	}
	return t.termTexts, nil
}

func (t *symbolTable) getNonTerminalTexts() ([]string, error) {
	if t.nonTermNum == nonTerminalNumMin || t.nonTermTexts[symbolStart.num().Int()] == "" {
		return nil, fmt.Errorf("symbol table has no terminals or no start symbol")
	}
	return t.nonTermTexts, nil
}
