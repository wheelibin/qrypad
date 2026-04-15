package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
)

func TestErrorPopupView(t *testing.T) {
	setupViewTest(t)

	t.Run("basic", func(t *testing.T) {
		m := component.NewErrorPopupModel()
		m.SetText("query syntax error near ';'")
		m.SetIsConnectionError(false)
		m.SetSize(60, 10)
		assertGolden(t, "ErrorPopup_basic", m.View())
	})

	t.Run("connection_error", func(t *testing.T) {
		m := component.NewErrorPopupModel()
		m.SetText("connection refused")
		m.SetIsConnectionError(true)
		m.SetSize(60, 15)
		assertGolden(t, "ErrorPopup_connection_error", m.View())
	})
}
