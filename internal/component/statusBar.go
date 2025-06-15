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

func (m StatusBarModel) View() string {
	barStyle := lipgloss.NewStyle().
		Background(colour.GetTheme().StatusBar.BG).
		Foreground(colour.GetTheme().StatusBar.FG).
		Padding(0, 2).
		Bold(true)

	barStyle = barStyle.Width(m.width)
	barStyle = barStyle.Height(m.height)

	content := lipgloss.NewStyle().
		Background(colour.GetTheme().StatusBar.BG).
		Foreground(colour.GetTheme().StatusBar.FG).
		Bold(true).
		Italic(true).
		Render(" " + m.selectedDatabase + " ")

	content2 := lipgloss.NewStyle().
		Background(colour.GetTheme().StatusBar.BG).
		Foreground(colour.GetTheme().StatusBar.FG).
		Bold(true).
		Render(fmt.Sprintf(" [%s] to switch", keys.DefaultKeyMap.SwitchDatabase.Help().Key))

	return barStyle.Render(content + content2)
}
