package ucd

import "io"

type PropertyValueAliases struct {
	GeneralCategory             map[string]string
	GeneralCategoryDefaultRange *CodePointRange
	GeneralCategoryDefaultValue string

	Script map[string]string
}

// ParsePropertyValueAliases parses the PropertyValueAliases.txt.
func ParsePropertyValueAliases(r io.Reader) (*PropertyValueAliases, error) {
	gcAbbs := map[string]string{}
	var defaultGCCPRange *CodePointRange
	var defaultGCVal string
	scAbbs := map[string]string{}
	p := newParser(r)
	for p.parse() {
		// https://www.unicode.org/reports/tr44/#Property_Value_Aliases
		// > In PropertyValueAliases.txt, the first field contains the abbreviated alias for a Unicode property,
		// > the second field specifies an abbreviated symbolic name for a value of that property, and the third
		// > field specifies the long symbolic name for that value of that property. These are the preferred
		// > aliases. Additional aliases for some property values may be specified in the fourth or subsequent
		// > fields.
		if len(p.fields) > 0 {
			switch p.fields[0].symbol() {
			case "gc":
				gcShort := p.fields[1].normalizedSymbol()
				gcLong := p.fields[2].normalizedSymbol()
				gcAbbs[gcShort] = gcShort
				gcAbbs[gcLong] = gcShort
				for _, f := range p.fields[3:] {
					gcShortOther := f.normalizedSymbol()
					gcAbbs[gcShortOther] = gcShort
				}
			case "sc":
				scShort := p.fields[1].normalizedSymbol()
				scLong := p.fields[2].normalizedSymbol()
				scAbbs[scShort] = scShort
				scAbbs[scLong] = scShort
				for _, f := range p.fields[3:] {
					scShortOther := f.normalizedSymbol()
					scAbbs[scShortOther] = scShort
				}
			}
		}

		// https://www.unicode.org/reports/tr44/#Missing_Conventions
		// > @missing lines are also supplied for many properties in the file PropertyValueAliases.txt.
		// > ...
		// > there are currently two syntactic patterns used for @missing lines, as summarized schematically below:
		// >     1. code_point_range; default_prop_val
		// >     2. code_point_range; property_name; default_prop_val
		// > ...
		// > Pattern #2 is used in PropertyValueAliases.txt and in DerivedNormalizationProps.txt, both of which
		// > contain values associated with many properties. For example:
		// >     # @missing: 0000..10FFFF; NFD_QC; Yes
		if len(p.defaultFields) > 0 && p.defaultFields[1].symbol() == "General_Category" {
			var err error
			defaultGCCPRange, err = p.defaultFields[0].codePointRange()
			if err != nil {
				return nil, err
			}
			defaultGCVal = p.defaultFields[2].normalizedSymbol()
		}
	}
	if p.err != nil {
		return nil, p.err
	}
	return &PropertyValueAliases{
		GeneralCategory:             gcAbbs,
		GeneralCategoryDefaultRange: defaultGCCPRange,
		GeneralCategoryDefaultValue: defaultGCVal,
		Script:                      scAbbs,
	}, nil
}

func (a *PropertyValueAliases) gcAbb(gc string) string {
	return a.GeneralCategory[gc]
}
