package aggregator

import (
	"github.com/Meesho/BharatMLStack/skye/internal/config"
	"github.com/Meesho/BharatMLStack/skye/internal/repositories/aggregator"
	"github.com/Meesho/BharatMLStack/skye/pkg/metric"
	"github.com/rs/zerolog/log"
)

var (
	aggregatorHandler Handler
)

type ScyllaAggregator struct {
	AggregatorDb  aggregator.Database
	configManager config.Manager
	LabelType     map[string]string
}

func initScyllaAggregator() Handler {
	if aggregatorHandler == nil {
		once.Do(func() {
			labelType := make(map[string]string)
			labelType["catalog_recommendation"] = "false"
			labelType["asp_segment"] = "false"
			labelType["catalog_listing_page"] = "false"
			labelType["for_you"] = "false"
			labelType["hidden"] = "1"
			labelType["sc"] = "NA"
			labelType["is_gst"] = "true"
			labelType["is_high_asp_enabled"] = "false"
			labelType["is_live_on_ads"] = "false"
			labelType["is_live_on_widget_ads"] = "false"
			labelType["is_melp_catalog"] = "false"
			labelType["text_search"] = "false"
			labelType["valid"] = "0"
			aggregatorHandler = &ScyllaAggregator{
				AggregatorDb:  aggregator.NewRepository(1),
				configManager: config.NewManager(config.DefaultVersion),
				LabelType:     labelType,
			}
		})
	}
	return aggregatorHandler
}

func (s *ScyllaAggregator) Process(payload Payload) (*Response, error) {
	entityConfig, err := s.configManager.GetEntityConfig(payload.Entity)
	if err != nil {
		log.Error().Msgf("Error getting entity config for entity %s: %v", payload.Entity, err)
		return nil, err
	}
	storeId := entityConfig.StoreId
	dbData, err := s.AggregatorDb.Query(storeId, &aggregator.Query{CandidateId: payload.CandidateId})
	if err != nil {
		log.Error().Msgf("Error querying aggregator db: %v", err)
		return nil, err
	}
	changedDbData := make(map[string]interface{})
	if payload.Columns != nil {
		for key, value := range payload.Columns {
			dbValue, exists := dbData[key]
			if !exists {
				metric.Incr("rt_delta_event_count_by_key_not_exists", []string{"label", key})
				changedDbData[key] = value
				continue
			}
			if dbValue == nil || dbValue == "" {
				if lt, ok := s.LabelType[key]; ok {
					dbValue = lt
				} else {
					metric.Incr("rt_delta_event_count_by_label_not_exists", []string{"label", key})
					changedDbData[key] = value
					continue
				}
			}
			if dbValue != value {
				metric.Incr("rt_delta_event_count_by_key", []string{"label", key})
				changedDbData[key] = value
			}
		}
	}
	if len(changedDbData) > 0 {
		err := s.AggregatorDb.Persist(storeId, payload.CandidateId, changedDbData)
		if err != nil {
			return nil, err
		}
	}
	return &Response{
		CompleteData: dbData,
		DeltaData:    changedDbData,
	}, nil
}
