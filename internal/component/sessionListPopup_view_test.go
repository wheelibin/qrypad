package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
)

func TestSessionListPopupView(t *testing.T) {
	setupViewTest(t)

	t.Run("basic", func(t *testing.T) {
		m := component.NewSessionListPopupModel()
		m.SetSize(40, 15)
		m.SetEntries([]component.SessionListEntry{
			{ConnName: "local", DBName: "mydb", IsActive: true},
			{ConnName: "staging", DBName: "staging_db"},
			{ConnName: "production", DBName: "prod_db"},
		})
		assertGolden(t, "SessionListPopup_basic", m.View())
	})
}
