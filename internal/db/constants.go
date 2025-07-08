package db

const (
	TimeoutConfigKey           = "queryTimeout"
	TableDataRowLimitConfigKey = "tableDataRowLimit"
)

type DriverNameType string

var DriverName = struct {
	Postgres DriverNameType
	MySQL    DriverNameType
	SQLite   DriverNameType
}{
	Postgres: "postgres",
	MySQL:    "mysql",
	SQLite:   "sqlite",
}
