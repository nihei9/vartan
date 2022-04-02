package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime/debug"

	"github.com/nihei9/vartan/driver"
	"github.com/nihei9/vartan/spec"
	"github.com/spf13/cobra"
)

var parseFlags = struct {
	source     *string
	onlyParse  *bool
	cst        *bool
	disableLAC *bool
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
	parseFlags.disableLAC = cmd.Flags().Bool("disable-lac", false, "disable LAC (lookahead correction)")
	rootCmd.AddCommand(cmd)
}

func runParse(cmd *cobra.Command, args []string) (retErr error) {
	defer func() {
		panicked := false
		v := recover()
		if v != nil {
			err, ok := v.(error)
			if !ok {
				retErr = fmt.Errorf("an unexpected error occurred: %v", v)
				fmt.Fprintf(os.Stderr, "%v:\n%v", retErr, string(debug.Stack()))
				return
			}

			retErr = err
			panicked = true
		}

		if retErr != nil {
			if panicked {
				fmt.Fprintf(os.Stderr, "%v:\n%v", retErr, string(debug.Stack()))
			} else {
				fmt.Fprintf(os.Stderr, "%v\n", retErr)
			}
		}
	}()

	if *parseFlags.onlyParse && *parseFlags.cst {
		return fmt.Errorf("You cannot enable --only-parse and --cst at the same time")
	}

	cg, err := readCompiledGrammar(args[0])
	if err != nil {
		return fmt.Errorf("Cannot read a compiled grammar: %w", err)
	}

	var p *driver.Parser
	var treeAct *driver.SyntaxTreeActionSet
	var tb *driver.DefaulSyntaxTreeBuilder
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

		gram := driver.NewGrammar(cg)

		var opts []driver.ParserOption
		{
			switch {
			case *parseFlags.cst:
				tb = driver.NewDefaultSyntaxTreeBuilder()
				treeAct = driver.NewCSTActionSet(gram, tb)
			case !*parseFlags.onlyParse:
				tb = driver.NewDefaultSyntaxTreeBuilder()
				treeAct = driver.NewASTActionSet(gram, tb)
			}
			if treeAct != nil {
				opts = append(opts, driver.SemanticAction(treeAct))
			}

			if *parseFlags.disableLAC {
				opts = append(opts, driver.DisableLAC())
			}
		}

		toks, err := driver.NewTokenStream(cg, src)
		if err != nil {
			return err
		}

		p, err = driver.NewParser(toks, gram, opts...)
		if err != nil {
			return err
		}
	}

	err = p.Parse()
	if err != nil {
		return err
	}

	synErrs := p.SyntaxErrors()
	for _, synErr := range synErrs {
		tok := synErr.Token

		var msg string
		switch {
		case tok.EOF():
			msg = "<eof>"
		case tok.Invalid():
			msg = fmt.Sprintf("'%v' (<invalid>)", string(tok.Lexeme()))
		default:
			t := cg.ParsingTable.Terminals[tok.TerminalID()]
			msg = fmt.Sprintf("'%v' (%v)", string(tok.Lexeme()), t)
		}

		fmt.Fprintf(os.Stderr, "%v:%v: %v: %v", synErr.Row+1, synErr.Col+1, synErr.Message, msg)
		if len(synErrs) > 0 {
			fmt.Fprintf(os.Stderr, "; expected: %v", synErr.ExpectedTerminals[0])
			for _, t := range synErr.ExpectedTerminals[1:] {
				fmt.Fprintf(os.Stderr, ", %v", t)
			}
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	if !*parseFlags.onlyParse {
		// A parser can construct a parse tree even if syntax errors occur.
		// When therer is a parse tree, print it.

		var tree *driver.Node
		if *parseFlags.cst {
			tree = tb.Tree()
		} else {
			tree = tb.Tree()
		}
		if tree != nil {
			if len(synErrs) > 0 {
				fmt.Println("")
			}
			driver.PrintTree(os.Stdout, tree)
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
	cg := &spec.CompiledGrammar{}
	err = json.Unmarshal(data, cg)
	if err != nil {
		return nil, err
	}
	return cg, nil
}
