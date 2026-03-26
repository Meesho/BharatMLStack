package connectionconfig

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type ConnectionConfigRepository interface {
	GetAll() ([]ConnectionConfig, error)
	Create(req *ConnectionConfig) error
	Update(req *ConnectionConfig) error
}

type connectionConfigRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (ConnectionConfigRepository, error) {
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

	return &connectionConfigRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *connectionConfigRepo) GetAll() ([]ConnectionConfig, error) {
	var requests []ConnectionConfig
	err := r.db.Where("active = ?", true).Find(&requests).Error
	return requests, err
}

func (r *connectionConfigRepo) Create(req *ConnectionConfig) error {
	return r.db.Create(req).Error
}

func (r *connectionConfigRepo) Update(req *ConnectionConfig) error {
	return r.db.Model(req).Where("id = ?", req.Id).Updates(req).Error
}
