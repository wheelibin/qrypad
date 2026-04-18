package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
	"github.com/wheelibin/qrypad/internal/db"
)

func TestTitleBarView(t *testing.T) {
	setupViewTest(t)

	t.Run("basic", func(t *testing.T) {
		m := component.NewTitleBarModel("prod", db.ConnectionConfig{
			Driver: "postgres",
			User:   "admin",
			Host:   "db.example.com",
			Port:   5432,
		})
		m.SetSize(80, 1)
		assertGolden(t, "TitleBar_basic", m.View())
	})

	t.Run("sqlite", func(t *testing.T) {
		m := component.NewTitleBarModel("local", db.ConnectionConfig{
			Driver: db.DriverName.SQLite,
		})
		m.SetSize(80, 1)
		assertGolden(t, "TitleBar_sqlite", m.View())
	})

	t.Run("after_setters_called", func(t *testing.T) {
		m := component.NewTitleBarModel("prod", db.ConnectionConfig{
			Driver: "postgres",
			User:   "admin",
			Host:   "db.example.com",
			Port:   5432,
		})
		m.SetSize(80, 1)
		m.SetConnectionName("staging")
		m.SetConn(db.ConnectionConfig{
			Driver: "postgres",
			User:   "readonly",
			Host:   "staging.example.com",
			Port:   5433,
		})
		assertGolden(t, "TitleBar_after_setters_called", m.View())
	})
}
