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
	synErrInvalidEscSeq     = newSyntaxError("invalid escape sequence")
	synErrIncompletedEscSeq = newSyntaxError("incompleted escape sequence; unexpected EOF following a backslash")
	synErrZeroPos           = newSyntaxError("a position must be greater than or equal to 1")

	// syntax errors
	synErrInvalidToken         = newSyntaxError("invalid token")
	synErrNoProduction         = newSyntaxError("a grammar must have at least one production")
	synErrNoProductionName     = newSyntaxError("a production name is missing")
	synErrNoColon              = newSyntaxError("the colon must precede alternatives")
	synErrNoSemicolon          = newSyntaxError("the semicolon is missing at the last of an alternative")
	synErrNoDirectiveName      = newSyntaxError("a directive needs a name")
	synErrProdDirNoNewline     = newSyntaxError("a production directive must be followed by a newline")
	synErrSemicolonNoNewline   = newSyntaxError("a semicolon must be followed by a newline")
	synErrFragmentNoPattern    = newSyntaxError("a fragment needs one pattern element")
	synErrTreeInvalidFirstElem = newSyntaxError("the first element of a tree structure must be an ID")
	synErrTreeUnclosed         = newSyntaxError("unclosed tree structure")
)
