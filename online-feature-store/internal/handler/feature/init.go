package feature

import (
	"github.com/Meesho/BharatMLStack/online-feature-store/pkg/proto/retrieve"
)

func GetHandler(version int) retrieve.FeatureServiceServer {
	switch version {
	case 1:
		return InitV1()
	default:
		return nil
	}
}
