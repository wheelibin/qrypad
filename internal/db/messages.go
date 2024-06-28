package db

type Data struct {
	Columns []string
	Rows    []map[string]any
}

type DataMsg *Data
type TableInfoDataMsg *Data
type SchemaTablesMsg *Data
