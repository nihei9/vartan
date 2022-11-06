package ucd

import "io"

type UnicodeData struct {
	GeneralCategory map[string][]*CodePointRange

	propValAliases *PropertyValueAliases
}

// ParseUnicodeData parses the UnicodeData.txt.
func ParseUnicodeData(r io.Reader, propValAliases *PropertyValueAliases) (*UnicodeData, error) {
	unicodeData := &UnicodeData{
		GeneralCategory: map[string][]*CodePointRange{},
		propValAliases:  propValAliases,
	}

	p := newParser(r)
	for p.parse() {
		if len(p.fields) == 0 {
			continue
		}
		cp, err := p.fields[0].codePointRange()
		if err != nil {
			return nil, err
		}
		gc := p.fields[2].normalizedSymbol()
		unicodeData.addGC(gc, cp)
	}
	if p.err != nil {
		return nil, p.err
	}

	return unicodeData, nil
}

func (u *UnicodeData) addGC(gc string, cp *CodePointRange) {
	// https://www.unicode.org/reports/tr44/#Empty_Fields
	// > The data file UnicodeData.txt defines many property values in each record. When a field in a data line
	// > for a code point is empty, that indicates that the property takes the default value for that code point.
	if gc == "" {
		return
	}

	cps, ok := u.GeneralCategory[u.propValAliases.gcAbb(gc)]
	if ok {
		c := cps[len(cps)-1]
		if cp.From-c.To == 1 {
			c.To = cp.To
		} else {
			u.GeneralCategory[u.propValAliases.gcAbb(gc)] = append(cps, cp)
		}
	} else {
		u.GeneralCategory[u.propValAliases.gcAbb(gc)] = []*CodePointRange{cp}
	}
}
