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

func TestValidateColumn_LengthCap(t *testing.T) {
	// A column name longer than the table cap but within the column cap is
	// valid for ValidateColumn but rejected by Validate.
	name := strings.Repeat("c", maxIdentLen+1)
	if err := ValidateColumn(name); err != nil {
		t.Errorf("ValidateColumn(%d chars) returned unexpected error: %v", len(name), err)
	}
	if err := Validate("table", name); err == nil {
		t.Errorf("Validate(%d chars) = nil, want length error", len(name))
	}

	// Exactly at the column cap is allowed; one over is not.
	if err := ValidateColumn(strings.Repeat("c", maxColumnLen)); err != nil {
		t.Errorf("ValidateColumn at maxColumnLen returned error: %v", err)
	}
	if err := ValidateColumn(strings.Repeat("c", maxColumnLen+1)); err == nil {
		t.Error("ValidateColumn over maxColumnLen = nil, want error")
	}

	// Injection is still rejected regardless of the larger cap.
	if err := ValidateColumn("c boolean; DROP TABLE x; --"); err == nil {
		t.Error("ValidateColumn(injection) = nil, want error")
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

// TestValidateColumn_NormalizedVariant mirrors the transform SkyeScylla applies
// to embedding variants (hyphen -> underscore, lower-cased, suffixed). The
// resulting column name must be accepted, while a variant carrying an injection
// payload must still be rejected after normalization.
func TestValidateColumn_NormalizedVariant(t *testing.T) {
	normalize := func(variant string) string {
		return strings.ToLower(strings.ReplaceAll(variant, "-", "_")) + "_to_be_indexed"
	}

	if err := ValidateColumn(normalize("model-v1")); err != nil {
		t.Errorf("normalized variant %q rejected: %v", "model-v1", err)
	}
	if err := ValidateColumn(normalize("v1; DROP")); err == nil {
		t.Error("normalized injection variant = nil, want error")
	}
}
