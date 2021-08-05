package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/nihei9/vartan/driver"
	"github.com/nihei9/vartan/spec"
	"github.com/spf13/cobra"
)

var parseFlags = struct {
	source    *string
	onlyParse *bool
	cst       *bool
}{}

func init() {
	cmd := &cobra.Command{
		Use:     "parse <grammar file path>",
		Short:   "Parse a text stream",
		Example: `  cat src | vartan parse grammar.json`,
		Args:    cobra.ExactArgs(1),
		RunE:    runParse,
	}
	parseFlags.source = cmd.Flags().StringP("source", "s", "", "source file path (default stdin)")
	parseFlags.onlyParse = cmd.Flags().Bool("only-parse", false, "when this option is enabled, the parser performs only parse and doesn't semantic actions")
	parseFlags.cst = cmd.Flags().Bool("cst", false, "when this option is enabled, the parser generates a CST")
	rootCmd.AddCommand(cmd)
}

func runParse(cmd *cobra.Command, args []string) (retErr error) {
	defer func() {
		v := recover()
		if v != nil {
			err, ok := v.(error)
			if !ok {
				retErr = fmt.Errorf("an unexpected error occurred: %v\n", v)
				fmt.Fprintln(os.Stderr, retErr)
				return
			}

			retErr = err
		}

		if retErr != nil {
			fmt.Fprintln(os.Stderr, retErr)
		}
	}()

	if *parseFlags.onlyParse && *parseFlags.cst {
		return fmt.Errorf("You cannot enable --only-parse and --cst at the same time")
	}

	cgram, err := readCompiledGrammar(args[0])
	if err != nil {
		return fmt.Errorf("Cannot read a compiled grammar: %w", err)
	}

	var p *driver.Parser
	{
		src := os.Stdin
		if *parseFlags.source != "" {
			f, err := os.Open(*parseFlags.source)
			if err != nil {
				return fmt.Errorf("Cannot open the source file %s: %w", *parseFlags.source, err)
			}
			defer f.Close()
			src = f
		}

		var opts []driver.ParserOption
		switch {
		case *parseFlags.cst:
			opts = append(opts, driver.MakeCST())
		case !*parseFlags.onlyParse:
			opts = append(opts, driver.MakeAST())
		}
		p, err = driver.NewParser(cgram, src, opts...)
		if err != nil {
			return err
		}
	}

	err = p.Parse()
	if err != nil {
		return err
	}

	fmt.Printf("Accepted\n")

	if !*parseFlags.onlyParse {
		var tree *driver.Node
		if *parseFlags.cst {
			tree = p.CST()
		} else {
			tree = p.AST()
		}
		driver.PrintTree(os.Stdout, tree)
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
