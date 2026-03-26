package filter_requests

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type FilterRequestRepository interface {
	Create(req *FilterRequest) error
	GetByID(requestID int) (*FilterRequest, error)
	GetByStatus(status string) ([]FilterRequest, error)
	UpdateStatus(requestID int, status string, approvedBy string) error
	GetAll() ([]FilterRequest, error)
}

type filterRequestRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (FilterRequestRepository, error) {
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

	return &filterRequestRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *filterRequestRepo) Create(req *FilterRequest) error {
	return r.db.Create(req).Error
}

func (r *filterRequestRepo) GetByID(requestID int) (*FilterRequest, error) {
	var request FilterRequest
	err := r.db.Where("request_id = ?", requestID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *filterRequestRepo) GetByStatus(status string) ([]FilterRequest, error) {
	var requests []FilterRequest
	err := r.db.Where("status = ?", status).Find(&requests).Error
	return requests, err
}

func (r *filterRequestRepo) UpdateStatus(requestID int, status string, approvedBy string) error {
	return r.db.Model(&FilterRequest{}).Where("request_id = ?", requestID).
		Updates(map[string]interface{}{
			"status":      status,
			"approved_by": approvedBy,
		}).Error
}

func (r *filterRequestRepo) GetAll() ([]FilterRequest, error) {
	var requests []FilterRequest
	err := r.db.Find(&requests).Error
	return requests, err
}
