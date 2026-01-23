package variant_onboarding_requests

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type VariantOnboardingRequestRepository interface {
	Create(req *VariantOnboardingRequest) error
	GetByID(requestID int) (*VariantOnboardingRequest, error)
	GetByStatus(status string) ([]VariantOnboardingRequest, error)
	UpdateStatus(requestID int, status string, approvedBy string) error
	GetAll() ([]VariantOnboardingRequest, error)
}

type variantOnboardingRequestRepo struct {
	db     *gorm.DB
	dbName string
}

func Repository(connection *infra.SQLConnection) (VariantOnboardingRequestRepository, error) {
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

	return &variantOnboardingRequestRepo{
		db:     session.(*gorm.DB),
		dbName: dbName,
	}, nil
}

func (r *variantOnboardingRequestRepo) Create(req *VariantOnboardingRequest) error {
	return r.db.Create(req).Error
}

func (r *variantOnboardingRequestRepo) GetByID(requestID int) (*VariantOnboardingRequest, error) {
	var request VariantOnboardingRequest
	err := r.db.Where("request_id = ?", requestID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (r *variantOnboardingRequestRepo) GetByStatus(status string) ([]VariantOnboardingRequest, error) {
	var requests []VariantOnboardingRequest
	err := r.db.Where("status = ?", status).Find(&requests).Error
	return requests, err
}

func (r *variantOnboardingRequestRepo) UpdateStatus(requestID int, status string, approvedBy string) error {
	return r.db.Model(&VariantOnboardingRequest{}).Where("request_id = ?", requestID).
		Updates(map[string]interface{}{
			"status":      status,
			"approved_by": approvedBy,
		}).Error
}

func (r *variantOnboardingRequestRepo) GetAll() ([]VariantOnboardingRequest, error) {
	var requests []VariantOnboardingRequest
	err := r.db.Find(&requests).Error
	return requests, err
}
