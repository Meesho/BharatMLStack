package utils

func IsEmptyString(in string) bool {
	return in == ""
}

func IsNilPointer[T any](ptr *T) bool {
	return ptr == nil
}

func IsEmptyMap[K comparable, V any](m map[K]V) bool {
	return len(m) == 0
}

func IsEmptySlice[V any](s []V) bool {
	return len(s) == 0
}
