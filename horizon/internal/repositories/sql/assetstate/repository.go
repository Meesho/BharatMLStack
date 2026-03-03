package assetstate

import (
	"context"
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AssetNecessityRow struct {
	AssetName       string
	Necessity       string
	ServingOverride *bool
	OverrideReason  *string
	OverrideBy      *string
	ResolvedAt      time.Time
	Reason          string
}

type Repository interface {
	BulkUpsert(ctx context.Context, rows []AssetNecessityRow) error
	Get(ctx context.Context, assetName string) (*AssetNecessityRow, error)
	GetAll(ctx context.Context) ([]AssetNecessityRow, error)
	GetByNecessity(ctx context.Context, necessity string) ([]AssetNecessityRow, error)
	SetServingOverride(ctx context.Context, assetName string, serving bool, reason string, by string) error
	ClearServingOverride(ctx context.Context, assetName string) error
}

type AssetState struct {
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

	return &AssetState{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func toTable(row AssetNecessityRow) Table {
	return Table{
		AssetName:       row.AssetName,
		Necessity:       row.Necessity,
		ServingOverride: row.ServingOverride,
		OverrideReason:  row.OverrideReason,
		OverrideBy:      row.OverrideBy,
		Reason:          row.Reason,
	}
}

func toRow(t Table) AssetNecessityRow {
	return AssetNecessityRow{
		AssetName:       t.AssetName,
		Necessity:       t.Necessity,
		ServingOverride: t.ServingOverride,
		OverrideReason:  t.OverrideReason,
		OverrideBy:      t.OverrideBy,
		ResolvedAt:      t.ResolvedAt,
		Reason:          t.Reason,
	}
}

func toRows(tables []Table) []AssetNecessityRow {
	rows := make([]AssetNecessityRow, len(tables))
	for i, t := range tables {
		rows[i] = toRow(t)
	}
	return rows
}

func (as *AssetState) BulkUpsert(ctx context.Context, rows []AssetNecessityRow) error {
	if len(rows) == 0 {
		return nil
	}
	records := make([]Table, len(rows))
	for i, r := range rows {
		records[i] = toTable(r)
	}
	result := as.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "asset_name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"necessity", "reason",
		}),
	}).Create(&records)
	return result.Error
}

func (as *AssetState) Get(ctx context.Context, assetName string) (*AssetNecessityRow, error) {
	var record Table
	result := as.db.WithContext(ctx).Where("asset_name = ?", assetName).First(&record)
	if result.Error != nil {
		return nil, result.Error
	}
	row := toRow(record)
	return &row, nil
}

func (as *AssetState) GetAll(ctx context.Context) ([]AssetNecessityRow, error) {
	var records []Table
	result := as.db.WithContext(ctx).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	return toRows(records), nil
}

func (as *AssetState) GetByNecessity(ctx context.Context, necessity string) ([]AssetNecessityRow, error) {
	var records []Table
	result := as.db.WithContext(ctx).Where("necessity = ?", necessity).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	return toRows(records), nil
}

func (as *AssetState) SetServingOverride(ctx context.Context, assetName string, serving bool, reason string, by string) error {
	result := as.db.WithContext(ctx).Model(&Table{}).
		Where("asset_name = ?", assetName).
		Updates(map[string]interface{}{
			"serving_override": serving,
			"override_reason":  reason,
			"override_by":      by,
		})
	return result.Error
}

func (as *AssetState) ClearServingOverride(ctx context.Context, assetName string) error {
	result := as.db.WithContext(ctx).Model(&Table{}).
		Where("asset_name = ?", assetName).
		Updates(map[string]interface{}{
			"serving_override": nil,
			"override_reason":  nil,
			"override_by":      nil,
		})
	return result.Error
}
