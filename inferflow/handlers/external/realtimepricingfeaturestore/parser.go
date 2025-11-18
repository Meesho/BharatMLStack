package realtimepricingfeaturestore

//
//import (
//	"github.com/Meesho/BharatMLStack/inferflow/pkg/utils"
//	"github.com/Meesho/price-aggregator-go/pricingfeatureretrieval/client/models"
//	"strings"
//)
//
//func FlattenResponseMap(fsResponse *models.EntityPayloads, flattenResKeys []string) map[string]map[string]string {
//
//	result := make(map[string]map[string]string)
//	if !utils.IsNilOrEmpty(fsResponse.Data) {
//		flattenResKeyIndexes := getFlattenResKeyIndexes(fsResponse.Data[0].Features, flattenResKeys)
//		if !utils.IsNilOrEmpty(flattenResKeyIndexes) {
//			for i, featureName := range fsResponse.Data[0].Features {
//				if i < fsResponse.KeySize {
//					continue // skip "entity_ids"
//				}
//				featureMap := make(map[string]string)
//				for _, data := range fsResponse.Data[1:] {
//					if !utils.IsNilOrEmpty(data) {
//						entityID := GetEntityIdForFsMap(data.Features, flattenResKeyIndexes)
//						featureValue := data.Features[i]
//						featureMap[entityID] = featureValue
//					}
//				}
//				result[fsResponse.EntityLabel+":"+featureName] = featureMap
//			}
//		}
//	}
//	return result
//}
//
//func getFlattenResKeyIndexes(features []string, flattenResKeys []string) []int {
//
//	indexes := make([]int, 0)
//	for _, key := range flattenResKeys {
//		if i := utils.FindIndex(features, key); i != -1 {
//			indexes = append(indexes, i)
//		}
//	}
//	return indexes
//}
//
//func GetEntityIdForFsMap(features []string, idIndexes []int) string {
//	var fsEntityIds []string
//
//	for _, index := range idIndexes {
//		if index >= 0 && index < len(features) {
//			entityId := features[index]
//			if !utils.IsNilOrEmpty(entityId) {
//				fsEntityIds = append(fsEntityIds, getLastElement(entityId, "::"))
//			}
//		}
//	}
//
//	return strings.Join(fsEntityIds, CACHE_KEY_SEPARATER)
//}
