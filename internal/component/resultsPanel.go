package component

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evertras/bubble-table/table"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

type resultsPanelKeymap struct {
	viewRow key.Binding
	filter  key.Binding
	copyRow key.Binding
}

//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
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
				key.WithKeys(keys.DefaultKeyMap.ViewData.Keys()...),
				key.WithHelp(keys.DefaultKeyMap.ViewData.Help().Key, "view row"),
			),
			filter: key.NewBinding(
				key.WithKeys("/"),
				key.WithHelp("/", "filter data"),
			),
			copyRow: key.NewBinding(
				key.WithKeys(keys.DefaultKeyMap.CopyValue.Keys()...),
				key.WithHelp(keys.DefaultKeyMap.CopyValue.Help().Key, "copy row as json"),
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
		w := getColumnWidth(c, *data, m.width/2)
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
		slog.Error("error converting row to json", "error", err)
	}
	return string(j)
}

func newTable(cols []table.Column) table.Model {
	return table.New(cols).
		WithBaseStyle(style.TableColumn()).
		HeaderStyle(style.GetTableHeaderStyle()).
		HighlightStyle(style.GetTableHighlightStyle()).
		WithBorderForeground(style.GetTableBorderForeground()).
		BorderRounded().
		WithHorizontalFreezeColumnCount(1).
		Filtered(true)
}

func (m ResultsPanelModel) helpView() string {
	return m.help.ShortHelpView([]key.Binding{
		m.keymap.viewRow,
		m.keymap.filter,
		m.keymap.copyRow,
	})
}

func (m ResultsPanelModel) View() string {
	panelStyle := style.GetBasePanelStyle().
		Width(m.width + 2).
		Height(m.height + 2)
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
