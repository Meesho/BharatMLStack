package features

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

// RepositoryInterface defines the interface for features table operations
type Repository interface {
	GetAll() ([]Table, error)
	Create(table *Table) (uint, error)
	Update(table *Table) error
	GetById(id int) (*Table, error)
	GetAllByUserId(userId string) ([]Table, error)
}

// Repository implements RepositoryInterface using a generic repository
type Features struct {
	db     *gorm.DB
	dbName string
}

// NewRepository creates a new features repository
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

	return &Features{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

// GetAll retrieves all features records from the database.
func (f *Features) GetAll() ([]Table, error) {
	var features []Table
	result := f.db.Find(&features)
	return features, result.Error
}

// GetById retrieves a feature by its ID.
func (f *Features) GetById(id int) (*Table, error) {
	var feature Table
	result := f.db.Where("request_id = ?", id).First(&feature)
	return &feature, result.Error
}

// GetAllByUserId retrieves all features created by a specific user
func (f *Features) GetAllByUserId(userId string) ([]Table, error) {
	var features []Table
	result := f.db.Where("created_by = ?", userId).Find(&features)
	return features, result.Error
}

// Create adds a new feature to the database.
func (f *Features) Create(table *Table) (uint, error) {
	result := f.db.Create(table)
	if result.Error != nil {
		return 0, result.Error
	}
	return table.RequestId, nil
}

// Update updates a feature's information in the database.
func (f *Features) Update(table *Table) error {
	result := f.db.Model(table).Where("request_id = ?", table.RequestId).Updates(table)
	return result.Error
}
