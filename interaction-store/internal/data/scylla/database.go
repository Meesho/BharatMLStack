package scylla

const (
	UpdateQuery           = "update %s.%s set %s WHERE user_id = ?"
	RetrieveQuery         = "select %s from %s.%s WHERE user_id = ?"
	UpdateMetadataQuery   = "update %s.%s set %s WHERE user_id = ?"
	RetrieveMetadataQuery = "select %s from %s.%s WHERE user_id = ?"
)

type Database interface {
	RetrieveInteractions(tableName string, userId string, columns []string) (map[string]interface{}, error)
	UpdateInteractions(tableName string, userId string, columns map[string]interface{}) error
	RetrieveMetadata(metadataTableName string, userId string, columns []string) (map[string]interface{}, error)
	UpdateMetadata(metadataTableName string, userId string, columns map[string]interface{}) error
}
