package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	verr "github.com/nihei9/vartan/error"
	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec"
	"github.com/spf13/cobra"
)

var compileFlags = struct {
	output *string
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

	cgram, report, err := grammar.Compile(gram, grammar.EnableReporting())
	if err != nil {
		return err
	}

	err = writeCompiledGrammarAndReport(cgram, report, *compileFlags.output)
	if err != nil {
		return fmt.Errorf("Cannot write an output files: %w", err)
	}

	var implicitlyResolvedCount int
	for _, s := range report.States {
		for _, c := range s.SRConflict {
			if c.ResolvedBy == grammar.ResolvedByShift.Int() {
				implicitlyResolvedCount++
			}
		}
		for _, c := range s.RRConflict {
			if c.ResolvedBy == grammar.ResolvedByProdOrder.Int() {
				implicitlyResolvedCount++
			}
		}
	}
	if implicitlyResolvedCount > 0 {
		fmt.Fprintf(os.Stdout, "%v conflicts\n", implicitlyResolvedCount)
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

// writeCompiledGrammarAndReport writes a compiled grammar and a report to a files located at a specified path.
// This function selects one of the following output methods depending on how the path is specified.
//
// 1. When the path is a directory path, this function writes the compiled grammar and the report to
//    <path>/<grammar-name>.json and <path>/<grammar-name>-report.json files, respectively.
//    <grammar-name>-report.json as the output files.
// 2. When the path is a file path or a non-exitent path, this function asumes that the path represents a file
//    path for the compiled grammar. Then it also writes the report in the same directory as the compiled grammar.
//    The report file is named <grammar-name>.json.
// 3. When the path is an empty string, this function writes the compiled grammar to the stdout and writes
//    the report to a file named <current-directory>/<grammar-name>-report.json.
func writeCompiledGrammarAndReport(cgram *spec.CompiledGrammar, report *spec.Report, path string) error {
	cgramPath, reportPath, err := makeOutputFilePaths(cgram.Name, path)
	if err != nil {
		return err
	}

	{
		var cgramW io.Writer
		if cgramPath != "" {
			cgramFile, err := os.OpenFile(cgramPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				return err
			}
			defer cgramFile.Close()
			cgramW = cgramFile
		} else {
			cgramW = os.Stdout
		}

		b, err := json.Marshal(cgram)
		if err != nil {
			return err
		}
		fmt.Fprintf(cgramW, "%v\n", string(b))
	}

	{
		reportFile, err := os.OpenFile(reportPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer reportFile.Close()

		b, err := json.Marshal(report)
		if err != nil {
			return err
		}
		fmt.Fprintf(reportFile, "%v\n", string(b))
	}

	return nil
}

func makeOutputFilePaths(gramName string, path string) (string, string, error) {
	reportFileName := gramName + "-report.json"

	if path == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", "", err
		}
		return "", filepath.Join(wd, reportFileName), nil
	}

	fi, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return "", "", err
	}
	if os.IsNotExist(err) || !fi.IsDir() {
		dir, _ := filepath.Split(path)
		return path, filepath.Join(dir, reportFileName), nil
	}

	return filepath.Join(path, gramName+".json"), filepath.Join(path, reportFileName), nil
}
