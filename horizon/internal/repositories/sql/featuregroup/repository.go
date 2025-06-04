package featuregroup

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

// Repository defines the interface for featuregroup table operations
type Repository interface {
	GetAll() ([]Table, error)
	Create(table *Table) (uint, error)
	Update(table *Table) error
	GetById(id int) (*Table, error)
	GetAllByUserId(userId string) ([]Table, error)
}

// Repository implements RepositoryInterface using a generic repository
type FeatureGroup struct {
	db     *gorm.DB
	dbName string
}

// NewRepository creates a new featuregroup repository
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

	return &FeatureGroup{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

// GetAll retrieves all featuregroup records from the database.
func (fg *FeatureGroup) GetAll() ([]Table, error) {
	var featuregroups []Table
	result := fg.db.Find(&featuregroups)
	return featuregroups, result.Error
}

// GetById retrieves a featuregroup by its ID.
func (fg *FeatureGroup) GetById(id int) (*Table, error) {
	var featuregroup Table
	result := fg.db.Where("request_id = ?", id).First(&featuregroup)
	return &featuregroup, result.Error
}

// GetAllByUserId retrieves all featuregroups created by a specific user
func (fg *FeatureGroup) GetAllByUserId(userId string) ([]Table, error) {
	var featuregroups []Table
	result := fg.db.Where("created_by = ?", userId).Find(&featuregroups)
	return featuregroups, result.Error
}

// Create adds a new featuregroup to the database.
func (fg *FeatureGroup) Create(table *Table) (uint, error) {
	result := fg.db.Create(table)
	if result.Error != nil {
		return 0, result.Error
	}
	return table.RequestId, nil
}

// Update updates a featuregroup's information in the database.
func (fg *FeatureGroup) Update(table *Table) error {
	result := fg.db.Model(table).Where("request_id = ?", table.RequestId).Updates(table)
	return result.Error
}
