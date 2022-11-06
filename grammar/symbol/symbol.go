package symbol

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

type SymbolNum uint16

func (n SymbolNum) Int() int {
	return int(n)
}

type Symbol uint16

func (s Symbol) String() string {
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

	SymbolNil   = Symbol(0)                                                 // 0000 0000 0000 0000
	symbolStart = Symbol(maskNonTerminal | maskStartOrEOF | symbolNumStart) // 0100 0000 0000 0001
	SymbolEOF   = Symbol(maskTerminal | maskStartOrEOF | symbolNumEOF)      // 1100 0000 0000 0001: The EOF symbol is treated as a terminal symbol.

	// The symbol name contains `<` and `>` to avoid conflicting with user-defined symbols.
	symbolNameEOF = "<eof>"

	nonTerminalNumMin = SymbolNum(2)           // The number 1 is used by a start symbol.
	terminalNumMin    = SymbolNum(2)           // The number 1 is used by the EOF symbol.
	symbolNumMax      = SymbolNum(0xffff) >> 2 // 0011 1111 1111 1111
)

func newSymbol(kind symbolKind, isStart bool, num SymbolNum) (Symbol, error) {
	if num > symbolNumMax {
		return SymbolNil, fmt.Errorf("a symbol number exceeds the limit; limit: %v, passed: %v", symbolNumMax, num)
	}
	if kind == symbolKindTerminal && isStart {
		return SymbolNil, fmt.Errorf("a start symbol must be a non-terminal symbol")
	}

	kindMask := maskNonTerminal
	if kind == symbolKindTerminal {
		kindMask = maskTerminal
	}
	startMask := maskNonStartAndEOF
	if isStart {
		startMask = maskStartOrEOF
	}
	return Symbol(kindMask | startMask | uint16(num)), nil
}

func (s Symbol) Num() SymbolNum {
	_, _, _, num := s.describe()
	return num
}

func (s Symbol) Byte() []byte {
	if s.IsNil() {
		return []byte{0, 0}
	}
	return []byte{byte(uint16(s) >> 8), byte(uint16(s) & 0x00ff)}
}

func (s Symbol) IsNil() bool {
	_, _, _, num := s.describe()
	return num == 0
}

func (s Symbol) IsStart() bool {
	if s.IsNil() {
		return false
	}
	_, isStart, _, _ := s.describe()
	return isStart
}

func (s Symbol) isEOF() bool {
	if s.IsNil() {
		return false
	}
	_, _, isEOF, _ := s.describe()
	return isEOF
}

func (s Symbol) isNonTerminal() bool {
	if s.IsNil() {
		return false
	}
	kind, _, _, _ := s.describe()
	return kind == symbolKindNonTerminal
}

func (s Symbol) IsTerminal() bool {
	if s.IsNil() {
		return false
	}
	return !s.isNonTerminal()
}

func (s Symbol) describe() (symbolKind, bool, bool, SymbolNum) {
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
	num := SymbolNum(uint16(s) & maskNumberPart)
	return kind, isStart, isEOF, num
}

type SymbolTable struct {
	text2Sym     map[string]Symbol
	sym2Text     map[Symbol]string
	nonTermTexts []string
	termTexts    []string
	nonTermNum   SymbolNum
	termNum      SymbolNum
}

type SymbolTableWriter struct {
	*SymbolTable
}

type SymbolTableReader struct {
	*SymbolTable
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		text2Sym: map[string]Symbol{
			symbolNameEOF: SymbolEOF,
		},
		sym2Text: map[Symbol]string{
			SymbolEOF: symbolNameEOF,
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

func (t *SymbolTable) Writer() *SymbolTableWriter {
	return &SymbolTableWriter{
		SymbolTable: t,
	}
}

func (t *SymbolTable) Reader() *SymbolTableReader {
	return &SymbolTableReader{
		SymbolTable: t,
	}
}

func (w *SymbolTableWriter) RegisterStartSymbol(text string) (Symbol, error) {
	w.text2Sym[text] = symbolStart
	w.sym2Text[symbolStart] = text
	w.nonTermTexts[symbolStart.Num().Int()] = text
	return symbolStart, nil
}

func (w *SymbolTableWriter) RegisterNonTerminalSymbol(text string) (Symbol, error) {
	if sym, ok := w.text2Sym[text]; ok {
		return sym, nil
	}
	sym, err := newSymbol(symbolKindNonTerminal, false, w.nonTermNum)
	if err != nil {
		return SymbolNil, err
	}
	w.nonTermNum++
	w.text2Sym[text] = sym
	w.sym2Text[sym] = text
	w.nonTermTexts = append(w.nonTermTexts, text)
	return sym, nil
}

func (w *SymbolTableWriter) RegisterTerminalSymbol(text string) (Symbol, error) {
	if sym, ok := w.text2Sym[text]; ok {
		return sym, nil
	}
	sym, err := newSymbol(symbolKindTerminal, false, w.termNum)
	if err != nil {
		return SymbolNil, err
	}
	w.termNum++
	w.text2Sym[text] = sym
	w.sym2Text[sym] = text
	w.termTexts = append(w.termTexts, text)
	return sym, nil
}

func (r *SymbolTableReader) ToSymbol(text string) (Symbol, bool) {
	if sym, ok := r.text2Sym[text]; ok {
		return sym, true
	}
	return SymbolNil, false
}

func (r *SymbolTableReader) ToText(sym Symbol) (string, bool) {
	text, ok := r.sym2Text[sym]
	return text, ok
}

func (r *SymbolTableReader) TerminalSymbols() []Symbol {
	syms := make([]Symbol, 0, r.termNum.Int()-terminalNumMin.Int())
	for sym := range r.sym2Text {
		if !sym.IsTerminal() || sym.IsNil() {
			continue
		}
		syms = append(syms, sym)
	}
	sort.Slice(syms, func(i, j int) bool {
		return syms[i] < syms[j]
	})
	return syms
}

func (r *SymbolTableReader) TerminalTexts() ([]string, error) {
	if r.termNum == terminalNumMin {
		return nil, fmt.Errorf("symbol table has no terminals")
	}
	return r.termTexts, nil
}

func (r *SymbolTableReader) NonTerminalSymbols() []Symbol {
	syms := make([]Symbol, 0, r.nonTermNum.Int()-nonTerminalNumMin.Int())
	for sym := range r.sym2Text {
		if !sym.isNonTerminal() || sym.IsNil() {
			continue
		}
		syms = append(syms, sym)
	}
	sort.Slice(syms, func(i, j int) bool {
		return syms[i] < syms[j]
	})
	return syms
}

func (r *SymbolTableReader) NonTerminalTexts() ([]string, error) {
	if r.nonTermNum == nonTerminalNumMin || r.nonTermTexts[symbolStart.Num().Int()] == "" {
		return nil, fmt.Errorf("symbol table has no terminals or no start symbol")
	}
	return r.nonTermTexts, nil
}
