package component_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/component"
)

func keyPress(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
}

// collectMsgs runs cmd and any batched cmds, returning every message produced.
func collectMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	var msgs []tea.Msg
	for _, c := range batch {
		msgs = append(msgs, collectMsgs(c)...)
	}
	return msgs
}

// Typing "x" into the session filter must not close the highlighted session.
func TestSessionListPopup_TypingInFilterDoesNotTriggerKeys(t *testing.T) {
	m := component.NewSessionListPopupModel()
	m.SetSize(40, 15)
	m.SetEntries([]component.SessionListEntry{
		{ConnName: "xlocal", IsActive: true},
		{ConnName: "xstaging"}, // names contain "x" so rows stay visible while filtering
	})

	m, _ = m.Update(keyPress("/"))
	_, cmd := m.Update(keyPress("x"))

	for _, msg := range collectMsgs(cmd) {
		if _, ok := msg.(commands.SessionCloseMsg); ok {
			t.Fatalf("typing x in the filter closed a session")
		}
	}
}

// Typing "]" into the table filter must not switch tabs.
func TestTablePanel_TypingInFilterDoesNotSwitchTab(t *testing.T) {
	m := component.NewTablePanelModel()
	m.SetActive(true)

	m, _ = m.Update(keyPress("/"))
	m, _ = m.Update(keyPress("]"))

	if got := m.GetActiveTabIndex(); got != 0 {
		t.Fatalf("typing ] in the filter switched tab to %d", got)
	}
}
