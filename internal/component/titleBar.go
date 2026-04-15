package component

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/theme"
)

//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
type TitleBarModel struct {
	width          int
	height         int
	connectionName string
	conn           db.ConnectionConfig
}

func NewTitleBarModel(connectionName string, conn db.ConnectionConfig) TitleBarModel {
	return TitleBarModel{
		connectionName: connectionName,
		conn:           conn,
	}
}

func (m TitleBarModel) Init() tea.Cmd {
	return nil
}

func (m TitleBarModel) Update(_ tea.Msg) (TitleBarModel, tea.Cmd) {
	var cmds []tea.Cmd

	return m, tea.Batch(cmds...)
}

func (m *TitleBarModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m TitleBarModel) View() string {
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
