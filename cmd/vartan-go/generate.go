package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	mldriver "github.com/nihei9/maleeni/driver"
	"github.com/nihei9/vartan/driver"
	spec "github.com/nihei9/vartan/spec/grammar"
	"github.com/spf13/cobra"
)

func Execute() error {
	return generateCmd.Execute()
}

var generateFlags = struct {
	pkgName *string
}{}

var generateCmd = &cobra.Command{
	Use:           "vartan-go",
	Short:         "Generate a parser for Go",
	Long:          `vartan-go generates a parser for Go.`,
	Example:       `  vartan-go grammar.json`,
	Args:          cobra.ExactArgs(1),
	RunE:          runGenerate,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	generateFlags.pkgName = generateCmd.Flags().StringP("package", "p", "main", "package name")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	cgram, err := readCompiledGrammar(args[0])
	if err != nil {
		return fmt.Errorf("Cannot read a compiled grammar: %w", err)
	}

	{
		b, err := mldriver.GenLexer(cgram.LexicalSpecification.Maleeni.Spec, *generateFlags.pkgName)
		if err != nil {
			return fmt.Errorf("Failed to generate a lexer: %w", err)
		}

		filePath := fmt.Sprintf("%v_lexer.go", cgram.Name)

		f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("Failed to create an output file: %v", err)
		}
		defer f.Close()

		_, err = f.Write(b)
		if err != nil {
			return fmt.Errorf("Failed to write lexer source code: %v", err)
		}
	}

	{
		b, err := driver.GenParser(cgram, *generateFlags.pkgName)
		if err != nil {
			return fmt.Errorf("Failed to generate a parser: %w", err)
		}

		filePath := fmt.Sprintf("%v_parser.go", cgram.Name)

		f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("Failed to create an output file: %v", err)
		}
		defer f.Close()

		_, err = f.Write(b)
		if err != nil {
			return fmt.Errorf("Failed to write parser source code: %v", err)
		}
	}

	{
		b, err := driver.GenSemanticAction(*generateFlags.pkgName)
		if err != nil {
			return fmt.Errorf("Failed to generate a semantic action set: %w", err)
		}

		filePath := fmt.Sprintf("%v_semantic_action.go", cgram.Name)

		f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("Failed to create an output file: %v", err)
		}
		defer f.Close()

		_, err = f.Write(b)
		if err != nil {
			return fmt.Errorf("Failed to write semantic action source code: %v", err)
		}
	}

	return nil
}

func readCompiledGrammar(path string) (*spec.CompiledGrammar, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	cgram := &spec.CompiledGrammar{}
	err = json.Unmarshal(data, cgram)
	if err != nil {
		return nil, err
	}
	return cgram, nil
}
