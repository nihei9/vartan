package grammar

import "errors"

var (
	semErrNoGrammarName         = errors.New("name is missing")
	semErrSpellingInconsistency = errors.New("the identifiers are treated as the same. please use the same spelling")
	semErrDuplicateAssoc        = errors.New("associativity and precedence cannot be specified multiple times for a symbol")
	semErrUndefinedPrec         = errors.New("symbol must has precedence")
	semErrUndefinedOrdSym       = errors.New("undefined ordered symbol")
	semErrUnusedProduction      = errors.New("unused production")
	semErrUnusedTerminal        = errors.New("unused terminal")
	semErrTermCannotBeSkipped   = errors.New("a terminal used in productions cannot be skipped")
	semErrNoProduction          = errors.New("a grammar needs at least one production")
	semErrUndefinedSym          = errors.New("undefined symbol")
	semErrDuplicateProduction   = errors.New("duplicate production")
	semErrDuplicateTerminal     = errors.New("duplicate terminal")
	semErrDuplicateFragment     = errors.New("duplicate fragment")
	semErrDuplicateName         = errors.New("duplicate names are not allowed between terminals and non-terminals")
	semErrErrSymIsReserved      = errors.New("symbol 'error' is reserved as a terminal symbol")
	semErrDuplicateLabel        = errors.New("a label must be unique in an alternative")
	semErrInvalidLabel          = errors.New("a label must differ from terminal symbols or non-terminal symbols")
	semErrDirInvalidName        = errors.New("invalid directive name")
	semErrDirInvalidParam       = errors.New("invalid parameter")
	semErrDuplicateDir          = errors.New("a directive must not be duplicated")
	semErrDuplicateElem         = errors.New("duplicate element")
	semErrAmbiguousElem         = errors.New("ambiguous element")
	semErrInvalidProdDir        = errors.New("invalid production directive")
	semErrInvalidAltDir         = errors.New("invalid alternative directive")
)
