package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
)

func TestStatusBarView(t *testing.T) {
	setupViewTest(t)

	t.Run("no_db", func(t *testing.T) {
		m := component.NewStatusBarModel("")
		m.SetSize(80, 1)
		assertGolden(t, "StatusBar_no_db", m.View())
	})

	t.Run("with_db", func(t *testing.T) {
		m := component.NewStatusBarModel("mydb")
		m.SetSize(80, 1)
		assertGolden(t, "StatusBar_with_db", m.View())
	})

	t.Run("with_copied_text", func(t *testing.T) {
		m := component.NewStatusBarModel("mydb")
		m.SetCopiedTextInfo("some_value")
		m.SetSize(80, 1)
		assertGolden(t, "StatusBar_with_copied_text", m.View())
	})
}
