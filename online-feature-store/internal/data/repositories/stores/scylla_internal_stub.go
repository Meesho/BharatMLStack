//go:build !meesho

package stores

import (
	"fmt"

	"github.com/gocql/gocql"
)

func getQueryV2(_ interface{}, _ string) (*gocql.Query, error) {
	return nil, fmt.Errorf("gocql_v2 not compiled: build with -tags meesho")
}

func retrieveV2(_ interface{}) ([]map[string]interface{}, error) {
	return nil, fmt.Errorf("gocql_v2 not compiled: build with -tags meesho")
}

func persistV2(_ interface{}) error {
	return fmt.Errorf("gocql_v2 not compiled: build with -tags meesho")
}

func bindV2(_ interface{}, _ []interface{}) interface{} {
	return nil
}
