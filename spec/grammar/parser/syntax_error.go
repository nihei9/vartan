package parser

type SyntaxError struct {
	message string
}

func newSyntaxError(message string) *SyntaxError {
	return &SyntaxError{
		message: message,
	}
}

func (e *SyntaxError) Error() string {
	return e.message
}

var (
	// lexical errors
	synErrIDInvalidChar            = newSyntaxError("an identifier can contain only the lower-case letter, the digits, and the underscore")
	synErrIDInvalidUnderscorePos   = newSyntaxError("the underscore cannot be placed at the beginning or end of an identifier")
	synErrIDConsecutiveUnderscores = newSyntaxError("the underscore cannot be placed consecutively")
	synErrIDInvalidDigitsPos       = newSyntaxError("the digits cannot be placed at the biginning of an identifier")
	synErrUnclosedTerminal         = newSyntaxError("unclosed terminal")
	synErrUnclosedString           = newSyntaxError("unclosed string")
	synErrIncompletedEscSeq        = newSyntaxError("incompleted escape sequence; unexpected EOF following a backslash")
	synErrEmptyPattern             = newSyntaxError("a pattern must include at least one character")
	synErrEmptyString              = newSyntaxError("a string must include at least one character")

	// syntax errors
	synErrInvalidToken           = newSyntaxError("invalid token")
	synErrTopLevelDirNoSemicolon = newSyntaxError("a top-level directive must be followed by ;")
	synErrNoProductionName       = newSyntaxError("a production name is missing")
	synErrNoColon                = newSyntaxError("the colon must precede alternatives")
	synErrNoSemicolon            = newSyntaxError("the semicolon is missing at the last of an alternative")
	synErrLabelWithNoSymbol      = newSyntaxError("a label must follow a symbol")
	synErrNoLabel                = newSyntaxError("an identifier that represents a label is missing after the label marker @")
	synErrNoDirectiveName        = newSyntaxError("a directive needs a name")
	synErrNoOrderedSymbolName    = newSyntaxError("an ordered symbol name is missing")
	synErrUnclosedDirGroup       = newSyntaxError("a directive group must be closed by )")
	synErrPatternInAlt           = newSyntaxError("a pattern literal cannot appear directly in an alternative. instead, please define a terminal symbol with the pattern literal")
	synErrStrayExpOp             = newSyntaxError("an expansion operator ... must be preceded by an identifier")
	synErrInvalidExpOperand      = newSyntaxError("an expansion operator ... can be applied to only an identifier")
	synErrSemicolonNoNewline     = newSyntaxError("a semicolon must be followed by a newline")
	synErrFragmentNoPattern      = newSyntaxError("a fragment needs one pattern element")
)
