package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/tester"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:     "test <grammar file path> <test file path>|<test directory path>",
		Short:   "Test a grammar",
		Example: `  vartan test grammar.vartan test`,
		Args:    cobra.ExactArgs(2),
		RunE:    runTest,
	}
	rootCmd.AddCommand(cmd)
}

func runTest(cmd *cobra.Command, args []string) error {
	g, err := readGrammar(args[0])
	if err != nil {
		return fmt.Errorf("Cannot read a grammar: %w", err)
	}
	cg, _, err := grammar.Compile(g)
	if err != nil {
		return fmt.Errorf("Cannot read a compiled grammar: %w", err)
	}

	var cs []*tester.TestCaseWithMetadata
	{
		cs = tester.ListTestCases(args[1])
		errOccurred := false
		for _, c := range cs {
			if c.Error != nil {
				fmt.Fprintf(os.Stderr, "Failed to read a test case or a directory: %v\n%v\n", c.FilePath, c.Error)
				errOccurred = true
			}
		}
		if errOccurred {
			return errors.New("Cannot run test")
		}
	}

	t := &tester.Tester{
		Grammar: cg,
		Cases:   cs,
	}
	rs := t.Run()
	testFailed := false
	for _, r := range rs {
		fmt.Fprintln(os.Stdout, r)
		if r.Error != nil {
			testFailed = true
		}
	}
	if testFailed {
		return errors.New("Test failed")
	}
	return nil
}
