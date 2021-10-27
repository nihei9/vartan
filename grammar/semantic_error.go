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
	semErrMDInvalidName       = newSemanticError("invalid meta data name")
	semErrMDInvalidParam      = newSemanticError("invalid parameter")
	semErrMDMissingName       = newSemanticError("name is missing")
	semErrUnusedProduction    = newSemanticError("unused production")
	semErrUnusedTerminal      = newSemanticError("unused terminal")
	semErrTermCannotBeSkipped = newSemanticError("a terminal used in productions cannot be skipped")
	semErrNoProduction        = newSemanticError("a grammar needs at least one production")
	semErrUndefinedSym        = newSemanticError("undefined symbol")
	semErrDuplicateProduction = newSemanticError("duplicate production")
	semErrDuplicateTerminal   = newSemanticError("duplicate terminal")
	semErrDuplicateName       = newSemanticError("duplicate names are not allowed between terminals and non-terminals")
	semErrErrSymIsReserved    = newSemanticError("symbol 'error' is reserved as a terminal symbol")
	semErrDirInvalidName      = newSemanticError("invalid directive name")
	semErrDirInvalidParam     = newSemanticError("invalid parameter")
)
