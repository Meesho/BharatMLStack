package featurestore

const (
	CacheKeySeparator = "|"
)

func isEmptyRow(row []string) bool {
	if row == nil || len(row) == 0 {
		return true
	}
	for _, v := range row {
		if v != "" {
			return false
		}
	}
	return true
}

// TODO -3 , no need of this, directly traverse the idMap matric while calling cache and jsut get key

// GeKey (){
// if isEmptyRow(row) {
// continue
// }
//
// nonEmpty := true
// for _, val := range row {
// if val == "" {
// nonEmpty = false
// break
// }
// }
// if !nonEmpty {
// continue
// }
//
// cacheKey := strings.Join(row, CacheKeySeparator)
// return cacheKEy
// }

//

//func GetCacheIdToFSKeysMap(values *matrix.ColumnValues) map[string]*onfs.Keys {
//	cacheMap := make(map[string]*onfs.Keys)
//
//	for _, row := range values.Values {
//		if isEmptyRow(row) {
//			continue
//		}
//
//		nonEmpty := true
//		for _, val := range row {
//			if val == "" {
//				nonEmpty = false
//				break
//			}
//		}
//		if !nonEmpty {
//			continue
//		}
//
//		cacheKey := strings.Join(row, CacheKeySeparator)
//		cacheMap[cacheKey] = &onfs.Keys{Cols: row}
//	}
//
//	return cacheMap
//}
