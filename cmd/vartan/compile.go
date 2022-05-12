package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	verr "github.com/nihei9/vartan/error"
	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec"
	"github.com/spf13/cobra"
)

var compileFlags = struct {
	output *string
	class  *string
}{}

func init() {
	cmd := &cobra.Command{
		Use:     "compile",
		Short:   "Compile grammar you defined into a parsing table",
		Example: `  vartan compile grammar.vartan -o grammar.json`,
		Args:    cobra.MaximumNArgs(1),
		RunE:    runCompile,
	}
	compileFlags.output = cmd.Flags().StringP("output", "o", "", "output file path (default stdout)")
	compileFlags.class = cmd.Flags().StringP("class", "", "lalr", "LALR or SLR")
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

	var grmPath string
	if len(args) > 0 {
		grmPath = args[0]
	}
	defer func() {
		if retErr != nil {
			specErrs, ok := retErr.(verr.SpecErrors)
			if ok {
				for _, err := range specErrs {
					if len(args) > 0 {
						err.FilePath = grmPath
						err.SourceName = grmPath
					} else {
						err.FilePath = grmPath
						err.SourceName = "stdin"
					}
				}
			}
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

		grmPath = filepath.Join(tmpDirPath, "stdin.vartan")
		err = ioutil.WriteFile(grmPath, src, 0600)
		if err != nil {
			return err
		}
	}

	gram, err := readGrammar(grmPath)
	if err != nil {
		return err
	}

	var reportFileName string
	{
		_, grmFileName := filepath.Split(grmPath)
		reportFileName = fmt.Sprintf("%v-report.json", strings.TrimSuffix(grmFileName, ".vartan"))
	}

	opts := []grammar.CompileOption{
		grammar.EnableReporting(reportFileName),
	}
	switch strings.ToLower(*compileFlags.class) {
	case "slr":
		opts = append(opts, grammar.SpecifyClass(grammar.ClassSLR))
	case "lalr":
		opts = append(opts, grammar.SpecifyClass(grammar.ClassLALR))
	default:
		return fmt.Errorf("invalid class: %v", *compileFlags.class)
	}

	cgram, err := grammar.Compile(gram, opts...)
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
