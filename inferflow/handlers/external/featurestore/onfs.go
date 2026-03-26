package featurestore

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/external/featurestore/proto"
	"github.com/Meesho/BharatMLStack/inferflow/handlers/models"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/configs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"

	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
)

var (
	fsConfig     FSConfig
	client       proto.FeatureServiceClient
	headers      metadata.MD
	connection   *grpc.ClientConn
	fsMetricTags = []string{"ext-service:featurestore", "endpoint:/FeatureService/retrieveFeatures"}
)

const (
	CALLER_ID               = "ONLINE-FEATURE-STORE-CALLER-ID"
	CALLER_TOKEN            = "ONLINE-FEATURE-STORE-AUTH-TOKEN"
	RESOLVER_DEFAULT_SCHEME = "dns"
)

func initFSConnection(fsConfig FSConfig) (*grpc.ClientConn, error) {
	resolver.SetDefaultScheme(RESOLVER_DEFAULT_SCHEME)
	var gConn *grpc.ClientConn
	var err error
	target := fsConfig.Host + ":" + fsConfig.Port
	if fsConfig.PLAIN_TEXT {
		gConn, err = grpc.NewClient(
			target,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
		)
	} else {
		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
		gConn, err = grpc.NewClient(
			target,
			grpc.WithTransportCredentials(creds),
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
		)
	}
	if err != nil {
		return nil, err
	}
	return gConn, nil
}

func InitFSHandler(configs *configs.AppConfigs) {
	err := createFsConfig(configs, &fsConfig)
	if err != nil {
		logger.Panic("Error while featurestore config creation.", err)
	}
	connection, err = initFSConnection(fsConfig)
	if err != nil {
		logger.Panic("Error while FS GRPC connection initialization.", err)
	}

	// Now that connection is valid, create the client here:
	client = proto.NewFeatureServiceClient(connection)

	headers = metadata.New(map[string]string{
		CALLER_ID:    fsConfig.CallerId,
		CALLER_TOKEN: fsConfig.CallerToken,
	})

	logger.Info("FS handler initialized")
}

type batchResult struct {
	batchIdx      int
	globalIndexes []int
	resultColumns *proto.Result
	err           error
}

func createFsConfig(configs *configs.AppConfigs, fsConfig *FSConfig) error {
	if configs.Configs.ExternalServiceOnFs_Host == "" || configs.Configs.ExternalServiceOnFs_Port == "" {
		return errors.New("featurestore host and port are not set")
	}
	fsConfig.Host = configs.Configs.ExternalServiceOnFs_Host
	fsConfig.Port = configs.Configs.ExternalServiceOnFs_Port
	fsConfig.PLAIN_TEXT = configs.Configs.ExternalServiceOnFs_GrpcPlainText
	fsConfig.CallerId = configs.Configs.ExternalServiceOnFs_CallerID
	fsConfig.CallerToken = configs.Configs.ExternalServiceOnFs_CallerToken
	fsConfig.DeadLine = configs.Configs.ExternalServiceOnFs_DeadLine
	fsConfig.BatchSize = configs.Configs.ExternalServiceOnFs_BatchSize
	return nil
}

func GetOnFsResponse(config config.FeatureComponentConfig, featureComponentBuilder *models.FeatureComponentBuilder, errLoggingPercent int, compMetricTags []string) {
	batchCount := (featureComponentBuilder.InMemoryMissCount + fsConfig.BatchSize - 1) / fsConfig.BatchSize
	batches, batchGlobalIndexes := buildQueryBatches(batchCount, config, featureComponentBuilder)

	metricTags := append(compMetricTags, fsMetricTags...)
	metrics.Count("inferflow.external.api.request.total", int64(batchCount), metricTags)
	metrics.Count("inferflow.external.api.request.batch", int64(featureComponentBuilder.InMemoryMissCount), metricTags)
	startTime := time.Now()
	resultCh := make(chan batchResult, batchCount)
	var wg sync.WaitGroup

	for batchIdx := range batches {
		wg.Add(1)
		go func(batchIdx int, q *proto.Query, gIndexes []int) {
			defer wg.Done()
			executeBatch(batchIdx, q, gIndexes, resultCh, metricTags, errLoggingPercent)
		}(batchIdx, &batches[batchIdx], batchGlobalIndexes[batchIdx])
	}

	// Close resultCh only after all goroutines finish
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	for res := range resultCh {
		if res.err != nil {
			continue
		}
		for ridx, gidx := range res.globalIndexes {
			featureComponentBuilder.Result[gidx] = res.resultColumns.Rows[ridx].Columns
		}
	}

	metrics.Timing("inferflow.external.api.request.latency", time.Since(startTime), metricTags)

}

// executeBatch performs the FS request and sends the result to resultCh.
func executeBatch(
	batchIdx int,
	query *proto.Query,
	globalIndexes []int,
	resultCh chan<- batchResult,
	metricTags []string,
	errLoggingPercent int,
) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(fsConfig.DeadLine)*time.Millisecond)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, headers)

	resp, err := client.RetrieveFeatures(ctx, query)
	if err != nil {
		handleBatchError(batchIdx, err, metricTags, errLoggingPercent)
		resultCh <- batchResult{batchIdx: batchIdx, err: err}
		return
	}

	resultCh <- batchResult{
		batchIdx:      batchIdx,
		globalIndexes: globalIndexes,
		resultColumns: resp,
		err:           nil,
	}
}

func buildQueryBatches(
	batchCount int,
	config config.FeatureComponentConfig,
	featureComponentBuilder *models.FeatureComponentBuilder,
) ([]proto.Query, [][]int) {
	entityLabel := config.FSRequest.Label
	keySchema := featureComponentBuilder.KeySchema
	featureGroups := buildFeatureGroups(config.FSRequest.FeatureGroups)
	batches := make([]proto.Query, batchCount)
	batchGlobalIndexes := make([][]int, len(batches))
	keys := make([]*proto.Keys, featureComponentBuilder.InMemoryMissCount)
	batchGlobalIndexesBuffer := make([]int, featureComponentBuilder.InMemoryMissCount)

	for i := 0; i < batchCount; i++ {
		start := i * fsConfig.BatchSize
		end := start + fsConfig.BatchSize
		if end > featureComponentBuilder.InMemoryMissCount {
			end = featureComponentBuilder.InMemoryMissCount
		}
		batchGlobalIndexes[i] = batchGlobalIndexesBuffer[start:end]
		batches[i] = proto.Query{
			EntityLabel:   entityLabel,
			KeysSchema:    keySchema,
			FeatureGroups: featureGroups,
			Keys:          keys[start:end],
		}
	}

	idx := 0
	for j, row := range featureComponentBuilder.UniqueEntityIds {
		if !featureComponentBuilder.InMemPresent[j] {
			batchGlobalIndexes[idx/fsConfig.BatchSize][idx%fsConfig.BatchSize] = j
			batches[idx/fsConfig.BatchSize].Keys[idx%fsConfig.BatchSize] = &proto.Keys{Cols: featureComponentBuilder.EntitiesToKeyValuesMap[row]}
			idx++
		}
	}
	return batches, batchGlobalIndexes
}

func buildFeatureGroups(fgs []config.FeatureGroup) []*proto.FeatureGroup {
	result := make([]*proto.FeatureGroup, len(fgs))
	for i, fg := range fgs {
		result[i] = &proto.FeatureGroup{
			Label:         fg.Label,
			FeatureLabels: fg.Features,
		}
	}
	return result
}

func handleBatchError(batchIdx int, err error, metricTags []string, errLoggingPercent int) {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		logger.PercentError(fmt.Sprintf("Timeout for batch %d", batchIdx), err, errLoggingPercent)
		metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, onFsApiReqTimeOut))
	case errors.Is(err, context.Canceled):
		logger.PercentError(fmt.Sprintf("Cancelled for batch %d", batchIdx), err, errLoggingPercent)
		metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, onFsApiReqCancelled))
	default:
		logger.PercentError(fmt.Sprintf("Error for batch %d", batchIdx), err, errLoggingPercent)
		metrics.Count("inferflow.component.execution.error", 1, append(metricTags, errorType, onFsApiError))
	}
}
