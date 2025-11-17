package stores

import (
	"fmt"

	"github.com/gocql/gocql"
)

func getQueryV1(session interface{}, query string) (*gocql.Query, error) {
	if gocqlSession, ok := session.(*gocql.Session); ok {
		return gocqlSession.Query(query), nil
	}
	return nil, fmt.Errorf("invalid gocql query type")
}

func retrieveV1(query interface{}) ([]map[string]interface{}, error) {
	if gocqlQuery, ok := query.(*gocql.Query); ok {
		return gocqlQuery.Iter().SliceMap()
	}
	return nil, fmt.Errorf("invalid gocql query type")
}

func persistV1(query interface{}) error {
	if gocqlQuery, ok := query.(*gocql.Query); ok {
		return gocqlQuery.Exec()
	}
	return fmt.Errorf("invalid gocql query type")
}

func bindV1(ps interface{}, bindKeys []interface{}) interface{} {
	if gocqlQuery, ok := ps.(*gocql.Query); ok {
		return gocqlQuery.Bind(bindKeys...).Consistency(gocql.One)
	}
	return nil
}
