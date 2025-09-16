package handler

import (
	ofsConfig "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config"
	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config/enums"
)

// RegisterStoreRequest is the request payload for RegisterStore
type RegisterStoreRequest struct {
	UserId      string   `json:"user-id"`
	Role        string   `json:"role"`
	ConfId      int      `json:"conf-id"`
	DbType      string   `json:"db-type"`
	Table       string   `json:"table"`
	PrimaryKeys []string `json:"primary-keys"`
	TableTtl    int      `json:"table-ttl"`
}

// RegisterEntityRequest is the request payload for RegisterEntity
type RegisterEntityRequest struct {
	EntityLabel      string                   `json:"entity-label"`
	KeyMap           map[string]ofsConfig.Key `json:"key-map"`
	DistributedCache ofsConfig.Cache          `json:"distributed-cache"`
	InMemoryCache    ofsConfig.Cache          `json:"in-memory-cache"`
	UserId           string                   `json:"user-id"`
	Role             string                   `json:"role"`
}

type EditEntityRequest struct {
	EntityLabel      string          `json:"entity-label"`
	DistributedCache ofsConfig.Cache `json:"distributed-cache"`
	InMemoryCache    ofsConfig.Cache `json:"in-memory-cache"`
	UserId           string          `json:"user-id"`
	Role             string          `json:"role"`
}

// RegisterFeatureGroupRequest is the request payload for RegisterFeatureGroup
type RegisterFeatureGroupRequest struct {
	UserId                  string         `json:"user-id"`
	Role                    string         `json:"role"`
	EntityLabel             string         `json:"entity-label"`
	FgLabel                 string         `json:"fg-label"`
	JobId                   string         `json:"job-id"`
	StoreId                 int            `json:"store-id"`
	TtlInSeconds            int            `json:"ttl-in-seconds"`
	InMemoryCacheEnabled    bool           `json:"in-memory-cache-enabled"`
	DistributedCacheEnabled bool           `json:"distributed-cache-enabled"`
	DataType                enums.DataType `json:"data-type"`
	Features                []Features     `json:"features"`
	LayoutVersion           int            `json:"layout-version"`
}

// EditFeatureGroupRequest is the request payload for EditFeatureGroup
type EditFeatureGroupRequest struct {
	UserId                  string `json:"user-id"`
	Role                    string `json:"role"`
	EntityLabel             string `json:"entity-label"`
	FgLabel                 string `json:"fg-label"`
	TtlInSeconds            int    `json:"ttl-in-seconds"`
	InMemoryCacheEnabled    bool   `json:"in-memory-cache-enabled"`
	DistributedCacheEnabled bool   `json:"distributed-cache-enabled"`
	LayoutVersion           int    `json:"layout-version"`
}

type Features struct {
	Labels           string `json:"labels"`
	DefaultValues    string `json:"default-values"`
	SourceBasePath   string `json:"source-base-path"`
	SourceDataColumn string `json:"source-data-column"`
	StorageProvider  string `json:"storage-provider"`
	StringLength     string `json:"string-length"`
	VectorLength     string `json:"vector-length"`
}

type FeatureResponse struct {
	Labels            string   `json:"labels"`
	DefaultValues     []string `json:"default-values"`
	SourceBasePaths   []string `json:"source-base-paths"`
	SourceDataColumns []string `json:"source-data-columns"`
	StorageProviders  []string `json:"storage-providers"`
	StringLengths     []uint16 `json:"string-lengths"`
	VectorLengths     []uint16 `json:"vector-lengths"`
}

// RegisterJobRequest is the request payload for RegisterJob
type RegisterJobRequest struct {
	UserId  string `json:"user-id"`
	Role    string `json:"role"`
	JobType string `json:"job-type"`
	JobId   string `json:"job-id"`
	Token   string `json:"token"`
}

// AddFeatureRequest is the request payload for AddFeatures
type AddFeatureRequest struct {
	UserId            string     `json:"user-id"`
	Role              string     `json:"role"`
	EntityLabel       string     `json:"entity-label"`
	FeatureGroupLabel string     `json:"feature-group-label"`
	Features          []Features `json:"features"`
}

// EditFeatureRequest is the request payload for EditFeatures
type EditFeatureRequest struct {
	UserId            string     `json:"user-id"`
	Role              string     `json:"role"`
	EntityLabel       string     `json:"entity-label"`
	FeatureGroupLabel string     `json:"feature-group-label"`
	Features          []Features `json:"features"`
}

type RetrieveEntityResponse struct {
	EntityLabel      string                   `json:"entity-label"`
	Keys             map[string]ofsConfig.Key `json:"keys"`
	InMemoryCache    ofsConfig.Cache          `json:"in-memory-cache"`
	DistributedCache ofsConfig.Cache          `json:"distributed-cache"`
}

type RetrieveFeatureGroupResponse struct {
	EntityLabel             string                     `json:"entity-label"`
	FeatureGroupLabel       string                     `json:"feature-group-label"`
	Id                      int                        `json:"id"`
	ActiveVersion           string                     `json:"active-version"`
	Features                map[string]FeatureResponse `json:"features"`
	StoreId                 string                     `json:"store-id"`
	DataType                enums.DataType             `json:"data-type"`
	TtlInSeconds            int                        `json:"ttl-in-seconds"`
	TtlInEpoch              uint64                     `json:"ttl-in-epoch"`
	JobId                   string                     `json:"job-id"`
	InMemoryCacheEnabled    bool                       `json:"in-memory-cache-enabled"`
	DistributedCacheEnabled bool                       `json:"distributed-cache-enabled"`
	LayoutVersion           int                        `json:"layout-version"`
}

type ProcessEntityRequest struct {
	ApproverId   string `json:"approver-id"`
	Role         string `json:"role"`
	RequestId    int    `json:"request-id"`
	Status       string `json:"status"`
	RejectReason string `json:"reject-reason"`
}

type ProcessFeatureGroupRequest struct {
	ApproverId   string `json:"approver-id"`
	Role         string `json:"role"`
	RequestId    int    `json:"request-id"`
	Status       string `json:"status"`
	RejectReason string `json:"reject-reason"`
}

type ProcessJobRequest struct {
	ApproverId   string `json:"approver-id"`
	Role         string `json:"role"`
	RequestId    int    `json:"request-id"`
	Status       string `json:"status"`
	RejectReason string `json:"reject-reason"`
}

type ProcessStoreRequest struct {
	ApproverId   string `json:"approver-id"`
	Role         string `json:"role"`
	RequestId    int    `json:"request-id"`
	Status       string `json:"status"`
	RejectReason string `json:"reject-reason"`
}

type ProcessAddFeatureRequest struct {
	ApproverId   string `json:"approver-id"`
	Role         string `json:"role"`
	RequestId    int    `json:"request-id"`
	Status       string `json:"status"`
	RejectReason string `json:"reject-reason"`
}

type RetrieveSourceMappingResponse struct {
	Data []Data              `json:"data"`
	Keys map[string][]string `json:"keys"`
}

type Data struct {
	StorageProvider string     `json:"storage-provider"`
	BasePath        []BasePath `json:"base-path"`
}

type BasePath struct {
	SourceBasePath string      `json:"source-base-path"`
	DataPaths      []DataPaths `json:"data-paths"`
}

type DataPaths struct {
	EntityLabel       string         `json:"entity-label"`
	FeatureGroupLabel string         `json:"feature-group-label"`
	FeatureLabel      string         `json:"feature-label"`
	SourceDataColumn  string         `json:"source-data-column"`
	DefaultValue      string         `json:"default-value"`
	DataType          enums.DataType `json:"data-type"`
}

type RetrieveSourceMappingRequest struct {
	EntityLabels []string `json:"entity-labels"`
}

type GetOnlineFeatureMappingRequest struct {
	OfflineFeatureList []string `json:"offline-feature-list"`
}

type GetOnlineFeatureMappingResponse struct {
	Error string   `json:"error"`
	Data  []string `json:"data"`
}

type FeatureGroupResponse struct {
	Label    string         `json:"label"`
	DataType enums.DataType `json:"data-type"`
}
