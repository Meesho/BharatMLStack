package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsEmptyString(t *testing.T) {
	var s string
	assert.Truef(t, IsEmptyString(s), "String is empty, but got not empty")
	s = "xyz"
	assert.Falsef(t, IsEmptyString(s), "String is not empty, but got empty")
}

func TestIsNilPointer(t *testing.T) {
	var ptr *int
	assert.Truef(t, IsNilPointer(ptr), "Pointer is not referencing, but got not nil")
	i := 34
	ptr = &i
	assert.Falsef(t, IsNilPointer(ptr), "Pointer is referencing, but got nil")
}

func TestIsEmptyMap(t *testing.T) {
	var mp map[int32]string
	assert.Truef(t, IsEmptyMap(mp), "Map is empty, but got non empty")
	mp = make(map[int32]string)
	assert.Truef(t, IsEmptyMap(mp), "Map is empty, but got non empty")
	mp[int32(23)] = "xyz"
	assert.Falsef(t, IsEmptyMap(mp), "Map is not empty, but got empty")
}

func TestIsEmptySlice(t *testing.T) {
	var sl []int32
	assert.Truef(t, IsEmptySlice(sl), "Slice is empty, but got non empty")
	sl = append(sl, 23)
	assert.Falsef(t, IsEmptySlice(sl), "Slice is not empty, but got empty")
}
