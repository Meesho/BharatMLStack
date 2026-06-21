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

// maxIdentLen is a defensive upper bound. Cassandra/Scylla limit unquoted
// keyspace and table names to 48 characters; we allow a little more headroom
// for generated column names while still bounding the value.
const maxIdentLen = 64

// identRe matches an unquoted CQL identifier: a letter or underscore followed
// by letters, digits or underscores. This intentionally rejects whitespace,
// quotes, semicolons, parentheses and every other character that could be used
// to break out of the intended statement.
var identRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Validate returns an error if name is not a safe, unquoted CQL identifier.
// kind is used only to produce a helpful error message (e.g. "table",
// "column", "primary key").
func Validate(kind, name string) error {
	if name == "" {
		return fmt.Errorf("invalid %s identifier: must not be empty", kind)
	}
	if len(name) > maxIdentLen {
		return fmt.Errorf("invalid %s identifier %q: must be at most %d characters", kind, name, maxIdentLen)
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
