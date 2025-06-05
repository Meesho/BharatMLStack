package store

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type Repository interface {
	GetAll() ([]Table, error)
	Create(table *Table) (uint, error)
	Update(table *Table) error
	GetById(id int) (*Table, error)
	GetAllByUserId(userId string) ([]Table, error)
}

type Store struct {
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

	return &Store{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

// GetAll retrieves all store records from the database.
func (s *Store) GetAll() ([]Table, error) {
	var stores []Table
	result := s.db.Find(&stores)
	return stores, result.Error
}

// GetById retrieves a store by its ID.
func (s *Store) GetById(id int) (*Table, error) {
	var store Table
	result := s.db.Where("request_id = ?", id).First(&store)
	return &store, result.Error
}

// GetAllByUserId retrieves all stores created by a specific user
func (s *Store) GetAllByUserId(userId string) ([]Table, error) {
	var stores []Table
	result := s.db.Where("created_by = ?", userId).Find(&stores)
	return stores, result.Error
}

// Create adds a new store to the database.
func (s *Store) Create(table *Table) (uint, error) {
	result := s.db.Create(table)
	if result.Error != nil {
		return 0, result.Error
	}
	return table.RequestId, nil
}

// Update updates a store's information in the database.
func (s *Store) Update(table *Table) error {
	result := s.db.Model(table).Where("request_id = ?", table.RequestId).Updates(table)
	return result.Error
}
