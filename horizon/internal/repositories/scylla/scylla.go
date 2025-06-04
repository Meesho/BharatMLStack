package scylla

type Store interface {
	CreateTable(tableName string, pkColumns []string, defaultTimeToLive int) error
	AddColumn(tableName string, column string) error
}
