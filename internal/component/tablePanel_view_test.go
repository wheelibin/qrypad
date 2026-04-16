package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
	"github.com/wheelibin/qrypad/internal/db"
)

func TestTablePanelView(t *testing.T) {
	setupViewTest(t)

	t.Run("inactive", func(t *testing.T) {
		m := component.NewTablePanelModel()
		m.SetActive(false)
		m.SetSize(80, 24)
		assertGolden(t, "TablePanel_inactive", m.View())
	})

	t.Run("active", func(t *testing.T) {
		m := component.NewTablePanelModel()
		m.SetActive(true)
		m.SetData(&db.Data{
			Columns: []string{"name", "rows"},
			Rows: []map[string]any{
				{"name": "users", "rows": "100"},
				{"name": "orders", "rows": "250"},
				{"name": "products", "rows": "50"},
			},
		})
		m.SetSize(80, 24)
		assertGolden(t, "TablePanel_active", m.View())
	})

	t.Run("with_schema", func(t *testing.T) {
		m := component.NewTablePanelModel()
		m.SetActive(true)
		m.SetData(&db.Data{
			Columns: []string{"schema", "name", "rows"},
			Rows: []map[string]any{
				{"schema": "public", "name": "users", "rows": "100"},
				{"schema": "public", "name": "orders", "rows": "250"},
				{"schema": "myschema", "name": "products", "rows": "50"},
			},
		})
		m.SetSize(80, 24)
		assertGolden(t, "TablePanel_with_schema", m.View())
	})

	t.Run("views_tab", func(t *testing.T) {
		m := component.NewTablePanelModel()
		m.SetActive(true)
		m.SetActiveTabIndex(component.TablePanelTabIndexViews)
		m.SetSize(80, 24)
		assertGolden(t, "TablePanel_views_tab", m.View())
	})
}
