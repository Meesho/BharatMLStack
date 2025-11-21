package skye

import "github.com/Meesho/go-core/grpcclient"

type ClientV1 struct {
	ClientConfigs *ClientConfig
	GrpcClient    *grpcclient.GRPCClient
}
