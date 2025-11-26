package utils

import "reflect"

func IsNilOrEmpty(v interface{}) bool {

	if v == nil {
		return true
	}
	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		return v.(string) == ""
	case reflect.Ptr, reflect.Slice, reflect.Map:
		return reflect.ValueOf(v).Len() == 0
	}
	return false
}
