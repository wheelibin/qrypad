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
		}, []string{"id", "name", "email"}, nil)
		assertGolden(t, "ResultRowPopup_basic", m.View())
	})

	t.Run("multiline_long_value", func(t *testing.T) {
		m := component.NewResultRowPopupModel()
		m.SetSize(60, 20)
		m.SetData(map[string]any{
			"id":   "42",
			"data": `{"user":{"name":"Alice","email":"alice@example.com","role":"admin","preferences":{"theme":"dark","language":"en","notifications":true}}}`,
		}, []string{"id", "data"}, nil)
		assertGolden(t, "ResultRowPopup_multiline", m.View())
	})
}
