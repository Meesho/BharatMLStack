package external

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	pb "github.com/Meesho/BharatMLStack/inferflow/handlers/external/interactionstore/client/grpc"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/logger"
	"github.com/Meesho/BharatMLStack/inferflow/pkg/metrics"
	"github.com/knadh/koanf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
)

const (
	CONFIG_PREFIX           = "externalServiceIS"
	RESOLVER_DEFAULT_SCHEME = "dns"
)

var (
	isConfig     ISConfig
	connection   *grpc.ClientConn
	headers      metadata.MD
	isMetricTags = []string{"ext-service:interaction_store", "endpoint:/InteractionService/getInteractions"}
)

func InitISHandler(kConfig *koanf.Koanf) {
	err := kConfig.Unmarshal(CONFIG_PREFIX, &isConfig)
	if err != nil {
		logger.Panic("Error while interaction store config unmarshal.", err)
	}
	connection, err = initISConnection(isConfig)
	if err != nil {
		logger.Panic("Error while IS GRPC connection intialization.", err)
	}
	logger.Info("IS handler initialized")
}

func initISConnection(isConfig ISConfig) (*grpc.ClientConn, error) {

	resolver.SetDefaultScheme(RESOLVER_DEFAULT_SCHEME)
	if isConfig.PLAIN_TEXT {
		return grpc.Dial(isConfig.Host+":"+isConfig.Port,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
	} else {
		credential := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
		return grpc.Dial(isConfig.Host+":"+isConfig.Port,
			grpc.WithTransportCredentials(credential),
			grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
	}
}

func GetISResponse(interactionReqProto *pb.InteractionRequestProto, compMetricTags []string) *ISResponse {

	metricTags := append(compMetricTags, isMetricTags...)
	client := pb.NewInteractionServiceClient(connection)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(isConfig.DeadLine)*time.Millisecond)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, headers)
	t := time.Now()
	metrics.Count("inferflow.external.api.request.total", 1, metricTags)
	res, err := client.GetInteractions(ctx, interactionReqProto)
	metrics.Timing("inferflow.external.api.request.latency", time.Duration(time.Since(t)), metricTags)
	if err != nil {
		logger.Error(fmt.Sprintf("Error getting IS response for component %v and ids %v", compMetricTags, interactionReqProto.UserId), err)
		metrics.Count("inferflow.component.execution.error", 1, append(metricTags, ERROR_TYPE, IS_API_ERR))
	} else if res == nil || res.InteractionTypeToInteractions == nil {
		logger.Error(fmt.Sprintf("Found nil/Empty IS response for component %v and ids %v", compMetricTags, interactionReqProto.UserId), err)
		metrics.Count("inferflow.component.execution.error", 1, append(metricTags, ERROR_TYPE, IS_API_ERR))
	}

	return processISProtoResponse(res, metricTags)
}

func processISProtoResponse(res *pb.InteractionResponseProto, metricTags []string) *ISResponse {

	isResponse := &ISResponse{
		InteractionTypeToInteractions: make([]InteractionTypeToInteraction, 0),
	}

	for interactionType, interactionList := range res.InteractionTypeToInteractions {
		interactions := make([]Interaction, 0)

		for _, interactionProto := range interactionList.Interactions {
			interaction := Interaction{
				Id:        interactionProto.Id,
				Type:      interactionProto.Type.String(),
				TimeStamp: fmt.Sprintf("%d", interactionProto.Timestamp),
			}
			interactions = append(interactions, interaction)
		}

		isResponse.InteractionTypeToInteractions = append(isResponse.InteractionTypeToInteractions, InteractionTypeToInteraction{
			Interactions:    interactions,
			InteractionType: interactionType,
		})
	}
	return isResponse
}
