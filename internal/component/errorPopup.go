package component

import (
	"fmt"
	"math"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
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
				return m, commands.RequestPasswordInput()
			}

		case key.Matches(msg, m.keymap.close):
			if m.isConnectionError {
				return m, tea.Quit
			} else {
				return m, commands.ClosePopup()
			}
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
	theme := theme.GetTheme()
	popupStyle := style.GetBasePanelStyle().
		Width(m.width).
		BorderForeground(theme.Error.FG)

	errStyle := lipgloss.NewStyle().
		Foreground(theme.Error.FG).
		Align(lipgloss.Left).
		Width(m.width - 2)

	msgWidth := int(math.Min(float64(m.width)-16, float64(len(m.text))))
	err := errStyle.
		Width(int(msgWidth)).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border.FG).
		Render(m.text)

	errHeight := lipgloss.Height(err)
	popupStyle = popupStyle.Height(errHeight + 2)

	var extraText string
	if m.isConnectionError {
		s := errStyle.MarginBottom(1).Align(lipgloss.Center).Foreground(theme.Text.FG)
		extraText = s.Render(
			fmt.Sprintf(
				"There was a problem connecting to the database server, check the connection details in your config, or press %s to update the saved password",
				keys.DefaultKeyMap.UpdatePassword.Help().Key,
			),
		)
	}

	title := style.Title(m.width-2, false).
		Background(theme.Error.FG).
		Foreground(theme.Error.BG).
		Align(lipgloss.Center).
		MarginRight(1).
		MarginBottom(1).
		Render("error")

	if extraText != "" {
		return popupStyle.Render(lipgloss.JoinVertical(lipgloss.Center,
			title,
			extraText,
			err,
			style.ShortHelp(m.width).Render(m.helpView()),
		))
	}
	return popupStyle.Render(lipgloss.JoinVertical(lipgloss.Center,
		title,
		err,
		style.ShortHelp(m.width).Render(m.helpView()),
	))
}
