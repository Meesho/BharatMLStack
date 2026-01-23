package model_requests

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type ModelRequestRepository interface {
	Create(req *ModelRequest) error
	GetByID(requestID int) (*ModelRequest, error)
	GetByStatus(status string) ([]ModelRequest, error)
	UpdateStatus(requestID int, status string, approvedBy string) error
	GetAll() ([]ModelRequest, error)
}

type modelRequestRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (ModelRequestRepository, error) {
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

	return &modelRequestRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *modelRequestRepo) Create(req *ModelRequest) error {
	return r.db.Create(req).Error
}

func (r *modelRequestRepo) GetByID(requestID int) (*ModelRequest, error) {
	var request ModelRequest
	err := r.db.Where("request_id = ?", requestID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *modelRequestRepo) GetByStatus(status string) ([]ModelRequest, error) {
	var requests []ModelRequest
	err := r.db.Where("status = ?", status).Find(&requests).Error
	return requests, err
}

func (r *modelRequestRepo) UpdateStatus(requestID int, status string, approvedBy string) error {
	return r.db.Model(&ModelRequest{}).Where("request_id = ?", requestID).
		Updates(map[string]interface{}{
			"status":      status,
			"approved_by": approvedBy,
		}).Error
}

func (r *modelRequestRepo) GetAll() ([]ModelRequest, error) {
	var requests []ModelRequest
	err := r.db.Find(&requests).Error
	return requests, err
}
