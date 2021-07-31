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
	semErrUnusedProduction    = newSemanticError("unused production")
	semErrUnusedTerminal      = newSemanticError("unused terminal")
	semErrTermCannotBeSkipped = newSemanticError("a terminal used in productions cannot be skipped")
	semErrNoProduction        = newSemanticError("a grammar needs at least one production")
	semErrUndefinedSym        = newSemanticError("undefined symbol")
	semErrDuplicateProduction = newSemanticError("duplicate production")
	semErrDuplicateTerminal   = newSemanticError("duplicate terminal")
	semErrDuplicateName       = newSemanticError("duplicate names are not allowed between terminals and non-terminals")
	semErrDirInvalidName      = newSemanticError("invalid directive name")
	semErrDirInvalidParam     = newSemanticError("invalid parameter")
)
