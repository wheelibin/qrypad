package component

import (
	"fmt"
	"math"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

type errorKeymap struct {
	updatePassword key.Binding
	close          key.Binding
	copy           key.Binding
}

//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
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
			copy: key.NewBinding(
				key.WithKeys(keys.DefaultKeyMap.CopyValue.Keys()...),
				key.WithHelp(keys.DefaultKeyMap.CopyValue.Help().Key, "copy error"),
			),
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
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, m.keymap.updatePassword):
			if m.isConnectionError {
				return m, commands.RequestPasswordInput()
			}

		case key.Matches(msg, m.keymap.close):
			if m.isConnectionError {
				return m, tea.Quit
			}
			return m, commands.ClosePopup()

		case key.Matches(msg, m.keymap.copy):
			var valDesc string
			val := m.text
			maxLengthForDesc := 30
			if len(val) > maxLengthForDesc {
				valDesc = val[0:maxLengthForDesc-3] + "..."
			}
			return m, commands.CopyValue(val, valDesc)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m ErrorPopupModel) helpView() string {
	if m.isConnectionError {
		return "\n" + m.help.ShortHelpView([]key.Binding{
			m.keymap.updatePassword,
			m.keymap.copy,
			m.keymap.close,
		})
	}
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.copy,
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
		Width(m.width + 2).
		BorderForeground(theme.Error.FG)

	errStyle := lipgloss.NewStyle().
		Foreground(theme.Error.FG).
		Align(lipgloss.Left).
		Width(m.width - 2)

	msgWidth := int(math.Min(float64(m.width)-16, float64(len(m.text)))) + 2
	err := errStyle.
		Width(msgWidth).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border.FG).
		Render(m.text)

	errHeight := lipgloss.Height(err)
	popupStyle = popupStyle.Height(errHeight + 4)

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
