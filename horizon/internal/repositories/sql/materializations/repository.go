package materializations

import (
	"context"
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type MaterializationRow struct {
	ID               int64
	AssetName        string
	PartitionKey     string
	ComputeKey       string
	ArtifactPath     string
	InputFingerprint string
	Status           string
	CreatedAt        time.Time
}

type Repository interface {
	Create(ctx context.Context, row MaterializationRow) error
	GetByComputeKey(ctx context.Context, assetName, partitionKey, computeKey string) (*MaterializationRow, error)
	GetLatest(ctx context.Context, assetName, partitionKey string) (*MaterializationRow, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	GetByAsset(ctx context.Context, assetName string) ([]MaterializationRow, error)
	Exists(ctx context.Context, computeKey string) (bool, error)
}

type Materializations struct {
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

	return &Materializations{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func toTable(row MaterializationRow) Table {
	return Table{
		AssetName:        row.AssetName,
		PartitionKey:     row.PartitionKey,
		ComputeKey:       row.ComputeKey,
		ArtifactPath:     row.ArtifactPath,
		InputFingerprint: row.InputFingerprint,
		Status:           row.Status,
	}
}

func toRow(t Table) MaterializationRow {
	return MaterializationRow{
		ID:               int64(t.ID),
		AssetName:        t.AssetName,
		PartitionKey:     t.PartitionKey,
		ComputeKey:       t.ComputeKey,
		ArtifactPath:     t.ArtifactPath,
		InputFingerprint: t.InputFingerprint,
		Status:           t.Status,
		CreatedAt:        t.CreatedAt,
	}
}

func toRows(tables []Table) []MaterializationRow {
	rows := make([]MaterializationRow, len(tables))
	for i, t := range tables {
		rows[i] = toRow(t)
	}
	return rows
}

func (m *Materializations) Create(ctx context.Context, row MaterializationRow) error {
	record := toTable(row)
	result := m.db.WithContext(ctx).Create(&record)
	return result.Error
}

func (m *Materializations) GetByComputeKey(ctx context.Context, assetName, partitionKey, computeKey string) (*MaterializationRow, error) {
	var record Table
	result := m.db.WithContext(ctx).
		Where("asset_name = ? AND partition_key = ? AND compute_key = ?", assetName, partitionKey, computeKey).
		First(&record)
	if result.Error != nil {
		return nil, result.Error
	}
	row := toRow(record)
	return &row, nil
}

func (m *Materializations) GetLatest(ctx context.Context, assetName, partitionKey string) (*MaterializationRow, error) {
	var record Table
	result := m.db.WithContext(ctx).
		Where("asset_name = ? AND partition_key = ?", assetName, partitionKey).
		Order("created_at DESC").
		First(&record)
	if result.Error != nil {
		return nil, result.Error
	}
	row := toRow(record)
	return &row, nil
}

func (m *Materializations) UpdateStatus(ctx context.Context, id int64, status string) error {
	result := m.db.WithContext(ctx).Model(&Table{}).
		Where("id = ?", id).
		Update("status", status)
	return result.Error
}

func (m *Materializations) GetByAsset(ctx context.Context, assetName string) ([]MaterializationRow, error) {
	var records []Table
	result := m.db.WithContext(ctx).Where("asset_name = ?", assetName).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	return toRows(records), nil
}

func (m *Materializations) Exists(ctx context.Context, computeKey string) (bool, error) {
	var count int64
	result := m.db.WithContext(ctx).Model(&Table{}).
		Where("compute_key = ?", computeKey).
		Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}
