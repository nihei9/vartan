package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime/debug"
	"strings"
	"text/template"

	"github.com/nihei9/vartan/grammar"
	"github.com/nihei9/vartan/spec"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:     "show",
		Short:   "Print a description file in readable format",
		Example: `  vartan show grammar-description.json`,
		Args:    cobra.ExactArgs(1),
		RunE:    runShow,
	}
	rootCmd.AddCommand(cmd)
}

func runShow(cmd *cobra.Command, args []string) (retErr error) {
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

	desc, err := readDescription(args[0])
	if err != nil {
		return err
	}

	err = writeDescription(os.Stdout, desc)
	if err != nil {
		return err
	}

	return nil
}

func readDescription(path string) (*spec.Description, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot open the description file %s: %w", path, err)
	}
	defer f.Close()

	d, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	desc := &spec.Description{}
	err = json.Unmarshal(d, desc)
	if err != nil {
		return nil, err
	}

	return desc, nil
}

const descTemplate = `# Class

{{ .Class }}

# Conflicts

{{ printConflictSummary . }}

# Terminals

{{ range slice .Terminals 1 -}}
{{ printTerminal . }}
{{ end }}
# Productions

{{ range slice .Productions 1 -}}
{{ printProduction . }}
{{ end }}
# States
{{ range .States }}
## State {{ .Number }}

{{ range .Kernel -}}
{{ printItem . }}
{{ end }}
{{ range .Shift -}}
{{ printShift . }}
{{ end -}}
{{ range .Reduce -}}
{{ printReduce . }}
{{ end -}}
{{ range .GoTo -}}
{{ printGoTo . }}
{{ end }}
{{ range .SRConflict -}}
{{ printSRConflict . }}
{{ end -}}
{{ range .RRConflict -}}
{{ printRRConflict . }}
{{ end -}}
{{ end }}`

func writeDescription(w io.Writer, desc *spec.Description) error {
	termName := func(sym int) string {
		if desc.Terminals[sym].Alias != "" {
			return desc.Terminals[sym].Alias
		}
		return desc.Terminals[sym].Name
	}

	nonTermName := func(sym int) string {
		return desc.NonTerminals[sym].Name
	}

	termAssoc := func(sym int) string {
		switch desc.Terminals[sym].Associativity {
		case "l":
			return "left"
		case "r":
			return "right"
		default:
			return "no"
		}
	}

	prodAssoc := func(prod int) string {
		switch desc.Productions[prod].Associativity {
		case "l":
			return "left"
		case "r":
			return "right"
		default:
			return "no"
		}
	}

	fns := template.FuncMap{
		"printConflictSummary": func(desc *spec.Description) string {
			var implicitlyResolvedCount int
			var explicitlyResolvedCount int
			for _, s := range desc.States {
				for _, c := range s.SRConflict {
					if c.ResolvedBy == grammar.ResolvedByShift.Int() {
						implicitlyResolvedCount++
					} else {
						explicitlyResolvedCount++
					}
				}
				for _, c := range s.RRConflict {
					if c.ResolvedBy == grammar.ResolvedByProdOrder.Int() {
						implicitlyResolvedCount++
					} else {
						explicitlyResolvedCount++
					}
				}
			}

			var b strings.Builder
			if implicitlyResolvedCount == 1 {
				fmt.Fprintf(&b, "%v conflict occurred and resolved implicitly.\n", implicitlyResolvedCount)
			} else if implicitlyResolvedCount > 1 {
				fmt.Fprintf(&b, "%v conflicts occurred and resolved implicitly.\n", implicitlyResolvedCount)
			}
			if explicitlyResolvedCount == 1 {
				fmt.Fprintf(&b, "%v conflict occurred and resolved explicitly.\n", explicitlyResolvedCount)
			} else if explicitlyResolvedCount > 1 {
				fmt.Fprintf(&b, "%v conflicts occurred and resolved explicitly.\n", explicitlyResolvedCount)
			}
			if implicitlyResolvedCount == 0 && explicitlyResolvedCount == 0 {
				fmt.Fprintf(&b, "No conflict")
			}
			return b.String()
		},
		"printTerminal": func(term spec.Terminal) string {
			var prec string
			if term.Precedence != 0 {
				prec = fmt.Sprintf("%2v", term.Precedence)
			} else {
				prec = " -"
			}

			var assoc string
			if term.Associativity != "" {
				assoc = term.Associativity
			} else {
				assoc = "-"
			}

			if term.Alias != "" {
				return fmt.Sprintf("%4v %v %v %v (%v)", term.Number, prec, assoc, term.Name, term.Alias)
			}
			return fmt.Sprintf("%4v %v %v %v", term.Number, prec, assoc, term.Name)
		},
		"printProduction": func(prod spec.Production) string {
			var prec string
			if prod.Precedence != 0 {
				prec = fmt.Sprintf("%2v", prod.Precedence)
			} else {
				prec = " -"
			}

			var assoc string
			if prod.Associativity != "" {
				assoc = prod.Associativity
			} else {
				assoc = "-"
			}

			var b strings.Builder
			fmt.Fprintf(&b, "%v →", nonTermName(prod.LHS))
			if len(prod.RHS) > 0 {
				for _, e := range prod.RHS {
					if e > 0 {
						fmt.Fprintf(&b, " %v", termName(e))
					} else {
						fmt.Fprintf(&b, " %v", nonTermName(e*-1))
					}
				}
			} else {
				fmt.Fprintf(&b, " ε")
			}

			return fmt.Sprintf("%4v %v %v %v", prod.Number, prec, assoc, b.String())
		},
		"printItem": func(item spec.Item) string {
			prod := desc.Productions[item.Production]

			var b strings.Builder
			fmt.Fprintf(&b, "%v →", nonTermName(prod.LHS))
			for i, e := range prod.RHS {
				if i == item.Dot {
					fmt.Fprintf(&b, " ・")
				}
				if e > 0 {
					fmt.Fprintf(&b, " %v", termName(e))
				} else {
					fmt.Fprintf(&b, " %v", nonTermName(e*-1))
				}
			}
			if item.Dot >= len(prod.RHS) {
				fmt.Fprintf(&b, " ・")
			}

			return fmt.Sprintf("%4v %v", prod.Number, b.String())
		},
		"printShift": func(tran spec.Transition) string {
			return fmt.Sprintf("shift  %4v on %v", tran.State, termName(tran.Symbol))
		},
		"printReduce": func(reduce spec.Reduce) string {
			var b strings.Builder
			{
				fmt.Fprintf(&b, "%v", termName(reduce.LookAhead[0]))
				for _, a := range reduce.LookAhead[1:] {
					fmt.Fprintf(&b, ", %v", termName(a))
				}
			}
			return fmt.Sprintf("reduce %4v on %v", reduce.Production, b.String())
		},
		"printGoTo": func(tran spec.Transition) string {
			return fmt.Sprintf("goto   %4v on %v", tran.State, nonTermName(tran.Symbol))
		},
		"printSRConflict": func(sr spec.SRConflict) string {
			var adopted string
			switch {
			case sr.AdoptedState != nil:
				adopted = fmt.Sprintf("shift %v", *sr.AdoptedState)
			case sr.AdoptedProduction != nil:
				adopted = fmt.Sprintf("reduce %v", *sr.AdoptedProduction)
			}
			var resolvedBy string
			switch sr.ResolvedBy {
			case grammar.ResolvedByPrec.Int():
				if sr.AdoptedState != nil {
					resolvedBy = fmt.Sprintf("symbol %v has higher precedence than production %v", termName(sr.Symbol), sr.Production)
				} else {
					resolvedBy = fmt.Sprintf("production %v has higher precedence than symbol %v", sr.Production, termName(sr.Symbol))
				}
			case grammar.ResolvedByAssoc.Int():
				if sr.AdoptedState != nil {
					resolvedBy = fmt.Sprintf("symbol %v and production %v has the same precedence, and symbol %v has %v associativity", termName(sr.Symbol), sr.Production, termName(sr.Symbol), termAssoc(sr.Symbol))
				} else {
					resolvedBy = fmt.Sprintf("production %v and symbol %v has the same precedence, and production %v has %v associativity", sr.Production, termName(sr.Symbol), sr.Production, prodAssoc(sr.Production))
				}
			case grammar.ResolvedByShift.Int():
				resolvedBy = fmt.Sprintf("symbol %v and production %v don't define a precedence comparison (default rule)", sr.Symbol, sr.Production)
			default:
				resolvedBy = "?" // This is a bug.
			}
			return fmt.Sprintf("shift/reduce conflict (shift %v, reduce %v) on %v: %v adopted because %v", sr.State, sr.Production, termName(sr.Symbol), adopted, resolvedBy)
		},
		"printRRConflict": func(rr spec.RRConflict) string {
			var resolvedBy string
			switch rr.ResolvedBy {
			case grammar.ResolvedByProdOrder.Int():
				resolvedBy = fmt.Sprintf("production %v and %v don't define a precedence comparison (default rule)", rr.Production1, rr.Production2)
			default:
				resolvedBy = "?" // This is a bug.
			}
			return fmt.Sprintf("reduce/reduce conflict (%v, %v) on %v: reduce %v adopted because %v", rr.Production1, rr.Production2, termName(rr.Symbol), rr.AdoptedProduction, resolvedBy)
		},
	}

	tmpl, err := template.New("").Funcs(fns).Parse(descTemplate)
	if err != nil {
		return err
	}

	err = tmpl.Execute(w, desc)
	if err != nil {
		return err
	}

	return nil
}
