package inferflow

import (
	"github.com/Meesho/BharatMLStack/go-sdk/pkg/clients/inferflow/client/grpc"
)

// InferflowClient exposes the Predict service APIs: PointWise, PairWise, and SlateWise.
type InferflowClient interface {
	InferPointWise(request *grpc.PointWiseRequest) (*grpc.PointWiseResponse, error)
	InferPairWise(request *grpc.PairWiseRequest) (*grpc.PairWiseResponse, error)
	InferSlateWise(request *grpc.SlateWiseRequest) (*grpc.SlateWiseResponse, error)
}
