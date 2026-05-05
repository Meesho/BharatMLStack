package aggregator

const (
	GenericRetrieveQuery = "SELECT * FROM %s.%s WHERE %s = ?"
	GenericPersistQuery  = "INSERT INTO %s.%s (%s) VALUES (%s)"
	Id                   = "candidate_id"
)

type Database interface {
	Query(storeId string, query *Query) (map[string]interface{}, error)
	Persist(storeId string, candidateId string, columns map[string]interface{}) error
}
