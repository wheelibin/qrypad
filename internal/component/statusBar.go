package component

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/theme"
)

//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
type StatusBarModel struct {
	width            int
	height           int
	selectedDatabase string
	copiedTextInfo   string
}

func NewStatusBarModel(selectedDatabase string) StatusBarModel {
	return StatusBarModel{selectedDatabase: selectedDatabase}
}

func (m StatusBarModel) Init() tea.Cmd {
	return nil
}

func (m StatusBarModel) Update(_ tea.Msg) (StatusBarModel, tea.Cmd) {
	var cmds []tea.Cmd

	return m, tea.Batch(cmds...)
}

func (m *StatusBarModel) SetSelectedDatabase(dbName string) {
	m.selectedDatabase = dbName
}

func (m *StatusBarModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *StatusBarModel) SetCopiedTextInfo(info string) {
	m.copiedTextInfo = info
}

func (m StatusBarModel) View() string {
	containerStyle := lipgloss.NewStyle().
		Background(theme.GetTheme().StatusBar.BG).
		Foreground(theme.GetTheme().StatusBar.FG)

	selectedDatabaseStyle := containerStyle
	helpTextStyle := containerStyle
	copiedTextInfoStyle := containerStyle.Foreground(theme.GetTheme().PanelTitleActive.BG)

	containerStyle = containerStyle.
		Width(m.width).
		Height(m.height).
		Padding(0, 2)

	selectedDatabase := selectedDatabaseStyle.
		Italic(true).
		Bold(true).
		Render(" " + m.selectedDatabase + " ")

	var helpText string
	if keys.DefaultKeyMap.SwitchDatabase.Enabled() {
		helpText = helpTextStyle.Render(fmt.Sprintf(" [%s] to switch database", keys.DefaultKeyMap.SwitchDatabase.Help().Key))
	}

	var copiedTextInfo string
	if m.copiedTextInfo != "" {
		copiedTextInfo = copiedTextInfoStyle.
			AlignHorizontal(lipgloss.Right).
			Width(m.width - lipgloss.Width(selectedDatabase+helpText) - 4).
			Render(fmt.Sprintf(`copied: "%s"`, m.copiedTextInfo))
	}

	return containerStyle.Render(lipgloss.JoinHorizontal(lipgloss.Center, selectedDatabase+helpText, copiedTextInfo))
}
