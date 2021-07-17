package error

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type SpecError struct {
	Cause      error
	FilePath   string
	SourceName string
	Row        int
}

func (e *SpecError) Error() string {
	var b strings.Builder
	if e.SourceName != "" {
		fmt.Fprintf(&b, "%v: ", e.SourceName)
	}
	if e.Row != 0 {
		fmt.Fprintf(&b, "%v: ", e.Row)
	}
	fmt.Fprintf(&b, "error: %v", e.Cause)

	line := readLine(e.FilePath, e.Row)
	if line != "" {
		fmt.Fprintf(&b, "\n    %v", line)
	}

	return b.String()
}

func readLine(filePath string, row int) string {
	if filePath == "" || row <= 0 {
		return ""
	}

	f, err := os.Open(filePath)
	if err != nil {
		return ""
	}

	i := 1
	s := bufio.NewScanner(f)
	for s.Scan() {
		if i == row {
			return s.Text()
		}
		i++
	}

	return ""
}
