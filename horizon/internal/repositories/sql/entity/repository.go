package entity

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

// RepositoryInterface defines the interface for table operations
type Repository interface {
	GetAll() ([]Table, error)
	Create(table *Table) (uint, error)
	Update(table *Table) error
	GetById(id int) (*Table, error)
	GetAllByUserId(userId string) ([]Table, error)
}

// Entity implements RepositoryInterface using a generic repository
type Entity struct {
	db     *gorm.DB
	dbName string
}

// NewRepository creates a new entity repository
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

	return &Entity{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

// GetAll retrieves all entity records from the database.
func (e *Entity) GetAll() ([]Table, error) {
	var entities []Table
	result := e.db.Find(&entities)
	return entities, result.Error
}

// GetById retrieves an entity by its ID.
func (e *Entity) GetById(id int) (*Table, error) {
	var entity Table
	result := e.db.Where("request_id = ?", id).First(&entity)
	return &entity, result.Error
}

// GetAllByUserId retrieves all entities created by a specific user
func (e *Entity) GetAllByUserId(userId string) ([]Table, error) {
	var entities []Table
	result := e.db.Where("created_by = ?", userId).Find(&entities)
	return entities, result.Error
}

// Create adds a new entity to the database.
func (e *Entity) Create(table *Table) (uint, error) {
	result := e.db.Create(table)
	if result.Error != nil {
		return 0, result.Error
	}
	return table.RequestId, nil
}

// Update updates an entity's information in the database.
func (e *Entity) Update(table *Table) error {
	result := e.db.Model(table).Where("request_id = ?", table.RequestId).Updates(table)
	return result.Error
}
