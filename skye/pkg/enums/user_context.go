package enum

import (
	"encoding/json"
	"fmt"
	"strings"
)

type UserContext uint8

const (
	UserContextUnknown UserContext = iota
	UserContextLoggedIn
	UserContextAnonymous
)

var (
	userContextName = map[uint8]string{
		0: "unknown",
		1: "logged_in",
		2: "anonymous",
	}

	userContextValue = map[string]UserContext{
		"unknown":   UserContextUnknown,
		"logged_in": UserContextLoggedIn,
		"anonymous": UserContextAnonymous,
	}
)

// String allows UserContext to implement fmt.Stringer
func (e UserContext) String() string {
	return userContextName[uint8(e)]
}

// Name allows UserContext to be backward compatible with the old enum
func (e UserContext) Name() string {
	return e.String()
}

// EnumIndex returns the integer representation of the UserContext.
func (s UserContext) EnumIndex() uint8 {
	return uint8(s)
}

// ShortValue returns the integer representation of the UserContext.
func (s UserContext) ShortValue() uint8 {
	return s.EnumIndex()
}

// MarshalJSON marshals the enum as a quoted json string
func (s UserContext) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON unmarshals a quoted json string to the enum value
func (s *UserContext) UnmarshalJSON(data []byte) (err error) {
	var suits string
	if err := json.Unmarshal(data, &suits); err != nil {
		return err
	}
	if *s, err = ParseUserContext(suits); err != nil {
		return err
	}
	return nil
}

// ParseUserContext Convert a string to a UserContext, returns an error if the string is unknown.
// NOTE: for JSON marshaling this must return a UserContext value not a pointer, which is
// common when using integer enumerations (or any primitive type alias).
func ParseUserContext(s string) (UserContext, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	value, ok := userContextValue[s]
	if !ok {
		return UserContext(0), fmt.Errorf("%q is not a valid UserContext", s)
	}
	return value, nil
}
