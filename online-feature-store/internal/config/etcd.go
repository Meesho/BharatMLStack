package config

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/circuitbreaker"
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/etcd"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type Etcd struct {
	instance etcd.Etcd
	appName  string
	env      string
}

var entities *NormalizedEntities
var registeredClients map[string]string

func NewEtcdConfig() Manager {
	return &Etcd{
		instance: etcd.Instance(),
		appName:  viper.GetString("APP_NAME"),
		env:      viper.GetString("APP_ENV"),
	}
}

func (e *Etcd) GetEtcdInstance() *FeatureRegistry {
	instance, ok := e.instance.GetConfigInstance().(*FeatureRegistry)
	if !ok {
		log.Panic().Msg("invalid etcd instance")
	}
	return instance
}

// GetEntity returns a pointer for a particular Entity
func (e *Etcd) GetEntity(entityLabel string) (*Entity, error) {
	featureRegistry := e.GetEtcdInstance()
	entity, ok := featureRegistry.Entities[entityLabel]
	if !ok {
		return nil, fmt.Errorf("entity %s not found", entityLabel)
	}
	return &entity, nil
}

// GetAllEntities returns all entities from the configuration
func (e *Etcd) GetAllEntities() []Entity {
	featureRegistry := e.GetEtcdInstance()
	entities := featureRegistry.Entities
	entityList := make([]Entity, 0)
	if len(entities) == 0 {
		return entityList
	}
	for _, entity := range entities {
		if !isEntityValid(&entity) {
			continue
		}
		entityList = append(entityList, entity)
	}
	return entityList
}

// GetFeatureGroup returns a pointer for a particular FeatureSchema Group
func (e *Etcd) GetFeatureGroup(entityLabel, fgLabel string) (*FeatureGroup, error) {
	entity, err := e.GetEntity(entityLabel)
	if err != nil {
		return nil, err
	}
	fg, ok := entity.FeatureGroups[fgLabel]
	if !ok {
		return nil, fmt.Errorf("feature group %s not found", fgLabel)
	}
	return &fg, nil
}

// GetActiveFeatureSchema returns a pointer for a particular FeatureSchema Schema
func (e *Etcd) GetActiveFeatureSchema(entityLabel, fgLabel string) (*FeatureSchema, error) {
	fg, err := e.GetFeatureGroup(entityLabel, fgLabel)
	if err != nil {
		return nil, err
	}
	activeVersion := fg.ActiveVersion
	feature, ok := fg.Features[activeVersion]
	if !ok {
		return nil, fmt.Errorf("feature schema for version: %s not found", activeVersion)
	}
	return &feature, nil
}

// GetStore returns a particular store config
func (e *Etcd) GetStore(storeId string) (*Store, error) {
	stores := e.GetEtcdInstance().Storage.Stores
	store, ok := stores[storeId]
	if !ok {
		return nil, fmt.Errorf("store id %s not found", storeId)
	}
	return &store, nil
}

func (e *Etcd) GetFeatureGroupColumn(entityLabel, fgLabel, columnLabel string) (*Column, error) {
	fg, err := e.GetFeatureGroup(entityLabel, fgLabel)
	if err != nil {
		return nil, err
	}
	column, ok := fg.Columns[columnLabel]
	if !ok {
		return nil, fmt.Errorf("column %s not found", columnLabel)
	}
	return &column, nil
}

func (e *Etcd) GetDistributedCacheConfForEntity(entityLabel string) (*Cache, error) {
	entity, err := e.GetEntity(entityLabel)
	if err != nil {
		return nil, err
	}
	return &entity.DistributedCache, nil
}

func (e *Etcd) GetInMemoryCacheConfForEntity(entityLabel string) (*Cache, error) {
	entity, err := e.GetEntity(entityLabel)
	if err != nil {
		return nil, err
	}
	return &entity.InMemoryCache, nil
}

func (e *Etcd) GetP2PCacheConfForEntity(entityLabel string) (*Cache, error) {
	entity, err := e.GetEntity(entityLabel)
	if err != nil {
		return nil, err
	}
	// TODO: use inmemory config until scaled up
	return &entity.InMemoryCache, nil
}

func (e *Etcd) GetP2PEnabledPercentage() int {
	return e.GetEtcdInstance().P2PEnabledPercentage
}

func (e *Etcd) GetStores() (*map[string]Store, error) {
	instance := e.GetEtcdInstance()
	return &instance.Storage.Stores, nil
}

// GetColumnsForEntityAndFG returns all columns for a given entity and feature group
func (e *Etcd) GetColumnsForEntityAndFG(entityLabel string, fgId int) ([]string, error) {
	entity, ok := entities.Entities[entityLabel]
	if !ok {
		return nil, fmt.Errorf("entity %s not found", entityLabel)
	}
	fg, ok := entity.FGs[fgId]
	if !ok {
		return nil, fmt.Errorf("feature group %d not found", fgId)
	}
	return fg.Columns, nil
}

func (e *Etcd) RegisterClients() error {
	instance := e.GetEtcdInstance()
	reader := instance.Security.Reader
	clients := make(map[string]string)
	for client, property := range reader {
		clients[client] = property.Token
	}
	registeredClients = clients
	return nil
}

func (e *Etcd) GetAllRegisteredClients() map[string]string {
	return registeredClients
}

// GetPKMapAndPKColumnsForEntity returns the primary key and primary key columns for a given entity
func (e *Etcd) GetPKMapAndPKColumnsForEntity(entityLabel string) (map[string]string, []string, error) {
	entity, ok := entities.Entities[entityLabel]
	if !ok {
		return nil, nil, fmt.Errorf("entity %s not found", entityLabel)
	}

	// Convert map[_pkId]string to map[string]string
	pkMap := make(map[string]string)
	for k, v := range entity.ColumnToPKIdMap {
		pkMap[k] = v
	}

	return pkMap, entity.PrimaryKeyColumnNames, nil
}

// GetFGDefaultValuesInBytesForVersion returns the default values in bytes for a given feature group and version
func (e *Etcd) GetFGDefaultValuesInBytesForVersion(entityLabel string, fgId int, version int) (map[string][]byte, error) {
	entity, ok := entities.Entities[entityLabel]
	if !ok {
		return nil, fmt.Errorf("entity %s not found", entityLabel)
	}
	fg, ok := entity.FGs[fgId]
	if !ok {
		return nil, fmt.Errorf("feature group %d not found", fgId)
	}
	defaults, ok := fg.DefaultsInBytes[_version(version)]
	if !ok {
		return nil, fmt.Errorf("version %d not found", version)
	}
	return defaults, nil
}

// GetEntities retrieves all entities from the configuration.
// Returns a map of entity names to their models or an error if entities are not found.
func (e *Etcd) GetEntities() (map[string]Entity, error) {
	featureRegistry := e.GetEtcdInstance()
	entities := featureRegistry.Entities
	if entities == nil {
		return nil, errors.New("entities not found in configuration")
	}
	return entities, nil
}

func (e *Etcd) GetMaxColumnSize(storeId string) (int, error) {
	store, err := e.GetStore(storeId)
	if err != nil {
		return 0, err
	}
	return store.MaxColumnSizeInBytes, nil
}

// GetDefaultValueByte returns the default value in bytes for a given feature
func (e *Etcd) GetDefaultValueByte(entityLabel string, fgId int, version int, featureLabel string) ([]byte, error) {
	entity, ok := entities.Entities[entityLabel]
	if !ok {
		return nil, fmt.Errorf("entity %s not found", entityLabel)
	}

	fg, ok := entity.FGs[fgId]
	if !ok {
		return nil, fmt.Errorf("feature group %d not found", fgId)
	}

	defaults, ok := fg.DefaultsInBytes[_version(version)]
	if !ok {
		return nil, fmt.Errorf("version %d not found", version)
	}

	defaultValue, ok := defaults[featureLabel]
	if !ok {
		return nil, fmt.Errorf("feature %s not found", featureLabel)
	}

	return defaultValue, nil
}

// GetSequenceNo returns the sequence number for a given feature
func (e *Etcd) GetSequenceNo(entityLabel string, fgId int, version int, featureLabel string) (int, error) {
	entity, ok := entities.Entities[entityLabel]
	if !ok {
		return 0, fmt.Errorf("entity %s not found", entityLabel)
	}

	fg, ok := entity.FGs[fgId]
	if !ok {
		return 0, fmt.Errorf("feature group %d not found for entity %s", fgId, entityLabel)
	}

	sequences, ok := fg.Sequences[_version(version)]
	if !ok {
		return 0, fmt.Errorf("version %d not found for entity %s, fg %d", version, entityLabel, fgId)
	}

	sequence, ok := sequences[featureLabel]
	if !ok {
		log.Warn().Msgf("feature %s not found for entity %s, fg %d, version %d", featureLabel, entityLabel, fgId, version)
		return -1, nil
	}

	return sequence, nil
}

// GetVectorLengths returns the vector lengths for a given feature
func (e *Etcd) GetVectorLengths(entityLabel string, fgId int, version int) ([]uint16, error) {
	entity, ok := entities.Entities[entityLabel]
	if !ok {
		return nil, fmt.Errorf("entity %s not found", entityLabel)
	}

	fg, ok := entity.FGs[fgId]
	if !ok {
		return nil, fmt.Errorf("feature group %d not found", fgId)
	}

	vectorLengths, ok := fg.VectorLengths[_version(version)]
	if !ok {
		return nil, fmt.Errorf("version %d not found", version)
	}

	return vectorLengths, nil
}

// GetStringLengths returns the string lengths for a given feature
func (e *Etcd) GetStringLengths(entityLabel string, fgId int, version int) ([]uint16, error) {
	entity, ok := entities.Entities[entityLabel]
	if !ok {
		return nil, fmt.Errorf("entity %s not found", entityLabel)
	}

	fg, ok := entity.FGs[fgId]
	if !ok {
		return nil, fmt.Errorf("feature group %d not found", fgId)
	}

	stringLengths, ok := fg.StringLengths[_version(version)]
	if !ok {
		return nil, fmt.Errorf("version %d not found", version)
	}

	return stringLengths, nil
}

// GetNumOfFeatures returns the number of features for a given feature group and version
func (e *Etcd) GetNumOfFeatures(entityLabel string, fgId int, version int) (int, error) {
	entity, ok := entities.Entities[entityLabel]
	if !ok {
		return 0, fmt.Errorf("entity %s not found", entityLabel)
	}

	fg, ok := entity.FGs[fgId]
	if !ok {
		return 0, fmt.Errorf("feature group %d not found", fgId)
	}

	numOfFeatures, ok := fg.NumofFeatures[_version(version)]
	if !ok {
		return 0, fmt.Errorf("version %d not found", version)
	}

	return numOfFeatures, nil
}

func (e *Etcd) GetActiveVersion(entityLabel string, fgId int) (int, error) {
	entity, ok := entities.Entities[entityLabel]
	if !ok {
		return 0, fmt.Errorf("entity %s not found", entityLabel)
	}
	fg, ok := entity.FGs[fgId]
	if !ok {
		return 0, fmt.Errorf("feature group %d not found", fgId)
	}
	return int(fg.ActiveVersion), nil
}

// TODO : register on initialize bootup and also on watcher path
func (e *Etcd) parseToNormalizedEntities(entities map[string]Entity) (*NormalizedEntities, error) {
	normalized := &NormalizedEntities{
		Entities: make(map[string]NormalizedEntity),
	}
	for entityLabel, entity := range entities {
		if !isEntityValid(&entity) {
			continue
		}
		// Parse primary key columns
		pkColumns := make([]string, len(entity.Keys))
		pkIdToColumn := make(map[string]string)
		// idx := 0
		for _, key := range entity.Keys {
			pkColumns[key.Sequence] = key.ColumnLabel
			// idx++
			pkIdToColumn[key.ColumnLabel] = key.EntityLabel

		}
		// sort.Strings(pkColumns)

		// Parse feature groups
		fgs := make(map[int]NormalizedFGConfig)
		for _, fg := range entity.FeatureGroups {
			// Create version mapping
			versions := make(map[_version][]string)
			defaultsInBytes := make(map[_version]map[string][]byte)
			versionSequences := make(map[_version]map[string]int)
			versionStringLengths := make(map[_version][]uint16)
			versionVectorLengths := make(map[_version][]uint16)
			numOfFeatures := make(map[_version]int)
			// Get all columns
			columns := make([]string, 0)
			for _, col := range fg.Columns {
				columns = append(columns, col.Label)
			}
			sort.Strings(columns)

			// Parse each version's features
			for version, schema := range fg.Features {
				versionInt, err := strconv.Atoi(version)
				if err != nil {
					return nil, fmt.Errorf("invalid version number: %s", version)
				}

				// Get feature labels for this version
				featureLabels := strings.Split(schema.Labels, ",")

				// Get default values and sequences for this version
				defaults := make(map[string][]byte)
				sequencesForVersion := make(map[string]int)
				sequenceToStringLengthMap := make(map[int]uint16)
				sequenceToVectorLengthMap := make(map[int]uint16)
				stringLengthsForVersion := make([]uint16, 0)
				vectorLengthsForVersion := make([]uint16, 0)
				for label, meta := range schema.FeatureMeta {
					defaults[label] = meta.DefaultValuesInBytes
					sequencesForVersion[label] = meta.Sequence
					sequenceToStringLengthMap[meta.Sequence] = meta.StringLength
					sequenceToVectorLengthMap[meta.Sequence] = meta.VectorLength
				}
				for seq := 0; seq < len(sequenceToStringLengthMap); seq++ {
					if length, ok := sequenceToStringLengthMap[seq]; ok {
						stringLengthsForVersion = append(stringLengthsForVersion, length)
					}
				}
				for seq := 0; seq < len(sequenceToVectorLengthMap); seq++ {
					if length, ok := sequenceToVectorLengthMap[seq]; ok {
						vectorLengthsForVersion = append(vectorLengthsForVersion, length)
					}
				}

				//Check if feature labels have default values
				featureLabelsWithDefaults := make([]string, 0)
				for _, label := range featureLabels {
					if _, ok := defaults[label]; !ok {
						log.Error().Msgf("default value not found for feature label: %s, fg: %d, entity: %s", label, fg.Id, entityLabel)
						continue
					}
					featureLabelsWithDefaults = append(featureLabelsWithDefaults, label)
				}
				versions[_version(versionInt)] = featureLabelsWithDefaults
				defaultsInBytes[_version(versionInt)] = defaults
				versionSequences[_version(versionInt)] = sequencesForVersion
				versionStringLengths[_version(versionInt)] = stringLengthsForVersion
				versionVectorLengths[_version(versionInt)] = vectorLengthsForVersion
				numOfFeatures[_version(versionInt)] = len(featureLabelsWithDefaults)
			}
			activeVersion, err := strconv.Atoi(fg.ActiveVersion)
			if err != nil {
				return nil, fmt.Errorf("invalid active version for entity: %s, fg: %d, active_version: %s", entityLabel, fg.Id, fg.ActiveVersion)
			}

			fgs[fg.Id] = NormalizedFGConfig{
				Versions:        versions,
				DefaultsInBytes: defaultsInBytes,
				Sequences:       versionSequences,
				Columns:         columns,
				ActiveVersion:   _version(activeVersion),
				StringLengths:   versionStringLengths,
				VectorLengths:   versionVectorLengths,
				NumofFeatures:   numOfFeatures,
			}
		}

		normalized.Entities[entityLabel] = NormalizedEntity{
			PrimaryKeyColumnNames: pkColumns,
			ColumnToPKIdMap:       pkIdToColumn,
			FGs:                   fgs,
		}
	}

	return normalized, nil
}

func (e *Etcd) GetNormalizedEntities() error {
	log.Debug().Msg("Getting normalized entities")
	allEntities, err := e.GetEntities()
	if err != nil {
		log.Error().Msgf("Error getting entities: %s", err)
		return err
	}
	tmpEntities, err := e.parseToNormalizedEntities(allEntities)
	log.Debug().Msgf("Normalized entities: %v", entities)
	if err != nil {
		log.Error().Msgf("Error parsing normalized entities: %s and entities : %v", err, tmpEntities)
		return err
	}
	entities = tmpEntities
	log.Debug().Msg("Normalized entities parsed")
	return nil
}

func (e *Etcd) GetAllFGIdsForEntity(entityLabel string) (map[int]bool, error) {
	entity, err := e.GetEntity(entityLabel)
	if err != nil {
		return nil, err
	}

	allFGIds := make(map[int]bool)
	for _, fg := range entity.FeatureGroups {
		allFGIds[fg.Id] = true
	}

	return allFGIds, nil
}

func isEntityValid(entity *Entity) bool {
	if entity == nil || entity.Label == "" || entity.Keys == nil || len(entity.Keys) == 0 {
		return false
	}
	return true
}

func (e *Etcd) GetCircuitBreakerConfigs() map[string]circuitbreaker.Config {
	instance := e.GetEtcdInstance()
	return instance.CircuitBreaker
}

func (e *Etcd) UpdateCBConfigs() error {
	cbConfigs := e.GetCircuitBreakerConfigs()
	for cbManagerName, cbConfig := range cbConfigs {
		activeCBs := make([]string, 0)
		inactiveCBs := make([]string, 0)
		for cbName, activeCBConfig := range cbConfig.ActiveCBKeys {
			if activeCBConfig.Enabled {
				activeCBs = append(activeCBs, cbName)
			} else {
				inactiveCBs = append(inactiveCBs, cbName)
			}
		}
		cbManager := circuitbreaker.GetManager(cbManagerName)
		cbManager.ActivateCBKey(activeCBs)
		cbManager.DeactivateCBKey(inactiveCBs)
		cbManager.UpdateCBConfig(cbConfig)

		// Handle force open/close based on etcd config
		for cbName, activeCBConfig := range cbConfig.ActiveCBKeys {
			if activeCBConfig.Enabled {
				switch activeCBConfig.ForcedState {
				case 1:
					cbManager.ForceOpenCB(cbName)
				case -1:
					cbManager.ForceCloseCB(cbName)
				case 0:
					cbManager.NormalExecutionModeCB(cbName)
				default:
					log.Error().Msgf("invalid forced state for circuit breaker %s. switching to normal execution mode", cbName)
					cbManager.NormalExecutionModeCB(cbName)
				}
			}
		}
	}
	return nil
}
