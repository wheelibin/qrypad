package db

// Exported wrappers for unexported functions and types, used only in tests.

var TruncateToSize = truncateToSize //nolint:gochecknoglobals

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
