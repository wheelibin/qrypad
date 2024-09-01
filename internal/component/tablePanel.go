package component

import (
	"math"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/wheelibin/qrypad/internal/colour"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/style"
)

type TablePanelModel struct {
	active        bool
	width         int
	height        int
	loading       bool
	spinner       spinner.Model
	table         table.Model
	selectedTable string
}

func NewTablePanelModel() TablePanelModel {
	t := table.New([]table.Column{}).
		WithBaseStyle(
			lipgloss.NewStyle().
				BorderForeground(colour.ResultsTableBorder).
				// Foreground(lipgloss.Color("#a7a")).
				Align(lipgloss.Left),
		).
		HeaderStyle(style.TableHeaderStyle).
		Filtered(true).
		Focused(true)

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(colour.Spinner)
	return TablePanelModel{table: t, spinner: s, active: true}
}

func (m TablePanelModel) Init(db db.DBConn) tea.Cmd {
	return tea.Batch(
		commands.GetSchemaTables(db),
		m.spinner.Tick,
	)
}

func (m TablePanelModel) Update(msg tea.Msg) (TablePanelModel, tea.Cmd) {
	// log.Println("tablePanel.model::Update", msg)
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg.(type) {
	case db.SchemaTablesMsg:
		m.selectedTable = m.table.HighlightedRow().Data["name"].(string)
		cmds = append(cmds, commands.TableSelectionChanged(m.selectedTable))
	}

	if m.active {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
		for _, e := range m.table.GetLastUpdateUserEvents() {
			switch e.(type) {
			case table.UserEventHighlightedIndexChanged:
				m.selectedTable = m.table.HighlightedRow().Data["name"].(string)
				cmds = append(cmds, commands.TableSelectionChanged(m.selectedTable))
			}
		}
	}

	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, m.spinner.Tick, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *TablePanelModel) SetData(data *db.Data) {
	if data == nil {
		return
	}

	cols := []table.Column{}
	rows := []table.Row{}

	// get cols
	// name
	cols = append(cols, table.NewFlexColumn(data.Columns[0], data.Columns[0], 1).WithFiltered(true))
	// rows
	cols = append(cols, table.NewColumn(data.Columns[1], data.Columns[1], 12).WithFiltered(true))

	for _, row := range data.Rows {
		rows = append(rows, table.Row{Data: row})
	}

	m.table = m.table.WithRows(rows)
	m.table = m.table.WithColumns(cols)
	m.loading = false
}

func (m TablePanelModel) GetSelectedTable() string {
	return m.selectedTable
}

func (m *TablePanelModel) SetActive(active bool) {
	m.table = m.table.Focused(active)
	m.active = active
}

func (m *TablePanelModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	rowsInTable := math.Max(float64(h-7), 1)
	m.table = m.table.WithPageSize(int(rowsInTable))
	m.table = m.table.WithMinimumHeight(h - 1)
	m.table = m.table.WithTargetWidth(w)
}

func (m *TablePanelModel) SetLoading(loading bool) {
	m.loading = loading
}

func (m TablePanelModel) View() string {
	var panelStyle = style.BasePanelStyle
	panelStyle = panelStyle.Width(m.width)
	panelStyle = panelStyle.Height(m.height)

	panelStyle = panelStyle.BorderForeground(colour.Border)
	if m.active {
		panelStyle = panelStyle.BorderForeground(colour.BorderActive)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, m.table.View())
	if m.loading {
		content = m.spinner.View()
	}
	title := style.Title(m.width-2, m.active).Render("tables")
	v := lipgloss.JoinVertical(lipgloss.Left, title, content)
	return panelStyle.Render(v)

}
