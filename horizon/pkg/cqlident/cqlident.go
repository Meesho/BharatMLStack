// Package cqlident provides strict validation for CQL identifiers
// (keyspace / table / column names).
//
// gocql cannot bind identifiers as query parameters, so any DDL that needs a
// dynamic table or column name has to build the statement with string
// interpolation. To keep that safe, every identifier that flows into such a
// statement MUST be validated against the allow-list below first, otherwise a
// crafted value (e.g. "t (id text PRIMARY KEY); DROP KEYSPACE x; --") could
// inject arbitrary CQL.
package cqlident

import (
	"fmt"
	"regexp"
)

const (
	// maxIdentLen bounds keyspace / table / primary-key names. Cassandra and
	// Scylla cap these unquoted identifiers at 48 characters, so we use the
	// same limit: a longer name would only pass this check and then fail at
	// DDL time, which is exactly the late failure this validator exists to
	// prevent.
	maxIdentLen = 48

	// maxColumnLen bounds column names. Feature-store and Skye column names are
	// generated from feature labels / variants and can legitimately be longer
	// than a table name, so they get a larger (but still finite) cap.
	maxColumnLen = 128
)

// identRe matches an unquoted CQL identifier: a letter or underscore followed
// by letters, digits or underscores. This intentionally rejects whitespace,
// quotes, semicolons, parentheses and every other character that could be used
// to break out of the intended statement.
var identRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Validate returns an error if name is not a safe, unquoted CQL identifier
// suitable for a keyspace, table or primary-key name. kind is used only to
// produce a helpful error message (e.g. "table", "primary key").
func Validate(kind, name string) error {
	return validate(kind, name, maxIdentLen)
}

// ValidateColumn validates a column identifier, allowing the larger column
// length cap. Use this for column names (including generated ones) rather than
// Validate, which uses the stricter table/keyspace cap.
func ValidateColumn(name string) error {
	return validate("column", name, maxColumnLen)
}

func validate(kind, name string, maxLen int) error {
	if name == "" {
		return fmt.Errorf("invalid %s identifier: must not be empty", kind)
	}
	if len(name) > maxLen {
		return fmt.Errorf("invalid %s identifier %q: must be at most %d characters", kind, name, maxLen)
	}
	if !identRe.MatchString(name) {
		return fmt.Errorf("invalid %s identifier %q: must match [a-zA-Z_][a-zA-Z0-9_]*", kind, name)
	}
	return nil
}

// ValidateAll validates every identifier in names, returning the first error.
func ValidateAll(kind string, names []string) error {
	for _, n := range names {
		if err := Validate(kind, n); err != nil {
			return err
		}
	}
	return nil
}
