package component

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/wheelibin/qrypad/internal/colour"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/style"
)

type ResultsPanelModel struct {
	active bool
	width  int
	height int
	// loading       bool
	// spinner       spinner.Model
	// stopwatch     stopwatch.Model
	table         table.Model
	lastQueryTime time.Duration
	help          help.Model
}

func NewResultsPanelModel() ResultsPanelModel {
	t := newTable([]table.Column{})
	// s := spinner.New()
	// s.Spinner = spinner.Meter
	// s.Style = style.Spinner
	return ResultsPanelModel{
		table: t,
		// spinner:   s,
		// stopwatch: stopwatch.New(),
		help: makeHelp(),
	}
}

func (m ResultsPanelModel) Init() tea.Cmd {
	return tea.Batch(
	// m.spinner.Tick,
	// m.stopwatch.Init(),
	)
}

func (m ResultsPanelModel) Update(msg tea.Msg) (ResultsPanelModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// switch msg := msg.(type) {
	// case spinner.TickMsg:
	// 	if m.loading {
	// 		m.spinner, cmd = m.spinner.Update(msg)
	// 		cmds = append(cmds, cmd)
	// 	}
	//
	// case commands.LoadingMsg:
	// 	m.loading = msg.Loading
	// 	if m.loading {
	// 		cmds = append(cmds, m.spinner.Tick)
	// 		cmds = append(cmds, tea.Sequence(m.stopwatch.Reset(), m.stopwatch.Start()))
	// 	} else {
	// 		// loading finished
	// 		cmds = append(cmds, m.stopwatch.Stop())
	// 	}
	// }

	if m.active {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	// if m.loading {
	// 	m.stopwatch, cmd = m.stopwatch.Update(msg)
	// 	cmds = append(cmds, cmd)
	// }

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
		w := m.getColumnWidth(c, *data)
		cols = append(cols, table.NewColumn(c, c, w).WithFiltered(true))
	}
	for _, row := range data.Rows {
		rows = append(rows, table.Row{Data: row})
	}

	m.table = newTable(cols).WithRows(rows)

	// m.loading = false
	m.SetSize(m.width, m.height)
	m.lastQueryTime = data.QueryTime
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

func (m ResultsPanelModel) GetSelectedRowJSON() string {
	j, err := json.Marshal(m.table.HighlightedRow().Data)
	if err != nil {
		log.Println("error converting row to json", err)
	}
	return string(j)
}

func (m ResultsPanelModel) View() string {
	panelStyle := style.BasePanelStyle.
		Width(m.width).
		Height(m.height)
	if m.active {
		panelStyle = panelStyle.BorderForeground(colour.GetTheme().BorderActive.FG)
	}

	title := style.Title(m.width-2, m.active).Render("results")
	if m.lastQueryTime > 0 {
		title = style.Title(m.width-2, m.active).Render(fmt.Sprintf("results (%s)", m.lastQueryTime.String()))
	}

	content := lipgloss.JoinVertical(lipgloss.Bottom, m.table.View())

	panel := panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		title,
		content,
	))

	return panel
}

func (m ResultsPanelModel) getColumnWidth(col string, data db.Data) int {
	maxAllowedLen := m.width / 2
	maxNeededLen := 0
	for _, c := range data.Columns {
		if len(c) > maxNeededLen {
			maxNeededLen = len(c)
		}
	}
	for _, r := range data.Rows {
		if len(fmt.Sprintf("%v", r[col])) > maxNeededLen {
			maxNeededLen = len(r[col].(string))
		}
	}
	padding := 1
	return int(math.Min(float64(maxNeededLen), float64(maxAllowedLen))) + padding
}

func newTable(cols []table.Column) table.Model {
	return table.New(cols).
		WithBaseStyle(style.TableColumn()).
		HeaderStyle(style.TableHeaderStyle).
		WithHorizontalFreezeColumnCount(1).
		Filtered(true)
}
