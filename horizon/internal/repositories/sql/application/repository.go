package application

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type ApplicationRepository interface {
	GetAll() ([]Application, error)
	Create(req *Application) error
	Update(req *Application) error
}

type applicationRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (ApplicationRepository, error) {
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

	return &applicationRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *applicationRepo) GetAll() ([]Application, error) {
	var requests []Application
	err := r.db.Where("active = ?", true).Find(&requests).Error
	return requests, err
}

func (r *applicationRepo) Create(req *Application) error {
	return r.db.Create(req).Error
}

func (r *applicationRepo) Update(req *Application) error {
	return r.db.Model(req).Where("app_token = ?", req.AppToken).Updates(req).Error
}
