package polars

import "testing"

// TestClassifyChangeConstantValues pins the ChangeClass constants to the values
// documented in compatibility.go.
func TestClassifyChangeConstantValues(t *testing.T) {
	cases := []struct {
		name string
		got  ChangeClass
		want ChangeClass
	}{
		{"ChangeClassCompatible", ChangeClassCompatible, "compatible"},
		{"ChangeClassDeprecating", ChangeClassDeprecating, "deprecating"},
		{"ChangeClassBreaking", ChangeClassBreaking, "breaking"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.got != c.want {
				t.Fatalf("%s = %q, want %q", c.name, c.got, c.want)
			}
		})
	}
}

// TestClassifyChangeClassifies exercises the documented heuristics plus the
// default (ambiguous) case that must return ChangeClassCompatible.
func TestClassifyChangeClassifies(t *testing.T) {
	cases := []struct {
		name        string
		description string
		want        ChangeClass
	}{
		{"breaking_keyword", "Breaking change in API surface", ChangeClassBreaking},
		{"remove_keyword", "Remove deprecated Foo()", ChangeClassBreaking},
		{"drop_keyword", "Drop support for old format", ChangeClassBreaking},
		{"deprecate_keyword", "Deprecate the legacy entry point", ChangeClassDeprecating},
		{"deprecating_keyword", "Deprecating the old serializer", ChangeClassDeprecating},
		{"default_compatible", "Fix off-by-one in null count", ChangeClassCompatible},
		{"empty_string_defaults_to_compatible", "", ChangeClassCompatible},
		{"case_insensitive_breaking", "BREAKING CHANGE to schema", ChangeClassBreaking},
		{"case_insensitive_deprecating", "DEPRECATE old function", ChangeClassDeprecating},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ClassifyChange(c.description); got != c.want {
				t.Fatalf("ClassifyChange(%q) = %q, want %q", c.description, got, c.want)
			}
		})
	}
}
