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

var lexFlags = struct {
	source *string
}{}

func init() {
	cmd := &cobra.Command{
		Use:     "parse <grammar file path>",
		Short:   "Parse a text stream",
		Example: `  cat src | vartan parse grammar.json`,
		Args:    cobra.ExactArgs(1),
		RunE:    runParse,
	}
	lexFlags.source = cmd.Flags().StringP("source", "s", "", "source file path (default stdin)")
	rootCmd.AddCommand(cmd)
}

func runParse(cmd *cobra.Command, args []string) (retErr error) {
	cgram, err := readCompiledGrammar(args[0])
	if err != nil {
		return fmt.Errorf("Cannot read a compiled grammar: %w", err)
	}

	var p *driver.Parser
	{
		src := os.Stdin
		if *lexFlags.source != "" {
			f, err := os.Open(*lexFlags.source)
			if err != nil {
				return fmt.Errorf("Cannot open the source file %s: %w", *lexFlags.source, err)
			}
			defer f.Close()
			src = f
		}
		p, err = driver.NewParser(cgram, src)
		if err != nil {
			return err
		}
	}

	err = p.Parse()
	if err != nil {
		return err
	}
	fmt.Printf("Accepted\n")

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
