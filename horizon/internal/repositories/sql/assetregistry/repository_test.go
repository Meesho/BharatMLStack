package assetregistry

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
	var _ Repository = (*AssetRegistry)(nil)
}

func TestToTableAndBack(t *testing.T) {
	schedule := "3h"
	freshness := "2h"
	checks := `["check1","check2"]`

	row := AssetSpecRow{
		AssetName:    "user_spend",
		AssetVersion: "abc123",
		EntityName:   "user",
		EntityKey:    "user_id",
		Notebook:     "user_features",
		PartitionKey: "ds",
		TriggerType:  "schedule_3h",
		Schedule:     &schedule,
		Serving:      true,
		Incremental:  false,
		Freshness:    &freshness,
		ChecksJSON:   &checks,
		SpecJSON:     `{"name":"user_spend"}`,
	}

	tbl := toTable(row)
	assert.Equal(t, row.AssetName, tbl.AssetName)
	assert.Equal(t, row.AssetVersion, tbl.AssetVersion)
	assert.Equal(t, row.EntityName, tbl.EntityName)
	assert.Equal(t, row.EntityKey, tbl.EntityKey)
	assert.Equal(t, row.Notebook, tbl.Notebook)
	assert.Equal(t, row.PartitionKey, tbl.PartitionKey)
	assert.Equal(t, row.TriggerType, tbl.TriggerType)
	assert.Equal(t, row.Schedule, tbl.Schedule)
	assert.Equal(t, row.Serving, tbl.Serving)
	assert.Equal(t, row.Incremental, tbl.Incremental)
	assert.Equal(t, row.Freshness, tbl.Freshness)
	assert.Equal(t, row.ChecksJSON, tbl.ChecksJSON)
	assert.Equal(t, row.SpecJSON, tbl.SpecJSON)

	tbl.ID = 42
	tbl.CreatedAt = time.Now()
	tbl.UpdatedAt = time.Now()

	back := toRow(tbl)
	assert.Equal(t, int64(42), back.ID)
	assert.Equal(t, row.AssetName, back.AssetName)
	assert.Equal(t, row.AssetVersion, back.AssetVersion)
	assert.Equal(t, row.EntityName, back.EntityName)
	assert.Equal(t, row.EntityKey, back.EntityKey)
	assert.Equal(t, row.Notebook, back.Notebook)
	assert.Equal(t, row.PartitionKey, back.PartitionKey)
	assert.Equal(t, row.TriggerType, back.TriggerType)
	assert.Equal(t, row.Schedule, back.Schedule)
	assert.Equal(t, row.Serving, back.Serving)
	assert.Equal(t, row.Incremental, back.Incremental)
	assert.Equal(t, row.Freshness, back.Freshness)
	assert.Equal(t, row.ChecksJSON, back.ChecksJSON)
	assert.Equal(t, row.SpecJSON, back.SpecJSON)
	assert.Equal(t, tbl.CreatedAt, back.CreatedAt)
	assert.Equal(t, tbl.UpdatedAt, back.UpdatedAt)
}

func TestToTableNullableFieldsNil(t *testing.T) {
	row := AssetSpecRow{
		AssetName:    "product_features",
		AssetVersion: "def456",
		EntityName:   "product",
		EntityKey:    "product_id",
		Notebook:     "product_notebook",
		PartitionKey: "ds",
		TriggerType:  "upstream",
		Schedule:     nil,
		Serving:      false,
		Incremental:  true,
		Freshness:    nil,
		ChecksJSON:   nil,
		SpecJSON:     `{}`,
	}

	tbl := toTable(row)
	assert.Nil(t, tbl.Schedule)
	assert.Nil(t, tbl.Freshness)
	assert.Nil(t, tbl.ChecksJSON)
	assert.False(t, tbl.Serving)
	assert.True(t, tbl.Incremental)
}

func TestToRows(t *testing.T) {
	tables := []Table{
		{ID: 1, AssetName: "asset_a", SpecJSON: "{}"},
		{ID: 2, AssetName: "asset_b", SpecJSON: "{}"},
		{ID: 3, AssetName: "asset_c", SpecJSON: "{}"},
	}

	rows := toRows(tables)
	assert.Len(t, rows, 3)
	assert.Equal(t, "asset_a", rows[0].AssetName)
	assert.Equal(t, "asset_b", rows[1].AssetName)
	assert.Equal(t, "asset_c", rows[2].AssetName)
	assert.Equal(t, int64(1), rows[0].ID)
	assert.Equal(t, int64(2), rows[1].ID)
	assert.Equal(t, int64(3), rows[2].ID)
}

func TestTableName(t *testing.T) {
	tbl := Table{}
	assert.Equal(t, "asset_specs", tbl.TableName())
}
