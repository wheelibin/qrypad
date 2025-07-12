package component

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

type helpKeymap struct {
	close key.Binding
}

type HelpPopupModel struct {
	width  int
	height int
	keymap helpKeymap
	help   help.Model
}

func NewHelpPopupModel() HelpPopupModel {
	return HelpPopupModel{
		help: makeHelp(),
		keymap: helpKeymap{
			close: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "close"),
			),
		},
	}
}

func (m HelpPopupModel) Init() tea.Cmd {
	return nil
}

func (m HelpPopupModel) Update(msg tea.Msg) (HelpPopupModel, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.close):
			cmds = append(cmds, commands.ClosePopup())
		}
	}

	return m, tea.Batch(cmds...)
}

func (m HelpPopupModel) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.close,
	})
}

func (m *HelpPopupModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m HelpPopupModel) View() string {
	popupStyle := style.GetBasePanelStyle().
		Width(m.width).
		Height(m.height).
		BorderForeground(theme.GetTheme().HelpPopup.BG)

	title := style.Title(m.width-2, false).
		Background(theme.GetTheme().HelpPopup.BG).
		Foreground(theme.GetTheme().HelpPopup.FG).
		MarginBottom(1).
		MarginRight(1).
		Align(lipgloss.Center).
		Render("help")

	return popupStyle.Render(lipgloss.JoinVertical(lipgloss.Center,
		title,
		m.help.View(keys.DefaultKeyMap),
		style.ShortHelp(m.width).Render(m.helpView()),
	))
}
