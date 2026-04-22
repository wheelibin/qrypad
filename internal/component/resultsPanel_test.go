package component_test

import (
	"testing"
	"time"

	"github.com/wheelibin/qrypad/internal/component"
	"github.com/wheelibin/qrypad/internal/db"
)

func makeResultsPanelWithData() component.ResultsPanelModel {
	m := component.NewResultsPanelModel()
	m.SetSize(80, 20)
	m.SetData(&db.Data{
		Columns: []string{"id", "name"},
		Rows: []map[string]any{
			{"id": 1, "name": "alice"},
			{"id": 2, "name": "bob"},
		},
		QueryTime: time.Millisecond,
	})
	return m
}

func TestResultsPanel_GetColumns_PreservesOrder(t *testing.T) {
	m := makeResultsPanelWithData()
	cols := m.GetColumns()
	want := []string{"id", "name"}
	if len(cols) != len(want) {
		t.Fatalf("len(cols) = %d, want %d", len(cols), len(want))
	}
	for i := range cols {
		if cols[i] != want[i] {
			t.Errorf("cols[%d] = %q, want %q", i, cols[i], want[i])
		}
	}
}

func TestResultsPanel_GetExportRows_NoFilter_ReturnsAll(t *testing.T) {
	m := makeResultsPanelWithData()
	rows := m.GetExportRows()
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0]["id"] != 1 || rows[0]["name"] != "alice" {
		t.Errorf("rows[0] = %v, want {id:1 name:alice}", rows[0])
	}
	if rows[1]["id"] != 2 || rows[1]["name"] != "bob" {
		t.Errorf("rows[1] = %v, want {id:2 name:bob}", rows[1])
	}
}

func TestResultsPanel_GetExportRows_Empty(t *testing.T) {
	m := component.NewResultsPanelModel()
	m.SetSize(80, 20)
	rows := m.GetExportRows()
	if len(rows) != 0 {
		t.Fatalf("expected empty slice, got %d rows", len(rows))
	}
}

func TestResultsPanel_SetData_CopiesColumns(t *testing.T) {
	m := component.NewResultsPanelModel()
	m.SetSize(80, 20)
	input := []string{"id", "name"}
	m.SetData(&db.Data{Columns: input, Rows: nil})
	input[0] = "mutated"
	if got := m.GetColumns(); got[0] != "id" {
		t.Errorf("SetData did not defensively copy columns: got[0] = %q", got[0])
	}
}
