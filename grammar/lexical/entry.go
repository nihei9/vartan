package lexical

import (
	"fmt"
	"sort"
	"strings"

	spec "github.com/nihei9/vartan/spec/grammar"
)

type LexEntry struct {
	Kind     spec.LexKindName
	Pattern  string
	Modes    []spec.LexModeName
	Push     spec.LexModeName
	Pop      bool
	Fragment bool
}

type LexSpec struct {
	Entries []*LexEntry
}

func (s *LexSpec) Validate() error {
	if len(s.Entries) <= 0 {
		return fmt.Errorf("the lexical specification must have at least one entry")
	}
	{
		ks := map[string]struct{}{}
		fks := map[string]struct{}{}
		for _, e := range s.Entries {
			// Allow duplicate names between fragments and non-fragments.
			if e.Fragment {
				if _, exist := fks[e.Kind.String()]; exist {
					return fmt.Errorf("kinds `%v` are duplicates", e.Kind)
				}
				fks[e.Kind.String()] = struct{}{}
			} else {
				if _, exist := ks[e.Kind.String()]; exist {
					return fmt.Errorf("kinds `%v` are duplicates", e.Kind)
				}
				ks[e.Kind.String()] = struct{}{}
			}
		}
	}
	{
		kinds := []string{}
		modes := []string{
			spec.LexModeNameDefault.String(), // This is a predefined mode.
		}
		for _, e := range s.Entries {
			if e.Fragment {
				continue
			}

			kinds = append(kinds, e.Kind.String())

			for _, m := range e.Modes {
				modes = append(modes, m.String())
			}
		}

		kindErrs := findSpellingInconsistenciesErrors(kinds, nil)
		modeErrs := findSpellingInconsistenciesErrors(modes, func(ids []string) error {
			if SnakeCaseToUpperCamelCase(ids[0]) == SnakeCaseToUpperCamelCase(spec.LexModeNameDefault.String()) {
				var b strings.Builder
				fmt.Fprintf(&b, "%+v", ids[0])
				for _, id := range ids[1:] {
					fmt.Fprintf(&b, ", %+v", id)
				}
				return fmt.Errorf("these identifiers are treated as the same. please use the same spelling as predefined '%v': %v", spec.LexModeNameDefault, b.String())
			}
			return nil
		})
		errs := append(kindErrs, modeErrs...)
		if len(errs) > 0 {
			var b strings.Builder
			fmt.Fprintf(&b, "%v", errs[0])
			for _, err := range errs[1:] {
				fmt.Fprintf(&b, "\n%v", err)
			}
			return fmt.Errorf(b.String())
		}
	}

	return nil
}

func findSpellingInconsistenciesErrors(ids []string, hook func(ids []string) error) []error {
	duplicated := FindSpellingInconsistencies(ids)
	if len(duplicated) == 0 {
		return nil
	}

	var errs []error
	for _, dup := range duplicated {
		if hook != nil {
			err := hook(dup)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}

		var b strings.Builder
		fmt.Fprintf(&b, "%+v", dup[0])
		for _, id := range dup[1:] {
			fmt.Fprintf(&b, ", %+v", id)
		}
		err := fmt.Errorf("these identifiers are treated as the same. please use the same spelling: %v", b.String())
		errs = append(errs, err)
	}

	return errs
}

// FindSpellingInconsistencies finds spelling inconsistencies in identifiers. The identifiers are considered to be the same
// if they are spelled the same when expressed in UpperCamelCase. For example, `left_paren` and `LeftParen` are spelled the same
// in UpperCamelCase. Thus they are considere to be spelling inconsistency.
func FindSpellingInconsistencies(ids []string) [][]string {
	m := map[string][]string{}
	for _, id := range removeDuplicates(ids) {
		c := SnakeCaseToUpperCamelCase(id)
		m[c] = append(m[c], id)
	}

	var duplicated [][]string
	for _, camels := range m {
		if len(camels) == 1 {
			continue
		}
		duplicated = append(duplicated, camels)
	}

	for _, dup := range duplicated {
		sort.Slice(dup, func(i, j int) bool {
			return dup[i] < dup[j]
		})
	}
	sort.Slice(duplicated, func(i, j int) bool {
		return duplicated[i][0] < duplicated[j][0]
	})

	return duplicated
}

func removeDuplicates(s []string) []string {
	m := map[string]struct{}{}
	for _, v := range s {
		m[v] = struct{}{}
	}

	var unique []string
	for v := range m {
		unique = append(unique, v)
	}

	return unique
}

func SnakeCaseToUpperCamelCase(snake string) string {
	elems := strings.Split(snake, "_")
	for i, e := range elems {
		if len(e) == 0 {
			continue
		}
		elems[i] = strings.ToUpper(string(e[0])) + e[1:]
	}

	return strings.Join(elems, "")
}
