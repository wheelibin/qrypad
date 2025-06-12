package db

type Data struct {
	Columns []string
	Rows    []map[string]any
}

type (
	DatabaseConnectedMsg    DBConn
	DataFetchedMsg          *Data
	TableInfoDataFetchedMsg *Data
	SchemaTablesFetchedMsg  *Data
	DatabaseListFetchedMsg  *Data
)
