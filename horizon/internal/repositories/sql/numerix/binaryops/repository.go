package binaryops

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type Repository interface {
	GetAll() ([]Table, error)
}

type NumerixBinaryOps struct {
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

	return &NumerixBinaryOps{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (g *NumerixBinaryOps) GetAll() ([]Table, error) {
	var tables []Table
	result := g.db.Find(&tables)
	return tables, result.Error
}
