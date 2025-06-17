package component

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/colour"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
)

type errorKeymap struct {
	updatePassword key.Binding
	close          key.Binding
}

type ErrorPopupModel struct {
	width             int
	height            int
	text              string
	isConnectionError bool
	keymap            errorKeymap
	help              help.Model
}

func NewErrorPopupModel() ErrorPopupModel {
	return ErrorPopupModel{
		help: makeHelp(),
		keymap: errorKeymap{
			updatePassword: keys.DefaultKeyMap.UpdatePassword,
			close: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "close"),
			),
		},
	}
}

func (m ErrorPopupModel) Init() tea.Cmd {
	return nil
}

func (m ErrorPopupModel) Update(msg tea.Msg) (ErrorPopupModel, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.updatePassword):
			if m.isConnectionError {
				cmds = append(cmds, commands.RequestPasswordInput())
			}

		case key.Matches(msg, m.keymap.close):
			cmds = append(cmds, commands.ClosePopup())
		}
	}

	return m, tea.Batch(cmds...)
}

func (m ErrorPopupModel) helpView() string {
	if m.isConnectionError {
		return "\n" + m.help.ShortHelpView([]key.Binding{
			m.keymap.updatePassword,
			m.keymap.close,
		})
	}
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.close,
	})
}

func (m *ErrorPopupModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *ErrorPopupModel) SetText(text string) {
	m.text = text
}

func (m *ErrorPopupModel) SetIsConnectionError(v bool) {
	m.isConnectionError = v
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

	return popupStyle.Render(lipgloss.JoinVertical(lipgloss.Center, title, err, m.helpView()))
}
