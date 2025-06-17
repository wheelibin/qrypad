package component

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/stopwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/wheelibin/qrypad/internal/colour"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
)

type ResultsPanelModel struct {
	active        bool
	width         int
	height        int
	loading       bool
	spinner       spinner.Model
	stopwatch     stopwatch.Model
	table         table.Model
	lastQueryTime time.Duration
	help          help.Model
}

func NewResultsPanelModel() ResultsPanelModel {
	t := table.New([]table.Column{}).
		WithBaseStyle(style.TableColumn()).
		HeaderStyle(style.TableHeaderStyle).
		WithHorizontalFreezeColumnCount(1).
		Filtered(true)

	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = style.Spinner
	return ResultsPanelModel{
		table:     t,
		spinner:   s,
		stopwatch: stopwatch.New(),
		help:      makeHelp(),
	}
}

func (m ResultsPanelModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.stopwatch.Init(),
	)
}

func (m ResultsPanelModel) Update(msg tea.Msg) (ResultsPanelModel, tea.Cmd) {
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
			cmds = append(cmds, tea.Sequence(m.stopwatch.Reset(), m.stopwatch.Start()))
		} else {
			// loading finished
			cmds = append(cmds, m.stopwatch.Stop())
		}
	}

	if m.active {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.loading {
		m.stopwatch, cmd = m.stopwatch.Update(msg)
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

func (m ResultsPanelModel) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		keys.DefaultKeyMap.CancelQuery,
	})
}

func (m ResultsPanelModel) View() string {
	panelStyle := style.BasePanelStyle.
		Width(m.width).
		Height(m.height).
		BorderForeground(colour.GetTheme().Border.FG)
	if m.active {
		panelStyle = panelStyle.BorderForeground(colour.GetTheme().BorderActive.FG)
	}

	title := style.Title(m.width-2, m.active).Render("results")
	if m.lastQueryTime > 0 {
		title = style.Title(m.width-2, m.active).Render(fmt.Sprintf("results (%s)", m.lastQueryTime.String()))
	}
	content := lipgloss.JoinVertical(lipgloss.Bottom, m.table.View())
	if m.loading {
		content = lipgloss.PlaceVertical(m.height-1, lipgloss.Center,
			lipgloss.PlaceHorizontal(m.width, lipgloss.Center,
				lipgloss.JoinVertical(lipgloss.Center, m.spinner.View(), m.stopwatch.Elapsed().String(), m.helpView())))
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
