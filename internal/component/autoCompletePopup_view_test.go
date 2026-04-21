package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
)

func TestAutoCompletePopupView(t *testing.T) {
	setupViewTest(t)

	t.Run("basic", func(t *testing.T) {
		m := component.NewAutoCompletePopupModel()
		m.SetActive(true)
		_ = m.SetItems([]string{"SELECT", "FROM", "WHERE", "INSERT"})
		assertGolden(t, "AutoCompletePopup_basic", m.View())
	})
}
