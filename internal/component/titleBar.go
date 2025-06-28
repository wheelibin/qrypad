package component

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/theme"
	"github.com/wheelibin/qrypad/internal/constants"
	"github.com/wheelibin/qrypad/internal/keys"
)

type TitlBarModel struct {
	width          int
	height         int
	text           string
	connectionName string
}

func NewTitlBarModel(connectionName string) TitlBarModel {
	return TitlBarModel{connectionName: connectionName}
}

func (m TitlBarModel) Init() tea.Cmd {
	return nil
}

func (m TitlBarModel) Update(msg tea.Msg) (TitlBarModel, tea.Cmd) {
	var cmds []tea.Cmd

	return m, tea.Batch(cmds...)
}

func (m *TitlBarModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *TitlBarModel) SetText(text string) {
	m.text = text
}

func (m TitlBarModel) View() string {
	baseStyle := lipgloss.NewStyle().
		Background(theme.GetTheme().TitleBar.BG).
		Foreground(theme.GetTheme().TitleBar.FG)

	helpText := baseStyle.
		Width(16).
		Render(fmt.Sprintf("[%s] Show help", keys.DefaultKeyMap.Help.Help().Key))

	barStyle := baseStyle.
		Padding(0, 2).
		Width(m.width - lipgloss.Width(helpText)).
		Height(m.height).
		Bold(true)

	return lipgloss.JoinHorizontal(lipgloss.Left, barStyle.Render(fmt.Sprintf("QryPad - %s (connection: %s)", constants.AppDesc, m.connectionName)), helpText)
}
