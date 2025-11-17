//go:build !meesho

package infra

import "errors"

func createSessionV2(_ interface{}) (interface{}, error) {
	return nil, errors.New("gocql_v2 not compiled: build with -tags meesho")
}

func isSessionClosedV2(_ interface{}) bool {
	return true // treat as closed
}

func buildGocqlV2ClusterConfig(_ []string, _ string, _ string) (interface{}, error) {
	return nil, errors.New("gocql_v2 not compiled: build with -tags meesho")
}
