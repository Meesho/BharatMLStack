package scylla

import (
	"strings"
	"testing"
)

// These tests verify that the Scylla DDL helpers reject unsafe identifiers
// *before* they reach the (here nil) gocql session. Because validation happens
// first, a malicious identifier returns a validation error rather than
// panicking on the nil session — which is exactly the guarantee we want.

func TestScylla_CreateTable_RejectsInjection(t *testing.T) {
	s := &Scylla{keySpace: "ks"} // session intentionally nil

	cases := []struct {
		name      string
		table     string
		pkColumns []string
	}{
		{"bad table", "t (id text PRIMARY KEY); DROP KEYSPACE ks; --", []string{"id"}},
		{"bad pk", "good_table", []string{"id text PRIMARY KEY) WITH x; --"}},
		{"empty table", "", []string{"id"}},
		{"pk with space", "good_table", []string{"id evil"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := s.CreateTable(c.table, c.pkColumns, 0); err == nil {
				t.Fatalf("CreateTable(%q, %v) = nil, want validation error", c.table, c.pkColumns)
			}
		})
	}
}

func TestScylla_AddColumn_RejectsInjection(t *testing.T) {
	s := &Scylla{keySpace: "ks"}

	cases := []struct {
		name   string
		table  string
		column string
	}{
		{"bad column", "good_table", "c blob; DROP TABLE good_table; --"},
		{"bad table", "bad table", "good_col"},
		{"empty column", "good_table", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := s.AddColumn(c.table, c.column); err == nil {
				t.Fatalf("AddColumn(%q, %q) = nil, want validation error", c.table, c.column)
			}
		})
	}
}

func TestSkyeScylla_RejectsInjection(t *testing.T) {
	s := &SkyeScylla{keySpace: "ks"} // session intentionally nil

	cases := []struct {
		name string
		// fn invokes one Skye helper with an unsafe input; it must return a
		// validation error before touching the nil session.
		fn func() error
	}{
		{
			"embedding column injection",
			func() error { return s.AddEmbeddingColumn("good_table", "c boolean; DROP TABLE x; --") },
		},
		{
			"aggregator column injection",
			func() error { return s.AddAggregatorColumn("good_table", "bad;col") },
		},
		{
			"embedding table injection",
			func() error { return s.CreateEmbeddingTable("t)", 0, []string{"v1"}) },
		},
		// NOTE: variant-identifier validation now happens on the table-creation
		// path (after the "already exists" check), so it can't be exercised here
		// with a nil session. That path is unit-tested in
		// pkg/cqlident: TestValidateColumn_NormalizedVariant.
		{
			"aggregator over-long table",
			func() error { return s.CreateAggregatorTable(strings.Repeat("a", 200), 0) },
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.fn(); err == nil {
				t.Fatalf("%s: got nil, want validation error", c.name)
			}
		})
	}
}
