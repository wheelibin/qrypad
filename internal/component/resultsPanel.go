package component

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

type resultsPanelKeymap struct {
	viewRow key.Binding
	filter  key.Binding
}

type ResultsPanelModel struct {
	active        bool
	width         int
	height        int
	table         table.Model
	lastQueryTime time.Duration
	help          help.Model
	keymap        resultsPanelKeymap
}

func NewResultsPanelModel() ResultsPanelModel {
	t := newTable([]table.Column{})
	return ResultsPanelModel{
		table: t,
		help:  makeHelp(),
		keymap: resultsPanelKeymap{
			viewRow: key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "view row"),
			),
			filter: key.NewBinding(
				key.WithKeys("/"),
				key.WithHelp("/", "filter data"),
			),
		},
	}
}

func (m ResultsPanelModel) Init() tea.Cmd {
	return nil
}

func (m ResultsPanelModel) Update(msg tea.Msg) (ResultsPanelModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

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
		w := m.getColumnWidth(c, *data)
		cols = append(cols, table.NewColumn(c, c, w).WithFiltered(true))
	}
	for _, row := range data.Rows {
		rows = append(rows, table.Row{Data: row})
	}

	m.table = newTable(cols).WithRows(rows)

	m.SetSize(m.width, m.height)
	m.lastQueryTime = data.QueryTime
}

func (m *ResultsPanelModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	rowsInTable := math.Ceil(math.Max(float64(h-9), 1))
	m.table = m.table.
		WithPageSize(int(rowsInTable)).
		WithMinimumHeight(h - 2).
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
		HeaderStyle(style.GetTableHeaderStyle()).
		WithHorizontalFreezeColumnCount(1).
		Filtered(true)
}

func (m ResultsPanelModel) helpView() string {
	return m.help.ShortHelpView([]key.Binding{
		m.keymap.viewRow,
		m.keymap.filter,
	})
}

func (m ResultsPanelModel) View() string {
	panelStyle := style.GetBasePanelStyle().
		Width(m.width).
		Height(m.height)
	if m.active {
		panelStyle = panelStyle.BorderForeground(theme.GetTheme().BorderActive.FG)
	}

	title := style.Title(m.width-2, m.active).Render("results")
	if m.lastQueryTime > 0 {
		title = style.Title(m.width-2, m.active).Render(fmt.Sprintf("results (%s)", m.lastQueryTime.String()))
	}

	content := lipgloss.JoinVertical(lipgloss.Bottom, m.table.View())

	helpView := ""
	if m.table.TotalRows() > 0 {
		helpView = style.ShortHelp(m.width).Render(m.helpView())
	}

	panel := panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		title,
		content,
		helpView,
	))

	return panel
}
