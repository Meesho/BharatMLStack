package request

import (
	"errors"
	"fmt"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type Repository interface {
	Transaction(fn func(tx *gorm.DB) error) error
	Create(table *Table) error
	Update(table *Table) error
	UpdateTx(tx *gorm.DB, table *Table) error
	GetAll() ([]Table, error)
	GetByUser(email string) ([]Table, error)
	DoesConfigIDExist(configID string) (bool, error)
	DoesRequestIDExist(requestID uint) (bool, error)
	DoesRequestIDExistWithStatus(requestID uint, status string) (bool, error)
	DoesConfigIdExistWithRequestType(configID string, requestType string) (bool, error)
	CurrentRequestStatus(requestID uint) (string, error)
	Deactivate(configID string) error
}

type InferflowRequest struct {
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

	return &InferflowRequest{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil

}

func (g *InferflowRequest) Transaction(fn func(tx *gorm.DB) error) error {
	return g.db.Transaction(fn)
}

func (g *InferflowRequest) Create(table *Table) error {
	result := g.db.Create(table)
	return result.Error
}

func (g *InferflowRequest) Update(table *Table) error {
	result := g.db.Model(table).Where("request_id = ?", table.RequestID).Updates(table)
	return result.Error
}

func (g *InferflowRequest) UpdateTx(tx *gorm.DB, table *Table) error {
	result := tx.Model(table).Where("request_id = ?", table.RequestID).Updates(table)
	if result.Error != nil {
		return result.Error
	}

	readResult := tx.First(table, table.RequestID)
	if readResult.Error != nil {
		return fmt.Errorf("failed to read back record after update: %w", readResult.Error)
	}
	return nil
}

func (g *InferflowRequest) GetAll() ([]Table, error) {
	var tables []Table
	result := g.db.Find(&tables)
	return tables, result.Error
}

func (g *InferflowRequest) GetByUser(email string) ([]Table, error) {
	var tables []Table
	result := g.db.Where("created_by = ?", email).Find(&tables)
	return tables, result.Error
}

func (g *InferflowRequest) DoesRecordExist(query string, args ...interface{}) (bool, error) {
	var table Table
	result := g.db.Where(query, args...).First(&table)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, result.Error
	}
	return true, nil
}

func (g *InferflowRequest) DoesConfigIDExist(configID string) (bool, error) {
	return g.DoesRecordExist("config_id = ?", configID)
}

func (g *InferflowRequest) DoesRequestIDExist(requestID uint) (bool, error) {
	return g.DoesRecordExist("request_id = ?", requestID)
}

func (g *InferflowRequest) DoesConfigIdExistWithRequestType(configID string, requestType string) (bool, error) {
	return g.DoesRecordExist("config_id = ? AND request_type = ? AND STATUS != 'REJECTED'", configID, requestType)
}

func (g *InferflowRequest) CurrentRequestStatus(requestID uint) (string, error) {
	var table Table
	result := g.db.Where("request_id = ?", requestID).First(&table)
	return table.Status, result.Error
}

func (g *InferflowRequest) DoesRequestIDExistWithStatus(requestID uint, status string) (bool, error) {
	return g.DoesRecordExist("request_id = ? AND status = ?", requestID, status)
}

func (g *InferflowRequest) Deactivate(configID string) error {
	return g.db.Model(&Table{}).Where("config_id = ?", configID).Update("active", false).Error
}
