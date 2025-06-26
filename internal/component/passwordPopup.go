package component

import (
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/colour"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/style"
)

type passwordKeymap struct {
	accept key.Binding
	cancel key.Binding
}

type PasswordPopupModel struct {
	width  int
	height int
	input  textinput.Model
	keymap passwordKeymap
	help   help.Model
}

func NewPasswordPopupModel() PasswordPopupModel {
	ti := textinput.New()
	ti.Placeholder = "password"
	ti.Cursor.SetMode(cursor.CursorBlink)
	ti.CharLimit = 0
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '·'
	ti.Focus()

	return PasswordPopupModel{
		input: ti,
		help:  makeHelp(),
		keymap: passwordKeymap{
			accept: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "accept"),
			),
			cancel: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "cancel"),
			),
		},
	}
}

func (m PasswordPopupModel) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.accept,
		m.keymap.cancel,
	})
}

func (m PasswordPopupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m PasswordPopupModel) Update(msg tea.Msg) (PasswordPopupModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.accept):
			cmd = commands.PasswordEntered(m.input.Value())
			cmds = append(cmds, cmd)

		case key.Matches(msg, m.keymap.cancel):
			cmd = commands.ClosePopup()
			cmds = append(cmds, cmd)
		}
	}

	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *PasswordPopupModel) Clear() {
	m.input.SetValue("")
}

func (m *PasswordPopupModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.input.Width = w / 2
}

func (m PasswordPopupModel) View() string {
	popupStyle := style.BasePanelStyle.
		Width(m.width).
		Height(m.height).
		BorderForeground(colour.GetTheme().Error.FG)

	title := style.Title(m.width-2, false).
		Background(colour.GetTheme().Error.FG).
		Foreground(colour.GetTheme().Error.BG).
		MarginBottom(1).
		MarginRight(1).
		Align(lipgloss.Center).
		Render("set password")

	return popupStyle.Render(lipgloss.JoinVertical(lipgloss.Center,
		title,
		m.input.View(),
		style.ShortHelp(m.width).Render(m.helpView()),
	))
}
