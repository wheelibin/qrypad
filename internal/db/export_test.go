package db

// Exported wrappers for unexported functions and types, used only in tests.

var (
	TruncateToSize      = truncateToSize      //nolint:gochecknoglobals
	NormalizeColumnType = normalizeColumnType //nolint:gochecknoglobals
)

// Test-only exports of private SQL builders on driver provider types.
// These let external tests (package db_test) assert on the generated SQL
// without keeping the builders as part of the public QueryProvider API.
var (
	MySQLTableIndexesSQL         = mysqlQueries{}.tableIndexesSQL         //nolint:gochecknoglobals
	PostgresTableIndexesSQL      = postgresQueries{}.tableIndexesSQL      //nolint:gochecknoglobals
	MySQLPrimaryKeyColumnsSQL    = mysqlQueries{}.primaryKeyColumnsSQL    //nolint:gochecknoglobals
	PostgresPrimaryKeyColumnsSQL = postgresQueries{}.primaryKeyColumnsSQL //nolint:gochecknoglobals
	SQLitePrimaryKeyColumnsSQL   = sqliteQueries{}.primaryKeyColumnsSQL   //nolint:gochecknoglobals
)

// QueriesForDriver returns the QueryProvider for the given driver name.
// Exported for tests only.
func QueriesForDriver(driver DriverNameType) QueryProvider {
	switch driver {
	case DriverName.MySQL:
		return mysqlQueries{}
	case DriverName.Postgres:
		return postgresQueries{}
	case DriverName.SQLite:
		return sqliteQueries{}
	}
	return nil
}
