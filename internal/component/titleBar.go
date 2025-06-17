package component

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/colour"
	"github.com/wheelibin/qrypad/internal/constants"
	"github.com/wheelibin/qrypad/internal/keys"
)

type TitlBarModel struct {
	width   int
	height  int
	text    string
	dbAlias string
}

func NewTitlBarModel(dbAlias string) TitlBarModel {
	return TitlBarModel{dbAlias: dbAlias}
}

func (m TitlBarModel) Init() tea.Cmd {
	return nil
}

func (m TitlBarModel) Update(msg tea.Msg) (TitlBarModel, tea.Cmd) {
	var (
		cmds []tea.Cmd
	)

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
		Background(colour.GetTheme().TitleBar.BG).
		Foreground(colour.GetTheme().TitleBar.FG)

	helpText := baseStyle.
		Width(16).
		Render(fmt.Sprintf("[%s] Show help", keys.DefaultKeyMap.Help.Help().Key))

	barStyle := baseStyle.
		Padding(0, 2).
		Width(m.width - lipgloss.Width(helpText)).
		Height(m.height).
		Bold(true)

	return lipgloss.JoinHorizontal(lipgloss.Left, barStyle.Render(fmt.Sprintf("QryPad - %s (config: %s)", constants.AppDesc, m.dbAlias)), helpText)
}
