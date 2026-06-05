package component

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

type exportPopupKeymap struct {
	json   key.Binding
	csv    key.Binding
	cancel key.Binding
}

//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
type ExportFormatPopupModel struct {
	width  int
	height int
	keymap exportPopupKeymap
	help   help.Model
}

func NewExportFormatPopupModel() ExportFormatPopupModel {
	return ExportFormatPopupModel{
		help: makeHelp(),
		keymap: exportPopupKeymap{
			json: key.NewBinding(
				key.WithKeys("j"),
				key.WithHelp("j", "JSON"),
			),
			csv: key.NewBinding(
				key.WithKeys("c"),
				key.WithHelp("c", "CSV"),
			),
			cancel: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "cancel"),
			),
		},
	}
}

func (m ExportFormatPopupModel) Init() tea.Cmd {
	return nil
}

func (m ExportFormatPopupModel) Update(msg tea.Msg) (ExportFormatPopupModel, tea.Cmd) {
	var cmds []tea.Cmd
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, m.keymap.json):
			// The popup is closed by handleCommandMessages when it processes
			// ExportRequestedMsg, so we don't emit ClosePopup here.
			cmds = append(cmds, commands.ExportRequested(commands.ExportFormatJSON))
		case key.Matches(msg, m.keymap.csv):
			cmds = append(cmds, commands.ExportRequested(commands.ExportFormatCSV))
		case key.Matches(msg, m.keymap.cancel):
			cmds = append(cmds, commands.ClosePopup())
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *ExportFormatPopupModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m ExportFormatPopupModel) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.json,
		m.keymap.csv,
		m.keymap.cancel,
	})
}

func (m ExportFormatPopupModel) View() string {
	// reuse the help popup colour since there isn't a dedicated theme entry
	popupStyle := style.GetBasePanelStyle().
		Width(m.width + 2).
		Height(m.height + 2).
		BorderForeground(theme.GetTheme().HelpPopup.BG)

	title := style.Title(m.width-2, false).
		Background(theme.GetTheme().HelpPopup.BG).
		Foreground(theme.GetTheme().HelpPopup.FG).
		MarginBottom(1).
		MarginRight(1).
		Align(lipgloss.Center).
		Render("export results")

	body := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.width).
		Render("choose a format")

	return popupStyle.Render(lipgloss.JoinVertical(lipgloss.Center,
		title,
		body,
		style.ShortHelp(m.width).Render(m.helpView()),
	))
}
