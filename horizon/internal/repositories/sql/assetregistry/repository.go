package assetregistry

import (
	"context"
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AssetSpecRow struct {
	ID           int64
	AssetName    string
	AssetVersion string
	EntityName   string
	EntityKey    string
	Notebook     string
	PartitionKey string
	TriggerType  string
	Schedule     *string
	Serving      bool
	Incremental  bool
	Freshness    *string
	ChecksJSON   *string
	SpecJSON     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository interface {
	Upsert(ctx context.Context, row AssetSpecRow) error
	BulkUpsert(ctx context.Context, rows []AssetSpecRow) error
	Get(ctx context.Context, assetName string) (*AssetSpecRow, error)
	GetByNotebook(ctx context.Context, notebook string) ([]AssetSpecRow, error)
	GetByEntity(ctx context.Context, entityName string) ([]AssetSpecRow, error)
	GetAll(ctx context.Context) ([]AssetSpecRow, error)
	Delete(ctx context.Context, assetName string) error
	GetByTriggerType(ctx context.Context, triggerType string) ([]AssetSpecRow, error)
}

type AssetRegistry struct {
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

	return &AssetRegistry{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func toTable(row AssetSpecRow) Table {
	return Table{
		AssetName:    row.AssetName,
		AssetVersion: row.AssetVersion,
		EntityName:   row.EntityName,
		EntityKey:    row.EntityKey,
		Notebook:     row.Notebook,
		PartitionKey: row.PartitionKey,
		TriggerType:  row.TriggerType,
		Schedule:     row.Schedule,
		Serving:      row.Serving,
		Incremental:  row.Incremental,
		Freshness:    row.Freshness,
		ChecksJSON:   row.ChecksJSON,
		SpecJSON:     row.SpecJSON,
	}
}

func toRow(t Table) AssetSpecRow {
	return AssetSpecRow{
		ID:           int64(t.ID),
		AssetName:    t.AssetName,
		AssetVersion: t.AssetVersion,
		EntityName:   t.EntityName,
		EntityKey:    t.EntityKey,
		Notebook:     t.Notebook,
		PartitionKey: t.PartitionKey,
		TriggerType:  t.TriggerType,
		Schedule:     t.Schedule,
		Serving:      t.Serving,
		Incremental:  t.Incremental,
		Freshness:    t.Freshness,
		ChecksJSON:   t.ChecksJSON,
		SpecJSON:     t.SpecJSON,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
}

func toRows(tables []Table) []AssetSpecRow {
	rows := make([]AssetSpecRow, len(tables))
	for i, t := range tables {
		rows[i] = toRow(t)
	}
	return rows
}

func (ar *AssetRegistry) Upsert(ctx context.Context, row AssetSpecRow) error {
	record := toTable(row)
	result := ar.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "asset_name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"asset_version", "entity_name", "entity_key", "notebook",
			"partition_key", "trigger_type", "schedule", "serving",
			"incremental", "freshness", "checks_json", "spec_json",
		}),
	}).Create(&record)
	return result.Error
}

func (ar *AssetRegistry) BulkUpsert(ctx context.Context, rows []AssetSpecRow) error {
	if len(rows) == 0 {
		return nil
	}
	records := make([]Table, len(rows))
	for i, r := range rows {
		records[i] = toTable(r)
	}
	result := ar.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "asset_name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"asset_version", "entity_name", "entity_key", "notebook",
			"partition_key", "trigger_type", "schedule", "serving",
			"incremental", "freshness", "checks_json", "spec_json",
		}),
	}).Create(&records)
	return result.Error
}

func (ar *AssetRegistry) Get(ctx context.Context, assetName string) (*AssetSpecRow, error) {
	var record Table
	result := ar.db.WithContext(ctx).Where("asset_name = ?", assetName).First(&record)
	if result.Error != nil {
		return nil, result.Error
	}
	row := toRow(record)
	return &row, nil
}

func (ar *AssetRegistry) GetByNotebook(ctx context.Context, notebook string) ([]AssetSpecRow, error) {
	var records []Table
	result := ar.db.WithContext(ctx).Where("notebook = ?", notebook).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	return toRows(records), nil
}

func (ar *AssetRegistry) GetByEntity(ctx context.Context, entityName string) ([]AssetSpecRow, error) {
	var records []Table
	result := ar.db.WithContext(ctx).Where("entity_name = ?", entityName).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	return toRows(records), nil
}

func (ar *AssetRegistry) GetAll(ctx context.Context) ([]AssetSpecRow, error) {
	var records []Table
	result := ar.db.WithContext(ctx).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	return toRows(records), nil
}

func (ar *AssetRegistry) Delete(ctx context.Context, assetName string) error {
	result := ar.db.WithContext(ctx).Where("asset_name = ?", assetName).Delete(&Table{})
	return result.Error
}

func (ar *AssetRegistry) GetByTriggerType(ctx context.Context, triggerType string) ([]AssetSpecRow, error) {
	var records []Table
	result := ar.db.WithContext(ctx).Where("trigger_type = ?", triggerType).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	return toRows(records), nil
}
