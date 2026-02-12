package embedding

import "github.com/Meesho/helix-clients/pkg/deployableclients/skye/client/grpc"

func GetHandler(version int) grpc.SkyeEmbeddingServiceServer {
	switch version {
	case 1:
		return InitV1Handler()
	default:
		return nil
	}
}
