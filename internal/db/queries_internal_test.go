package db

import (
	"strings"
	"testing"
)

func TestTableIndexesSQL_Builders(t *testing.T) {
	tests := []struct {
		name           string
		build          func(TableReference) string
		ref            TableReference
		wantContain    []string
		wantNotContain []string
	}{
		{
			name:  "MySQL no schema",
			build: mysqlQueries{}.tableIndexesSQL,
			ref:   TableReference{Name: "my_table"},
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
			build: mysqlQueries{}.tableIndexesSQL,
			ref:   TableReference{Schema: "myschema", Name: "my_table"},
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
			build: postgresQueries{}.tableIndexesSQL,
			ref:   TableReference{Name: "my_table"},
			wantContain: []string{
				"pg_index",
				"t.relname like 'my_table'",
				"array_agg(a.attname ORDER BY array_position(ix.indkey::int[], a.attnum::int))",
			},
			wantNotContain: []string{"n.nspname ="},
		},
		{
			name:  "Postgres with schema",
			build: postgresQueries{}.tableIndexesSQL,
			ref:   TableReference{Schema: "public", Name: "my_table"},
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
		builder     func(TableReference) string
		ref         TableReference
		wantContain string
	}{
		{
			name:        "MySQL no schema",
			builder:     mysqlQueries{}.primaryKeyColumnsSQL,
			ref:         TableReference{Name: "my_table"},
			wantContain: "KEY_COLUMN_USAGE",
		},
		{
			name:        "MySQL with schema",
			builder:     mysqlQueries{}.primaryKeyColumnsSQL,
			ref:         TableReference{Schema: "myschema", Name: "my_table"},
			wantContain: "myschema",
		},
		{
			name:        "Postgres no schema",
			builder:     postgresQueries{}.primaryKeyColumnsSQL,
			ref:         TableReference{Name: "my_table"},
			wantContain: "pg_index",
		},
		{
			name:        "Postgres with schema",
			builder:     postgresQueries{}.primaryKeyColumnsSQL,
			ref:         TableReference{Schema: "public", Name: "my_table"},
			wantContain: "public",
		},
		{
			name:        "SQLite",
			builder:     sqliteQueries{}.primaryKeyColumnsSQL,
			ref:         TableReference{Name: "my_table"},
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
