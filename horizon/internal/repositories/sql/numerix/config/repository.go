package numerix_config

import (
	"errors"
	"strconv"
	"time"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type Repository interface {
	GetAll() ([]Table, error)
	GetAllPaginated(page int, pageSize int, configID int) ([]Table, int64, error)
	Create(table *Table) error
	CreateTx(tx *gorm.DB, table *Table) error
	UpdateTx(tx *gorm.DB, table *Table) error
	DoesConfigIDExist(configID uint) (bool, error)
	GetExpression(configID uint) (string, error)
	GetByConfigID(configID string) (Table, error)
	Update(table *Table) error
	FindByCreatedBefore(daysAgo int) ([]Table, error)
	Deactivate(configID string) error
}

type NumerixConfig struct {
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

	return &NumerixConfig{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil

}

func (g *NumerixConfig) GetAll() ([]Table, error) {
	var configs []Table
	result := g.db.Where("active = ?", true).Find(&configs)
	return configs, result.Error
}

func (g *NumerixConfig) GetAllPaginated(page int, pageSize int, configID int) ([]Table, int64, error) {
	var configs []Table
	var totalCount int64

	// Build base query
	query := g.db.Model(&Table{}).Where("active = ?", true)

	// Add search filter if provided
	if configID != 0 {
		query = query.Where("CAST(config_id AS CHAR) LIKE ?", strconv.Itoa(configID)+"%")
	}

	// Get total count
	countResult := query.Count(&totalCount)
	if countResult.Error != nil {
		return nil, 0, countResult.Error
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated results
	result := query.
		Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&configs)

	return configs, totalCount, result.Error
}

func (g *NumerixConfig) Create(table *Table) error {
	result := g.db.Create(table)
	return result.Error
}

func (g *NumerixConfig) CreateTx(tx *gorm.DB, table *Table) error {
	result := tx.Create(table)
	return result.Error
}

func (g *NumerixConfig) DoesConfigIDExist(configID uint) (bool, error) {
	var table Table
	result := g.db.Where("config_id = ?", configID).First(&table)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

func (g *NumerixConfig) GetExpression(configID uint) (string, error) {
	var config Table
	result := g.db.Where("config_id = ?", configID).First(&config)
	return config.ConfigValue.PostfixExpression, result.Error
}

func (g *NumerixConfig) GetByConfigID(configID string) (Table, error) {
	var config Table
	result := g.db.Where("config_id = ?", configID).First(&config)
	return config, result.Error
}

func (g *NumerixConfig) Update(table *Table) error {
	result := g.db.Model(table).Where("config_id = ?", table.ConfigID).Updates(table)
	return result.Error
}

func (g *NumerixConfig) UpdateTx(tx *gorm.DB, table *Table) error {
	result := tx.Model(table).Where("config_id = ?", table.ConfigID).Updates(table)
	return result.Error
}

func (g *NumerixConfig) FindByCreatedBefore(daysAgo int) ([]Table, error) {
	var configs []Table

	cutoffDate := time.Now().AddDate(0, 0, -daysAgo)

	err := g.db.Where("updated_at < ?", cutoffDate).
		Find(&configs).Error

	return configs, err
}

func (g *NumerixConfig) Deactivate(configID string) error {
	return g.db.Model(&Table{}).Where("config_id = ?", configID).Update("active", false).Error
}
