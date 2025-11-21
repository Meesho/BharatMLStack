package handler

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/scylla"
	"github.com/Meesho/BharatMLStack/horizon/pkg/infra"

	config2 "github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config"
	"github.com/Meesho/BharatMLStack/horizon/internal/online-feature-store/config/enums"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/entity"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/featuregroup"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/features"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/job"
	"github.com/Meesho/BharatMLStack/horizon/internal/repositories/sql/store"
	"github.com/rs/zerolog/log"
)

type OnlineFeatureStore struct {
	Config       config2.Manager
	entityRepo   entity.Repository
	fgRepo       featuregroup.Repository
	featureRepo  features.Repository
	storeRepo    store.Repository
	jobRepo      job.Repository
	scyllaStores map[int]scylla.Store //ConfigId to scylla store
}

var FeatureList = []string{"PARENT", "PCTR_CALIBRATION", "PCVR_CALIBRATION"}

const (
	storageScyllaPrefix        = "SCYLLA_"
	activeConfIds              = "ACTIVE_CONFIG_IDS"
	storageRedisFailoverPrefix = "REDIS_FAILOVER_"
	distributedCachePrefix     = "DISTRIBUTED_CACHE_"
	inMemoryCachePrefix        = "IN_MEMORY_CACHE_"
)

var (
	confIdToDbTypeMap                 = make(map[string]string)
	distributedCacheConfIdToDbTypeMap = make(map[string]string)
	inMemoryCacheConfIdToDbTypeMap    = make(map[string]string)
)

func InitV1ConfigHandler() Config {
	if config == nil {
		once.Do(func() {
			scyllaActiveConfIdsStr := viper.GetString(storageScyllaPrefix + activeConfIds)
			redisFailoverActiveConfIdsStr := viper.GetString(storageRedisFailoverPrefix + activeConfIds)
			distributedCacheActiveConfIdsStr := viper.GetString(distributedCachePrefix + activeConfIds)
			inMemoryCacheActiveConfIdsStr := viper.GetString(inMemoryCachePrefix + activeConfIds)
			scyllaStores := make(map[int]scylla.Store)

			if scyllaActiveConfIdsStr != "" {
				scyllaActiveIds := strings.Split(scyllaActiveConfIdsStr, ",")
				scyllaStores = make(map[int]scylla.Store, len(scyllaActiveIds))
				for _, configIdStr := range scyllaActiveIds {
					confIdToDbTypeMap[configIdStr] = "scylla"
					activeConfigId, err := strconv.Atoi(configIdStr)
					if err != nil {
						log.Error().Msgf("Error in converting config id %s to int", configIdStr)
						continue
					}
					connFacade, _ := infra.Scylla.GetConnection(activeConfigId)
					conn := connFacade.(*infra.ScyllaClusterConnection)
					scyllaStore, err2 := scylla.NewRepository(conn)
					if err2 != nil {
						log.Error().Msgf("Error in creating scylla store")
					}
					scyllaStores[activeConfigId] = scyllaStore
				}
			} else {
				log.Warn().Msg("SCYLLA_ACTIVE_CONFIG_IDS not configured, running without Scylla stores")
			}
			if redisFailoverActiveConfIdsStr != "" {
				redisFailoverActiveIds := strings.Split(redisFailoverActiveConfIdsStr, ",")
				for _, redisFailoverActiveId := range redisFailoverActiveIds {
					confIdToDbTypeMap[redisFailoverActiveId] = "redis_failover"
				}
			}

			if distributedCacheActiveConfIdsStr != "" {
				distributedCacheActiveIds := strings.Split(distributedCacheActiveConfIdsStr, ",")
				for _, distributedCacheActiveId := range distributedCacheActiveIds {
					distributedCacheConfIdToDbTypeMap[distributedCacheActiveId] = "distributed_cache"
				}
			}

			if inMemoryCacheActiveConfIdsStr != "" {
				inMemoryCacheActiveIds := strings.Split(inMemoryCacheActiveConfIdsStr, ",")
				for _, inMemoryCacheActiveId := range inMemoryCacheActiveIds {
					inMemoryCacheConfIdToDbTypeMap[inMemoryCacheActiveId] = "in_memory_cache"
				}
			}

			connection, _ := infra.SQL.GetConnection()
			sqlConn := connection.(*infra.SQLConnection)
			entityRepo, err := entity.NewRepository(sqlConn)
			if err != nil {
				log.Error().Msgf("Error in creating entity repository")
			}
			fgRepo, err := featuregroup.NewRepository(sqlConn)
			if err != nil {
				log.Error().Msgf("Error in creating feature group repository")
			}
			featureRepo, err := features.NewRepository(sqlConn)
			if err != nil {
				log.Error().Msgf("Error in creating feature repository")
			}
			storeRepo, err := store.NewRepository(sqlConn)
			if err != nil {
				log.Error().Msgf("Error in creating store repository")
			}
			jobRepo, err := job.NewRepository(sqlConn)
			if err != nil {
				log.Error().Msgf("Error in creating job repository")
			}
			config = &OnlineFeatureStore{
				Config:       config2.NewEtcdConfig(),
				entityRepo:   entityRepo,
				fgRepo:       fgRepo,
				featureRepo:  featureRepo,
				storeRepo:    storeRepo,
				jobRepo:      jobRepo,
				scyllaStores: scyllaStores,
			}
		})
	}
	return config
}

func (o *OnlineFeatureStore) RegisterStore(request *RegisterStoreRequest) (uint, error) {
	//check if confId is valid
	if _, ok := confIdToDbTypeMap[strconv.Itoa(request.ConfId)]; !ok {
		return 0, fmt.Errorf("invalid conf id :%d", request.ConfId)
	}

	//check if dbType is valid
	if confIdToDbTypeMap[strconv.Itoa(request.ConfId)] != request.DbType {
		return 0, fmt.Errorf("invalid db type %s for conf id %d", request.DbType, request.ConfId)
	}

	payload, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error:", err)
		return 0, err
	}
	requestId, err := o.storeRepo.Create(&store.Table{
		Payload:    string(payload),
		CreatedBy:  request.UserId,
		ApprovedBy: "",
		Status:     "PENDING APPROVAL",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Service:    "Online Feature Store",
	})
	if err != nil {
		log.Error().Msgf("Error Registering Store for %s , %s", request.DbType, request.Table)
		return 0, err
	}
	return requestId, nil
}

func (o *OnlineFeatureStore) ProcessStore(request *ProcessStoreRequest) error {
	e, err := o.storeRepo.GetById(request.RequestId)
	if e == nil || err != nil {
		log.Error().Msgf("Error Approving Store for %d", request.RequestId)
		return err
	}
	if e.Status == "APPROVED" || e.Status == "REJECTED" {
		log.Error().Msgf("Store is already Processed for %d", request.RequestId)
		return fmt.Errorf("store is already Processed for %d", request.RequestId)
	}
	var payload RegisterStoreRequest
	if request.Status != "REJECTED" {
		json.Unmarshal([]byte(e.Payload), &payload)
		storeMap, err := o.Config.RegisterStore(payload.ConfId, payload.DbType, payload.Table, payload.PrimaryKeys, payload.TableTtl)
		if err != nil {
			log.Error().Msgf("Error Registering Store for %s , %s", payload.DbType, payload.Table)
			return err
		}

		err = o.Config.CreateStore(storeMap)
		if err != nil {
			log.Error().Msgf("Error Creating Store for %s , %s", payload.DbType, payload.Table)
			return err
		}

		if payload.DbType != "scylla" {
			// Table creation is not required
			return nil
		}

		err = o.scyllaStores[payload.ConfId].CreateTable(payload.Table, payload.PrimaryKeys, payload.TableTtl)
		if err != nil {
			log.Error().Msgf("Error Creating Table for %s , %s", payload.DbType, payload.Table)
			return err
		}
	}
	err = o.storeRepo.Update(&store.Table{
		RequestId:    uint(request.RequestId),
		ApprovedBy:   request.ApproverId,
		Status:       request.Status,
		Service:      "Online Feature Store",
		RejectReason: request.RejectReason,
	})
	if err != nil {
		log.Error().Msgf("Error Approving Store for %v", payload)
		return err
	}
	return nil
}

func (o *OnlineFeatureStore) RegisterEntity(request *RegisterEntityRequest) (uint, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error:", err)
		return 0, err
	}
	requestId, err := o.entityRepo.Create(&entity.Table{
		Payload:     string(payload),
		CreatedBy:   request.UserId,
		EntityLabel: request.EntityLabel,
		ApprovedBy:  "",
		Status:      "PENDING APPROVAL",
		RequestType: "CREATE",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Service:     "Online Feature Store",
	})
	if err != nil {
		log.Error().Msgf("Error Registering Entity for %s , %v", request.EntityLabel, request.KeyMap)
		return 0, err
	}
	return requestId, nil
}

func (o *OnlineFeatureStore) EditEntity(request *EditEntityRequest) (uint, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error:", err)
		return 0, err
	}
	requestId, err := o.entityRepo.Create(&entity.Table{
		Payload:     string(payload),
		CreatedBy:   request.UserId,
		EntityLabel: request.EntityLabel,
		ApprovedBy:  "",
		Status:      "PENDING APPROVAL",
		RequestType: "EDIT",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Service:     "Online Feature Store",
	})
	if err != nil {
		log.Error().Msgf("Error Registering Entity for %s", request.EntityLabel)
		return 0, err
	}
	return requestId, nil
}

func (o *OnlineFeatureStore) GetAllEntities() ([]string, error) {
	return o.Config.GetAllEntities()
}

func (o *OnlineFeatureStore) GetFeatureGroupLabelsForEntity(entityLabel string) ([]string, error) {
	return o.Config.GetAllFeatureGroupByEntityLabel(entityLabel)
}

func (o *OnlineFeatureStore) GetJobsByJobType(jobType string) ([]string, error) {
	return o.Config.GetJobs(jobType)
}

func (o *OnlineFeatureStore) GetStores() (map[string]config2.Store, error) {
	return o.Config.GetStores()
}

func (o *OnlineFeatureStore) GetConfig() (map[string]string, error) {
	return confIdToDbTypeMap, nil
}

func (o *OnlineFeatureStore) ProcessEntity(request *ProcessEntityRequest) error {
	e, err := o.entityRepo.GetById(request.RequestId)
	if e == nil || err != nil {
		log.Error().Msgf("Error Approving Entity for %d", request.RequestId)
		return err
	}
	if e.Status == "APPROVED" {
		log.Error().Msgf("Entity is already Processed for %d", request.RequestId)
		return fmt.Errorf("entity is already Processed for %d", request.RequestId)
	}
	var registerPayload RegisterEntityRequest
	var editPayload EditEntityRequest
	if request.Status != "REJECTED" {
		if e.RequestType == "CREATE" {
			json.Unmarshal([]byte(e.Payload), &registerPayload)
			err = o.Config.RegisterEntity(registerPayload.EntityLabel, registerPayload.KeyMap, registerPayload.DistributedCache, registerPayload.InMemoryCache)
			if err != nil {
				log.Error().Msgf("Error Creating Entity for %s , %v", registerPayload.EntityLabel, registerPayload.KeyMap)
				return err
			}
		} else if e.RequestType == "EDIT" {
			json.Unmarshal([]byte(e.Payload), &editPayload)
			err = o.Config.EditEntity(editPayload.EntityLabel, editPayload.DistributedCache, editPayload.InMemoryCache)
			if err != nil {
				log.Error().Msgf("Error Editing Entity for %s", editPayload.EntityLabel)
				return err
			}
		} else {
			return fmt.Errorf("invalid request type")
		}

	}
	err = o.entityRepo.Update(&entity.Table{
		RequestId:    uint(request.RequestId),
		ApprovedBy:   request.ApproverId,
		Status:       request.Status,
		Service:      "Online Feature Store",
		RejectReason: request.RejectReason,
	})
	if err != nil {
		log.Error().Msgf("Error Approving Entity for %s", registerPayload.EntityLabel)
		return err
	}
	return nil
}

func (o *OnlineFeatureStore) RegisterFeatureGroup(request *RegisterFeatureGroupRequest) (uint, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error:", err)
		return 0, err
	}
	requestId, err := o.fgRepo.Create(&featuregroup.Table{
		Payload:           payload,
		CreatedBy:         request.UserId,
		EntityLabel:       request.EntityLabel,
		FeatureGroupLabel: request.FgLabel,
		ApprovedBy:        "",
		Status:            "PENDING APPROVAL",
		RequestType:       "CREATE",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Service:           "Online Feature Store",
	})
	if err != nil {
		log.Error().Msgf("Error Registering Feature Group for %s", request.FgLabel)
		return 0, err
	}
	return requestId, nil
}

func (o *OnlineFeatureStore) EditFeatureGroup(request *EditFeatureGroupRequest) (uint, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error:", err)
		return 0, err
	}
	requestId, err := o.fgRepo.Create(&featuregroup.Table{
		Payload:           payload,
		CreatedBy:         request.UserId,
		EntityLabel:       request.EntityLabel,
		FeatureGroupLabel: request.FgLabel,
		ApprovedBy:        "",
		Status:            "PENDING APPROVAL",
		RequestType:       "EDIT",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Service:           "Online Feature Store",
	})
	if err != nil {
		log.Error().Msgf("Error Editing Feature Group for %s", request.FgLabel)
		return 0, err
	}
	return requestId, nil
}

func (o *OnlineFeatureStore) ProcessFeatureGroup(request *ProcessFeatureGroupRequest) error {
	fg, err := o.fgRepo.GetById(request.RequestId)
	if fg == nil || err != nil {
		log.Error().Msgf("Error Approving Feature Group for %d", request.RequestId)
		return err
	}
	if fg.Status == "APPROVED" {
		log.Error().Msgf("Feature Group is already Processed for %d", request.RequestId)
		return fmt.Errorf("feature group is already Processed for %d", request.RequestId)
	}
	var registerPayload RegisterFeatureGroupRequest
	var editPayload EditFeatureGroupRequest
	if request.Status != "REJECTED" {
		if fg.RequestType == "CREATE" {
			json.Unmarshal(fg.Payload, &registerPayload)
			var featureLabels, featureDefaultValues, storageProvider, basePaths, dataPaths, stringLength, vectorLength []string

			hasValidFeatures := false
			for _, feature := range registerPayload.Features {
				if feature.Labels != "" {
					hasValidFeatures = true
					featureLabels = append(featureLabels, feature.Labels)
					featureDefaultValues = append(featureDefaultValues, feature.DefaultValues)
					storageProvider = append(storageProvider, feature.StorageProvider)
					basePaths = append(basePaths, feature.SourceBasePath)
					dataPaths = append(dataPaths, feature.SourceDataColumn)
					stringLength = append(stringLength, feature.StringLength)
					vectorLength = append(vectorLength, feature.VectorLength)
				}
			}

			if !hasValidFeatures {
				log.Info().Msgf("Creating feature group %s without features - no valid features provided", registerPayload.FgLabel)
			}

			columns, store, paths, err := o.Config.RegisterFeatureGroup(registerPayload.EntityLabel, registerPayload.FgLabel, registerPayload.JobId, registerPayload.StoreId, registerPayload.TtlInSeconds, registerPayload.InMemoryCacheEnabled, registerPayload.DistributedCacheEnabled,
				registerPayload.DataType, featureLabels, featureDefaultValues, storageProvider, basePaths, dataPaths, stringLength, vectorLength, registerPayload.LayoutVersion)
			if err != nil {
				log.Error().Msgf("Error Registering Feature Group for %s , %s, %s", registerPayload.EntityLabel, registerPayload.FgLabel, err)
				return err
			}
			err = o.Config.CreateFeatureGroup(paths)
			if err != nil {
				log.Error().Msgf("Error Creating Feature Group for %s , %s, %s", registerPayload.EntityLabel, registerPayload.FgLabel, err)
				return err
			}
			if store.DbType != "scylla" {
				// Column creation is not required. Still update approval status for non-scylla stores (e.g., redis_failover)
				err = o.fgRepo.Update(&featuregroup.Table{
					RequestId:    uint(request.RequestId),
					ApprovedBy:   request.ApproverId,
					Status:       request.Status,
					RejectReason: request.RejectReason,
				})
				if err != nil {
					log.Error().Msg("Error Approving Feature Group")
					return err
				}
				return nil
			}
			for _, column := range columns {
				err = o.scyllaStores[store.ConfId].AddColumn(store.Table, column)
				if err != nil {
					log.Error().Msgf("Error Adding Column for %s", err)
					return err
				}
			}
		} else if fg.RequestType == "EDIT" {
			json.Unmarshal(fg.Payload, &editPayload)
			err = o.Config.EditFeatureGroup(editPayload.EntityLabel, editPayload.FgLabel, editPayload.TtlInSeconds, editPayload.InMemoryCacheEnabled, editPayload.DistributedCacheEnabled, editPayload.LayoutVersion)
			if err != nil {
				log.Error().Msgf("Error Registering Feature Group for %s , %s, %s", editPayload.EntityLabel, editPayload.FgLabel, err)
				return err
			}
		} else {
			return fmt.Errorf("invalid request type")
		}
	}
	err = o.fgRepo.Update(&featuregroup.Table{
		RequestId:    uint(request.RequestId),
		ApprovedBy:   request.ApproverId,
		Status:       request.Status,
		RejectReason: request.RejectReason,
	})
	if err != nil {
		log.Error().Msg("Error Approving Feature Group")
		return err
	}
	return nil
}

func (o *OnlineFeatureStore) RegisterJob(request *RegisterJobRequest) (uint, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error:", err)
		return 0, err
	}
	requestId, err := o.jobRepo.Create(&job.Table{
		Payload:    string(payload),
		JobId:      request.JobId,
		CreatedBy:  request.UserId,
		ApprovedBy: "",
		Status:     "PENDING APPROVAL",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Service:    "Online Feature Store",
	})
	if err != nil {
		log.Error().Msgf("Error Registering Job for %s ", request.JobId)
		return 0, err
	}
	return requestId, nil

}

func (o *OnlineFeatureStore) ProcessJob(request *ProcessJobRequest) error {
	fg, err := o.jobRepo.GetById(request.RequestId)
	if fg == nil || err != nil {
		log.Error().Msgf("Error Approving Feature Group for %d", request.RequestId)
		return err
	}
	if fg.Status != "PENDING APPROVAL" {
		log.Error().Msgf("Feature Group is already Processed for %d", request.RequestId)
		return fmt.Errorf("feature group is already Processed for %d", request.RequestId)
	}
	var payload RegisterJobRequest
	if request.Status != "REJECTED" {
		json.Unmarshal([]byte(fg.Payload), &payload)
		err = o.Config.RegisterJob(payload.JobType, payload.JobId, payload.Token)
		if err != nil {
			log.Error().Msgf("Error Registering Job for %s , %s", payload.JobType, payload.JobId)
			return err
		}
	}
	err = o.jobRepo.Update(&job.Table{
		RequestId:    uint(request.RequestId),
		ApprovedBy:   request.ApproverId,
		Status:       request.Status,
		Service:      "Online Feature Store",
		RejectReason: request.RejectReason,
	})
	if err != nil {
		log.Error().Msgf("Error Approving Job for %s ", payload.JobId)
		return err
	}
	return nil
}

func (o *OnlineFeatureStore) AddFeatures(request *AddFeatureRequest) (uint, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error:", err)
		return 0, err
	}
	requestId, err := o.featureRepo.Create(&features.Table{
		Payload:           payload,
		CreatedBy:         request.UserId,
		EntityLabel:       request.EntityLabel,
		FeatureGroupLabel: request.FeatureGroupLabel,
		ApprovedBy:        "",
		Status:            "PENDING APPROVAL",
		RequestType:       "CREATE",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Service:           "Online Feature Store",
	})
	if err != nil {
		log.Error().Msgf("Error Adding Features for %s , %s", request.EntityLabel, request.FeatureGroupLabel)
		return 0, err
	}
	return requestId, nil
}

func (o *OnlineFeatureStore) EditFeatures(request *EditFeatureRequest) (uint, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error:", err)
		return 0, err
	}
	requestId, err := o.featureRepo.Create(&features.Table{
		Payload:           payload,
		CreatedBy:         request.UserId,
		EntityLabel:       request.EntityLabel,
		FeatureGroupLabel: request.FeatureGroupLabel,
		ApprovedBy:        "",
		Status:            "PENDING APPROVAL",
		RequestType:       "EDIT",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Service:           "Online Feature Store",
	})
	if err != nil {
		log.Error().Msgf("Error Adding Features for %s , %s", request.EntityLabel, request.FeatureGroupLabel)
		return 0, err
	}
	return requestId, nil
}

func (o *OnlineFeatureStore) DeleteFeatures(request *DeleteFeaturesRequest) (uint, error) {
	fg, err := o.Config.GetFeatureGroup(request.EntityLabel, request.FeatureGroupLabel)
	if err != nil {
		return 0, err
	}

	featureMetaMap := fg.Features[fg.ActiveVersion].FeatureMeta
	existingLabelSet := make(map[string]struct{})
	for key := range featureMetaMap {
		existingLabelSet[key] = struct{}{}
	}

	var missingFeatures []string
	for _, label := range request.FeatureLabels {
		if _, exists := existingLabelSet[label]; !exists {
			missingFeatures = append(missingFeatures, label)
		}
	}

	if len(missingFeatures) > 0 {
		return 0, fmt.Errorf("features not found in the active version schema: %s", strings.Join(missingFeatures, ", "))
	}

	payload, err := json.Marshal(request)
	if err != nil {
		fmt.Println("Error:", err)
		return 0, err
	}

	requestId, err := o.featureRepo.Create(&features.Table{
		Payload:           payload,
		CreatedBy:         request.UserId,
		EntityLabel:       request.EntityLabel,
		FeatureGroupLabel: request.FeatureGroupLabel,
		ApprovedBy:        "",
		Status:            "PENDING APPROVAL",
		RequestType:       "DELETE",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Service:           "Online Feature Store",
	})
	if err != nil {
		log.Error().Msgf("Error Deleting Features for %s , %s", request.EntityLabel, request.FeatureGroupLabel)
		return 0, err
	}
	return requestId, nil
}

func (o *OnlineFeatureStore) ProcessAddFeature(request *ProcessAddFeatureRequest) error {
	fg, err := o.featureRepo.GetById(request.RequestId)
	if fg == nil || err != nil {
		log.Error().Msgf("Error Approving Feature Group for %d", request.RequestId)
		return err
	}
	if fg.Status == "APPROVED" {
		log.Error().Msgf("Feature Group is already Processed for %d", request.RequestId)
		return fmt.Errorf("feature group is already Processed for %d", request.RequestId)
	}
	var addPayload AddFeatureRequest
	var editPayload EditFeatureRequest
	if request.Status != "REJECTED" {
		var featureLabels, featureDefaultValues, storageProvider, basePaths, dataPaths, stringLength, vectorLength []string
		if fg.RequestType == "CREATE" {
			json.Unmarshal(fg.Payload, &addPayload)
			for _, feature := range addPayload.Features {
				featureLabels = append(featureLabels, feature.Labels)
				featureDefaultValues = append(featureDefaultValues, feature.DefaultValues)
				storageProvider = append(storageProvider, feature.StorageProvider)
				basePaths = append(basePaths, feature.SourceBasePath)
				dataPaths = append(dataPaths, feature.SourceDataColumn)
				stringLength = append(stringLength, feature.StringLength)
				vectorLength = append(vectorLength, feature.VectorLength)
			}
			columns, store, pathsToUpdate, paths, err := o.Config.AddFeatures(addPayload.EntityLabel, addPayload.FeatureGroupLabel, featureLabels, featureDefaultValues, storageProvider, basePaths, dataPaths, stringLength, vectorLength)
			if err != nil {
				log.Error().Msgf("Error Adding Features for %s , %s", addPayload.EntityLabel, addPayload.FeatureGroupLabel)
				return err
			}

			err = o.Config.CreateAddFeaturesNodes(paths, pathsToUpdate)
			if err != nil {
				log.Error().Msgf("Error Creating Add Features Nodes for %s , %s", addPayload.EntityLabel, addPayload.FeatureGroupLabel)
				return err
			}

			if store.DbType != "scylla" {
				// Column creation is not required
				err = o.featureRepo.Update(&features.Table{
					RequestId:    uint(request.RequestId),
					ApprovedBy:   request.ApproverId,
					Status:       request.Status,
					Service:      "Online Feature Store",
					RejectReason: request.RejectReason,
				})
				if err != nil {
					log.Error().Msg("Error Approving Features")
					return err
				}
				return nil
			}

			for _, column := range columns {
				err = o.scyllaStores[store.ConfId].AddColumn(store.Table, column)
				if err != nil {
					log.Error().Msgf("Error Adding Column for %s", err)
					return err
				}
			}
		} else if fg.RequestType == "EDIT" {
			json.Unmarshal(fg.Payload, &editPayload)
			for _, feature := range editPayload.Features {
				featureLabels = append(featureLabels, feature.Labels)
				featureDefaultValues = append(featureDefaultValues, feature.DefaultValues)
				storageProvider = append(storageProvider, feature.StorageProvider)
				basePaths = append(basePaths, feature.SourceBasePath)
				dataPaths = append(dataPaths, feature.SourceDataColumn)
				stringLength = append(stringLength, feature.StringLength)
				vectorLength = append(vectorLength, feature.VectorLength)
			}
			columnsToAdd, store, err := o.Config.EditFeatures(editPayload.EntityLabel, editPayload.FeatureGroupLabel, featureLabels, featureDefaultValues, storageProvider, basePaths, dataPaths, stringLength, vectorLength)
			if err != nil {
				log.Error().Msgf("Error Editing Features for %s , %s", editPayload.EntityLabel, editPayload.FeatureGroupLabel)
				return err
			}
			if len(columnsToAdd) > 0 && store.DbType == "scylla" {
				for _, column := range columnsToAdd {
					err = o.scyllaStores[store.ConfId].AddColumn(store.Table, column)
					if err != nil {
						log.Error().Msgf("Error Adding Column for %s", err)
						return err
					}
				}
			}

		} else {
			return fmt.Errorf("invalid request type")
		}
	}
	err = o.featureRepo.Update(&features.Table{
		RequestId:    uint(request.RequestId),
		ApprovedBy:   request.ApproverId,
		Status:       request.Status,
		Service:      "Online Feature Store",
		RejectReason: request.RejectReason,
	})
	if err != nil {
		log.Error().Msg("Error Approving Features")
		return err
	}
	return nil
}

func (o *OnlineFeatureStore) ProcessDeleteFeatures(request *ProcessDeleteFeaturesRequest) error {
	fg, err := o.featureRepo.GetById(request.RequestId)
	if fg == nil || err != nil {
		log.Error().Msgf("Error Approving Feature Delete for %d", request.RequestId)
		return err
	}
	if fg.Status == "APPROVED" {
		log.Error().Msgf("Delete features request is already Processed for request id %d", request.RequestId)
		return fmt.Errorf("delete features request is already Processed for request id %d", request.RequestId)
	}

	if request.Status != "REJECTED" {
		var deletePayload DeleteFeaturesRequest
		json.Unmarshal(fg.Payload, &deletePayload)

		fg, err := o.Config.GetFeatureGroup(deletePayload.EntityLabel, deletePayload.FeatureGroupLabel)
		if err != nil {
			return err
		}

		featureMetaMap := fg.Features[fg.ActiveVersion].FeatureMeta
		existingLabelSet := make(map[string]struct{})
		for key := range featureMetaMap {
			existingLabelSet[key] = struct{}{}
		}

		var missingFeatures []string
		for _, label := range deletePayload.FeatureLabels {
			if _, exists := existingLabelSet[label]; !exists {
				missingFeatures = append(missingFeatures, label)
			}
		}

		if len(missingFeatures) > 0 {
			return fmt.Errorf("features not found in the active version schema: %s", strings.Join(missingFeatures, ", "))
		}

		err = o.Config.DeleteFeatures(deletePayload.EntityLabel, deletePayload.FeatureGroupLabel, deletePayload.FeatureLabels)
		if err != nil {
			log.Error().Msgf("Error Deleting Features for %s , %s", deletePayload.EntityLabel, deletePayload.FeatureGroupLabel)
			return err
		}
	}

	err = o.featureRepo.Update(&features.Table{
		RequestId:    uint(request.RequestId),
		ApprovedBy:   request.ApproverId,
		Status:       request.Status,
		Service:      "Online Feature Store",
		RejectReason: request.RejectReason,
	})
	if err != nil {
		log.Error().Msgf("Error updating delete features request status for request ID %d: %v", request.RequestId, err)
		return err
	}
	return nil
}

func (o *OnlineFeatureStore) RetrieveEntities() (*[]RetrieveEntityResponse, error) {
	entities, err := o.Config.GetEntities()
	var response []RetrieveEntityResponse
	for entityLabel, entity := range entities {
		response = append(response, RetrieveEntityResponse{
			EntityLabel:      entityLabel,
			Keys:             entity.Keys,
			InMemoryCache:    entity.InMemoryCache,
			DistributedCache: entity.DistributedCache,
		})
	}
	if err != nil {
		log.Error().Msgf("Error Retrieving Entities")
		return nil, err
	}
	return &response, nil
}

func (o *OnlineFeatureStore) RetrieveFeatureGroups(entityLabel string) (*[]RetrieveFeatureGroupResponse, error) {
	featureGroups, err := o.Config.GetFeatureGroups(entityLabel)
	if err != nil {
		log.Error().Msgf("Error Retrieving Feature Groups for %s", entityLabel)
		return nil, err
	}
	source, err := o.Config.GetSource()
	if err != nil {
		log.Error().Msgf("Error Retrieving Source Mapping")
		return nil, err
	}
	response := make([]RetrieveFeatureGroupResponse, 0)

	for fgLabel, featureGroup := range featureGroups {
		features := make(map[string]FeatureResponse)
		for version, feature := range featureGroup.Features {
			var sourceBasePaths []string
			var sourceDataColumns []string
			var storageProviders []string
			var stringLengths []uint16
			var vectorLengths []uint16
			if version != featureGroup.ActiveVersion {
				continue
			}
			featureResponse := FeatureResponse{
				Labels: feature.Labels,
			}
			featureLabels := strings.Split(feature.Labels, ",")
			for _, featureLabel := range featureLabels {
				stringLengths = append(stringLengths, feature.FeatureMeta[featureLabel].StringLength)
				vectorLengths = append(vectorLengths, feature.FeatureMeta[featureLabel].VectorLength)

				sourceKey := fmt.Sprintf("%s|%s|%s", entityLabel, fgLabel, featureLabel)
				sourceValue, exists := source[sourceKey]
				if exists && sourceValue != "" {
					valueParts := strings.Split(sourceValue, "|")
					if len(valueParts) == 4 {
						storageProviders = append(storageProviders, valueParts[0])
						sourceBasePaths = append(sourceBasePaths, valueParts[1])
						sourceDataColumns = append(sourceDataColumns, valueParts[2])
					} else {
						// If source data exists but is malformed, add empty values to maintain array alignment
						storageProviders = append(storageProviders, "")
						sourceBasePaths = append(sourceBasePaths, "")
						sourceDataColumns = append(sourceDataColumns, "")
					}
				} else {
					continue
				}
			}
			featureResponse.StorageProviders = storageProviders
			featureResponse.SourceBasePaths = sourceBasePaths
			featureResponse.SourceDataColumns = sourceDataColumns
			featureResponse.DefaultValues, _ = splitOnCommaOutsideBrackets(feature.DefaultValues)
			featureResponse.StringLengths = stringLengths
			featureResponse.VectorLengths = vectorLengths
			features[version] = featureResponse
		}
		response = append(response, RetrieveFeatureGroupResponse{
			EntityLabel:             entityLabel,
			FeatureGroupLabel:       fgLabel,
			Id:                      featureGroup.Id,
			ActiveVersion:           featureGroup.ActiveVersion,
			Features:                features,
			StoreId:                 featureGroup.StoreId,
			DataType:                featureGroup.DataType,
			TtlInSeconds:            int(featureGroup.TtlInSeconds),
			JobId:                   featureGroup.JobId,
			InMemoryCacheEnabled:    featureGroup.InMemoryCacheEnabled,
			DistributedCacheEnabled: featureGroup.DistributedCacheEnabled,
			LayoutVersion:           featureGroup.LayoutVersion,
		})
	}
	return &response, nil
}

func (o *OnlineFeatureStore) GetAllEntitiesRequest(email, role string) ([]entity.Table, error) {
	var response []entity.Table
	var err error
	if role == "user" {
		response, err = o.entityRepo.GetAllByUserId(email)
	} else if role == "admin" {
		response, err = o.entityRepo.GetAll()
	} else {
		return response, fmt.Errorf("invalid role")
	}
	if err != nil {
		log.Error().Msgf("Error Retrieving Entities for %s", email)
		return nil, err
	}
	return response, nil
}

func (o *OnlineFeatureStore) GetAllFeatureGroupsRequest(email, role string) ([]featuregroup.Table, error) {
	var response []featuregroup.Table
	var err error
	if role == "user" {
		response, err = o.fgRepo.GetAllByUserId(email)
	} else if role == "admin" {
		response, err = o.fgRepo.GetAll()
	} else {
		return response, fmt.Errorf("invalid role")
	}
	if err != nil {
		log.Error().Msgf("Error Retrieving Entities for %s", email)
		return nil, err
	}
	return response, nil
}

func (o *OnlineFeatureStore) GetAllJobsRequest(email, role string) ([]job.Table, error) {
	var response []job.Table
	var err error
	if role == "user" {
		response, err = o.jobRepo.GetAllByUserId(email)
	} else if role == "admin" {
		response, err = o.jobRepo.GetAll()
	} else {
		return response, fmt.Errorf("invalid role")
	}
	if err != nil {
		log.Error().Msgf("Error Retrieving Entities for %s", email)
		return nil, err
	}
	return response, nil
}

func (o *OnlineFeatureStore) GetAllStoresRequest(email, role string) ([]store.Table, error) {
	var response []store.Table
	var err error
	if role == "user" {
		response, err = o.storeRepo.GetAllByUserId(email)
	} else if role == "admin" {
		response, err = o.storeRepo.GetAll()
	} else {
		return response, fmt.Errorf("invalid role")
	}
	if err != nil {
		log.Error().Msgf("Error Retrieving Entities for %s", email)
		return nil, err
	}
	return response, nil
}

func (o *OnlineFeatureStore) GetAllFeaturesRequest(email, role string) ([]features.Table, error) {
	var response []features.Table
	var err error
	if role == "user" {
		response, err = o.featureRepo.GetAllByUserId(email)
	} else if role == "admin" {
		response, err = o.featureRepo.GetAll()
	} else {
		return response, fmt.Errorf("invalid role")
	}
	if err != nil {
		log.Error().Msgf("Error Retrieving Entities for %s", email)
		return nil, err
	}
	return response, nil
}

func (o *OnlineFeatureStore) RetrieveSourceMapping(jobId, jobToken string) (*RetrieveSourceMappingResponse, error) {
	token, err := o.Config.GetJobsToken("writer", jobId)
	if err != nil {
		log.Error().Msgf("Error Retrieving Source Mapping")
		return nil, err
	}
	if token != jobToken {
		return nil, fmt.Errorf("invalid token")
	}
	sourceData, err := o.Config.GetSource()
	if err != nil {
		log.Error().Msgf("Error Retrieving Source Mapping")
		return nil, err
	}
	response := RetrieveSourceMappingResponse{}
	storageMap := make(map[string]map[string][]DataPaths)
	entityToKey := make(map[string][]string)
	uniqueEntityLabels := make(map[string]bool)
	for key, value := range sourceData {
		// Split the key
		keyParts := strings.Split(key, "|")
		if len(keyParts) != 3 {
			fmt.Println("Skipping invalid key:", key)
			continue
		}
		entityLabel, featureGroupLabel, featureLabel := keyParts[0], keyParts[1], keyParts[2]
		fg, err := o.Config.GetFeatureGroup(entityLabel, featureGroupLabel)
		if err != nil {
			fmt.Println("Error Getting Fg", key)
			continue
		}
		if fg.JobId != jobId {
			continue
		}
		if _, exists := uniqueEntityLabels[entityLabel]; !exists {
			uniqueEntityLabels[entityLabel] = true
		}
		// Split the value
		valueParts := strings.Split(value, "|")
		if len(valueParts) != 4 {
			fmt.Println("Skipping invalid value:", value)
			continue
		}
		storageType, basePath, columnName, defaultValue := valueParts[0], valueParts[1], valueParts[2], valueParts[3]

		// Create DataPaths entry
		dataPath := DataPaths{
			EntityLabel:       entityLabel,
			FeatureGroupLabel: featureGroupLabel,
			FeatureLabel:      featureLabel,
			SourceDataColumn:  columnName,
			DefaultValue:      defaultValue,
			DataType:          fg.DataType,
		}

		// Group by storage type
		if _, exists := storageMap[storageType]; !exists {
			storageMap[storageType] = make(map[string][]DataPaths)
		}
		storageMap[storageType][basePath] = append(storageMap[storageType][basePath], dataPath)
	}
	for entityLabel := range uniqueEntityLabels {
		entityKeys, err := o.Config.GetEntityKeys(entityLabel)
		for _, value := range entityKeys {
			entityToKey[entityLabel] = append(entityToKey[entityLabel], value.EntityLabel)
		}
		if err != nil {
			fmt.Println("Error Getting Entity Keys", entityLabel)
			continue
		}
	}
	response.Keys = entityToKey
	// Convert the grouped map into the response struct
	for storageType, basePathMap := range storageMap {
		var basePaths []BasePath
		for basePath, dataPaths := range basePathMap {
			basePaths = append(basePaths, BasePath{
				SourceBasePath: basePath,
				DataPaths:      dataPaths,
			})
		}
		response.Data = append(response.Data, Data{
			StorageProvider: storageType,
			BasePath:        basePaths,
		})
	}

	return &response, nil
}

func splitOnCommaOutsideBrackets(input string) ([]string, error) {
	var parts []string
	var currentPart []rune
	openBrackets := 0

	for _, char := range input {
		switch char {
		case '[':
			openBrackets++
		case ']':
			openBrackets--
			if openBrackets < 0 {
				return nil, fmt.Errorf("mismatched brackets in input")
			}
		case ',':
			if openBrackets == 0 {
				parts = append(parts, string(currentPart))
				currentPart = nil
				continue
			}
		}
		currentPart = append(currentPart, char)
	}

	if openBrackets != 0 {
		return nil, fmt.Errorf("mismatched brackets in input")
	}

	// Append the last part
	if len(currentPart) > 0 {
		parts = append(parts, string(currentPart))
	}

	return parts, nil
}

func (o *OnlineFeatureStore) GetOnlineFeatureMapping(request GetOnlineFeatureMappingRequest) (GetOnlineFeatureMappingResponse, error) {
	onlineFeatureList := make(map[string]string)

	sourceData, err := o.Config.GetSource()
	if err != nil {
		log.Error().Msgf("Error Retrieving Source Mapping")
		return GetOnlineFeatureMappingResponse{}, err
	}

	sourceDataMap := make(map[string]string)

	for key, value := range sourceData {
		keyParts := strings.Split(key, "|")
		if len(keyParts) != 3 {
			fmt.Println("Skipping invalid key:", key)
			continue
		}

		valueParts := strings.Split(value, "|")
		if len(valueParts) != 4 {
			fmt.Println("Skipping invalid value:", value)
			continue
		}

		storageType, basePath, columnName := valueParts[0], valueParts[1], valueParts[2]
		ckey := keyParts[0] + ":" + keyParts[1] + ":" + keyParts[2]
		sourceDataMap[storageType+"|"+basePath+"|"+columnName] = ckey
	}

	for _, feature := range request.OfflineFeatureList {
		parts := strings.Split(feature, "|")
		prefix := strings.ToLower(parts[0])
		if slices.Contains(FeatureList, parts[0]) {
			if sourceDataMap[strings.Join(parts[1:], "|")] != "" {
				onlineFeatureList[feature] = prefix + ":" + sourceDataMap[strings.Join(parts[1:], "|")]
			}
		} else if sourceDataMap[feature] != "" {
			onlineFeatureList[feature] = sourceDataMap[feature]
		}
	}
	return GetOnlineFeatureMappingResponse{
		Error: "",
		Data:  onlineFeatureList,
	}, nil
}

func (o *OnlineFeatureStore) GetCacheConfig(request GetCacheConfigRequest) (GetCacheConfigResponse, error) {
	cacheConfig := make(map[string]string)
	if request.CacheType == enums.CacheTypeDistributed {
		cacheConfig = distributedCacheConfIdToDbTypeMap
	} else if request.CacheType == enums.CacheTypeInMemory {
		cacheConfig = inMemoryCacheConfIdToDbTypeMap
	} else {
		return GetCacheConfigResponse{
			Error: fmt.Sprintf("invalid cache type: %s, valid types are: distributed, in-memory", request.CacheType),
			Data:  map[string]string{},
		}, fmt.Errorf("invalid cache type: %s", request.CacheType)
	}
	return GetCacheConfigResponse{
		Error: "",
		Data:  cacheConfig,
	}, nil
}
