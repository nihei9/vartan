package parser

import "fmt"

var (
	ParseErr = fmt.Errorf("parse error")

	// lexical errors
	synErrIncompletedEscSeq     = fmt.Errorf("incompleted escape sequence; unexpected EOF following \\")
	synErrInvalidEscSeq         = fmt.Errorf("invalid escape sequence")
	synErrInvalidCodePoint      = fmt.Errorf("code points must consist of just 4 or 6 hex digits")
	synErrCharPropInvalidSymbol = fmt.Errorf("invalid character property symbol")
	SynErrFragmentInvalidSymbol = fmt.Errorf("invalid fragment symbol")

	// syntax errors
	synErrUnexpectedToken        = fmt.Errorf("unexpected token")
	synErrNullPattern            = fmt.Errorf("a pattern must be a non-empty byte sequence")
	synErrUnmatchablePattern     = fmt.Errorf("a pattern cannot match any characters")
	synErrAltLackOfOperand       = fmt.Errorf("an alternation expression must have operands")
	synErrRepNoTarget            = fmt.Errorf("a repeat expression must have an operand")
	synErrGroupNoElem            = fmt.Errorf("a grouping expression must include at least one character")
	synErrGroupUnclosed          = fmt.Errorf("unclosed grouping expression")
	synErrGroupNoInitiator       = fmt.Errorf(") needs preceding (")
	synErrGroupInvalidForm       = fmt.Errorf("invalid grouping expression")
	synErrBExpNoElem             = fmt.Errorf("a bracket expression must include at least one character")
	synErrBExpUnclosed           = fmt.Errorf("unclosed bracket expression")
	synErrBExpInvalidForm        = fmt.Errorf("invalid bracket expression")
	synErrRangeInvalidOrder      = fmt.Errorf("a range expression with invalid order")
	synErrRangePropIsUnavailable = fmt.Errorf("a property expression is unavailable in a range expression")
	synErrRangeInvalidForm       = fmt.Errorf("invalid range expression")
	synErrCPExpInvalidForm       = fmt.Errorf("invalid code point expression")
	synErrCPExpOutOfRange        = fmt.Errorf("a code point must be between U+0000 to U+10FFFF")
	synErrCharPropExpInvalidForm = fmt.Errorf("invalid character property expression")
	synErrCharPropUnsupported    = fmt.Errorf("unsupported character property")
	synErrFragmentExpInvalidForm = fmt.Errorf("invalid fragment expression")
)
