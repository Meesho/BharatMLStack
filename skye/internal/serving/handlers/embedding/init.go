package embedding

import "github.com/Meesho/BharatMLStack/helix-client/pkg/clients/skye/client/grpc"

func GetHandler(version int) grpc.SkyeEmbeddingServiceServer {
	switch version {
	case 1:
		return InitV1Handler()
	default:
		return nil
	}
}
