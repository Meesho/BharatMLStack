package inferflow

import "github.com/Meesho/BharatMLStack/go-sdk/pkg/grpcclient"

type ClientV1 struct {
	ClientConfigs *ClientConfig
	GrpcClient    *grpcclient.GRPCClient
}
