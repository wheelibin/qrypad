package db_test

import (
	"strings"
	"testing"

	"github.com/wheelibin/qrypad/internal/db"
)

func makeConn(driver db.DriverNameType) db.DBConn {
	return db.DBConn{DriverName: driver, Queries: db.QueriesForDriver(driver)}
}

func makeRef(schema, name string) db.TableReference {
	return db.TableReference{Schema: schema, Name: name}
}

func TestDatabases(t *testing.T) {
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
			got := makeConn(tt.driver).Queries.Databases()
			if tt.wantEmpty && got != "" {
				t.Errorf("expected empty string, got %q", got)
			}
			if !tt.wantEmpty && !strings.Contains(got, tt.wantContain) {
				t.Errorf("expected SQL to contain %q, got %q", tt.wantContain, got)
			}
		})
	}
}

func TestSchemaTables(t *testing.T) {
	tests := []struct {
		name        string
		driver      db.DriverNameType
		wantContain string
		wantSchema  bool
	}{
		{name: "MySQL", driver: db.DriverName.MySQL, wantContain: "information_schema", wantSchema: true},
		{name: "Postgres", driver: db.DriverName.Postgres, wantContain: "pg_stat_user_tables", wantSchema: true},
		{name: "SQLite", driver: db.DriverName.SQLite, wantContain: "sqlite_master", wantSchema: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeConn(tt.driver).Queries.SchemaTables()
			if got == "" {
				t.Fatalf("expected non-empty SQL for driver %q", tt.driver)
			}
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("expected SQL to contain %q, got %q", tt.wantContain, got)
			}
			hasSchema := strings.Contains(strings.ToLower(got), "schema")
			if tt.wantSchema && !hasSchema {
				t.Errorf("expected SQL to select a schema column, got %q", got)
			}
			if !tt.wantSchema && hasSchema {
				t.Errorf("expected SQL to NOT select a schema column for SQLite, got %q", got)
			}
		})
	}
}

func TestSchemaViews(t *testing.T) {
	tests := []struct {
		name        string
		driver      db.DriverNameType
		wantContain string
		wantSchema  bool
	}{
		{name: "MySQL", driver: db.DriverName.MySQL, wantContain: "information_schema.views", wantSchema: true},
		{name: "Postgres", driver: db.DriverName.Postgres, wantContain: "information_schema.views", wantSchema: true},
		{name: "SQLite", driver: db.DriverName.SQLite, wantContain: "sqlite_master", wantSchema: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeConn(tt.driver).Queries.SchemaViews()
			if got == "" {
				t.Fatalf("expected non-empty SQL for driver %q", tt.driver)
			}
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("expected SQL to contain %q, got %q", tt.wantContain, got)
			}
			hasSchema := strings.Contains(strings.ToLower(got), "schema")
			if tt.wantSchema && !hasSchema {
				t.Errorf("expected SQL to select a schema column, got %q", got)
			}
			if !tt.wantSchema && hasSchema {
				t.Errorf("expected SQL to NOT select a schema column for SQLite, got %q", got)
			}
		})
	}
}

func TestTableColumns(t *testing.T) {
	tests := []struct {
		name        string
		driver      db.DriverNameType
		ref         db.TableReference
		wantContain string
	}{
		{name: "MySQL no schema", driver: db.DriverName.MySQL, ref: makeRef("", "my_table"), wantContain: "INFORMATION_SCHEMA.COLUMNS"},
		{name: "MySQL with schema", driver: db.DriverName.MySQL, ref: makeRef("myschema", "my_table"), wantContain: "myschema"},
		{name: "Postgres no schema", driver: db.DriverName.Postgres, ref: makeRef("", "my_table"), wantContain: "INFORMATION_SCHEMA.COLUMNS"},
		{name: "Postgres with schema", driver: db.DriverName.Postgres, ref: makeRef("public", "my_table"), wantContain: "public"},
		{name: "SQLite", driver: db.DriverName.SQLite, ref: makeRef("", "my_table"), wantContain: "pragma_table_info"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeConn(tt.driver).Queries.TableColumns(tt.ref)
			if got == "" {
				t.Fatalf("expected non-empty SQL for driver %q", tt.driver)
			}
			if !strings.Contains(got, tt.ref.Name) {
				t.Errorf("expected SQL to contain table name %q, got %q", tt.ref.Name, got)
			}
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("expected SQL to contain %q, got %q", tt.wantContain, got)
			}
		})
	}
}

func TestTableConstraints(t *testing.T) {
	tests := []struct {
		name        string
		driver      db.DriverNameType
		ref         db.TableReference
		wantContain string
	}{
		{name: "MySQL no schema", driver: db.DriverName.MySQL, ref: makeRef("", "my_table"), wantContain: "TABLE_CONSTRAINTS"},
		{name: "MySQL with schema", driver: db.DriverName.MySQL, ref: makeRef("myschema", "my_table"), wantContain: "myschema"},
		{name: "Postgres no schema", driver: db.DriverName.Postgres, ref: makeRef("", "my_table"), wantContain: "pg_constraint"},
		{name: "Postgres with schema", driver: db.DriverName.Postgres, ref: makeRef("public", "my_table"), wantContain: "public"},
		{name: "SQLite", driver: db.DriverName.SQLite, ref: makeRef("", "my_table"), wantContain: "foreign_key_list"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeConn(tt.driver).Queries.TableConstraints(tt.ref)
			if got == "" {
				t.Fatalf("expected non-empty SQL for driver %q", tt.driver)
			}
			if !strings.Contains(got, tt.ref.Name) {
				t.Errorf("expected SQL to contain table name %q, got %q", tt.ref.Name, got)
			}
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("expected SQL to contain %q, got %q", tt.wantContain, got)
			}
		})
	}
}

func TestTableRows(t *testing.T) {
	tests := []struct {
		name              string
		driver            db.DriverNameType
		ref               db.TableReference
		primaryKeyColumns []string
		sortOrder         string
		wantContain       []string
		wantNotContain    []string
	}{
		{
			name:              "no schema no primary keys",
			driver:            db.DriverName.Postgres,
			ref:               makeRef("", "users"),
			primaryKeyColumns: []string{},
			sortOrder:         "ASC",
			wantContain:       []string{`SELECT * FROM "users"`, "LIMIT"},
			wantNotContain:    []string{"ORDER BY"},
		},
		{
			name:              "no schema single primary key ASC",
			driver:            db.DriverName.Postgres,
			ref:               makeRef("", "users"),
			primaryKeyColumns: []string{"id"},
			sortOrder:         "ASC",
			wantContain:       []string{`SELECT * FROM "users"`, "ORDER BY id ASC", "LIMIT"},
		},
		{
			name:              "with schema qualifies FROM clause (postgres double-quote)",
			driver:            db.DriverName.Postgres,
			ref:               makeRef("public", "users"),
			primaryKeyColumns: []string{"id"},
			sortOrder:         "ASC",
			wantContain:       []string{`SELECT * FROM "public"."users"`, "ORDER BY id ASC", "LIMIT"},
		},
		{
			name:              "mysql with schema uses backtick quoting",
			driver:            db.DriverName.MySQL,
			ref:               makeRef("myschema", "orders"),
			primaryKeyColumns: []string{"id"},
			sortOrder:         "DESC",
			wantContain:       []string{"SELECT * FROM `myschema`.`orders`", "ORDER BY id DESC", "LIMIT"},
		},
		{
			name:              "multiple primary keys DESC",
			driver:            db.DriverName.Postgres,
			ref:               makeRef("", "order_items"),
			primaryKeyColumns: []string{"order_id", "item_id"},
			sortOrder:         "DESC",
			wantContain:       []string{`SELECT * FROM "order_items"`, "ORDER BY order_id, item_id DESC", "LIMIT"},
		},
		{
			name:              "sqlite no schema no quoting",
			driver:            db.DriverName.SQLite,
			ref:               makeRef("", "users"),
			primaryKeyColumns: []string{"id"},
			sortOrder:         "ASC",
			wantContain:       []string{"SELECT * FROM users", "ORDER BY id ASC", "LIMIT"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeConn(tt.driver).Queries.TableRows(tt.ref, tt.primaryKeyColumns, tt.sortOrder)
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

func TestTableIndexesSQL_Builders(t *testing.T) {
	tests := []struct {
		name           string
		build          func(db.TableReference) string
		ref            db.TableReference
		wantContain    []string
		wantNotContain []string
	}{
		{
			name:  "MySQL no schema",
			build: db.MySQLTableIndexesSQL,
			ref:   db.TableReference{Name: "my_table"},
			wantContain: []string{
				"INFORMATION_SCHEMA.statistics",
				"TABLE_SCHEMA = DATABASE()",
				"TABLE_NAME = 'my_table'",
				"GROUP_CONCAT(column_name ORDER BY seq_in_index)",
				"order by index_name",
			},
		},
		{
			name:  "MySQL with schema",
			build: db.MySQLTableIndexesSQL,
			ref:   db.TableReference{Schema: "myschema", Name: "my_table"},
			wantContain: []string{
				"INFORMATION_SCHEMA.statistics",
				"TABLE_SCHEMA = 'myschema'",
				"TABLE_NAME = 'my_table'",
				"GROUP_CONCAT(column_name ORDER BY seq_in_index)",
				"order by index_name",
			},
			wantNotContain: []string{"DATABASE()"},
		},
		{
			name:  "Postgres no schema",
			build: db.PostgresTableIndexesSQL,
			ref:   db.TableReference{Name: "my_table"},
			wantContain: []string{
				"pg_index",
				"t.relname like 'my_table'",
				"array_agg(a.attname ORDER BY array_position(ix.indkey::int[], a.attnum::int))",
			},
			wantNotContain: []string{"n.nspname ="},
		},
		{
			name:  "Postgres with schema",
			build: db.PostgresTableIndexesSQL,
			ref:   db.TableReference{Schema: "public", Name: "my_table"},
			wantContain: []string{
				"pg_index",
				"t.relname like 'my_table'",
				"n.nspname = 'public'",
				"array_agg(a.attname ORDER BY array_position(ix.indkey::int[], a.attnum::int))",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.build(tt.ref)
			if got == "" {
				t.Fatalf("expected non-empty SQL")
			}
			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("expected SQL to contain %q, got:\n%s", want, got)
				}
			}
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("expected SQL NOT to contain %q, got:\n%s", notWant, got)
				}
			}
		})
	}
}

func TestPrimaryKeyColumnsSQL_Builders(t *testing.T) {
	tests := []struct {
		name        string
		builder     func(db.TableReference) string
		ref         db.TableReference
		wantContain string
	}{
		{
			name:        "MySQL no schema",
			builder:     db.MySQLPrimaryKeyColumnsSQL,
			ref:         db.TableReference{Name: "my_table"},
			wantContain: "KEY_COLUMN_USAGE",
		},
		{
			name:        "MySQL with schema",
			builder:     db.MySQLPrimaryKeyColumnsSQL,
			ref:         db.TableReference{Schema: "myschema", Name: "my_table"},
			wantContain: "myschema",
		},
		{
			name:        "Postgres no schema",
			builder:     db.PostgresPrimaryKeyColumnsSQL,
			ref:         db.TableReference{Name: "my_table"},
			wantContain: "pg_index",
		},
		{
			name:        "Postgres with schema",
			builder:     db.PostgresPrimaryKeyColumnsSQL,
			ref:         db.TableReference{Schema: "public", Name: "my_table"},
			wantContain: "public",
		},
		{
			name:        "SQLite",
			builder:     db.SQLitePrimaryKeyColumnsSQL,
			ref:         db.TableReference{Name: "my_table"},
			wantContain: "pragma_table_info",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.builder(tt.ref)
			if got == "" {
				t.Fatalf("expected non-empty SQL")
			}
			if !strings.Contains(got, tt.ref.Name) {
				t.Errorf("expected SQL to contain table name %q, got %q", tt.ref.Name, got)
			}
			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("expected SQL to contain %q, got %q", tt.wantContain, got)
			}
		})
	}
}
