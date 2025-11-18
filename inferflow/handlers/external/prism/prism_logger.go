package external

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	"github.com/Meesho/go-core/mq/producer"
	"github.com/Meesho/go-core/mq/server"
)

const (
	PRISM_EVENT_NAME     = "inferflow_logs"
	CONFIG_PREFIX        = "externalServicePrism"
	IOP_ID               = "iop_id"
	USER_ID              = "user_id"
	MODEL_ID             = "inferflow_config_id"
	ITEMS                = "items"
	ITEMS_META           = "items_meta"
	CREATED_AT           = "created_at"
	MEESHO_COUNTRY_CODE  = "MEESHO-ISO-COUNTRY-CODE"
	IN_COUNTRY_CODE      = "IN"
	AUTH                 = "Authorization"
	CONTENT_TYPE_HEADER  = "Content-Type"
	JSON_CONTENT_TYPE    = "application/json"
	PRISM_EVENT_PUSH_API = "/api/v1/track"
	MQ_KEY               = "MQ"
)

var (
	prismConfig       PrismConfig
	client            *http.Client
	PrismEventMqId    int
	prismMetricTags   = []string{"ext-service:prism", PRISM_EVENT_PUSH_API}
	prismMqMetricTags = []string{"ext-service:prism", MQ_KEY}
)

func InitPrismHandler(configs *configs.AppConfigs) {
	err := createPrismConfig(configs, &prismConfig)
	if err != nil {
		logger.Panic("Error while creating prism configuration:", err)
	}
	client = &http.Client{
		Timeout: time.Duration(prismConfig.Timeout) * time.Millisecond,
	}
	PrismEventMqId = prismConfig.PrismEventMqId
	logger.Info("Prism handler initialized")
}

func createPrismConfig(configs *configs.AppConfigs, prismConfig *PrismConfig) error {
	if configs.Configs.ExternalServicePrism_Host == "" || configs.Configs.ExternalServicePrism_Port == "" {
		return errors.New("prism host and port are not set")
	}
	var err error
	prismConfig.Host = configs.Configs.ExternalServicePrism_Host
	prismConfig.Port = configs.Configs.ExternalServicePrism_Port
	prismConfig.PrismEventMqId, err = strconv.Atoi(configs.Configs.ExternalServicePrism_EventMqId)
	if err != nil {
		return nil
	}
	prismConfig.Timeout, err = strconv.Atoi(configs.Configs.ExternalServicePrism_Timeout)
	if err != nil {
		return nil
	}
	prismConfig.Username = configs.Configs.ExternalServicePrism_Username
	prismConfig.Password = configs.Configs.ExternalServicePrism_Password
	return nil
}

func SendPrismEvent(itemLoggingData *ItemsLoggingData) {
	pEvents, err := getPrismEventRequest(itemLoggingData)
	if err != nil {
		logger.Error("Error creating prism event request:", err)
		return
	}

	jsonBody, err := json.Marshal(pEvents)
	if err != nil {
		logger.Error("Error marshaling prism event json:", err)
		return
	}

	url := fmt.Sprintf("%s:%s%s", prismConfig.Host, prismConfig.Port, PRISM_EVENT_PUSH_API)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		logger.Error("Error creating prism HTTP request:", err)
		return
	}

	req.Header.Set(CONTENT_TYPE_HEADER, JSON_CONTENT_TYPE)
	req.Header.Set(AUTH, getBasicAuthorization(prismConfig))
	req.Header.Set(MEESHO_COUNTRY_CODE, IN_COUNTRY_CODE)

	t := time.Now()
	metrics.Count("inferflow.external.api.request.total", 1, prismMetricTags)
	resp, err := client.Do(req)
	metrics.Timing("inferflow.external.api.request.latency", time.Duration(time.Since(t)), prismMetricTags)
	if err != nil {
		logger.Error("Error sending event to prism:", err)
		metrics.Count("inferflow.component.execution.error", 1, append(prismMetricTags, ERROR_TYPE, PRISM_API_ERR))
		return
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
}

func SendPrismEventUsingMq(itemLoggingData *ItemsLoggingData, logBatchSize int) {
	if logBatchSize <= 0 {
		logBatchSize = 500
	}

	headers := createHeaders()
	totalItems := len(itemLoggingData.Items)
	if totalItems == 0 {
		return
	}

	for i := 0; i < totalItems; i += logBatchSize {
		end := i + logBatchSize
		if end > totalItems {
			end = totalItems
		}

		chunkedItems := itemLoggingData.Items[i:end]
		chunkedItemLoggingData := &ItemsLoggingData{
			IopId:             itemLoggingData.IopId,
			UserId:            itemLoggingData.UserId,
			InferflowConfigId: itemLoggingData.InferflowConfigId,
			ItemsMeta:         itemLoggingData.ItemsMeta,
			Items:             chunkedItems,
		}

		propertyMap, err := getPropertiesMap(chunkedItemLoggingData)
		if err != nil {
			logger.Error("Error while fetching properties map for chunk:", err)
			continue
		}
		err = producer.SendToPrismBulk(
			PrismEventMqId,
			[]server.PrismEvent{
				{
					Event:           PRISM_EVENT_NAME,
					Properties:      propertyMap,
					PipelineId:      0,
					PrismIngestedAt: time.Now().UnixMilli(),
				}},
			headers,
		)
		if err != nil {
			logger.Error("Error sending event to prism:", err)
			metrics.Count("inferflow.component.execution.error", 1, append(prismMqMetricTags, ERROR_TYPE, PRISM_MQ_ERR))
		}
	}
}

func createHeaders() map[string][]byte {
	headers := make(map[string][]byte)
	headers[CONTENT_TYPE_HEADER] = []byte(JSON_CONTENT_TYPE)
	headers[AUTH] = []byte(getBasicAuthorization(prismConfig))
	headers[MEESHO_COUNTRY_CODE] = []byte(IN_COUNTRY_CODE)
	return headers
}

func getPropertiesMap(itemsLoggingData *ItemsLoggingData) (map[string]any, error) {
	itemsJSON, err := json.Marshal(itemsLoggingData.Items)
	if err != nil {
		return map[string]any{}, err
	}
	itemsMetaJSON, err := json.Marshal(itemsLoggingData.ItemsMeta)
	if err != nil {
		return map[string]any{}, err
	}
	return map[string]any{
		IOP_ID:     itemsLoggingData.IopId,
		USER_ID:    itemsLoggingData.UserId,
		MODEL_ID:   itemsLoggingData.InferflowConfigId,
		ITEMS:      string(itemsJSON),
		ITEMS_META: string(itemsMetaJSON),
		CREATED_AT: time.Now().UTC(),
	}, nil
}

func getPrismEventRequest(itemsLoggingData *ItemsLoggingData) ([]PrismEvent, error) {
	prismEvents := make([]PrismEvent, 0)
	pEvent := PrismEvent{
		Event:      PRISM_EVENT_NAME,
		Properties: make(map[string]any),
	}

	itemsJSON, err := json.Marshal(itemsLoggingData.Items)
	if err != nil {
		return prismEvents, err
	}
	itemsMetaJSON, err := json.Marshal(itemsLoggingData.ItemsMeta)
	if err != nil {
		return prismEvents, err
	}

	pEvent.Properties[IOP_ID] = itemsLoggingData.IopId
	pEvent.Properties[USER_ID] = itemsLoggingData.UserId
	pEvent.Properties[MODEL_ID] = itemsLoggingData.InferflowConfigId
	pEvent.Properties[ITEMS] = string(itemsJSON)
	pEvent.Properties[ITEMS_META] = string(itemsMetaJSON)
	pEvent.Properties[CREATED_AT] = time.Now().UTC()
	prismEvents = append(prismEvents, pEvent)
	return prismEvents, nil
}

func getBasicAuthorization(prismConfig PrismConfig) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", prismConfig.Username, prismConfig.Password)))
	return fmt.Sprintf("Basic %s", encoded)
}
