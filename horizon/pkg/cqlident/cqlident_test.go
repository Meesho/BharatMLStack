package cqlident

import (
	"strings"
	"testing"
)

func TestValidate_Valid(t *testing.T) {
	valid := []string{
		"users",
		"feature_group_1",
		"_internal",
		"seg_42",
		"a",
		"A",
		"Col123",
		strings.Repeat("a", maxIdentLen), // exactly at the limit
	}
	for _, name := range valid {
		if err := Validate("table", name); err != nil {
			t.Errorf("Validate(%q) returned unexpected error: %v", name, err)
		}
	}
}

func TestValidate_Invalid(t *testing.T) {
	// These are the injection / malformed cases that MUST be rejected.
	invalid := []string{
		"",                                    // empty
		"1abc",                                // starts with a digit
		"foo bar",                             // whitespace
		"foo;DROP KEYSPACE x",                 // statement separator
		"foo)",                                // closing paren breakout
		"t (id text PRIMARY KEY); DROP TABLE", // classic stacked-query payload
		"foo-bar",                             // hyphen not allowed (unquoted)
		"foo.bar",                             // dotted name
		`foo"`,                                // quote
		"foo--comment",                        // CQL comment
		"naïve",                               // non-ASCII
		strings.Repeat("a", maxIdentLen+1),    // over the length cap
	}
	for _, name := range invalid {
		if err := Validate("column", name); err == nil {
			t.Errorf("Validate(%q) = nil, want error", name)
		}
	}
}

func TestValidateAll(t *testing.T) {
	if err := ValidateAll("primary key", []string{"id", "tenant_id"}); err != nil {
		t.Errorf("ValidateAll(valid) returned error: %v", err)
	}
	if err := ValidateAll("primary key", []string{"id", "bad;name"}); err == nil {
		t.Error("ValidateAll(with invalid) = nil, want error")
	}
	if err := ValidateAll("primary key", nil); err != nil {
		t.Errorf("ValidateAll(nil) returned error: %v", err)
	}
}
