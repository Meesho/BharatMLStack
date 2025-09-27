package numerix_request

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

const (
	onboardRequestType = "ONBOARD"
)

type Repository interface {
	Transaction(fn func(tx *gorm.DB) error) error
	Create(table *Table) error
	Update(table *Table) error
	UpdateTx(tx *gorm.DB, table *Table) (Table, error)
	GetAll() ([]Table, error)
	GetByUser(email string) ([]Table, error)
	DoesConfigIDExist(configID uint) (bool, error)
	DoesRequestIDExistWithStatus(requestID uint, status string) (bool, error)
	DoesConfigIdExistWithRequestType(configID uint, requestType string) (bool, error)
}

type NumerixRequest struct {
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

	return &NumerixRequest{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil

}

func (g *NumerixRequest) Transaction(fn func(tx *gorm.DB) error) error {
	return g.db.Transaction(fn)
}

func (g *NumerixRequest) Create(table *Table) error {
	newEntry := g.db.Create(table)
	if newEntry.Error != nil {
		return newEntry.Error
	}

	if table.RequestType == onboardRequestType {
		err := g.db.Model(table).Update("config_id", table.RequestID).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *NumerixRequest) Update(table *Table) error {
	result := g.db.Model(table).Where("request_id = ?", table.RequestID).Updates(table)
	return result.Error
}

func (g *NumerixRequest) UpdateTx(tx *gorm.DB, table *Table) (Table, error) {
	result := tx.Model(table).Where("request_id = ?", table.RequestID).Updates(table)
	if result.Error != nil {
		return Table{}, result.Error
	}

	var updatedTable Table
	if err := tx.Where("request_id = ?", table.RequestID).First(&updatedTable).Error; err != nil {
		return Table{}, err
	}
	return updatedTable, nil
}

func (g *NumerixRequest) GetAll() ([]Table, error) {
	var tables []Table
	result := g.db.Find(&tables)
	return tables, result.Error
}

func (g *NumerixRequest) GetByUser(email string) ([]Table, error) {
	var tables []Table
	result := g.db.Where("created_by = ?", email).Find(&tables)
	return tables, result.Error
}

func (g *NumerixRequest) DoesRecordExist(query string, args ...interface{}) (bool, error) {
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

func (g *NumerixRequest) DoesConfigIDExist(configID uint) (bool, error) {
	return g.DoesRecordExist("config_id = ?", configID)
}

func (g *NumerixRequest) DoesRequestIDExistWithStatus(requestID uint, status string) (bool, error) {
	return g.DoesRecordExist("request_id = ? AND status = ?", requestID, status)
}

func (g *NumerixRequest) DoesConfigIdExistWithRequestType(configID uint, requestType string) (bool, error) {
	return g.DoesRecordExist("config_id = ? AND request_type = ?", configID, requestType)
}
