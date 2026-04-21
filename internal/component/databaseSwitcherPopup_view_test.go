package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
	"github.com/wheelibin/qrypad/internal/db"
)

func TestDatabaseSwitcherPopupView(t *testing.T) {
	setupViewTest(t)

	t.Run("basic", func(t *testing.T) {
		m := component.NewDatabaseSwitcherPopupModel()
		m.SetSize(40, 15)
		m.SetData(&db.Data{
			Columns: []string{"name"},
			Rows: []map[string]any{
				{"name": "production"},
				{"name": "staging"},
				{"name": "development"},
			},
		})
		assertGolden(t, "DatabaseSwitcherPopup_basic", m.View())
	})
}
