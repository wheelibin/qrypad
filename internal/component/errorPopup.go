package component

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/dbee/internal/colour"
	"github.com/wheelibin/dbee/internal/style"
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
	popupStyle := style.BasePanelStyle
	popupStyle = popupStyle.Width(m.width)
	popupStyle = popupStyle.Height(m.height)

	var errStyle = lipgloss.NewStyle().
		Foreground(colour.Error).
		Padding(0, 2).
		Align(lipgloss.Center)
	err := errStyle.Render(m.text)

	title := style.Title(m.width-2, false).
		Background(colour.Error).
		Foreground(colour.PanelTitleActiveFG).
		MarginBottom(1).
		Align(lipgloss.Center).
		Render("error")

	return popupStyle.Render(lipgloss.JoinVertical(lipgloss.Center, title, err))
}
