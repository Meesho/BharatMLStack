package datasetpartitions

import (
	"context"
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DatasetPartitionRow struct {
	ID           int64
	DatasetName  string
	PartitionKey string
	DeltaVersion *int64
	CommitTS     *time.Time
	DQStatus     string
	IsReady      bool
	UpdatedAt    time.Time
}

type DatasetPartitionInput struct {
	DatasetName  string
	PartitionKey string
}

type Repository interface {
	Upsert(ctx context.Context, row DatasetPartitionRow) error
	Get(ctx context.Context, datasetName, partitionKey string) (*DatasetPartitionRow, error)
	GetPartitions(ctx context.Context, datasetName string) ([]DatasetPartitionRow, error)
	IsReady(ctx context.Context, datasetName, partitionKey string) (bool, error)
	AreAllReady(ctx context.Context, inputs []DatasetPartitionInput) (bool, error)
	MarkReady(ctx context.Context, datasetName, partitionKey string, deltaVersion int64) error
}

type DatasetPartitions struct {
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

	return &DatasetPartitions{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func toTable(row DatasetPartitionRow) Table {
	return Table{
		DatasetName:  row.DatasetName,
		PartitionKey: row.PartitionKey,
		DeltaVersion: row.DeltaVersion,
		CommitTS:     row.CommitTS,
		DQStatus:     row.DQStatus,
		IsReady:      row.IsReady,
	}
}

func toRow(t Table) DatasetPartitionRow {
	return DatasetPartitionRow{
		ID:           int64(t.ID),
		DatasetName:  t.DatasetName,
		PartitionKey: t.PartitionKey,
		DeltaVersion: t.DeltaVersion,
		CommitTS:     t.CommitTS,
		DQStatus:     t.DQStatus,
		IsReady:      t.IsReady,
		UpdatedAt:    t.UpdatedAt,
	}
}

func toRows(tables []Table) []DatasetPartitionRow {
	rows := make([]DatasetPartitionRow, len(tables))
	for i, t := range tables {
		rows[i] = toRow(t)
	}
	return rows
}

func (dp *DatasetPartitions) Upsert(ctx context.Context, row DatasetPartitionRow) error {
	record := toTable(row)
	result := dp.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "dataset_name"}, {Name: "partition_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"delta_version", "commit_ts", "dq_status", "is_ready",
		}),
	}).Create(&record)
	return result.Error
}

func (dp *DatasetPartitions) Get(ctx context.Context, datasetName, partitionKey string) (*DatasetPartitionRow, error) {
	var record Table
	result := dp.db.WithContext(ctx).
		Where("dataset_name = ? AND partition_key = ?", datasetName, partitionKey).
		First(&record)
	if result.Error != nil {
		return nil, result.Error
	}
	row := toRow(record)
	return &row, nil
}

func (dp *DatasetPartitions) GetPartitions(ctx context.Context, datasetName string) ([]DatasetPartitionRow, error) {
	var records []Table
	result := dp.db.WithContext(ctx).Where("dataset_name = ?", datasetName).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	return toRows(records), nil
}

func (dp *DatasetPartitions) IsReady(ctx context.Context, datasetName, partitionKey string) (bool, error) {
	var record Table
	result := dp.db.WithContext(ctx).
		Where("dataset_name = ? AND partition_key = ? AND is_ready = ?", datasetName, partitionKey, true).
		First(&record)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

func (dp *DatasetPartitions) AreAllReady(ctx context.Context, inputs []DatasetPartitionInput) (bool, error) {
	if len(inputs) == 0 {
		return true, nil
	}
	for _, input := range inputs {
		ready, err := dp.IsReady(ctx, input.DatasetName, input.PartitionKey)
		if err != nil {
			return false, err
		}
		if !ready {
			return false, nil
		}
	}
	return true, nil
}

func (dp *DatasetPartitions) MarkReady(ctx context.Context, datasetName, partitionKey string, deltaVersion int64) error {
	now := time.Now()
	record := Table{
		DatasetName:  datasetName,
		PartitionKey: partitionKey,
		DeltaVersion: &deltaVersion,
		CommitTS:     &now,
		DQStatus:     "passed",
		IsReady:      true,
	}
	result := dp.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "dataset_name"}, {Name: "partition_key"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"delta_version", "commit_ts", "dq_status", "is_ready",
		}),
	}).Create(&record)
	return result.Error
}
