//go:build integration

package db_test

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"

	"github.com/wheelibin/qrypad/internal/db"
)

// openPostgresConn opens a connection to the Postgres test DB seeded by
// test-db/postgres/init.sql. Registers a cleanup to close the connection.
func openPostgresConn(t *testing.T) db.DBConn {
	t.Helper()
	conn, err := db.Connect(db.ConnectionConfig{
		Driver:   db.DriverName.Postgres,
		Host:     "localhost",
		Port:     50403,
		User:     "qrypad",
		Database: "qrypad",
	}, "qrypad")
	if err != nil {
		t.Fatalf("failed to connect to Postgres test DB: %v", err)
	}
	t.Cleanup(func() { conn.DB.Close() })
	return conn
}

// openMySQLConn opens a connection to the MySQL test DB seeded by
// test-db/mysql/init.sql. Registers a cleanup to close the connection.
func openMySQLConn(t *testing.T) db.DBConn {
	t.Helper()
	conn, err := db.Connect(db.ConnectionConfig{
		Driver:   db.DriverName.MySQL,
		Host:     "localhost",
		Port:     50306,
		User:     "qrypad",
		Database: "qrypad",
	}, "qrypad")
	if err != nil {
		t.Fatalf("failed to connect to MySQL test DB: %v", err)
	}
	t.Cleanup(func() { conn.DB.Close() })
	return conn
}

// openSQLiteConn opens a connection to the SQLite test DB seeded by
// test-db/sqlite/init.sql. Registers a cleanup to close the connection.
func openSQLiteConn(t *testing.T) db.DBConn {
	t.Helper()
	conn, err := db.Connect(db.ConnectionConfig{
		Driver:   db.DriverName.SQLite,
		Database: "../../test-db/sqlite/sqlite.db",
	}, "")
	if err != nil {
		t.Fatalf("failed to connect to SQLite test DB: %v", err)
	}
	t.Cleanup(func() { conn.DB.Close() })
	return conn
}

// columnNames extracts the "name" field from each row as a []string.
// Useful for asserting which columns/tables are returned.
func columnNames(rows []map[string]any) []string {
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		if name, ok := row["name"].(string); ok {
			names = append(names, name)
		}
	}
	return names
}

// rowsContainName reports whether any row has a "name" field equal to target.
func rowsContainName(rows []map[string]any, target string) bool {
	for _, row := range rows {
		if name, ok := row["name"].(string); ok && name == target {
			return true
		}
	}
	return false
}

// rowsContainAllNames reports whether every target name appears in at least
// one row's "name" field.
func rowsContainAllNames(rows []map[string]any, targets ...string) bool {
	for _, target := range targets {
		if !rowsContainName(rows, target) {
			return false
		}
	}
	return true
}
