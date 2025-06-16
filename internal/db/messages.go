package db

import "time"

type Data struct {
	Columns   []string
	Rows      []map[string]any
	QueryTime time.Duration
}

type (
	DatabaseConnectedMsg     DBConn
	DataFetchedMsg           *Data
	TableInfoDataFetchedMsg  *Data
	SchemaEntitiesFetchedMsg *Data
	DatabaseListFetchedMsg   *Data
)
