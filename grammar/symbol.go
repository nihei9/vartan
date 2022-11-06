package grammar

import (
	"fmt"
	"sort"
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

	symbolNumStart = uint16(0x0001) // 0000 0000 0000 0001
	symbolNumEOF   = uint16(0x0001) // 0000 0000 0000 0001

	symbolNil   = symbol(0)                                                 // 0000 0000 0000 0000
	symbolStart = symbol(maskNonTerminal | maskStartOrEOF | symbolNumStart) // 0100 0000 0000 0001
	symbolEOF   = symbol(maskTerminal | maskStartOrEOF | symbolNumEOF)      // 1100 0000 0000 0001: The EOF symbol is treated as a terminal symbol.

	// The symbol name contains `<` and `>` to avoid conflicting with user-defined symbols.
	symbolNameEOF = "<eof>"

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
	return kind == symbolKindNonTerminal
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

type symbolTableWriter struct {
	*symbolTable
}

type symbolTableReader struct {
	*symbolTable
}

func newSymbolTable() *symbolTable {
	return &symbolTable{
		text2Sym: map[string]symbol{
			symbolNameEOF: symbolEOF,
		},
		sym2Text: map[symbol]string{
			symbolEOF: symbolNameEOF,
		},
		termTexts: []string{
			"",            // Nil
			symbolNameEOF, // EOF
		},
		nonTermTexts: []string{
			"", // Nil
			"", // Start Symbol
		},
		nonTermNum: nonTerminalNumMin,
		termNum:    terminalNumMin,
	}
}

func (t *symbolTable) writer() *symbolTableWriter {
	return &symbolTableWriter{
		symbolTable: t,
	}
}

func (t *symbolTable) reader() *symbolTableReader {
	return &symbolTableReader{
		symbolTable: t,
	}
}

func (w *symbolTableWriter) registerStartSymbol(text string) (symbol, error) {
	w.text2Sym[text] = symbolStart
	w.sym2Text[symbolStart] = text
	w.nonTermTexts[symbolStart.num().Int()] = text
	return symbolStart, nil
}

func (w *symbolTableWriter) registerNonTerminalSymbol(text string) (symbol, error) {
	if sym, ok := w.text2Sym[text]; ok {
		return sym, nil
	}
	sym, err := newSymbol(symbolKindNonTerminal, false, w.nonTermNum)
	if err != nil {
		return symbolNil, err
	}
	w.nonTermNum++
	w.text2Sym[text] = sym
	w.sym2Text[sym] = text
	w.nonTermTexts = append(w.nonTermTexts, text)
	return sym, nil
}

func (w *symbolTableWriter) registerTerminalSymbol(text string) (symbol, error) {
	if sym, ok := w.text2Sym[text]; ok {
		return sym, nil
	}
	sym, err := newSymbol(symbolKindTerminal, false, w.termNum)
	if err != nil {
		return symbolNil, err
	}
	w.termNum++
	w.text2Sym[text] = sym
	w.sym2Text[sym] = text
	w.termTexts = append(w.termTexts, text)
	return sym, nil
}

func (r *symbolTableReader) toSymbol(text string) (symbol, bool) {
	if sym, ok := r.text2Sym[text]; ok {
		return sym, true
	}
	return symbolNil, false
}

func (r *symbolTableReader) toText(sym symbol) (string, bool) {
	text, ok := r.sym2Text[sym]
	return text, ok
}

func (r *symbolTableReader) terminalSymbols() []symbol {
	syms := make([]symbol, 0, r.termNum.Int()-terminalNumMin.Int())
	for sym := range r.sym2Text {
		if !sym.isTerminal() || sym.isNil() {
			continue
		}
		syms = append(syms, sym)
	}
	sort.Slice(syms, func(i, j int) bool {
		return syms[i] < syms[j]
	})
	return syms
}

func (r *symbolTableReader) terminalTexts() ([]string, error) {
	if r.termNum == terminalNumMin {
		return nil, fmt.Errorf("symbol table has no terminals")
	}
	return r.termTexts, nil
}

func (r *symbolTableReader) nonTerminalSymbols() []symbol {
	syms := make([]symbol, 0, r.nonTermNum.Int()-nonTerminalNumMin.Int())
	for sym := range r.sym2Text {
		if !sym.isNonTerminal() || sym.isNil() {
			continue
		}
		syms = append(syms, sym)
	}
	sort.Slice(syms, func(i, j int) bool {
		return syms[i] < syms[j]
	})
	return syms
}

func (r *symbolTableReader) nonTerminalTexts() ([]string, error) {
	if r.nonTermNum == nonTerminalNumMin || r.nonTermTexts[symbolStart.num().Int()] == "" {
		return nil, fmt.Errorf("symbol table has no terminals or no start symbol")
	}
	return r.nonTermTexts, nil
}
