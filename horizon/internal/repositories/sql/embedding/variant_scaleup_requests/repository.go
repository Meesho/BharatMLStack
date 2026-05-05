package variant_scaleup_requests

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type VariantScaleUpRequestRepository interface {
	Create(req *VariantScaleUpRequest) error
	GetByID(requestID int) (*VariantScaleUpRequest, error)
	GetByStatus(status string) ([]VariantScaleUpRequest, error)
	UpdateStatus(requestID int, status string, approvedBy string) error
	GetAll() ([]VariantScaleUpRequest, error)
}

type variantScaleUpRequestRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (VariantScaleUpRequestRepository, error) {
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

	return &variantScaleUpRequestRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *variantScaleUpRequestRepo) Create(req *VariantScaleUpRequest) error {
	return r.db.Create(req).Error
}

func (r *variantScaleUpRequestRepo) GetByID(requestID int) (*VariantScaleUpRequest, error) {
	var request VariantScaleUpRequest
	err := r.db.Where("request_id = ?", requestID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *variantScaleUpRequestRepo) GetByStatus(status string) ([]VariantScaleUpRequest, error) {
	var requests []VariantScaleUpRequest
	err := r.db.Where("status = ?", status).Find(&requests).Error
	return requests, err
}

func (r *variantScaleUpRequestRepo) UpdateStatus(requestID int, status string, approvedBy string) error {
	return r.db.Model(&VariantScaleUpRequest{}).Where("request_id = ?", requestID).
		Updates(map[string]interface{}{
			"status":      status,
			"approved_by": approvedBy,
		}).Error
}

func (r *variantScaleUpRequestRepo) GetAll() ([]VariantScaleUpRequest, error) {
	var requests []VariantScaleUpRequest
	err := r.db.Find(&requests).Error
	return requests, err
}
