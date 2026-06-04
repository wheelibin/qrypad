package component

import (
	"encoding/json"
	"fmt"
	"log/slog"
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
	export  key.Binding
}

//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
type ResultsPanelModel struct {
	active        bool
	width         int
	height        int
	table         table.Model
	columns       []string
	columnTypes   []string
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
			export: key.NewBinding(
				key.WithKeys(keys.DefaultKeyMap.ExportResults.Keys()...),
				key.WithHelp(keys.DefaultKeyMap.ExportResults.Help().Key, "export"),
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

	m.columns = append([]string(nil), data.Columns...)
	m.columnTypes = append([]string(nil), data.ColumnTypes...)

	cols := []table.Column{}
	rows := []table.Row{}

	// get cols
	for _, c := range data.Columns {
		w := getColumnWidth(c, *data, m.width/2)
		cols = append(cols, table.NewColumn(c, c, w).WithFiltered(true))
	}
	for _, row := range data.Rows {
		styledRow := make(map[string]any, len(row))
		for colIdx, colName := range data.Columns {
			val := row[colName]
			valStr := fmt.Sprintf("%v", val)
			category := "unknown"
			if colIdx < len(data.ColumnTypes) {
				category = data.ColumnTypes[colIdx]
			}

			switch {
			case valStr == "NULL":
				styledRow[colName] = table.NewStyledCell(val, style.NullStyle())
			case category == "json":
				highlighted := style.HighlightJSON(valStr)
				styledRow[colName] = table.NewStyledCell(jsonCellData{Raw: valStr, highlighted: highlighted}, style.JSONBaseStyle())
			default:
				styledRow[colName] = table.NewStyledCell(val, style.ResultCellStyle(category))
			}
		}
		rows = append(rows, table.Row{Data: styledRow})
	}

	m.table = newTable(cols).WithRows(rows)

	m.SetSize(m.width, m.height)
	m.lastQueryTime = data.QueryTime
}

func (m *ResultsPanelModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	tableHeight := max(h-2, 1)
	m.table = m.table.
		WithTargetHeight(tableHeight).
		WithMinimumHeight(tableHeight).
		WithMaxTotalWidth(w - 1)
}

func (m *ResultsPanelModel) SetActive(active bool) {
	m.table = m.table.Focused(active)
	m.active = active
}

func (m ResultsPanelModel) GetSelectedRow() map[string]any {
	raw := m.table.HighlightedRow().Data
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		out[k] = unwrapCellData(v)
	}
	return out
}

func (m ResultsPanelModel) GetSelectedRowJSON() string {
	raw := m.table.HighlightedRow().Data
	plain := make(map[string]any, len(raw))
	for k, v := range raw {
		plain[k] = unwrapCellData(v)
	}
	j, err := json.Marshal(plain)
	if err != nil {
		slog.Error("error converting row to json", "error", err)
	}
	return string(j)
}

// GetColumns returns the column names in the order they were supplied by the
// most recent SetData call.
func (m ResultsPanelModel) GetColumns() []string {
	return append([]string(nil), m.columns...)
}

// GetColumnTypes returns the column type categories from the most recent SetData call.
func (m ResultsPanelModel) GetColumnTypes() []string {
	return append([]string(nil), m.columnTypes...)
}

// GetExportRows returns the rows currently visible in the table, honouring
// any active filter and sort order. When no filter is active this is the
// full result set in its current display order.
func (m ResultsPanelModel) GetExportRows() []map[string]any {
	src := m.table.GetVisibleRows()
	out := make([]map[string]any, 0, len(src))
	for _, r := range src {
		plain := make(map[string]any, len(r.Data))
		for k, v := range r.Data {
			plain[k] = unwrapCellData(v)
		}
		out = append(out, plain)
	}
	return out
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
		m.keymap.export,
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
