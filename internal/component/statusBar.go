package component

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/colour"
	"github.com/wheelibin/qrypad/internal/keys"
)

type StatusBarModel struct {
	width            int
	height           int
	text             string
	selectedDatabase string
	copiedTextInfo   string
}

func NewStatusBarModel(selectedDatabase string) StatusBarModel {
	return StatusBarModel{selectedDatabase: selectedDatabase}
}

func (m StatusBarModel) Init() tea.Cmd {
	return nil
}

func (m StatusBarModel) Update(msg tea.Msg) (StatusBarModel, tea.Cmd) {
	// log.Println("statusBar.model::Update", msg)
	var (
		cmds []tea.Cmd
	)

	return m, tea.Batch(cmds...)
}

func (m *StatusBarModel) SetSelectedDatabase(dbName string) {
	m.selectedDatabase = dbName
}

func (m *StatusBarModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *StatusBarModel) SetText(text string) {
	m.text = text
}

func (m *StatusBarModel) SetCopiedTextInfo(info string) {
	m.copiedTextInfo = info
}

func (m StatusBarModel) View() string {
	containerStyle := lipgloss.NewStyle().
		Background(colour.GetTheme().StatusBar.BG).
		Foreground(colour.GetTheme().StatusBar.FG)

	selectedDatabaseStyle := containerStyle
	helpTextStyle := containerStyle
	copiedTextInfoStyle := containerStyle.Foreground(colour.GetTheme().PanelTitleActive.BG)

	containerStyle = containerStyle.
		Width(m.width).
		Height(m.height).
		Padding(0, 2)

	selectedDatabase := selectedDatabaseStyle.
		Italic(true).
		Bold(true).
		Render(" " + m.selectedDatabase + " ")

	helpText := helpTextStyle.Render(fmt.Sprintf(" [%s] to switch", keys.DefaultKeyMap.SwitchDatabase.Help().Key))

	var copiedTextInfo string
	if m.copiedTextInfo != "" {
		copiedTextInfo = copiedTextInfoStyle.
			AlignHorizontal(lipgloss.Right).
			Width(m.width - lipgloss.Width(selectedDatabase+helpText) - 4).
			Render(fmt.Sprintf("copied: %s", m.copiedTextInfo))
	}

	return containerStyle.Render(lipgloss.JoinHorizontal(lipgloss.Center, selectedDatabase+helpText, copiedTextInfo))
}
