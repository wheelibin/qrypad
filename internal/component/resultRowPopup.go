package component

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evertras/bubble-table/table"

	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

const maxRowLines = 5

// fallbackCharsPerLine is used as a fallback column width when the popup
// width has not yet been set (i.e. width == 0).
const fallbackCharsPerLine = 200

// clampToLines returns s with at most n newline-separated lines.
// If s has more than n lines, the excess is removed and "…" is appended.
func clampToLines(s string, n int) string {
	lines := strings.SplitN(s, "\n", n+1)
	if len(lines) > n {
		lines = lines[:n]
		return strings.Join(lines, "\n") + "…"
	}
	return s
}

type resultRowPopupKeymap struct {
	copy  key.Binding
	close key.Binding
}

//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
type ResultRowPopupModel struct {
	width  int
	height int
	table  table.Model
	keymap resultRowPopupKeymap
	help   help.Model
}

func NewResultRowPopupModel() ResultRowPopupModel {
	t := table.New([]table.Column{}).
		WithBaseStyle(style.TableColumn()).
		HighlightStyle(style.GetTableHighlightStyle()).
		WithBorderForeground(style.GetTableBorderForeground()).
		BorderRounded().
		WithHeaderVisibility(false).
		Filtered(true).
		Focused(true).
		WithMultiline(true)
	t = withRowBorders(t)

	return ResultRowPopupModel{
		table: t,
		help:  makeHelp(),
		keymap: resultRowPopupKeymap{
			copy: keys.DefaultKeyMap.CopyValue,
			close: key.NewBinding(
				key.WithKeys("esc"),
				key.WithHelp("esc", "close"),
			),
		},
	}
}

func (m ResultRowPopupModel) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.copy,
	})
}

func (m ResultRowPopupModel) Init() tea.Cmd {
	return nil
}

func (m ResultRowPopupModel) Update(msg tea.Msg) (ResultRowPopupModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, m.keymap.copy):
			var valDesc string
			val := m.GetSelectedValue()
			maxLengthForDesc := 30
			if len(val) > maxLengthForDesc {
				valDesc = val[0:maxLengthForDesc-3] + "..."
			}
			cmds = append(cmds, commands.CopyValue(val, valDesc))

		case key.Matches(msg, m.keymap.close):
			cmds = append(cmds, commands.ClosePopup())
		}
	}

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m ResultRowPopupModel) GetSelectedValue() string {
	row := m.table.HighlightedRow()
	if row.Data == nil {
		return ""
	}
	val, ok := row.Data["value"]
	if !ok {
		return ""
	}
	// Unwrap StyledCell if present
	if sc, ok := val.(table.StyledCell); ok {
		if jc, ok := sc.Data.(jsonCellData); ok {
			return jc.Raw
		}
		val = sc.Data
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}

func (m *ResultRowPopupModel) SetData(data map[string]any, columns []string, columnTypes []string) {
	if data == nil {
		return
	}

	cols := []table.Column{
		table.NewFlexColumn("field", "field", 1).WithFiltered(true),
		table.NewFlexColumn("value", "value", 3).WithFiltered(true),
	}
	rows := []table.Row{}

	// Build a column-name-to-type lookup
	typeMap := make(map[string]string, len(columns))
	for i, col := range columns {
		if i < len(columnTypes) {
			typeMap[col] = columnTypes[i]
		}
	}

	// Use the provided column order; fall back to sorted keys when unavailable
	// (e.g. table info panel, which passes nil columns).
	keys := columns
	if len(keys) == 0 {
		for k := range data {
			keys = append(keys, k)
		}
		slices.Sort(keys)
	}

	for _, k := range keys {
		v, ok := data[k]
		if !ok {
			continue
		}
		val := fmt.Sprintf("%v", v)
		// Pre-truncate long single-line values so word-wrap cannot create
		// more than maxRowLines lines regardless of column width.
		displayVal := val
		maxChars := m.width * maxRowLines
		if maxChars <= 0 {
			maxChars = fallbackCharsPerLine * maxRowLines // fallback when width not yet set
		}
		if runeCount := len([]rune(displayVal)); runeCount > maxChars {
			displayVal = string([]rune(displayVal)[:maxChars]) + "…"
		}
		displayVal = clampToLines(displayVal, maxRowLines)

		// Determine style for value cell
		category := typeMap[k]
		var styledValue any
		switch {
		case fmt.Sprintf("%v", v) == "NULL":
			styledValue = table.NewStyledCell(displayVal, style.NullStyle())
		case category == "json":
			highlighted := style.HighlightJSON(displayVal)
			styledValue = table.NewStyledCell(jsonCellData{Raw: val, highlighted: highlighted}, style.JSONBaseStyle())
		default:
			styledValue = table.NewStyledCell(displayVal, style.ResultCellStyle(category))
		}

		rows = append(rows, table.Row{Data: map[string]any{"field": k, "value": styledValue}})
	}

	m.table = m.table.WithRows(rows)
	m.table = m.table.WithColumns(cols)
}

func (m *ResultRowPopupModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	tableHeight := max(h-1, 1)
	m.table = m.table.WithTargetHeight(tableHeight)
	m.table = m.table.WithMinimumHeight(tableHeight)
	m.table = m.table.WithTargetWidth(w)
}

func (m ResultRowPopupModel) View() string {
	panelStyle := style.GetBasePanelStyle()
	panelStyle = panelStyle.Width(m.width + 2)
	panelStyle = panelStyle.Height(m.height + 2)

	panelStyle = panelStyle.BorderForeground(theme.GetTheme().RowDetailsPopup.BG)

	content := lipgloss.JoinVertical(lipgloss.Left, m.table.View())

	title := style.Title(m.width-2, false).
		Background(theme.GetTheme().RowDetailsPopup.BG).
		Foreground(theme.GetTheme().RowDetailsPopup.FG).
		Align(lipgloss.Center).
		Render("record details")

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		title,
		content,
		style.ShortHelp(m.width).Render(m.helpView()),
	))
}
