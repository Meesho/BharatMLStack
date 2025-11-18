package realtimepricingfeaturestore

//
//import (
//	"fmt"
//	"github.com/Meesho/BharatMLStack/inferflow/handlers/config"
//	"github.com/Meesho/BharatMLStack/inferflow/pkg/utils"
//	pricingFeatureClient "github.com/Meesho/price-aggregator-go/pricingfeatureretrieval/client/models"
//
//	"strings"
//)
//
//const (
//	CACHE_KEY_SEPARATER = "|"
//)
//
//func GetCacheIdToFsEntityIdMap(m []map[interface{}][]string) map[string]pricingFeatureClient.EntityId {
//
//	entityIdsMap := make(map[string]pricingFeatureClient.EntityId)
//	if len(m) == 1 && !utils.IsNilOrEmpty(m[0]) {
//		for iFsIdConfig, values := range m[0] {
//			if !utils.IsNilOrEmpty(iFsIdConfig) && len(values) > 0 {
//				fsIdConfig := iFsIdConfig.(config.FsId)
//				for _, value := range values {
//					if !utils.IsNilOrEmpty(value) {
//						entityId := &pricingFeatureClient.EntityId{}
//						entityKey := &pricingFeatureClient.EntityKey{
//							Type:  fsIdConfig.Type,
//							Value: fmt.Sprintf(fsIdConfig.IdRegex, value),
//						}
//						entityId.EntityKeys = append(entityId.EntityKeys, *entityKey)
//						entityIdsMap[entityKey.Value] = *entityId
//					}
//				}
//			}
//		}
//	} else if len(m) == 2 && !utils.IsNilOrEmpty(m[0]) && !utils.IsNilOrEmpty(m[1]) {
//		for iFsIdConfig1, values1 := range m[0] {
//			for iFsIdConfig2, values2 := range m[1] {
//				if !utils.IsNilOrEmpty(iFsIdConfig1) && !utils.IsNilOrEmpty(iFsIdConfig2) && len(values1) > 0 && len(values2) > 0 {
//					for i := 0; i < len(values1) && i < len(values2); i++ {
//						value1 := values1[i]
//						value2 := values2[i]
//						if !utils.IsNilOrEmpty(value1) && !utils.IsNilOrEmpty(value2) {
//							entityId := &pricingFeatureClient.EntityId{}
//							entityKey1 := &pricingFeatureClient.EntityKey{
//								Type:  iFsIdConfig1.(config.FsId).Type,
//								Value: fmt.Sprintf(iFsIdConfig1.(config.FsId).IdRegex, value1),
//							}
//							entityKey2 := &pricingFeatureClient.EntityKey{
//								Type:  iFsIdConfig2.(config.FsId).Type,
//								Value: fmt.Sprintf(iFsIdConfig2.(config.FsId).IdRegex, value2),
//							}
//							entityId.EntityKeys = append(entityId.EntityKeys, *entityKey1, *entityKey2)
//							entityIdsMap[entityKey1.Value+CACHE_KEY_SEPARATER+entityKey2.Value] = *entityId
//						}
//					}
//				}
//			}
//		}
//	}
//	return entityIdsMap
//}
//
//func getLastElement(str, separator string) string {
//	sl := strings.Split(str, separator)
//	return sl[len(sl)-1]
//}
