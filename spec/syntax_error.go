package spec

import "fmt"

type SyntaxError struct {
	message string
}

func newSyntaxError(message string) *SyntaxError {
	return &SyntaxError{
		message: message,
	}
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("syntax error: %s", e.message)
}

var (
	// lexical errors
	synErrUnclosedTerminal  = newSyntaxError("unclosed terminal")
	synErrInvalidEscSeq     = newSyntaxError("invalid escape sequence")
	synErrIncompletedEscSeq = newSyntaxError("incompleted escape sequence; unexpected EOF following \\")

	// syntax errors
	synErrInvalidToken     = newSyntaxError("invalid token")
	synErrNoProduction     = newSyntaxError("a grammar must have at least one production")
	synErrNoProductionName = newSyntaxError("a production name is missing")
	synErrNoColon          = newSyntaxError("the colon must precede alternatives")
	synErrNoSemicolon      = newSyntaxError("the semicolon is missing at the last of an alternative")
	synErrNoModifierName   = newSyntaxError("a modifier needs a name")
	synErrNoActionName     = newSyntaxError("an action needs a name")
)
