package driver

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	goToken "go/token"
	"strconv"
	"strings"
	"text/template"

	spec "github.com/nihei9/vartan/spec/grammar"
)

//go:embed parser.go
var parserCoreSrc string

//go:embed semantic_action.go
var semActSrc string

func GenParser(cgram *spec.CompiledGrammar, pkgName string) ([]byte, error) {
	var parserSrc string
	{
		fset := goToken.NewFileSet()
		f, err := parser.ParseFile(fset, "parser.go", parserCoreSrc, parser.ParseComments)
		if err != nil {
			return nil, err
		}

		var b strings.Builder
		err = format.Node(&b, fset, f)
		if err != nil {
			return nil, err
		}

		parserSrc = b.String()
	}

	var grammarSrc string
	{
		t, err := template.New("").Funcs(genGrammarTemplateFuncs(cgram)).Parse(grammarSrcTmplate)
		if err != nil {
			return nil, err
		}

		var b strings.Builder
		err = t.Execute(&b, map[string]interface{}{
			"initialState":     cgram.ParsingTable.InitialState,
			"startProduction":  cgram.ParsingTable.StartProduction,
			"terminalCount":    cgram.ParsingTable.TerminalCount,
			"nonTerminalCount": cgram.ParsingTable.NonTerminalCount,
			"eofSymbol":        cgram.ParsingTable.EOFSymbol,
			"errorSymbol":      cgram.ParsingTable.ErrorSymbol,
		})
		if err != nil {
			return nil, err
		}

		grammarSrc = b.String()
	}

	var lexerSrc string
	{
		t, err := template.New("").Funcs(genLexerTemplateFuncs(cgram)).Parse(lexerSrcTmplate)
		if err != nil {
			return nil, err
		}

		var b strings.Builder
		err = t.Execute(&b, nil)
		if err != nil {
			return nil, err
		}

		lexerSrc = b.String()
	}

	var src string
	{
		tmpl := `// Code generated by vartan-go. DO NOT EDIT.
{{ .parserSrc }}

{{ .grammarSrc }}

{{ .lexerSrc }}
`
		t, err := template.New("").Parse(tmpl)
		if err != nil {
			return nil, err
		}

		var b strings.Builder
		err = t.Execute(&b, map[string]string{
			"parserSrc":  parserSrc,
			"grammarSrc": grammarSrc,
			"lexerSrc":   lexerSrc,
		})
		if err != nil {
			return nil, err
		}

		src = b.String()
	}

	fset := goToken.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	f.Name = ast.NewIdent(pkgName)

	// Complete an import statement.
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}
		gd.Specs = append(gd.Specs, &ast.ImportSpec{
			Path: &ast.BasicLit{
				Value: `"io"`,
			},
		})
		break
	}

	var b bytes.Buffer
	err = format.Node(&b, fset, f)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

const grammarSrcTmplate = `
type grammarImpl struct {
	recoverProductions      []int
	action                  []int
	goTo                    []int
	alternativeSymbolCounts []int
	errorTrapperStates      []int
	nonTerminals            []string
	lhsSymbols              []int
	terminals               []string
	terminalSkip            []int
	astActions              [][]int
}

func NewGrammar() *grammarImpl {
	return &grammarImpl{
		recoverProductions:      {{ genRecoverProductions }},
		action:                  {{ genAction }},
		goTo:                    {{ genGoTo }},
		alternativeSymbolCounts: {{ genAlternativeSymbolCounts }},
		errorTrapperStates:      {{ genErrorTrapperStates }},
		nonTerminals:            {{ genNonTerminals }},
		lhsSymbols:              {{ genLHSSymbols }},
		terminals:               {{ genTerminals }},
		terminalSkip:            {{ genTerminalSkip }},
		astActions:              {{ genASTActions }},
	}
}

func (g *grammarImpl) InitialState() int {
	return {{ .initialState }}
}

func (g *grammarImpl) StartProduction() int {
	return {{ .startProduction }}
}

func (g *grammarImpl) RecoverProduction(prod int) bool {
	return g.recoverProductions[prod] != 0
}

func (g *grammarImpl) Action(state int, terminal int) int {
	return g.action[state*{{ .terminalCount }}+terminal]
}

func (g *grammarImpl) GoTo(state int, lhs int) int {
	return g.goTo[state*{{ .nonTerminalCount }}+lhs]
}

func (g *grammarImpl) AlternativeSymbolCount(prod int) int {
	return g.alternativeSymbolCounts[prod]
}

func (g *grammarImpl) TerminalCount() int {
	return {{ .terminalCount }}
}

func (g *grammarImpl) SkipTerminal(terminal int) bool {
	return g.terminalSkip[terminal] == 1
}

func (g *grammarImpl) ErrorTrapperState(state int) bool {
	return g.errorTrapperStates[state] != 0
}

func (g *grammarImpl) NonTerminal(nonTerminal int) string {
	return g.nonTerminals[nonTerminal]
}

func (g *grammarImpl) LHS(prod int) int {
	return g.lhsSymbols[prod]
}

func (g *grammarImpl) EOF() int {
	return {{ .eofSymbol }}
}

func (g *grammarImpl) Error() int {
	return {{ .errorSymbol }}
}

func (g *grammarImpl) Terminal(terminal int) string {
	return g.terminals[terminal]
}

func (g *grammarImpl) ASTAction(prod int) []int {
	return g.astActions[prod]
}
`

func genGrammarTemplateFuncs(cgram *spec.CompiledGrammar) template.FuncMap {
	return template.FuncMap{
		"genRecoverProductions": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]int{\n")
			c := 1
			for _, v := range cgram.ParsingTable.RecoverProductions {
				fmt.Fprintf(&b, "%v, ", v)
				if c == 20 {
					fmt.Fprintf(&b, "\n")
					c = 1
				} else {
					c++
				}
			}
			if c > 1 {
				fmt.Fprintf(&b, "\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genAction": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]int{\n")
			c := 1
			for _, v := range cgram.ParsingTable.Action {
				fmt.Fprintf(&b, "%v, ", v)
				if c == 20 {
					fmt.Fprintf(&b, "\n")
					c = 1
				} else {
					c++
				}
			}
			if c > 1 {
				fmt.Fprintf(&b, "\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genGoTo": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]int{\n")
			c := 1
			for _, v := range cgram.ParsingTable.GoTo {
				fmt.Fprintf(&b, "%v, ", v)
				if c == 20 {
					fmt.Fprintf(&b, "\n")
					c = 1
				} else {
					c++
				}
			}
			if c > 1 {
				fmt.Fprintf(&b, "\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genAlternativeSymbolCounts": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]int{\n")
			c := 1
			for _, v := range cgram.ParsingTable.AlternativeSymbolCounts {
				fmt.Fprintf(&b, "%v, ", v)
				if c == 20 {
					fmt.Fprintf(&b, "\n")
					c = 1
				} else {
					c++
				}
			}
			if c > 1 {
				fmt.Fprintf(&b, "\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genErrorTrapperStates": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]int{\n")
			c := 1
			for _, v := range cgram.ParsingTable.ErrorTrapperStates {
				fmt.Fprintf(&b, "%v, ", v)
				if c == 20 {
					fmt.Fprintf(&b, "\n")
					c = 1
				} else {
					c++
				}
			}
			if c > 1 {
				fmt.Fprintf(&b, "\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genNonTerminals": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]string{\n")
			for _, v := range cgram.ParsingTable.NonTerminals {
				fmt.Fprintf(&b, "%v,\n", strconv.Quote(v))
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genLHSSymbols": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]int{\n")
			c := 1
			for _, v := range cgram.ParsingTable.LHSSymbols {
				fmt.Fprintf(&b, "%v, ", v)
				if c == 20 {
					fmt.Fprintf(&b, "\n")
					c = 1
				} else {
					c++
				}
			}
			if c > 1 {
				fmt.Fprintf(&b, "\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genTerminals": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]string{\n")
			for _, v := range cgram.ParsingTable.Terminals {
				fmt.Fprintf(&b, "%v,\n", strconv.Quote(v))
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genTerminalSkip": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]int{\n")
			c := 1
			for _, v := range cgram.ParsingTable.TerminalSkip {
				fmt.Fprintf(&b, "%v, ", v)
				if c == 20 {
					fmt.Fprintf(&b, "\n")
					c = 1
				} else {
					c++
				}
			}
			if c > 1 {
				fmt.Fprintf(&b, "\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
		"genASTActions": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[][]int{\n")
			for _, entries := range cgram.ASTAction.Entries {
				if len(entries) == 0 {
					fmt.Fprintf(&b, "nil,\n")
					continue
				}

				fmt.Fprintf(&b, "{\n")
				c := 1
				for _, v := range entries {
					fmt.Fprintf(&b, "%v, ", v)
					if c == 20 {
						fmt.Fprintf(&b, "\n")
						c = 1
					} else {
						c++
					}
				}
				if c > 1 {
					fmt.Fprintf(&b, "\n")
				}
				fmt.Fprintf(&b, "},\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
	}
}

const lexerSrcTmplate = `
type vToken struct {
	terminalID int
	tok        *Token
}

func (t *vToken) TerminalID() int {
	return t.terminalID
}

func (t *vToken) Lexeme() []byte {
	return t.tok.Lexeme
}

func (t *vToken) EOF() bool {
	return t.tok.EOF
}

func (t *vToken) Invalid() bool {
	return t.tok.Invalid
}

func (t *vToken) Position() (int, int) {
	return t.tok.Row, t.tok.Col
}

var kindToTerminal = {{ genKindToTerminal }}

type tokenStream struct {
	lex            *Lexer
	kindToTerminal []int
}

func NewTokenStream(src io.Reader) (*tokenStream, error) {
	lex, err := NewLexer(NewLexSpec(), src)
	if err != nil {
		return nil, err
	}

	return &tokenStream{
		lex: lex,
	}, nil
}

func (t *tokenStream) Next() (VToken, error) {
	tok, err := t.lex.Next()
	if err != nil {
		return nil, err
	}
	return &vToken{
		terminalID: kindToTerminal[tok.KindID],
		tok:        tok,
	}, nil
}
`

func genLexerTemplateFuncs(cgram *spec.CompiledGrammar) template.FuncMap {
	return template.FuncMap{
		"genKindToTerminal": func() string {
			var b strings.Builder
			fmt.Fprintf(&b, "[]int{\n")
			c := 1
			for _, v := range cgram.LexicalSpecification.Maleeni.KindToTerminal {
				fmt.Fprintf(&b, "%v, ", v)
				if c == 20 {
					fmt.Fprintf(&b, "\n")
					c = 1
				} else {
					c++
				}
			}
			if c > 1 {
				fmt.Fprintf(&b, "\n")
			}
			fmt.Fprintf(&b, "}")
			return b.String()
		},
	}
}

func GenSemanticAction(pkgName string) ([]byte, error) {
	var src string
	{
		tmpl := `// Code generated by vartan-go. DO NOT EDIT.
{{ .semActSrc }}
`
		t, err := template.New("").Parse(tmpl)
		if err != nil {
			return nil, err
		}

		var b strings.Builder
		err = t.Execute(&b, map[string]string{
			"semActSrc": semActSrc,
		})
		if err != nil {
			return nil, err
		}

		src = b.String()
	}

	fset := goToken.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	f.Name = ast.NewIdent(pkgName)

	var b bytes.Buffer
	err = format.Node(&b, fset, f)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
