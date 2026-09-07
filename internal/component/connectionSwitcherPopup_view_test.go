package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
	"github.com/wheelibin/qrypad/internal/db"
)

func TestConnectionSwitcherPopupView(t *testing.T) {
	setupViewTest(t)

	t.Run("basic", func(t *testing.T) {
		m := component.NewConnectionSwitcherPopupModel()
		m.SetSize(50, 15)
		m.SetData(&db.Data{
			Columns: []string{"name", "driver", "host"},
			Rows: []map[string]any{
				{"name": "local", "driver": "postgres", "host": "localhost"},
				{"name": "staging", "driver": "postgres", "host": "staging.example.com"},
				{"name": "production", "driver": "postgres", "host": "prod.example.com"},
			},
		})
		assertGolden(t, "ConnectionSwitcherPopup_basic", m.View())
	})
}
