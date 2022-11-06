//go:generate go run ../cmd/ucdgen/main.go
//go:generate go fmt codepoint.go

package ucd

import (
	"fmt"
	"strings"
)

const (
	// https://www.unicode.org/versions/Unicode13.0.0/ch03.pdf
	// 3.4  Characters and Encoding
	// > D9 Unicode codespace: A range of integers from 0 to 10FFFF16.
	codePointMin = 0x0
	codePointMax = 0x10FFFF
)

func NormalizeCharacterProperty(propName, propVal string) (string, error) {
	if propName == "" {
		propName = "gc"
	}

	name, ok := propertyNameAbbs[normalizeSymbolicValue(propName)]
	if !ok {
		return "", fmt.Errorf("unsupported character property name: %v", propName)
	}
	props, ok := derivedCoreProperties[name]
	if !ok {
		return "", nil
	}
	var b strings.Builder
	yes, ok := binaryValues[normalizeSymbolicValue(propVal)]
	if !ok {
		return "", fmt.Errorf("unsupported character property value: %v", propVal)
	}
	if yes {
		fmt.Fprint(&b, "[")
	} else {
		fmt.Fprint(&b, "[^")
	}
	for _, prop := range props {
		fmt.Fprint(&b, prop)
	}
	fmt.Fprint(&b, "]")

	return b.String(), nil
}

func IsContributoryProperty(propName string) bool {
	if propName == "" {
		return false
	}

	for _, p := range contributoryProperties {
		if propName == p {
			return true
		}
	}
	return false
}

func FindCodePointRanges(propName, propVal string) ([]*CodePointRange, bool, error) {
	if propName == "" {
		propName = "gc"
	}

	name, ok := propertyNameAbbs[normalizeSymbolicValue(propName)]
	if !ok {
		return nil, false, fmt.Errorf("unsupported character property name: %v", propName)
	}
	switch name {
	case "gc":
		val, ok := generalCategoryValueAbbs[normalizeSymbolicValue(propVal)]
		if !ok {
			return nil, false, fmt.Errorf("unsupported character property value: %v", propVal)
		}
		if val == generalCategoryValueAbbs[normalizeSymbolicValue(generalCategoryDefaultValue)] {
			var allCPs []*CodePointRange
			if generalCategoryDefaultRange.From > codePointMin {
				allCPs = append(allCPs, &CodePointRange{
					From: codePointMin,
					To:   generalCategoryDefaultRange.From - 1,
				})
			}
			if generalCategoryDefaultRange.To < codePointMax {
				allCPs = append(allCPs, &CodePointRange{
					From: generalCategoryDefaultRange.To + 1,
					To:   codePointMax,
				})
			}
			for _, cp := range generalCategoryCodePoints {
				allCPs = append(allCPs, cp...)
			}
			return allCPs, true, nil
		}
		vals, ok := compositGeneralCategories[val]
		if !ok {
			vals = []string{val}
		}
		var ranges []*CodePointRange
		for _, v := range vals {
			rs, ok := generalCategoryCodePoints[v]
			if !ok {
				return nil, false, fmt.Errorf("invalid value of the General_Category property: %v", v)
			}
			ranges = append(ranges, rs...)
		}
		return ranges, false, nil
	case "sc":
		val, ok := scriptValueAbbs[normalizeSymbolicValue(propVal)]
		if !ok {
			return nil, false, fmt.Errorf("unsupported character property value: %v", propVal)
		}
		if val == scriptValueAbbs[normalizeSymbolicValue(scriptDefaultValue)] {
			var allCPs []*CodePointRange
			if scriptDefaultRange.From > codePointMin {
				allCPs = append(allCPs, &CodePointRange{
					From: codePointMin,
					To:   scriptDefaultRange.From - 1,
				})
			}
			if scriptDefaultRange.To < codePointMax {
				allCPs = append(allCPs, &CodePointRange{
					From: scriptDefaultRange.To + 1,
					To:   codePointMax,
				})
			}
			for _, cp := range scriptCodepoints {
				allCPs = append(allCPs, cp...)
			}
			return allCPs, true, nil
		}
		return scriptCodepoints[val], false, nil
	case "oalpha":
		yes, ok := binaryValues[normalizeSymbolicValue(propVal)]
		if !ok {
			return nil, false, fmt.Errorf("unsupported character property value: %v", propVal)
		}
		if yes {
			return otherAlphabeticCodePoints, false, nil
		} else {
			return otherAlphabeticCodePoints, true, nil
		}
	case "olower":
		yes, ok := binaryValues[normalizeSymbolicValue(propVal)]
		if !ok {
			return nil, false, fmt.Errorf("unsupported character property value: %v", propVal)
		}
		if yes {
			return otherLowercaseCodePoints, false, nil
		} else {
			return otherLowercaseCodePoints, true, nil
		}
	case "oupper":
		yes, ok := binaryValues[normalizeSymbolicValue(propVal)]
		if !ok {
			return nil, false, fmt.Errorf("unsupported character property value: %v", propVal)
		}
		if yes {
			return otherUppercaseCodePoints, false, nil
		} else {
			return otherUppercaseCodePoints, true, nil
		}
	case "wspace":
		yes, ok := binaryValues[normalizeSymbolicValue(propVal)]
		if !ok {
			return nil, false, fmt.Errorf("unsupported character property value: %v", propVal)
		}
		if yes {
			return whiteSpaceCodePoints, false, nil
		} else {
			return whiteSpaceCodePoints, true, nil
		}
	}

	// If the process reaches this code, it's a bug. We must handle all of the properties registered with
	// the `propertyNameAbbs`.
	return nil, false, fmt.Errorf("character property '%v' is unavailable", propName)
}
