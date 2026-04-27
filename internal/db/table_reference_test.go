package db_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/db"
)

func TestTableReferenceQualifiedName(t *testing.T) {
	tests := []struct {
		name string
		ref  db.TableReference
		want string
	}{
		{"no schema", db.TableReference{Schema: "", Name: "users"}, "users"},
		{"with schema", db.TableReference{Schema: "public", Name: "users"}, "public.users"},
		{"myschema", db.TableReference{Schema: "myschema", Name: "orders"}, "myschema.orders"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ref.QualifiedName(); got != tt.want {
				t.Errorf("QualifiedName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTableReferenceIsDefaultSchema(t *testing.T) {
	tests := []struct {
		name        string
		ref         db.TableReference
		driver      db.DriverNameType
		connectedDB string
		want        bool
	}{
		{"postgres public is default", db.TableReference{Schema: "public", Name: "users"}, db.DriverName.Postgres, "mydb", true},
		{"postgres other schema is not default", db.TableReference{Schema: "myschema", Name: "users"}, db.DriverName.Postgres, "mydb", false},
		{"mysql schema matches connectedDB", db.TableReference{Schema: "mydb", Name: "users"}, db.DriverName.MySQL, "mydb", true},
		{"mysql schema differs from connectedDB", db.TableReference{Schema: "other", Name: "users"}, db.DriverName.MySQL, "mydb", false},
		{"sqlite empty schema is default", db.TableReference{Schema: "", Name: "users"}, db.DriverName.SQLite, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ref.IsDefaultSchema(tt.driver, tt.connectedDB); got != tt.want {
				t.Errorf("IsDefaultSchema() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTableReferenceAutocompleteInsert(t *testing.T) {
	tests := []struct {
		name        string
		ref         db.TableReference
		driver      db.DriverNameType
		connectedDB string
		want        string
	}{
		{"default schema inserts name only", db.TableReference{Schema: "public", Name: "users"}, db.DriverName.Postgres, "mydb", "users"},
		{
			"non-default schema inserts qualified name",
			db.TableReference{Schema: "myschema", Name: "orders"},
			db.DriverName.Postgres,
			"mydb",
			"myschema.orders",
		},
		{"mysql default schema inserts name only", db.TableReference{Schema: "mydb", Name: "users"}, db.DriverName.MySQL, "mydb", "users"},
		{"mysql non-default inserts qualified", db.TableReference{Schema: "other", Name: "users"}, db.DriverName.MySQL, "mydb", "other.users"},
		{"sqlite no schema inserts name only", db.TableReference{Schema: "", Name: "items"}, db.DriverName.SQLite, "", "items"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ref.AutocompleteInsert(tt.driver, tt.connectedDB); got != tt.want {
				t.Errorf("AutocompleteInsert() = %q, want %q", got, tt.want)
			}
		})
	}
}
