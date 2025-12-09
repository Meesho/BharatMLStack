package apiresolver

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type Repository interface {
	GetResolver(method, path string) (ApiResolver, error)
	CreateResolver(resolver *ApiResolver) (uint, error)
}

type ServiceDiscoveryRepository struct {
	db *gorm.DB
}

func NewRepository(connection *infra.SQLConnection) (Repository, error) {
	if connection == nil {
		return nil, errors.New("connection cannot be nil")
	}
	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	return &ServiceDiscoveryRepository{
		db: session.(*gorm.DB),
	}, nil
}

func (s ServiceDiscoveryRepository) GetResolver(method, path string) (ApiResolver, error) {
	var resolver ApiResolver
	result := s.db.Where("method = ? AND api_path = ?", method, path).First(&resolver)
	return resolver, result.Error
}

func (s ServiceDiscoveryRepository) CreateResolver(resolver *ApiResolver) (uint, error) {
	result := s.db.Create(resolver)
	if result.Error != nil {
		return 0, result.Error
	}
	return resolver.ID, nil
}
