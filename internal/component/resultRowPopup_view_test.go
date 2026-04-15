package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
)

func TestResultRowPopupView(t *testing.T) {
	setupViewTest(t)

	t.Run("basic", func(t *testing.T) {
		m := component.NewResultRowPopupModel()
		m.SetSize(60, 15)
		m.SetData(map[string]any{
			"id":    "1",
			"name":  "Alice",
			"email": "alice@example.com",
		})
		assertGolden(t, "ResultRowPopup_basic", m.View())
	})
}
