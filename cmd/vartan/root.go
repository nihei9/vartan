package main

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vartan",
	Short: "Generate a portable LALR(1) parsing table from grammar you defined",
	Long: `vartan provides two features:
- Generate a portable LALR(1) parsing table from grammar you defined.
- Parse a text stream according to the grammar.`,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() error {
	return rootCmd.Execute()
}
