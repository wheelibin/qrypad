package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
)

func TestPasswordPopupView(t *testing.T) {
	setupViewTest(t)

	t.Run("basic", func(t *testing.T) {
		m := component.NewPasswordPopupModel()
		m.SetSize(40, 10)
		assertGolden(t, "PasswordPopup_basic", m.View())
	})
}
