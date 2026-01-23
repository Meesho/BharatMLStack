package variant_requests

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type VariantRequestRepository interface {
	Create(req *VariantRequest) error
	GetByID(requestID int) (*VariantRequest, error)
	GetByStatus(status string) ([]VariantRequest, error)
	UpdateStatus(requestID int, status string, approvedBy string) error
	GetAll() ([]VariantRequest, error)
}

type variantRequestRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (VariantRequestRepository, error) {
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

	return &variantRequestRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *variantRequestRepo) Create(req *VariantRequest) error {
	return r.db.Create(req).Error
}

func (r *variantRequestRepo) GetByID(requestID int) (*VariantRequest, error) {
	var request VariantRequest
	err := r.db.Where("request_id = ?", requestID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *variantRequestRepo) GetByStatus(status string) ([]VariantRequest, error) {
	var requests []VariantRequest
	err := r.db.Where("status = ?", status).Find(&requests).Error
	return requests, err
}

func (r *variantRequestRepo) UpdateStatus(requestID int, status string, approvedBy string) error {
	return r.db.Model(&VariantRequest{}).Where("request_id = ?", requestID).
		Updates(map[string]interface{}{
			"status":      status,
			"approved_by": approvedBy,
		}).Error
}

func (r *variantRequestRepo) GetAll() ([]VariantRequest, error) {
	var requests []VariantRequest
	err := r.db.Find(&requests).Error
	return requests, err
}
