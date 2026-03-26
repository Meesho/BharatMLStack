package vector

import (
	"github.com/Meesho/BharatMLStack/skye/internal/config/enums"
)

func GetRepository(vectorDbType enums.VectorDbType) Database {
	switch vectorDbType {
	case enums.QDRANT:
		return initQdrantInstance()
	case enums.NGT:
		return initNgtInstance()
	case enums.EIGENIX:
		return initEigenixInstance()
	default:
		return nil
	}
}
