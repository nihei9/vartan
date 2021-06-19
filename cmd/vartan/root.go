package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vartan",
	Short: "Generate a portable parsing table from a grammar",
	Long: `vartan provides two features:
- Generates a portable parsing table from a grammar.
- Tokenizes a text stream according to the grammar.
  This feature is primarily aimed at debugging the grammar.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return err
	}
	return nil
}
