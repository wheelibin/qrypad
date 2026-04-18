package component

import (
	"math"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evertras/bubble-table/table"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

type connSwitcherKeymap struct {
	connect key.Binding
	cancel  key.Binding
}

//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
type ConnectionSwitcherPopupModel struct {
	width   int
	height  int
	loading bool
	spinner spinner.Model
	table   table.Model
	keymap  connSwitcherKeymap
	help    help.Model
}

func NewConnectionSwitcherPopupModel() ConnectionSwitcherPopupModel {
	t := table.New([]table.Column{}).
		WithBaseStyle(style.TableColumn()).
		HighlightStyle(style.GetTableHighlightStyle()).
		WithBorderForeground(style.GetTableBorderForeground()).
		BorderRounded().
		HeaderStyle(style.GetTableHeaderStyle()).
		Filtered(true).
		Focused(true)

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = style.GetSpinnerStyle()
	return ConnectionSwitcherPopupModel{
		table:   t,
		spinner: s,
		help:    makeHelp(),
		keymap: connSwitcherKeymap{
			connect: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "connect"),
			),
			cancel: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "cancel"),
			),
		},
	}
}

func (m ConnectionSwitcherPopupModel) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.connect,
		m.keymap.cancel,
	})
}

func (m ConnectionSwitcherPopupModel) Init() tea.Cmd {
	return nil
}

func (m ConnectionSwitcherPopupModel) Update(msg tea.Msg) (ConnectionSwitcherPopupModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, m.keymap.connect):
			cmd = commands.ConnectionSelectionChanged(m.GetSelectedConnection())
			cmds = append(cmds, cmd)

		case key.Matches(msg, m.keymap.cancel):
			cmd = commands.ClosePopup()
			cmds = append(cmds, cmd)
		}
	}

	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, m.spinner.Tick, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *ConnectionSwitcherPopupModel) SetLoading(loading bool) {
	m.loading = loading
}

func (m ConnectionSwitcherPopupModel) GetSelectedConnection() string {
	name, _ := m.table.HighlightedRow().Data["name"].(string)
	return name
}

func (m *ConnectionSwitcherPopupModel) SetData(data *db.Data) {
	if data == nil {
		return
	}

	cols := []table.Column{}
	rows := []table.Row{}

	// get cols
	for _, c := range data.Columns {
		cols = append(cols, table.NewFlexColumn(c, c, 1).WithFiltered(true))
	}
	for _, row := range data.Rows {
		rows = append(rows, table.Row{Data: row})
	}

	m.table = m.table.
		WithRows(rows).
		WithColumns(cols)

	m.loading = false
	m.SetSize(m.width, m.height)
}

func (m *ConnectionSwitcherPopupModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	rowsInTable := math.Max(float64(h-6), 1)
	m.table = m.table.WithPageSize(int(rowsInTable))
	m.table = m.table.WithMinimumHeight(h - 1)
	m.table = m.table.WithTargetWidth(w)
}

func (m ConnectionSwitcherPopupModel) View() string {
	panelStyle := style.GetBasePanelStyle()
	panelStyle = panelStyle.Width(m.width + 2)
	panelStyle = panelStyle.Height(m.height + 2)

	panelStyle = panelStyle.BorderForeground(theme.GetTheme().ConnectionSwitcherPopup.BG)

	content := lipgloss.JoinVertical(lipgloss.Left, m.table.View())
	if m.loading {
		content = m.spinner.View()
	}

	title := style.Title(m.width-2, false).
		Background(theme.GetTheme().ConnectionSwitcherPopup.BG).
		Foreground(theme.GetTheme().ConnectionSwitcherPopup.FG).
		Align(lipgloss.Center).
		Render("switch connection")

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		title,
		content,
		style.ShortHelp(m.width).Render(m.helpView())),
	)
}
