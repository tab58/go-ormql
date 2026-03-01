package strutil

import "testing"

// === L-strutil: Table-driven tests for Capitalize and PluralLower ===

// TestCapitalize verifies that Capitalize uppercases the first character
// and leaves the rest unchanged.
// Expected: each input maps to the expected output.
func TestCapitalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Empty string returns empty string unchanged.
		{"empty string", "", ""},
		// Single lowercase letter is uppercased.
		{"single lowercase char", "a", "A"},
		// Single uppercase letter remains unchanged.
		{"single uppercase char", "A", "A"},
		// Already capitalized string stays the same.
		{"already capitalized", "Hello", "Hello"},
		// All lowercase string has first char uppercased.
		{"all lowercase", "hello", "Hello"},
		// Multi-word string only capitalizes the first character.
		{"multi-word", "hello world", "Hello world"},
		// All uppercase string stays the same (first char already upper).
		{"all uppercase", "HELLO", "HELLO"},
		// Numeric first character is unchanged (ToUpper is no-op on digits).
		{"numeric first char", "123abc", "123abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Capitalize(tt.input)
			if got != tt.expected {
				t.Errorf("Capitalize(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestPluralLower verifies that PluralLower lowercases the first character
// and appends "s" to produce a naive plural form.
// Expected: each input maps to the expected output.
func TestPluralLower(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Empty string returns empty string unchanged.
		{"empty string", "", ""},
		// Single uppercase letter is lowercased and pluralized.
		{"single char uppercase", "A", "as"},
		// Single lowercase letter stays lowercase and is pluralized.
		{"single char lowercase", "a", "as"},
		// Standard GraphQL type name "Movie" → "movies".
		{"standard type Movie", "Movie", "movies"},
		// Standard GraphQL type name "Person" → "persons".
		{"standard type Person", "Person", "persons"},
		// Already lowercase input stays lowercase.
		{"already lowercase", "movie", "movies"},
		// Input ending in 's' still appends 's' (naive pluralization).
		{"ends in s", "Status", "statuss"},
		// Multi-word input only lowercases the first character.
		{"multi-word", "MovieGenre", "movieGenres"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PluralLower(tt.input)
			if got != tt.expected {
				t.Errorf("PluralLower(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
