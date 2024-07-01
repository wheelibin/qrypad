package component

import (
	"math"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/wheelibin/dbee/internal/colour"
	"github.com/wheelibin/dbee/internal/db"
	"github.com/wheelibin/dbee/internal/style"
)

type ResultsPanelModel struct {
	active  bool
	width   int
	height  int
	loading bool
	spinner spinner.Model
	table   table.Model
}

func NewResultsPanelModel() ResultsPanelModel {
	t := table.New([]table.Column{}).
		WithBaseStyle(
			lipgloss.NewStyle().
				BorderForeground(colour.ResultsTableBorder).
				// Foreground(lipgloss.Color("#a7a")).
				Align(lipgloss.Left),
		).
		HeaderStyle(style.TableHeaderStyle).
		Filtered(true)

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = lipgloss.NewStyle().Foreground(colour.Spinner)
	return ResultsPanelModel{table: t, spinner: s}
}

func (m ResultsPanelModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
	)
}

func (m ResultsPanelModel) Update(msg tea.Msg) (ResultsPanelModel, tea.Cmd) {
	// log.Println("resultsPanel.model::Update", msg)
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, m.spinner.Tick, cmd)
	}

	if m.active {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *ResultsPanelModel) SetData(data *db.Data) {
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

	m.table = m.table.WithRows(rows)
	m.table = m.table.WithColumns(cols)
	m.loading = false
	m.SetSize(m.width, m.height)
}

func (m *ResultsPanelModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	rowsInTable := math.Ceil(math.Max(float64(h-7), 1))
	m.table = m.table.WithPageSize(int(rowsInTable))
	m.table = m.table.WithMinimumHeight(h - 1)
	m.table = m.table.WithTargetWidth(w)
}

func (m *ResultsPanelModel) SetLoading(loading bool) {
	m.loading = loading
}

func (m *ResultsPanelModel) SetActive(active bool) {
	m.table = m.table.Focused(active)
	m.active = active
}

func (m ResultsPanelModel) View() string {
	var panelStyle = style.BasePanelStyle
	panelStyle = panelStyle.Width(m.width)
	panelStyle = panelStyle.Height(m.height)

	panelStyle = panelStyle.BorderForeground(colour.Border)
	if m.active {
		panelStyle = panelStyle.BorderForeground(colour.BorderActive)
	}

	title := style.Title(m.width-2, m.active).Render("results")
	content := lipgloss.JoinVertical(lipgloss.Bottom, m.table.View())
	if m.loading {
		content = m.spinner.View()
	}
	v := lipgloss.JoinVertical(lipgloss.Left, title, content)
	return panelStyle.Render(v)

}
