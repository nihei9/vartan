package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/nihei9/vartan/driver"
	spec "github.com/nihei9/vartan/spec/grammar"
	"github.com/spf13/cobra"
)

var parseFlags = struct {
	source     *string
	onlyParse  *bool
	cst        *bool
	disableLAC *bool
	json       *bool
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
	parseFlags.json = cmd.Flags().Bool("json", false, "enable JSON output")
	rootCmd.AddCommand(cmd)
}

func runParse(cmd *cobra.Command, args []string) error {
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
			if *parseFlags.json {
				b, err := json.Marshal(tree)
				if err != nil {
					return err
				}
				fmt.Fprintln(os.Stdout, string(b))
			} else {
				driver.PrintTree(os.Stdout, tree)
			}
		}
	}

	if len(p.SyntaxErrors()) > 0 {
		var b strings.Builder
		synErrs := p.SyntaxErrors()
		writeSyntaxErrorMessage(&b, cg, synErrs[0])
		for _, synErr := range synErrs[1:] {
			fmt.Fprintf(&b, "\n")
			writeSyntaxErrorMessage(&b, cg, synErr)
		}
		if b.Len() > 0 {
			return fmt.Errorf(b.String())
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

func writeSyntaxErrorMessage(b *strings.Builder, cgram *spec.CompiledGrammar, synErr *driver.SyntaxError) {
	fmt.Fprintf(b, "%v:%v: %v: ", synErr.Row+1, synErr.Col+1, synErr.Message)

	tok := synErr.Token
	switch {
	case tok.EOF():
		fmt.Fprintf(b, "<eof>")
	case tok.Invalid():
		fmt.Fprintf(b, "'%v' (<invalid>)", string(tok.Lexeme()))
	default:
		fmt.Fprintf(b, "'%v' (%v)", string(tok.Lexeme()), cgram.ParsingTable.Terminals[tok.TerminalID()])
	}

	fmt.Fprintf(b, "; expected: %v", synErr.ExpectedTerminals[0])
	for _, t := range synErr.ExpectedTerminals[1:] {
		fmt.Fprintf(b, ", %v", t)
	}
}
