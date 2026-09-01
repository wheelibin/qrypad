package db_test

import (
	"context"
	"database/sql"
	"slices"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"

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
	return db.DBConn{
		DB:         sqlDB,
		DriverName: db.DriverName.SQLite,
		Queries:    db.QueriesForDriver(db.DriverName.SQLite),
	}
}

func TestSQLiteTableIndexes_MultiPragmaWalk(t *testing.T) {
	conn := openTestDB(t)
	ctx := context.Background()

	// Create a table with two indexes: one single-column, one composite.
	stmts := []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT, first_name TEXT, last_name TEXT)",
		"CREATE INDEX idx_users_email ON users(email)",
		"CREATE INDEX idx_users_name ON users(first_name, last_name)",
	}
	for _, s := range stmts {
		if _, err := conn.DB.ExecContext(ctx, s); err != nil {
			t.Fatalf("setup: %q: %v", s, err)
		}
	}

	data, err := conn.Queries.TableIndexes(ctx, conn, db.TableReference{Name: "users"})
	if err != nil {
		t.Fatalf("TableIndexes: unexpected error: %v", err)
	}
	if data == nil {
		t.Fatalf("TableIndexes: nil data")
		return
	}

	wantCols := []string{"name", "cols"}
	if !slices.Equal(data.Columns, wantCols) {
		t.Errorf("Columns: got %v, want %v", data.Columns, wantCols)
	}

	// Build a map of index name -> columns for order-independent assertion.
	got := make(map[string][]string)
	for _, row := range data.Rows {
		name, _ := row["name"].(string)
		cols, _ := row["cols"].([]string)
		got[name] = cols
	}

	wantEmail := []string{"email"}
	if cols, ok := got["idx_users_email"]; !ok {
		t.Errorf("expected index idx_users_email in results, got: %v", got)
	} else if !slices.Equal(cols, wantEmail) {
		t.Errorf("idx_users_email cols: got %v, want %v", cols, wantEmail)
	}

	wantName := []string{"first_name", "last_name"}
	if cols, ok := got["idx_users_name"]; !ok {
		t.Errorf("expected index idx_users_name in results, got: %v", got)
	} else if !slices.Equal(cols, wantName) {
		t.Errorf("idx_users_name cols: got %v, want %v", cols, wantName)
	}
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
