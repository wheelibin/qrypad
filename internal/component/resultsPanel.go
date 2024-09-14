package component

import (
	"fmt"
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
		WithHorizontalFreezeColumnCount(1).
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

	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case commands.LoadingMsg:
		m.loading = msg.Loading
		if m.loading {
			cmds = append(cmds, m.spinner.Tick)
		}
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
		w := getColumnWidth(c, *data)
		cols = append(cols, table.NewColumn(c, c, w).WithFiltered(true))
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

func (m *ResultsPanelModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	rowsInTable := math.Ceil(math.Max(float64(h-7), 1))
	m.table = m.table.
		WithPageSize(int(rowsInTable)).
		WithMinimumHeight(h - 1).
		WithMaxTotalWidth(w - 1)
}

func (m *ResultsPanelModel) SetActive(active bool) {
	m.table = m.table.Focused(active)
	m.active = active
}

func (m ResultsPanelModel) GetSelectedRow() map[string]any {
	return m.table.HighlightedRow().Data
}

func (m ResultsPanelModel) View() string {
	panelStyle := style.BasePanelStyle.
		Width(m.width).
		Height(m.height).
		BorderForeground(colour.Border)
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

func getColumnWidth(col string, data db.Data) int {
	maxLen := 0
	for _, c := range data.Columns {
		if len(c) > maxLen {
			maxLen = len(c)
		}
	}
	for _, r := range data.Rows {
		if len(fmt.Sprintf("%v", r[col])) > maxLen {
			maxLen = len(r[col].(string))
		}
	}
	padding := 1
	return maxLen + padding
}
