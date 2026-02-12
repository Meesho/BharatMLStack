package utils

import (
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/spaolacci/murmur3"
)

func IsNilOrEmpty(v interface{}) bool {

	if v == nil {
		return true
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		return v.(string) == ""
	case reflect.Ptr:
		return reflect.ValueOf(v).IsNil() || IsNilOrEmpty(reflect.Indirect(reflect.ValueOf(v)).Interface())
	case reflect.Slice, reflect.Map:
		return reflect.ValueOf(v).Len() == 0
	}
	return false
}

func PartitionSliceOfSlice(ids [][]string, batchSize int) [][][]string {

	var idLists [][][]string
	n := len(ids)
	for i := 0; i < n; i += batchSize {
		end := i + batchSize
		if end > n {
			end = n
		}
		idLists = append(idLists, ids[i:end])
	}
	return idLists
}

func FindIndex(s []string, target string) int {

	for i := 0; i < len(s); i++ {
		if s[i] == target {
			return i
		}
	}
	return -1 // target not found
}

func IsZeroValue(value string) bool {
	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return false
	}
	return num == 0
}

// func MergeMaps(maps ...map[string]map[string][]byte) map[string]map[string][]byte {
// 	finalMap := make(map[string]map[string][]byte)
// 	for _, m := range maps {
// 		for feature, featureValue := range m {
// 			if _, ok := finalMap[feature]; !ok {
// 				finalMap[feature] = make(map[string][]byte)
// 			}
// 			for key, value := range featureValue {
// 				finalMap[feature][key] = value
// 			}
// 		}
// 	}
// 	return finalMap
// }

func IsEnableForUserForToday(userId string, percentageUsers int) bool {

	// MM-DD-YYYY
	currentDate := time.Now().Format("02-01-2006")
	hashValue := GetMurMurHash(userId + strings.ReplaceAll(currentDate, "-", ""))
	return (int(math.Abs(float64(hashValue))))%100 < percentageUsers
}

// IsV2LoggingEnableForUserForToday uses reversed hash input (date + userId)
// to provide independent sampling from IsEnableForUserForToday.
func IsV2LoggingEnableForUserForToday(userId string, percentageUsers int) bool {
	currentDate := time.Now().Format("02-01-2006")
	hashValue := GetMurMurHash(strings.ReplaceAll(currentDate, "-", "") + userId)
	return (int(math.Abs(float64(hashValue))))%100 < percentageUsers
}

func GetMurMurHash(key string) int32 {
	h := murmur3.New32()
	h.Write([]byte(key))
	return int32(h.Sum32() - math.MaxUint32 - 1)
}

func ConvertToString(bytes []byte, dataType string) string {
	return ""
}
