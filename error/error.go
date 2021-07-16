package error

import "fmt"

type SpecError struct {
	Cause error
	Row   int
}

func (e *SpecError) Error() string {
	if e.Row == 0 {
		return fmt.Sprintf("error: %v", e.Cause)
	}
	return fmt.Sprintf("%v: error: %v", e.Row, e.Cause)
}
