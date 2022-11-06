package ucd

import (
	"fmt"
	"io"
)

type Scripts struct {
	Script             map[string][]*CodePointRange
	ScriptDefaultRange *CodePointRange
	ScriptDefaultValue string
}

// ParseScripts parses the Scripts.txt.
func ParseScripts(r io.Reader, propValAliases *PropertyValueAliases) (*Scripts, error) {
	ss := map[string][]*CodePointRange{}
	var defaultRange *CodePointRange
	var defaultValue string
	p := newParser(r)
	for p.parse() {
		if len(p.fields) > 0 {
			cp, err := p.fields[0].codePointRange()
			if err != nil {
				return nil, err
			}

			name, ok := propValAliases.Script[p.fields[1].normalizedSymbol()]
			if !ok {
				return nil, fmt.Errorf("unknown property: %v", p.fields[1].symbol())
			}
			ss[name] = append(ss[name], cp)
		}

		if len(p.defaultFields) > 0 {
			var err error
			defaultRange, err = p.defaultFields[0].codePointRange()
			if err != nil {
				return nil, err
			}
			defaultValue = p.defaultFields[1].normalizedSymbol()
		}
	}
	if p.err != nil {
		return nil, p.err
	}

	return &Scripts{
		Script:             ss,
		ScriptDefaultRange: defaultRange,
		ScriptDefaultValue: defaultValue,
	}, nil
}
