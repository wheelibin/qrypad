package component

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/colour"
	"github.com/wheelibin/qrypad/internal/constants"
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
	// log.Println("titleBar.model::Update", msg)
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
	barStyle := lipgloss.NewStyle().
		Background(colour.TitleBarBG).
		Foreground(colour.TitleBarFG).
		Padding(0, 2).
		Bold(true)

	barStyle = barStyle.Width(m.width)
	barStyle = barStyle.Height(m.height)

	return barStyle.Render(fmt.Sprintf("QryPad - %s [%s]", constants.AppDesc, m.dbAlias))
}
