package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
)

func TestQueryPanelInit(t *testing.T) {
	t.Run("returns nil when per-database mode and database name is empty", func(t *testing.T) {
		// singleQueryFile=false, databaseName="" means we don't yet know the database
		m := component.NewQueryPanelModel("test-conn", "", false, false)
		cmd := m.Init()
		if cmd != nil {
			t.Error("expected Init() to return nil when databaseName is empty in per-database mode")
		}
	})

	t.Run("returns cmd when single file mode even with empty database name", func(t *testing.T) {
		// singleQueryFile=true means we don't need a database name
		m := component.NewQueryPanelModel("test-conn", "", true, false)
		cmd := m.Init()
		if cmd == nil {
			t.Error("expected Init() to return a non-nil cmd in single file mode")
		}
	})

	t.Run("returns cmd when per-database mode and database name is set", func(t *testing.T) {
		m := component.NewQueryPanelModel("test-conn", "mydb", false, false)
		cmd := m.Init()
		if cmd == nil {
			t.Error("expected Init() to return a non-nil cmd when databaseName is set")
		}
	})
}

func TestQueryPanelSetters(t *testing.T) {
	t.Run("SetDatabaseName enables Init to return cmd", func(t *testing.T) {
		m := component.NewQueryPanelModel("test-conn", "", false, false)
		// Before setting database name, Init returns nil
		if m.Init() != nil {
			t.Fatal("expected nil before SetDatabaseName")
		}
		m.SetDatabaseName("mydb")
		// After setting database name, Init should return a cmd
		if m.Init() == nil {
			t.Error("expected non-nil cmd after SetDatabaseName")
		}
	})
}
