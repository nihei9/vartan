package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec"
	"github.com/spf13/cobra"
)

var compileFlags = struct {
	grammar *string
	output  *string
}{}

func init() {
	cmd := &cobra.Command{
		Use:     "compile",
		Short:   "Compile a grammar into a parsing table",
		Example: `  cat grammar | vartan compile -o grammar.json`,
		RunE:    runCompile,
	}
	compileFlags.grammar = cmd.Flags().StringP("grammar", "g", "", "grammar file path (default stdin)")
	compileFlags.output = cmd.Flags().StringP("output", "o", "", "output file path (default stdout)")
	rootCmd.AddCommand(cmd)
}

func runCompile(cmd *cobra.Command, args []string) (retErr error) {
	gram, err := readGrammar(*compileFlags.grammar)
	if err != nil {
		return err
	}

	cgram, err := grammar.Compile(gram)
	if err != nil {
		return err
	}

	err = writeCompiledGrammar(cgram, *compileFlags.output)
	if err != nil {
		return fmt.Errorf("Cannot write a compiled grammar: %w", err)
	}

	return nil
}

func readGrammar(path string) (*grammar.Grammar, error) {
	r := os.Stdin
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("Cannot open the grammar file %s: %w", path, err)
		}
		defer f.Close()
		r = f
	}
	ast, err := spec.Parse(r)
	if err != nil {
		return nil, err
	}
	return grammar.NewGrammar(ast)
}

func writeCompiledGrammar(cgram *spec.CompiledGrammar, path string) error {
	out, err := json.Marshal(cgram)
	if err != nil {
		return err
	}
	w := os.Stdout
	if path != "" {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("Cannot open the output file %s: %w", path, err)
		}
		defer f.Close()
		w = f
	}
	fmt.Fprintf(w, "%v\n", string(out))
	return nil
}
