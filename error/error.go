package error

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
)

type SpecErrors []*SpecError

func (e SpecErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	sorted := make([]*SpecError, len(e))
	copy(sorted, e)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Row < sorted[j].Row
	})
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].FilePath < sorted[j].FilePath
	})

	var b strings.Builder
	fmt.Fprintf(&b, "%v", sorted[0])
	for _, err := range sorted[1:] {
		fmt.Fprintf(&b, "\n%v", err)
	}

	return b.String()
}

type SpecError struct {
	Cause      error
	Detail     string
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
	if e.Detail != "" {
		fmt.Fprintf(&b, ": %v", e.Detail)
	}

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
