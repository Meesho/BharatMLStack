package handler

import (
	ofsConfig "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/entity"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/featuregroup"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/features"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/job"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/store"
)

type Config interface {
	RegisterStore(*RegisterStoreRequest) (uint, error)
	RegisterEntity(*RegisterEntityRequest) (uint, error)
	EditEntity(*EditEntityRequest) (uint, error)
	RegisterFeatureGroup(*RegisterFeatureGroupRequest) (uint, error)
	EditFeatureGroup(*EditFeatureGroupRequest) (uint, error)
	RegisterJob(*RegisterJobRequest) (uint, error)
	GetAllEntities() ([]string, error)
	GetFeatureGroupLabelsForEntity(entityLabel string) ([]string, error)
	GetJobsByJobType(jobType string) ([]string, error)
	GetStores() (map[string]ofsConfig.Store, error)
	GetConfig() (map[string]string, error)
	AddFeatures(*AddFeatureRequest) (uint, error)
	EditFeatures(*EditFeatureRequest) (uint, error)
	RetrieveEntities() (*[]RetrieveEntityResponse, error)
	RetrieveFeatureGroups(entityLabel string) (*[]RetrieveFeatureGroupResponse, error)
	ProcessEntity(*ProcessEntityRequest) error
	ProcessFeatureGroup(*ProcessFeatureGroupRequest) error
	ProcessStore(*ProcessStoreRequest) error
	ProcessJob(*ProcessJobRequest) error
	ProcessAddFeature(*ProcessAddFeatureRequest) error
	GetAllEntitiesRequest(email, role string) ([]entity.Table, error)
	GetAllFeatureGroupsRequest(email, role string) ([]featuregroup.Table, error)
	GetAllJobsRequest(email, role string) ([]job.Table, error)
	GetAllStoresRequest(email, role string) ([]store.Table, error)
	GetAllFeaturesRequest(email, role string) ([]features.Table, error)
	RetrieveSourceMapping(jobId, jobToken string) (*RetrieveSourceMappingResponse, error)
	GetOnlineFeatureMapping(request GetOnlineFeatureMappingRequest) (GetOnlineFeatureMappingResponse, error)
}
