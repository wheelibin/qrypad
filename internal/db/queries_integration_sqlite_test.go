//go:build integration

package db_test

import (
	"context"
	"slices"
	"testing"

	"github.com/wheelibin/qrypad/internal/db"
)

func TestIntegration_SQLite_Databases(t *testing.T) {
	conn := openSQLiteConn(t)
	// SQLite has no concept of listing databases; Databases() returns "".
	got := conn.Queries.Databases()
	if got != "" {
		t.Errorf("Databases: expected empty string for SQLite, got %q", got)
	}
}

func TestIntegration_SQLite_SchemaTables(t *testing.T) {
	conn := openSQLiteConn(t)
	data, err := db.ExecuteQuery(context.Background(), conn, conn.Queries.SchemaTables())
	if err != nil {
		t.Fatalf("ExecuteQuery: %v", err)
	}
	if !rowsContainAllNames(data.Rows, "species", "staff", "animals", "adoptions") {
		t.Errorf("SchemaTables: expected all seeded tables; got names=%v", columnNames(data.Rows))
	}
}

func TestIntegration_SQLite_SchemaViews(t *testing.T) {
	conn := openSQLiteConn(t)
	data, err := db.ExecuteQuery(context.Background(), conn, conn.Queries.SchemaViews())
	if err != nil {
		t.Fatalf("ExecuteQuery: %v", err)
	}
	if !rowsContainName(data.Rows, "adopted_animals") {
		t.Errorf("SchemaViews: expected 'adopted_animals' view; got %v", columnNames(data.Rows))
	}
}

func TestIntegration_SQLite_TableColumns(t *testing.T) {
	conn := openSQLiteConn(t)
	ref := db.TableReference{Name: "animals"}
	data, err := db.ExecuteQuery(context.Background(), conn, conn.Queries.TableColumns(ref))
	if err != nil {
		t.Fatalf("ExecuteQuery: %v", err)
	}
	got := columnNames(data.Rows)
	want := []string{"id", "name", "species_id", "age", "arrived_on", "adopted"}
	for _, expected := range want {
		if !slices.Contains(got, expected) {
			t.Errorf("TableColumns(animals): expected column %q in results %v", expected, got)
		}
	}
	// Verify 'id' row has pk=1 (pragma_table_info returns pk as string via fetchRows).
	for _, row := range data.Rows {
		if name, _ := row["name"].(string); name == "id" {
			pk, _ := row["pk"].(string)
			if pk != "1" {
				t.Errorf("TableColumns(animals).id: expected pk='1', got %q", pk)
			}
		}
	}
}

func TestIntegration_SQLite_TableConstraints(t *testing.T) {
	conn := openSQLiteConn(t)
	ref := db.TableReference{Name: "animals"}
	data, err := db.ExecuteQuery(context.Background(), conn, conn.Queries.TableConstraints(ref))
	if err != nil {
		t.Fatalf("ExecuteQuery: %v", err)
	}
	// SQLite's pragma_foreign_key_list returns only FK rows; animals has FK to species.
	if len(data.Rows) == 0 {
		t.Fatal("TableConstraints(animals): expected at least one FK row from pragma_foreign_key_list")
	}
	// pragma_foreign_key_list has a "table" column with the referenced table name.
	foundSpeciesFK := false
	for _, row := range data.Rows {
		refTable, _ := row["table"].(string)
		if refTable == "species" {
			foundSpeciesFK = true
			break
		}
	}
	if !foundSpeciesFK {
		t.Errorf("TableConstraints(animals): expected FK to 'species'; got rows=%+v", data.Rows)
	}
}

func TestIntegration_SQLite_TableRows(t *testing.T) {
	conn := openSQLiteConn(t)
	ref := db.TableReference{Name: "animals"}

	ascData, err := db.ExecuteQuery(context.Background(), conn, conn.Queries.TableRows(ref, []string{"id"}, "ASC"))
	if err != nil {
		t.Fatalf("TableRows ASC: %v", err)
	}
	if len(ascData.Rows) == 0 {
		t.Fatal("TableRows ASC: expected rows")
	}
	firstID, _ := ascData.Rows[0]["id"].(string)
	if firstID != "1" {
		t.Errorf("TableRows ASC: expected first id='1', got %q", firstID)
	}

	descData, err := db.ExecuteQuery(context.Background(), conn, conn.Queries.TableRows(ref, []string{"id"}, "DESC"))
	if err != nil {
		t.Fatalf("TableRows DESC: %v", err)
	}
	if len(descData.Rows) == 0 {
		t.Fatal("TableRows DESC: expected rows")
	}
	firstDescID, _ := descData.Rows[0]["id"].(string)
	if firstDescID <= "100" {
		t.Errorf("TableRows DESC: expected first id > '100', got %q", firstDescID)
	}
}

func TestIntegration_SQLite_TableIndexes(t *testing.T) {
	conn := openSQLiteConn(t)
	ref := db.TableReference{Name: "animals"}
	data, err := conn.Queries.TableIndexes(context.Background(), conn, ref)
	if err != nil {
		t.Fatalf("TableIndexes: %v", err)
	}
	// For INTEGER PRIMARY KEY AUTOINCREMENT, SQLite may or may not surface an
	// index via pragma_index_list. Assert loose — no error and any rows present
	// are well-formed with name + cols fields.
	for _, row := range data.Rows {
		if _, ok := row["name"]; !ok {
			t.Errorf("TableIndexes row missing 'name': %+v", row)
		}
		if _, ok := row["cols"]; !ok {
			t.Errorf("TableIndexes row missing 'cols': %+v", row)
		}
	}
}

func TestIntegration_SQLite_PrimaryKeyColumns(t *testing.T) {
	conn := openSQLiteConn(t)
	ref := db.TableReference{Name: "animals"}
	cols, err := conn.Queries.PrimaryKeyColumns(context.Background(), conn, ref)
	if err != nil {
		t.Fatalf("PrimaryKeyColumns: %v", err)
	}
	want := []string{"id"}
	if !slices.Equal(cols, want) {
		t.Errorf("PrimaryKeyColumns(animals): got %v, want %v", cols, want)
	}
}

func TestIntegration_SQLite_AllTableColumns(t *testing.T) {
	conn := openSQLiteConn(t)
	sql := conn.Queries.AllTableColumns()
	if sql != "" {
		t.Fatalf("SQLite AllTableColumns should return empty string, got %q", sql)
	}
}

func TestIntegration_SQLite_SequentialColumnPreload(t *testing.T) {
	conn := openSQLiteConn(t)

	// Get table list
	data, err := db.ExecuteQuery(context.Background(), conn, conn.Queries.SchemaTables())
	if err != nil {
		t.Fatalf("SchemaTables: %v", err)
	}

	refs := make([]db.TableReference, 0, len(data.Rows))
	for _, row := range data.Rows {
		name, _ := row["name"].(string)
		refs = append(refs, db.TableReference{Name: name})
	}

	if len(refs) == 0 {
		t.Fatal("no tables in test SQLite database")
	}

	// Simulate sequential preload: fetch columns for each table
	cache := db.NewSchemaCache()
	for _, ref := range refs {
		query := conn.Queries.TableColumns(ref)
		colData, err := db.ExecuteQuery(context.Background(), conn, query)
		if err != nil {
			t.Fatalf("TableColumns(%s): %v", ref.Name, err)
		}
		cache.SetTableInfo(ref, "cols", colData)
	}

	// Verify cache is populated for all tables
	for _, ref := range refs {
		cached, ok := cache.GetTableInfo(ref, "cols")
		if !ok {
			t.Errorf("expected cache hit for table %q after preload", ref.Name)
			continue
		}
		if len(cached.Rows) == 0 {
			t.Errorf("table %q has no columns in cache", ref.Name)
		}
	}
}

func TestIntegration_SQLite_ConnectPopulatesQueries(t *testing.T) {
	conn := openSQLiteConn(t)
	if conn.Queries == nil {
		t.Fatal("Connect did not populate Queries")
	}
	// For SQLite, Databases() returns "" — assert that's what we get.
	if sql := conn.Queries.Databases(); sql != "" {
		t.Errorf("Queries.Databases() expected empty string for SQLite, got %q", sql)
	}
}
