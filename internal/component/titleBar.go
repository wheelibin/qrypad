package component

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/theme"
)

type TitlBarModel struct {
	width          int
	height         int
	text           string
	connectionName string
	conn           db.ConnectionConfig
}

func NewTitlBarModel(connectionName string, conn db.ConnectionConfig) TitlBarModel {
	return TitlBarModel{
		connectionName: connectionName,
		conn:           conn,
	}
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

	altText := lipgloss.NewStyle().Foreground(theme.GetTheme().TitleBarAlt.FG)

	var connDetails string
	if m.conn.Driver == db.DriverName.SQLite {
		connDetails = fmt.Sprintf("[%s]", m.connectionName)
	} else {
		connDetails = fmt.Sprintf("[%s: %s@%s:%d]", m.connectionName, m.conn.User, m.conn.Host, m.conn.Port)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center,
		barStyle.Render(fmt.Sprintf("QryPad %s", altText.Render(connDetails))),
		helpText)
}
