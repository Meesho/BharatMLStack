package enum

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ClientType uint8

const (
	ClientTypeUnknown ClientType = iota
	ClientTypeAndroid
	ClientTypeIOS
	ClientTypeWeb
)

var (
	clientTypeName = map[uint8]string{
		0: "unknown",
		1: "android",
		2: "ios",
		3: "web",
	}

	clientTypeValue = map[string]ClientType{
		"unknown": ClientTypeUnknown,
		"android": ClientTypeAndroid,
		"ios":     ClientTypeIOS,
		"web":     ClientTypeWeb,
	}
)

// String allows ClientType to implement fmt.Stringer
func (e ClientType) String() string {
	return clientTypeName[uint8(e)]
}

// Name allows ClientType to be backward compatible with the old enum
func (e ClientType) Name() string {
	return e.String()
}

// EnumIndex returns the integer representation of the ClientType.
func (s ClientType) EnumIndex() uint8 {
	return uint8(s)
}

// ShortValue returns the integer representation of the ClientType.
func (s ClientType) ShortValue() uint8 {
	return s.EnumIndex()
}

// MarshalJSON marshals the enum as a quoted json string
func (s ClientType) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON unmarshals a quoted json string to the enum value
func (s *ClientType) UnmarshalJSON(data []byte) (err error) {
	var suits string
	if err := json.Unmarshal(data, &suits); err != nil {
		return err
	}
	if *s, err = ParseClientType(suits); err != nil {
		return err
	}
	return nil
}

// ParseClientType Convert a string to a ClientType, returns an error if the string is unknown.
// NOTE: for JSON marshaling this must return a ClientType value not a pointer, which is
// common when using integer enumerations (or any primitive type alias).
func ParseClientType(s string) (ClientType, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	value, ok := clientTypeValue[s]
	if !ok {
		return ClientType(0), fmt.Errorf("%q is not a valid ClientType", s)
	}
	return value, nil
}
