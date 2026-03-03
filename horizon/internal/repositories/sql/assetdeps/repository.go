package assetdeps

import (
	"context"
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type AssetDependencyRow struct {
	ID           int64
	AssetName    string
	InputName    string
	InputType    string
	PartitionKey string
	WindowSize   *int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository interface {
	ReplaceForAsset(ctx context.Context, assetName string, deps []AssetDependencyRow) error
	GetDependencies(ctx context.Context, assetName string) ([]AssetDependencyRow, error)
	GetDependents(ctx context.Context, inputName string) ([]AssetDependencyRow, error)
	GetAllEdges(ctx context.Context) ([]AssetDependencyRow, error)
	DeleteForAsset(ctx context.Context, assetName string) error
}

type AssetDeps struct {
	db     *gorm.DB
	dbName string
}

func NewRepository(connection *infra.SQLConnection) (Repository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}
	meta, err := connection.GetMeta()
	if err != nil {
		return nil, err
	}
	dbName := meta["db_name"].(string)

	return &AssetDeps{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func toTable(row AssetDependencyRow) Table {
	return Table{
		AssetName:    row.AssetName,
		InputName:    row.InputName,
		InputType:    row.InputType,
		PartitionKey: row.PartitionKey,
		WindowSize:   row.WindowSize,
	}
}

func toRow(t Table) AssetDependencyRow {
	return AssetDependencyRow{
		ID:           int64(t.ID),
		AssetName:    t.AssetName,
		InputName:    t.InputName,
		InputType:    t.InputType,
		PartitionKey: t.PartitionKey,
		WindowSize:   t.WindowSize,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}

func toRows(tables []Table) []AssetDependencyRow {
	rows := make([]AssetDependencyRow, len(tables))
	for i, t := range tables {
		rows[i] = toRow(t)
	}
	return rows
}

// ReplaceForAsset transactionally deletes all existing dependencies for the asset
// and inserts the new set of dependencies.
func (ad *AssetDeps) ReplaceForAsset(ctx context.Context, assetName string, deps []AssetDependencyRow) error {
	return ad.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("asset_name = ?", assetName).Delete(&Table{}).Error; err != nil {
			return err
		}
		if len(deps) == 0 {
			return nil
		}
		records := make([]Table, len(deps))
		for i, d := range deps {
			records[i] = toTable(d)
			records[i].AssetName = assetName
		}
		return tx.Create(&records).Error
	})
}

func (ad *AssetDeps) GetDependencies(ctx context.Context, assetName string) ([]AssetDependencyRow, error) {
	var records []Table
	result := ad.db.WithContext(ctx).Where("asset_name = ?", assetName).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	return toRows(records), nil
}

func (ad *AssetDeps) GetDependents(ctx context.Context, inputName string) ([]AssetDependencyRow, error) {
	var records []Table
	result := ad.db.WithContext(ctx).Where("input_name = ?", inputName).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	return toRows(records), nil
}

func (ad *AssetDeps) GetAllEdges(ctx context.Context) ([]AssetDependencyRow, error) {
	var records []Table
	result := ad.db.WithContext(ctx).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	return toRows(records), nil
}

func (ad *AssetDeps) DeleteForAsset(ctx context.Context, assetName string) error {
	result := ad.db.WithContext(ctx).Where("asset_name = ?", assetName).Delete(&Table{})
	return result.Error
}
