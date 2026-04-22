package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
)

func TestExportFormatPopupView(t *testing.T) {
	setupViewTest(t)

	t.Run("basic", func(t *testing.T) {
		m := component.NewExportFormatPopupModel()
		m.SetSize(40, 5)
		assertGolden(t, "ExportFormatPopup_basic", m.View())
	})
}
