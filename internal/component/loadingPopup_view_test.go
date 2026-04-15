package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
)

func TestLoadingPopupView(t *testing.T) {
	setupViewTest(t)

	t.Run("basic", func(t *testing.T) {
		m := component.NewLoadingPopupModel()
		m.SetSize(40, 10)
		assertGolden(t, "LoadingPopup_basic", m.View())
	})
}
