package component_test

import (
	"testing"
	"time"

	"github.com/wheelibin/qrypad/internal/component"
	"github.com/wheelibin/qrypad/internal/db"
)

func TestResultsPanelView(t *testing.T) {
	setupViewTest(t)

	t.Run("empty", func(t *testing.T) {
		m := component.NewResultsPanelModel()
		m.SetSize(80, 24)
		assertGolden(t, "ResultsPanel_empty", m.View())
	})

	t.Run("with_data", func(t *testing.T) {
		m := component.NewResultsPanelModel()
		m.SetSize(80, 24)
		data := &db.Data{
			Columns:     []string{"id", "name", "email"},
			ColumnTypes: []string{"number", "string", "string"},
			Rows: []map[string]any{
				{"id": "1", "name": "Alice", "email": "alice@example.com"},
				{"id": "2", "name": "Bob", "email": "bob@example.com"},
			},
			QueryTime: 42 * time.Millisecond,
		}
		m.SetData(data)
		assertGolden(t, "ResultsPanel_with_data", m.View())
	})

	t.Run("with_data_row_borders", func(t *testing.T) {
		setRowBorders(t)
		m := component.NewResultsPanelModel()
		m.SetSize(80, 24)
		data := &db.Data{
			Columns:     []string{"id", "name", "email"},
			ColumnTypes: []string{"number", "string", "string"},
			Rows: []map[string]any{
				{"id": "1", "name": "Alice", "email": "alice@example.com"},
				{"id": "2", "name": "Bob", "email": "bob@example.com"},
			},
			QueryTime: 42 * time.Millisecond,
		}
		m.SetData(data)
		assertGolden(t, "ResultsPanel_with_data_row_borders", m.View())
	})
}
