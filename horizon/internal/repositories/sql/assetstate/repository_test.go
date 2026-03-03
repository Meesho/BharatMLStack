package assetstate

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRepository_NilConnection(t *testing.T) {
	repo, err := NewRepository(nil)
	assert.Nil(t, repo)
	assert.EqualError(t, err, "connection cannot be nil")
}

func TestInterfaceCompliance(t *testing.T) {
	var _ Repository = (*AssetState)(nil)
}

func TestToTableAndBack(t *testing.T) {
	servingOverride := true
	overrideReason := "manual override for testing"
	overrideBy := "admin@example.com"

	row := AssetNecessityRow{
		AssetName:       "user_spend",
		Necessity:       "active",
		ServingOverride: &servingOverride,
		OverrideReason:  &overrideReason,
		OverrideBy:      &overrideBy,
		Reason:          "serving=true, no override",
	}

	tbl := toTable(row)
	assert.Equal(t, row.AssetName, tbl.AssetName)
	assert.Equal(t, row.Necessity, tbl.Necessity)
	assert.Equal(t, row.ServingOverride, tbl.ServingOverride)
	assert.Equal(t, row.OverrideReason, tbl.OverrideReason)
	assert.Equal(t, row.OverrideBy, tbl.OverrideBy)
	assert.Equal(t, row.Reason, tbl.Reason)

	tbl.ResolvedAt = time.Now()

	back := toRow(tbl)
	assert.Equal(t, row.AssetName, back.AssetName)
	assert.Equal(t, row.Necessity, back.Necessity)
	assert.True(t, *back.ServingOverride)
	assert.Equal(t, "manual override for testing", *back.OverrideReason)
	assert.Equal(t, "admin@example.com", *back.OverrideBy)
	assert.Equal(t, row.Reason, back.Reason)
	assert.Equal(t, tbl.ResolvedAt, back.ResolvedAt)
}

func TestToTableNullableFieldsNil(t *testing.T) {
	row := AssetNecessityRow{
		AssetName:       "transient_asset",
		Necessity:       "transient",
		ServingOverride: nil,
		OverrideReason:  nil,
		OverrideBy:      nil,
		Reason:          "needed by downstream active asset",
	}

	tbl := toTable(row)
	assert.Nil(t, tbl.ServingOverride)
	assert.Nil(t, tbl.OverrideReason)
	assert.Nil(t, tbl.OverrideBy)

	back := toRow(tbl)
	assert.Nil(t, back.ServingOverride)
	assert.Nil(t, back.OverrideReason)
	assert.Nil(t, back.OverrideBy)
}

func TestToRows(t *testing.T) {
	tables := []Table{
		{AssetName: "asset_a", Necessity: "active", Reason: "serving=true"},
		{AssetName: "asset_b", Necessity: "transient", Reason: "downstream needs it"},
		{AssetName: "asset_c", Necessity: "skipped", Reason: "no downstream"},
	}

	rows := toRows(tables)
	assert.Len(t, rows, 3)

	tests := []struct {
		name      string
		necessity string
		reason    string
	}{
		{"asset_a", "active", "serving=true"},
		{"asset_b", "transient", "downstream needs it"},
		{"asset_c", "skipped", "no downstream"},
	}

	for i, tt := range tests {
		assert.Equal(t, tt.name, rows[i].AssetName)
		assert.Equal(t, tt.necessity, rows[i].Necessity)
		assert.Equal(t, tt.reason, rows[i].Reason)
	}
}

func TestTableName(t *testing.T) {
	tbl := Table{}
	assert.Equal(t, "asset_necessity", tbl.TableName())
}

func TestNecessityValues(t *testing.T) {
	validNecessities := []string{"active", "transient", "skipped"}
	for _, n := range validNecessities {
		row := AssetNecessityRow{Necessity: n}
		tbl := toTable(row)
		assert.Equal(t, n, tbl.Necessity)
	}
}
