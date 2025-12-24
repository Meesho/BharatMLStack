package deployablemetadata

import (
	"errors"

	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"
	"gorm.io/gorm"
)

type DeployableMetadataRepository interface {
	GetGroupedActiveMetadata() (map[string][]string, error)
}

type repositoryImpl struct {
	db *gorm.DB
}

func NewRepository(connection *infra.SQLConnection) (DeployableMetadataRepository, error) {
	if connection == nil {
		return nil, errors.New("SQL connection cannot be nil")
	}

	session, err := connection.GetConn()
	if err != nil {
		return nil, err
	}

	return &repositoryImpl{
		db: session.(*gorm.DB),
	}, nil
}

func (r *repositoryImpl) GetGroupedActiveMetadata() (map[string][]string, error) {
	var items []DeployableMetadata
	err := r.db.Where("active = ?", true).Find(&items).Error
	if err != nil {
		return nil, err
	}

	grouped := make(map[string][]string)
	for _, item := range items {
		grouped[item.Key] = append(grouped[item.Key], item.Value)
	}

	return grouped, nil
}
