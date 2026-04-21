//go:build integration

package db_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/wheelibin/qrypad/internal/db"
)

func TestIntegration_MySQL_Databases(t *testing.T) {
	conn := openMySQLConn(t)
	data, err := db.ExecuteQuery(context.Background(), conn, conn.Queries.Databases())
	if err != nil {
		t.Fatalf("ExecuteQuery: %v", err)
	}
	if len(data.Rows) == 0 {
		t.Fatal("Databases: expected at least one row, got none")
	}
	if !rowsContainName(data.Rows, "qrypad") {
		t.Errorf("Databases: expected 'qrypad' in results, got %v", columnNames(data.Rows))
	}
}

func TestIntegration_MySQL_SchemaTables(t *testing.T) {
	conn := openMySQLConn(t)
	data, err := db.ExecuteQuery(context.Background(), conn, conn.Queries.SchemaTables())
	if err != nil {
		t.Fatalf("ExecuteQuery: %v", err)
	}
	if !rowsContainAllNames(data.Rows, "species", "staff", "animals", "adoptions") {
		t.Errorf("SchemaTables: expected all seeded tables; got names=%v", columnNames(data.Rows))
	}
	for _, row := range data.Rows {
		if schema, ok := row["schema"].(string); !ok || schema == "" {
			t.Errorf("SchemaTables row %+v: expected non-empty schema column", row)
			break
		}
	}
}

func TestIntegration_MySQL_SchemaViews(t *testing.T) {
	conn := openMySQLConn(t)
	data, err := db.ExecuteQuery(context.Background(), conn, conn.Queries.SchemaViews())
	if err != nil {
		t.Fatalf("ExecuteQuery: %v", err)
	}
	if !rowsContainName(data.Rows, "adopted_animals") {
		t.Errorf("SchemaViews: expected 'adopted_animals' view; got %v", columnNames(data.Rows))
	}
	for _, row := range data.Rows {
		if name, ok := row["name"].(string); ok && name == "adopted_animals" {
			schema, _ := row["schema"].(string)
			if schema != "qrypad" {
				t.Errorf("SchemaViews adopted_animals: expected schema 'qrypad', got %q", schema)
			}
			return
		}
	}
}

func TestIntegration_MySQL_TableColumns(t *testing.T) {
	conn := openMySQLConn(t)
	ref := db.TableReference{Schema: "qrypad", Name: "animals"}
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
	// Verify nullable column for 'id' is NOT NULL.
	for _, row := range data.Rows {
		if name, _ := row["name"].(string); name == "id" {
			nullable, _ := row["nullable"].(string)
			if nullable != "NOT NULL" {
				t.Errorf("TableColumns(animals).id: expected NOT NULL, got %q", nullable)
			}
		}
	}
}

func TestIntegration_MySQL_TableConstraints(t *testing.T) {
	conn := openMySQLConn(t)
	ref := db.TableReference{Schema: "qrypad", Name: "animals"}
	data, err := db.ExecuteQuery(context.Background(), conn, conn.Queries.TableConstraints(ref))
	if err != nil {
		t.Fatalf("ExecuteQuery: %v", err)
	}
	if len(data.Rows) == 0 {
		t.Fatal("TableConstraints(animals): expected at least one constraint")
	}
	// MySQL returns type as string: "PRIMARY KEY", "FOREIGN KEY", "CHECK", etc.
	hasPrimary := false
	hasForeign := false
	for _, row := range data.Rows {
		ctype, _ := row["type"].(string)
		if strings.Contains(ctype, "PRIMARY") {
			hasPrimary = true
		}
		if strings.Contains(ctype, "FOREIGN") {
			hasForeign = true
		}
	}
	if !hasPrimary {
		t.Error("TableConstraints(animals): expected a PRIMARY KEY constraint")
	}
	if !hasForeign {
		t.Error("TableConstraints(animals): expected a FOREIGN KEY constraint")
	}
}

func TestIntegration_MySQL_TableRows(t *testing.T) {
	conn := openMySQLConn(t)
	ref := db.TableReference{Schema: "qrypad", Name: "animals"}

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

func TestIntegration_MySQL_TableIndexes(t *testing.T) {
	conn := openMySQLConn(t)
	ref := db.TableReference{Schema: "qrypad", Name: "animals"}
	data, err := conn.Queries.TableIndexes(context.Background(), conn, ref)
	if err != nil {
		t.Fatalf("TableIndexes: %v", err)
	}
	if len(data.Rows) == 0 {
		t.Fatal("TableIndexes(animals): expected at least one row (PK index)")
	}
	foundIDIndex := false
	for _, row := range data.Rows {
		cols, _ := row["cols"].(string)
		if cols == "id" {
			foundIDIndex = true
			break
		}
	}
	if !foundIDIndex {
		t.Errorf("TableIndexes(animals): expected an index with cols='id'; got rows=%+v", data.Rows)
	}
}

func TestIntegration_MySQL_PrimaryKeyColumns(t *testing.T) {
	conn := openMySQLConn(t)
	ref := db.TableReference{Schema: "qrypad", Name: "animals"}
	cols, err := conn.Queries.PrimaryKeyColumns(context.Background(), conn, ref)
	if err != nil {
		t.Fatalf("PrimaryKeyColumns: %v", err)
	}
	want := []string{"id"}
	if !slices.Equal(cols, want) {
		t.Errorf("PrimaryKeyColumns(animals): got %v, want %v", cols, want)
	}
}

func TestIntegration_MySQL_ConnectPopulatesQueries(t *testing.T) {
	conn := openMySQLConn(t)
	if conn.Queries == nil {
		t.Fatal("Connect did not populate Queries")
	}
	if sql := conn.Queries.Databases(); sql == "" {
		t.Error("Queries.Databases() returned empty string; expected MySQL SQL")
	}
}
