package grammar

type SemanticError struct {
	message string
}

func newSemanticError(message string) *SemanticError {
	return &SemanticError{
		message: message,
	}
}

func (e *SemanticError) Error() string {
	return e.message
}

var (
	semErrNoProduction    = newSemanticError("a grammar needs at least one production")
	semErrUndefinedSym    = newSemanticError("undefined symbol")
	semErrDirInvalidName  = newSemanticError("invalid directive name")
	semErrDirInvalidParam = newSemanticError("invalid parameter")
)
