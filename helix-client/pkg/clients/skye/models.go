package skye

import "github.com/Meesho/BharatMLStack/helix-client/pkg/grpcclient"

type ClientV1 struct {
	ClientConfigs *ClientConfig
	GrpcClient    *grpcclient.GRPCClient
}
