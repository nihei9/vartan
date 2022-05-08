package spec

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
	synErrAutoGenID         = newSyntaxError("you cannot define an identifier beginning with an underscore")
	synErrUnclosedTerminal  = newSyntaxError("unclosed terminal")
	synErrUnclosedString    = newSyntaxError("unclosed string")
	synErrIncompletedEscSeq = newSyntaxError("incompleted escape sequence; unexpected EOF following a backslash")
	synErrEmptyPattern      = newSyntaxError("a pattern must include at least one character")
	synErrEmptyString       = newSyntaxError("a string must include at least one character")

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
	synErrSemicolonNoNewline     = newSyntaxError("a semicolon must be followed by a newline")
	synErrFragmentNoPattern      = newSyntaxError("a fragment needs one pattern element")
)
