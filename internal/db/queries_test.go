package db_test

import (
	"strings"
	"testing"

	"github.com/wheelibin/qrypad/internal/db"
)

func makeConn(driver db.DriverNameType) db.DBConn {
	return db.DBConn{DriverName: driver}
}

func TestGetDatabasesSQL(t *testing.T) {
	tests := []struct {
		name        string
		driver      db.DriverNameType
		wantEmpty   bool
		wantContain string
	}{
		{name: "MySQL returns non-empty SQL", driver: db.DriverName.MySQL, wantContain: "SCHEMATA"},
		{name: "Postgres returns non-empty SQL", driver: db.DriverName.Postgres, wantContain: "pg_database"},
		{name: "SQLite returns empty string", driver: db.DriverName.SQLite, wantEmpty: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := db.GetDatabasesSQL(makeConn(tt.driver))
			if tt.wantEmpty && got != "" {
				t.Errorf("expected empty string, got %q", got)
			}
			if !tt.wantEmpty && !strings.Contains(got, tt.wantContain) {
				t.Errorf("expected SQL to contain %q, got %q", tt.wantContain, got)
			}
		})
	}
}

func TestGetSchemaTablesSQL(t *testing.T) {
	tests := []struct {
		name        string
		driver      db.DriverNameType
		wantContain string
	}{
		{name: "MySQL", driver: db.DriverName.MySQL, wantContain: "information_schema"},
		{name: "Postgres", driver: db.DriverName.Postgres, wantContain: "pg_stat_user_tables"},
		{name: "SQLite", driver: db.DriverName.SQLite, wantContain: "sqlite_master"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := db.GetSchemaTablesSQL(makeConn(tt.driver))
			if got == "" {
				t.Fatalf("expected non-empty SQL for driver %q", tt.driver)
			}
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("expected SQL to contain %q, got %q", tt.wantContain, got)
			}
		})
	}
}

func TestGetSchemaViewsSQL(t *testing.T) {
	tests := []struct {
		name        string
		driver      db.DriverNameType
		wantContain string
	}{
		{name: "MySQL", driver: db.DriverName.MySQL, wantContain: "information_schema.views"},
		{name: "Postgres", driver: db.DriverName.Postgres, wantContain: "information_schema.views"},
		{name: "SQLite", driver: db.DriverName.SQLite, wantContain: "sqlite_master"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := db.GetSchemaViewsSQL(makeConn(tt.driver))
			if got == "" {
				t.Fatalf("expected non-empty SQL for driver %q", tt.driver)
			}
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("expected SQL to contain %q, got %q", tt.wantContain, got)
			}
		})
	}
}

func TestGetTableColumnsSQL(t *testing.T) {
	tableName := "my_table"
	tests := []struct {
		name        string
		driver      db.DriverNameType
		wantContain string
	}{
		{name: "MySQL", driver: db.DriverName.MySQL, wantContain: "INFORMATION_SCHEMA.COLUMNS"},
		{name: "Postgres", driver: db.DriverName.Postgres, wantContain: "INFORMATION_SCHEMA.COLUMNS"},
		{name: "SQLite", driver: db.DriverName.SQLite, wantContain: "pragma_table_info"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := db.GetTableColumnsSQL(makeConn(tt.driver), tableName)
			if got == "" {
				t.Fatalf("expected non-empty SQL for driver %q", tt.driver)
			}
			if !strings.Contains(got, tableName) {
				t.Errorf("expected SQL to contain table name %q, got %q", tableName, got)
			}
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("expected SQL to contain %q, got %q", tt.wantContain, got)
			}
		})
	}
}

func TestGetTableIndexesSQL(t *testing.T) {
	tableName := "my_table"
	tests := []struct {
		name        string
		driver      db.DriverNameType
		wantContain string
	}{
		{name: "MySQL", driver: db.DriverName.MySQL, wantContain: "INFORMATION_SCHEMA.statistics"},
		{name: "Postgres", driver: db.DriverName.Postgres, wantContain: "pg_index"},
		{name: "SQLite", driver: db.DriverName.SQLite, wantContain: "pragma_index_list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := db.GetTableIndexesSQL(makeConn(tt.driver), tableName)
			if got == "" {
				t.Fatalf("expected non-empty SQL for driver %q", tt.driver)
			}
			if !strings.Contains(got, tableName) {
				t.Errorf("expected SQL to contain table name %q, got %q", tableName, got)
			}
		})
	}
}

func TestGetTableConstraintsSQL(t *testing.T) {
	tableName := "my_table"
	tests := []struct {
		name        string
		driver      db.DriverNameType
		wantContain string
	}{
		{name: "MySQL", driver: db.DriverName.MySQL, wantContain: "TABLE_CONSTRAINTS"},
		{name: "Postgres", driver: db.DriverName.Postgres, wantContain: "pg_constraint"},
		{name: "SQLite", driver: db.DriverName.SQLite, wantContain: "foreign_key_list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := db.GetTableConstraintsSQL(makeConn(tt.driver), tableName)
			if got == "" {
				t.Fatalf("expected non-empty SQL for driver %q", tt.driver)
			}
			if !strings.Contains(got, tableName) {
				t.Errorf("expected SQL to contain table name %q, got %q", tableName, got)
			}
		})
	}
}

func TestGetTableRowsSQL(t *testing.T) {
	tests := []struct {
		name              string
		tableName         string
		primaryKeyColumns []string
		sortOrder         string
		wantContain       []string
		wantNotContain    []string
	}{
		{
			name:              "no primary keys - no ORDER BY",
			tableName:         "users",
			primaryKeyColumns: []string{},
			sortOrder:         "ASC",
			wantContain:       []string{"SELECT * FROM users", "LIMIT"},
			wantNotContain:    []string{"ORDER BY"},
		},
		{
			name:              "single primary key ASC",
			tableName:         "users",
			primaryKeyColumns: []string{"id"},
			sortOrder:         "ASC",
			wantContain:       []string{"SELECT * FROM users", "ORDER BY id ASC", "LIMIT"},
		},
		{
			name:              "multiple primary keys DESC",
			tableName:         "order_items",
			primaryKeyColumns: []string{"order_id", "item_id"},
			sortOrder:         "DESC",
			wantContain:       []string{"ORDER BY order_id, item_id DESC", "LIMIT"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := db.GetTableRowsSQL(tt.tableName, tt.primaryKeyColumns, tt.sortOrder)
			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("expected SQL to contain %q, got: %q", want, got)
				}
			}
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("expected SQL NOT to contain %q, got: %q", notWant, got)
				}
			}
		})
	}
}

func TestTruncateToSize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxBytes int
		wantFull bool
	}{
		{
			name:     "string shorter than limit returned as-is",
			input:    "hello",
			maxBytes: 100,
			wantFull: true,
		},
		{
			name:     "string at exact limit returned as-is",
			input:    "hello",
			maxBytes: 5,
			wantFull: true,
		},
		{
			name:     "string over limit is truncated",
			input:    "hello world",
			maxBytes: 5,
			wantFull: false,
		},
		{
			name:     "multi-byte UTF-8 truncated at rune boundary",
			input:    "héllo", // 'é' is 2 bytes (0xC3 0xA9)
			maxBytes: 3,
			wantFull: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := db.TruncateToSize(tt.input, tt.maxBytes)
			if tt.wantFull {
				if got != tt.input {
					t.Errorf("expected full string %q, got %q", tt.input, got)
				}
			} else {
				if len(got) > tt.maxBytes {
					t.Errorf("result length %d exceeds maxBytes %d: %q", len(got), tt.maxBytes, got)
				}
				if got == tt.input {
					t.Errorf("expected truncation but got full string: %q", got)
				}
				// Verify it's valid UTF-8 (no broken rune sequences)
				for i, r := range got {
					if r == '\uFFFD' {
						t.Errorf("invalid UTF-8 rune at index %d in truncated result %q", i, got)
					}
				}
			}
		})
	}
}
