package entity_requests

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type EntityRequestRepository interface {
	Create(req *EntityRequest) error
	GetByID(requestID int) (*EntityRequest, error)
	GetByStatus(status string) ([]EntityRequest, error)
	UpdateStatus(requestID int, status string, approvedBy string) error
	GetAll() ([]EntityRequest, error)
}

type entityRequestRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (EntityRequestRepository, error) {
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

	return &entityRequestRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *entityRequestRepo) Create(req *EntityRequest) error {
	return r.db.Create(req).Error
}

func (r *entityRequestRepo) GetByID(requestID int) (*EntityRequest, error) {
	var request EntityRequest
	err := r.db.Where("request_id = ?", requestID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *entityRequestRepo) GetByStatus(status string) ([]EntityRequest, error) {
	var requests []EntityRequest
	err := r.db.Where("status = ?", status).Find(&requests).Error
	return requests, err
}

func (r *entityRequestRepo) UpdateStatus(requestID int, status string, approvedBy string) error {
	return r.db.Model(&EntityRequest{}).Where("request_id = ?", requestID).
		Updates(map[string]interface{}{
			"status":      status,
			"approved_by": approvedBy,
		}).Error
}

func (r *entityRequestRepo) GetAll() ([]EntityRequest, error) {
	var requests []EntityRequest
	err := r.db.Find(&requests).Error
	return requests, err
}
