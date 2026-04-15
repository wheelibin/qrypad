package db_test

import (
	"slices"
	"context"
	"database/sql"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/wheelibin/qrypad/internal/db"
)

// openTestDB opens an in-memory SQLite database for testing.
// The caller is responsible for closing it.
func openTestDB(t *testing.T) db.DBConn {
	t.Helper()
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("openTestDB: sql.Open: %v", err)
	}
	if err := sqlDB.PingContext(context.Background()); err != nil {
		t.Fatalf("openTestDB: Ping: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("openTestDB: Close: %v", err)
		}
	})
	return db.DBConn{DB: sqlDB, DriverName: db.DriverName.SQLite}
}

func TestExecuteQuery_Routing(t *testing.T) {
	conn := openTestDB(t)
	ctx := context.Background()

	// Create a test table once, used across subtests
	_, err := conn.DB.ExecContext(ctx, "CREATE TABLE test_items (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("setup: CREATE TABLE: %v", err)
	}

	tests := []struct {
		name          string
		query         string
		wantStatement bool // true = routes to execStatement (rows affected), false = routes to fetchRows (columns)
	}{
		{
			name:          "SELECT routes to fetchRows",
			query:         "SELECT * FROM test_items",
			wantStatement: false,
		},
		{
			name:          "INSERT routes to execStatement",
			query:         "INSERT INTO test_items (name) VALUES ('foo')",
			wantStatement: true,
		},
		{
			name:          "UPDATE routes to execStatement",
			query:         "UPDATE test_items SET name = 'bar' WHERE id = 1",
			wantStatement: true,
		},
		{
			name:          "DELETE routes to execStatement",
			query:         "DELETE FROM test_items WHERE id = 999",
			wantStatement: true,
		},
		{
			name:          "CREATE TABLE routes to execStatement",
			query:         "CREATE TABLE IF NOT EXISTS other_items (id INTEGER PRIMARY KEY)",
			wantStatement: true,
		},
		{
			name:          "DROP TABLE routes to execStatement",
			query:         "DROP TABLE IF EXISTS other_items",
			wantStatement: true,
		},
		{
			name:          "TRUNCATE-like DELETE routes to execStatement",
			query:         "DELETE FROM test_items",
			wantStatement: true,
		},
		{
			name:          "SELECT with leading whitespace routes to fetchRows",
			query:         "   SELECT id FROM test_items",
			wantStatement: false,
		},
		{
			name:          "lowercase select routes to fetchRows",
			query:         "select * from test_items",
			wantStatement: false,
		},
		{
			name:          "INSERT with RETURNING routes to fetchRows",
			query:         "INSERT INTO test_items (name) VALUES ('returning_test') RETURNING id, name",
			wantStatement: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := db.ExecuteQuery(ctx, conn, tt.query)
			if err != nil {
				t.Fatalf("ExecuteQuery(%q): unexpected error: %v", tt.query, err)
			}
			if data == nil {
				t.Fatalf("ExecuteQuery(%q): returned nil data", tt.query)
			}

			if tt.wantStatement {
				// execStatement returns a "Rows Affected" column
				hasRowsAffected := slices.Contains(data.Columns, "Rows Affected")
				if !hasRowsAffected {
					t.Errorf("ExecuteQuery(%q): expected statement result with 'Rows Affected' column, got columns: %v",
						tt.query, data.Columns)
				}
			} else {
				// fetchRows returns actual table columns (not "Rows Affected")
				for _, col := range data.Columns {
					if col == "Rows Affected" {
						t.Errorf("ExecuteQuery(%q): expected row-fetching result but got 'Rows Affected' column",
							tt.query)
					}
				}
			}
		})
	}
}
