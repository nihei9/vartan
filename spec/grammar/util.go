package grammar

import "strings"

var rep = strings.NewReplacer(
	`.`, `\.`,
	`*`, `\*`,
	`+`, `\+`,
	`?`, `\?`,
	`|`, `\|`,
	`(`, `\(`,
	`)`, `\)`,
	`[`, `\[`,
	`\`, `\\`,
)

// EscapePattern escapes the special characters.
// For example, EscapePattern(`+`) returns `\+`.
func EscapePattern(s string) string {
	return rep.Replace(s)
}
