package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
)

func TestHelpPopupView(t *testing.T) {
	setupViewTest(t)

	t.Run("basic", func(t *testing.T) {
		m := component.NewHelpPopupModel()
		m.SetSize(60, 20)
		assertGolden(t, "HelpPopup_basic", m.View())
	})
}
