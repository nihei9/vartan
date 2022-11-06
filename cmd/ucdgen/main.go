package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/nihei9/vartan/ucd"
)

func main() {
	err := gen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func gen() error {
	var propValAliases *ucd.PropertyValueAliases
	{
		resp, err := http.Get("https://www.unicode.org/Public/13.0.0/ucd/PropertyValueAliases.txt")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		propValAliases, err = ucd.ParsePropertyValueAliases(resp.Body)
		if err != nil {
			return err
		}
	}
	var unicodeData *ucd.UnicodeData
	{
		resp, err := http.Get("https://www.unicode.org/Public/13.0.0/ucd/UnicodeData.txt")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		unicodeData, err = ucd.ParseUnicodeData(resp.Body, propValAliases)
		if err != nil {
			return err
		}
	}
	var scripts *ucd.Scripts
	{
		resp, err := http.Get("https://www.unicode.org/Public/13.0.0/ucd/Scripts.txt")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		scripts, err = ucd.ParseScripts(resp.Body, propValAliases)
		if err != nil {
			return err
		}
	}
	var propList *ucd.PropList
	{
		resp, err := http.Get("https://www.unicode.org/Public/13.0.0/ucd/PropList.txt")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		propList, err = ucd.ParsePropList(resp.Body)
		if err != nil {
			return err
		}
	}
	tmpl, err := template.ParseFiles("../ucd/codepoint.go.tmpl")
	if err != nil {
		return err
	}
	var b strings.Builder
	err = tmpl.Execute(&b, struct {
		GeneratorName        string
		UnicodeData          *ucd.UnicodeData
		Scripts              *ucd.Scripts
		PropList             *ucd.PropList
		PropertyValueAliases *ucd.PropertyValueAliases
	}{
		GeneratorName:        "generator/main.go",
		UnicodeData:          unicodeData,
		Scripts:              scripts,
		PropList:             propList,
		PropertyValueAliases: propValAliases,
	})
	if err != nil {
		return err
	}
	f, err := os.OpenFile("../ucd/codepoint.go", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprint(f, b.String())
	return nil
}
