package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	verr "github.com/nihei9/vartan/error"
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
	var tmpDirPath string
	defer func() {
		if tmpDirPath == "" {
			return
		}
		os.RemoveAll(tmpDirPath)
	}()

	grmPath := *compileFlags.grammar
	defer func() {
		v := recover()
		if v != nil {
			err, ok := v.(error)
			if !ok {
				retErr = fmt.Errorf("an unexpected error occurred: %v", v)
				fmt.Fprintf(os.Stderr, "%v:\n%v", retErr, string(debug.Stack()))
				return
			}

			retErr = err
		}

		if retErr != nil {
			specErrs, ok := retErr.(verr.SpecErrors)
			if ok {
				for _, err := range specErrs {
					if *compileFlags.grammar != "" {
						err.FilePath = grmPath
						err.SourceName = grmPath
					} else {
						err.FilePath = grmPath
						err.SourceName = "stdin"
					}
				}
			}

			fmt.Fprintf(os.Stderr, "%v:\n%v", retErr, string(debug.Stack()))
		}
	}()

	if grmPath == "" {
		var err error
		tmpDirPath, err = os.MkdirTemp("", "vartan-compile-*")
		if err != nil {
			return err
		}

		src, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return err
		}

		grmPath = filepath.Join(tmpDirPath, "stdin.vr")
		err = ioutil.WriteFile(grmPath, src, 0600)
		if err != nil {
			return err
		}
	}

	gram, err := readGrammar(grmPath)
	if err != nil {
		return err
	}

	var descFileName string
	{
		_, grmFileName := filepath.Split(grmPath)
		descFileName = fmt.Sprintf("%v.desc", strings.TrimSuffix(grmFileName, ".vr"))
	}

	cgram, err := grammar.Compile(gram, grammar.EnableDescription(descFileName))
	if err != nil {
		return err
	}

	err = writeCompiledGrammar(cgram, *compileFlags.output)
	if err != nil {
		return fmt.Errorf("Cannot write a compiled grammar: %w", err)
	}

	return nil
}

func readGrammar(path string) (grm *grammar.Grammar, retErr error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot open the grammar file %s: %w", path, err)
	}
	defer f.Close()

	ast, err := spec.Parse(f)
	if err != nil {
		return nil, err
	}

	b := grammar.GrammarBuilder{
		AST: ast,
	}
	return b.Build()
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
