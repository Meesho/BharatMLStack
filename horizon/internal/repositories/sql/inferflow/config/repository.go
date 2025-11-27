package config

import (
	"errors"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type Repository interface {
	CreateTx(tx *gorm.DB, table *Table) error
	UpdateTx(tx *gorm.DB, table *Table) error
	DeleteTx(tx *gorm.DB, configID string) error
	GetAll() ([]Table, error)
	GetByID(configID string) (table *Table, err error)
	DoesConfigIDExist(configID string) (bool, error)
	Update(table *Table) error
	FindByDiscoveryIDsAndCreatedBefore(discoveryIDs []int, daysAgo int) ([]Table, error)
	Deactivate(configID string) error
}

type InferflowConfig struct {
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

	return &InferflowConfig{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil

}

func (g *InferflowConfig) CreateTx(tx *gorm.DB, table *Table) error {
	result := tx.Create(table)
	return result.Error
}

func (g *InferflowConfig) UpdateTx(tx *gorm.DB, table *Table) error {
	if err := tx.Model(table).Where("config_id = ?", table.ConfigID).Updates(table).Error; err != nil {
		return err
	}

	if !table.Active {
		if err := tx.Model(table).Where("config_id = ?", table.ConfigID).Update("active", table.Active).Error; err != nil {
			return err
		}
	}

	return nil
}

func (g *InferflowConfig) DeleteTx(tx *gorm.DB, configID string) error {
	result := tx.Where("config_id = ?", configID).Delete(&Table{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("no config found with the given config_id")
	}
	return nil
}

func (g *InferflowConfig) GetAll() ([]Table, error) {
	var configs []Table
	result := g.db.Where("active = ?", true).Find(&configs)
	return configs, result.Error
}

func (g *InferflowConfig) GetByID(configID string) (table *Table, err error) {
	result := g.db.Where("config_id = ? and active = ?", configID, true).First(&table)
	return table, result.Error
}

func (g *InferflowConfig) DoesConfigIDExist(configID string) (bool, error) {
	var table Table
	result := g.db.Where("config_id = ? and active = ?", configID, true).First(&table)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

func (g *InferflowConfig) Update(table *Table) error {
	result := g.db.Model(table).Where("config_id = ?", table.ConfigID).Updates(table)
	return result.Error
}

func (r *InferflowConfig) FindByDiscoveryIDsAndCreatedBefore(discoveryIDs []int, daysAgo int) ([]Table, error) {
	var configs []Table

	cutoffDate := time.Now().AddDate(0, 0, -daysAgo)

	err := r.db.Where("discovery_id IN ? AND updated_at < ?", discoveryIDs, cutoffDate).
		Find(&configs).Error

	return configs, err
}

func (g *InferflowConfig) Deactivate(configID string) error {
	return g.db.Model(&Table{}).Where("config_id = ?", configID).Update("active", false).Error
}
