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
	s := &SkyeScylla{keySpace: "ks"}

	if err := s.AddEmbeddingColumn("good_table", "c boolean; DROP TABLE x; --"); err == nil {
		t.Error("AddEmbeddingColumn with injected column = nil, want error")
	}
	if err := s.AddAggregatorColumn("good_table", "bad;col"); err == nil {
		t.Error("AddAggregatorColumn with injected column = nil, want error")
	}
	if err := s.CreateEmbeddingTable("t)", 0, []string{"v1"}); err == nil {
		t.Error("CreateEmbeddingTable with injected table = nil, want error")
	}
	if err := s.CreateEmbeddingTable("good_table", 0, []string{"v1; DROP"}); err == nil {
		t.Error("CreateEmbeddingTable with injected variant = nil, want error")
	}
	if err := s.CreateAggregatorTable(strings.Repeat("a", 200), 0); err == nil {
		t.Error("CreateAggregatorTable with over-long table = nil, want error")
	}
}
