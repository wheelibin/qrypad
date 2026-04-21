package db

const (
	TimeoutConfigKey           = "queryTimeout"
	TableDataRowLimitConfigKey = "tableDataRowLimit"
)

type DriverNameType string

//nolint:gochecknoglobals // singleton enum struct used as namespaced constants
var DriverName = struct {
	Postgres DriverNameType
	MySQL    DriverNameType
	SQLite   DriverNameType
}{
	Postgres: "postgres",
	MySQL:    "mysql",
	SQLite:   "sqlite",
}
