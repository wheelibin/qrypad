package component_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
)

func TestQueryPanelView(t *testing.T) {
	t.Run("inactive", func(t *testing.T) {
		setupViewTest(t)
		m := component.NewQueryPanelModel("test-conn", "", true, false)
		m.SetSize(80, 24)
		assertGolden(t, "QueryPanel_inactive", m.View())
	})

	t.Run("active", func(t *testing.T) {
		setupViewTest(t)
		m := component.NewQueryPanelModel("test-conn", "", true, false)
		m.SetActive(true)
		m.SetValue("SELECT id FROM users;")
		m.SetSize(80, 24)
		assertGolden(t, "QueryPanel_active", m.View())
	})

	t.Run("dirty", func(t *testing.T) {
		setupViewTest(t)
		m := component.NewQueryPanelModel("test-conn", "", true, false)
		m.SetActive(true)
		m.SetDirty(true)
		m.SetValue("SELECT 1;")
		m.SetSize(80, 24)
		assertGolden(t, "QueryPanel_dirty", m.View())
	})

	t.Run("with_filename", func(t *testing.T) {
		setupViewTest(t)
		home, _ := os.UserHomeDir()
		m := component.NewQueryPanelModel("test-conn", "", true, false)
		m.SetActive(true)
		m.SetFilename(filepath.Join(home, ".local", "share", "qrypad", "test-conn.sql"))
		m.SetSize(80, 24)
		assertGolden(t, "QueryPanel_with_filename", m.View())
	})

	t.Run("without_filename", func(t *testing.T) {
		setupViewTest(t)
		m := component.NewQueryPanelModel("test-conn", "", true, false)
		m.SetActive(true)
		m.SetSize(80, 24)
		assertGolden(t, "QueryPanel_without_filename", m.View())
	})
}
