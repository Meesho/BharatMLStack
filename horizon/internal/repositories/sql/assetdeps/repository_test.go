package assetdeps

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
	var _ Repository = (*AssetDeps)(nil)
}

func TestToTableAndBack(t *testing.T) {
	windowSize := 7

	row := AssetDependencyRow{
		AssetName:    "user_spend",
		InputName:    "silver.orders",
		InputType:    "external",
		PartitionKey: "ds",
		WindowSize:   &windowSize,
	}

	tbl := toTable(row)
	assert.Equal(t, row.AssetName, tbl.AssetName)
	assert.Equal(t, row.InputName, tbl.InputName)
	assert.Equal(t, row.InputType, tbl.InputType)
	assert.Equal(t, row.PartitionKey, tbl.PartitionKey)
	assert.Equal(t, row.WindowSize, tbl.WindowSize)
	assert.Equal(t, 7, *tbl.WindowSize)

	tbl.ID = 10
	tbl.CreatedAt = time.Now()
	tbl.UpdatedAt = time.Now()

	back := toRow(tbl)
	assert.Equal(t, int64(10), back.ID)
	assert.Equal(t, row.AssetName, back.AssetName)
	assert.Equal(t, row.InputName, back.InputName)
	assert.Equal(t, row.InputType, back.InputType)
	assert.Equal(t, row.PartitionKey, back.PartitionKey)
	assert.Equal(t, 7, *back.WindowSize)
	assert.Equal(t, tbl.CreatedAt, back.CreatedAt)
	assert.Equal(t, tbl.UpdatedAt, back.UpdatedAt)
}

func TestToTableNullWindowSize(t *testing.T) {
	row := AssetDependencyRow{
		AssetName:    "user_spend",
		InputName:    "fg.user_profile",
		InputType:    "internal",
		PartitionKey: "ds",
		WindowSize:   nil,
	}

	tbl := toTable(row)
	assert.Nil(t, tbl.WindowSize)

	back := toRow(tbl)
	assert.Nil(t, back.WindowSize)
}

func TestToRows(t *testing.T) {
	tables := []Table{
		{ID: 1, AssetName: "asset_a", InputName: "input_x"},
		{ID: 2, AssetName: "asset_a", InputName: "input_y"},
		{ID: 3, AssetName: "asset_b", InputName: "input_z"},
	}

	rows := toRows(tables)
	assert.Len(t, rows, 3)
	assert.Equal(t, "input_x", rows[0].InputName)
	assert.Equal(t, "input_y", rows[1].InputName)
	assert.Equal(t, "input_z", rows[2].InputName)
}

func TestTableName(t *testing.T) {
	tbl := Table{}
	assert.Equal(t, "asset_dependencies", tbl.TableName())
}
