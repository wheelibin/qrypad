package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
)

func TestQueryPanelView(t *testing.T) {
	t.Run("inactive", func(t *testing.T) {
		setupViewTest(t)
		m := component.NewQueryPanelModel("test-conn", false)
		m.SetSize(80, 24)
		assertGolden(t, "QueryPanel_inactive", m.View())
	})

	t.Run("active", func(t *testing.T) {
		setupViewTest(t)
		m := component.NewQueryPanelModel("test-conn", false)
		m.SetActive(true)
		m.SetValue("SELECT id FROM users;")
		m.SetSize(80, 24)
		assertGolden(t, "QueryPanel_active", m.View())
	})

	t.Run("dirty", func(t *testing.T) {
		setupViewTest(t)
		m := component.NewQueryPanelModel("test-conn", false)
		m.SetActive(true)
		m.SetDirty(true)
		m.SetValue("SELECT 1;")
		m.SetSize(80, 24)
		assertGolden(t, "QueryPanel_dirty", m.View())
	})
}
