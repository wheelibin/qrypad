package component

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/colour"
	"github.com/wheelibin/qrypad/internal/style"
)

type ErrorPopupModel struct {
	width  int
	height int
	text   string
}

func NewErrorPopupModel() ErrorPopupModel {
	return ErrorPopupModel{}
}

func (m ErrorPopupModel) Init() tea.Cmd {
	return nil
}

func (m ErrorPopupModel) Update(msg tea.Msg) (ErrorPopupModel, tea.Cmd) {
	// log.Println("queryPanel.update", msg)
	var (
		cmds []tea.Cmd
	)

	// switch msg := msg.(type) {
	// case tea.KeyMsg:
	// 	// any key
	//
	// }
	return m, tea.Batch(cmds...)
}

func (m *ErrorPopupModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *ErrorPopupModel) SetText(text string) {
	m.text = text
}

func (m ErrorPopupModel) View() string {
	popupStyle := style.BasePanelStyle.
		Width(m.width).
		BorderForeground(colour.GetTheme().Error.FG)

	errStyle := lipgloss.NewStyle().
		Foreground(colour.GetTheme().Error.FG).
		Padding(0, 2).
		Align(lipgloss.Center).
		Width(m.width - 2)
	err := errStyle.Render(m.text)

	errHeight := lipgloss.Height(err)
	popupStyle = popupStyle.Height(errHeight + 3)

	title := style.Title(m.width-2, false).
		Background(colour.GetTheme().Error.FG).
		Foreground(colour.GetTheme().Error.BG).
		MarginBottom(1).
		Align(lipgloss.Center).
		Render("error")

	return popupStyle.Render(lipgloss.JoinVertical(lipgloss.Center, title, err))
}
