package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
	"github.com/wheelibin/qrypad/internal/db"
)

func TestTableInfoPanelView(t *testing.T) {
	setupViewTest(t)

	tableInfoData := &db.Data{
		Columns: []string{"name", "type", "nullable"},
		Rows: []map[string]any{
			{"name": "id", "type": "integer", "nullable": "false"},
			{"name": "email", "type": "text", "nullable": "false"},
			{"name": "created_at", "type": "timestamp", "nullable": "true"},
		},
	}

	t.Run("inactive", func(t *testing.T) {
		m := component.NewTableInfoPanelModel()
		m.SetActive(false)
		m.SetSize(80, 24)
		assertGolden(t, "TableInfoPanel_inactive", m.View())
	})

	t.Run("active_cols", func(t *testing.T) {
		m := component.NewTableInfoPanelModel()
		m.SetActive(true)
		m.SetActiveTabIndex(component.TableInfoTabIndexColumns)
		m.SetData(tableInfoData)
		m.SetSize(80, 24)
		assertGolden(t, "TableInfoPanel_active_cols", m.View())
	})

	t.Run("active_inds", func(t *testing.T) {
		m := component.NewTableInfoPanelModel()
		m.SetActive(true)
		m.SetActiveTabIndex(component.TableInfoTabIndexIndexes)
		m.SetSize(80, 24)
		assertGolden(t, "TableInfoPanel_active_inds", m.View())
	})

	t.Run("active_cons", func(t *testing.T) {
		m := component.NewTableInfoPanelModel()
		m.SetActive(true)
		m.SetActiveTabIndex(component.TableInfoTabIndexConstraints)
		m.SetSize(80, 24)
		assertGolden(t, "TableInfoPanel_active_cons", m.View())
	})

	t.Run("active_cols_row_borders", func(t *testing.T) {
		setRowBorders(t)
		m := component.NewTableInfoPanelModel()
		m.SetActive(true)
		m.SetActiveTabIndex(component.TableInfoTabIndexColumns)
		m.SetData(tableInfoData)
		m.SetSize(80, 24)
		assertGolden(t, "TableInfoPanel_active_cols_row_borders", m.View())
	})
}
