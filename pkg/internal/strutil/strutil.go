package strutil

import "strings"

// Capitalize returns the string with its first character uppercased.
// GraphQL type names are ASCII, so byte-level uppercasing is safe.
func Capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// PluralLower returns the lowercase-first plural form.
// e.g. "Movie" → "movies".
func PluralLower(name string) string {
	if name == "" {
		return name
	}
	return strings.ToLower(name[:1]) + name[1:] + "s"
}
