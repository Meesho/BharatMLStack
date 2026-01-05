package scylla

const (
	UpdateQuery           = "update %s.%s set %s = ? WHERE user_id = ?"
	PersistQuery          = "insert into %s.%s (%s) values (%s)"
	RetrieveQuery         = "select %s from %s.%s WHERE user_id = ?"
	UpdateMetadataQuery   = "update %s.%s set count = ? WHERE user_id = ?"
	PersistMetadataQuery  = "insert into %s.%s (count, user_id) values (?, ?)"
	RetrieveMetadataQuery = "select count from %s.%s WHERE user_id = ?"
)

type Database interface {
	RetrieveInteractions(tableName string, userId string, columns []string) (map[string]interface{}, error)
	PersistInteractions(tableName string, userId string, columns map[string]interface{}) error
	UpdateInteractions(tableName string, userId string, column string, value interface{}) error
	RetrieveMetadata(metadataTableName string, userId string) (map[string]interface{}, error)
	PersistMetadata(metadataTableName string, userId string, weeklyEventCount interface{}) error
	UpdateMetadata(metadataTableName string, userId string, weeklyEventCount interface{}) error
}
