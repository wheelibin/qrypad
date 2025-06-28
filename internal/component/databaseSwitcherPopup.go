package component

import (
	"math"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

type dbSwitcherKeymap struct {
	connect key.Binding
	cancel  key.Binding
}

type DatabaseSwitcherPopupModel struct {
	width   int
	height  int
	loading bool
	spinner spinner.Model
	table   table.Model
	keymap  dbSwitcherKeymap
	help    help.Model
}

func NewDatabaseSwitcherPopupModel() DatabaseSwitcherPopupModel {
	t := table.New([]table.Column{}).
		WithBaseStyle(style.TableColumn()).
		WithHeaderVisibility(false).
		Filtered(true).
		Focused(true)

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = style.GetSpinnerStyle()
	return DatabaseSwitcherPopupModel{
		table:   t,
		spinner: s,
		help:    makeHelp(),
		keymap: dbSwitcherKeymap{
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

func (m DatabaseSwitcherPopupModel) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.connect,
		m.keymap.cancel,
	})
}

func (m DatabaseSwitcherPopupModel) Init() tea.Cmd {
	return nil
}

func (m DatabaseSwitcherPopupModel) Update(msg tea.Msg) (DatabaseSwitcherPopupModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.connect):
			cmd = commands.DatabaseSelectionChanged(m.GetSelectedDatabase())
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

func (m *DatabaseSwitcherPopupModel) SetLoading(loading bool) {
	m.loading = loading
}

func (m DatabaseSwitcherPopupModel) GetSelectedDatabase() string {
	return m.table.HighlightedRow().Data["name"].(string)
}

func (m *DatabaseSwitcherPopupModel) SetData(data *db.Data) {
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

func (m *DatabaseSwitcherPopupModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	rowsInTable := math.Max(float64(h-6), 1)
	m.table = m.table.WithPageSize(int(rowsInTable))
	m.table = m.table.WithMinimumHeight(h - 1)
	m.table = m.table.WithTargetWidth(w)
}

func (m DatabaseSwitcherPopupModel) View() string {
	panelStyle := style.GetBasePanelStyle()
	panelStyle = panelStyle.Width(m.width)
	panelStyle = panelStyle.Height(m.height)

	panelStyle = panelStyle.BorderForeground(theme.GetTheme().DatabaseSwitcherPopup.BG)

	content := lipgloss.JoinVertical(lipgloss.Left, m.table.View())
	if m.loading {
		content = m.spinner.View()
	}

	title := style.Title(m.width-2, false).
		Background(theme.GetTheme().DatabaseSwitcherPopup.BG).
		Foreground(theme.GetTheme().DatabaseSwitcherPopup.FG).
		Align(lipgloss.Center).
		Render("switch database")

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		title,
		content,
		style.ShortHelp(m.width).Render(m.helpView())),
	)
}
