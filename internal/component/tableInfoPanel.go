package component

import (
	"math"
	"slices"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

const (
	TableInfoTabCount            = 3
	TableInfoTabIndexColumns     = 0
	TableInfoTabIndexIndexes     = 1
	TableInfoTabIndexConstraints = 2
)

type tableInfoKeymap struct {
	viewRow key.Binding
	copy    key.Binding
}

type TableInfoPanelModel struct {
	active         bool
	width          int
	height         int
	table          table.Model
	activeTabIndex int
	help           help.Model
	keymap         tableInfoKeymap
}

func NewTableInfoPanelModel() TableInfoPanelModel {
	t := table.New([]table.Column{}).
		WithBaseStyle(style.TableColumn()).
		HeaderStyle(style.GetTableHeaderStyle()).
		Filtered(true)

	return TableInfoPanelModel{
		table: t,
		help:  makeHelp(),
		keymap: tableInfoKeymap{
			viewRow: key.NewBinding(
				key.WithKeys(keys.DefaultKeyMap.ViewData.Keys()...),
				key.WithHelp(keys.DefaultKeyMap.ViewData.Help().Key, "view row"),
			),
			copy: key.NewBinding(
				key.WithKeys(keys.DefaultKeyMap.CopyValue.Keys()...),
				key.WithHelp(keys.DefaultKeyMap.CopyValue.Help().Key, "copy name"),
			),
		},
	}
}

func (m TableInfoPanelModel) Init() tea.Cmd {
	return nil
}

func (m TableInfoPanelModel) Update(msg tea.Msg) (TableInfoPanelModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	if m.active {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:

		switch {
		case key.Matches(msg, keys.DefaultKeyMap.NextTab):
			m.activeTabIndex = (m.activeTabIndex + 1) % TableInfoTabCount
			cmd = commands.SetActiveTableInfoTab(m.activeTabIndex)
			cmds = append(cmds, cmd)

		case key.Matches(msg, keys.DefaultKeyMap.PrevTab):
			i := m.activeTabIndex - 1
			if i < 0 {
				i = TableInfoTabCount - 1
			}
			m.activeTabIndex = i
			cmd = commands.SetActiveTableInfoTab(m.activeTabIndex)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *TableInfoPanelModel) SetData(data *db.Data) {
	if data == nil {
		return
	}

	cols := []table.Column{}
	rows := []table.Row{}

	autoWidthCols := []string{"type"}
	fixedWidthCols := map[string]int{
		"unique":   8,
		"primary":  8,
		"nullable": 9,
	}

	// get cols
	for _, c := range data.Columns {
		if slices.Contains(autoWidthCols, c) {
			w := getColumnWidth(c, *data, m.width/2)
			cols = append(cols, table.NewColumn(c, c, w).WithFiltered(true))
			continue
		}
		if w, ok := fixedWidthCols[c]; ok {
			cols = append(cols, table.NewColumn(c, c, w).WithFiltered(true))
			continue
		}
		cols = append(cols, table.NewFlexColumn(c, c, 12).WithFiltered(true))
	}
	for _, row := range data.Rows {
		rows = append(rows, table.Row{Data: row})
	}

	m.table = m.table.WithRows(rows)
	m.table = m.table.WithColumns(cols)
}

func (m *TableInfoPanelModel) SetActive(active bool) {
	m.table = m.table.Focused(active)
	m.active = active
}

func (m *TableInfoPanelModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	rowsInTable := math.Max(float64(h-9), 1)
	m.table = m.table.WithPageSize(int(rowsInTable))
	m.table = m.table.WithMinimumHeight(h - 2)
	m.table = m.table.WithMaxTotalWidth(w)
	m.table = m.table.WithTargetWidth(w)
}

func (m TableInfoPanelModel) GetActiveTabIndex() int {
	return m.activeTabIndex
}

func (m TableInfoPanelModel) GetSelectedRow() map[string]any {
	return m.table.HighlightedRow().Data
}

func (m TableInfoPanelModel) helpView() string {
	return m.help.ShortHelpView([]key.Binding{
		m.keymap.viewRow,
		m.keymap.copy,
	})
}

func (m TableInfoPanelModel) View() string {
	panelStyle := style.GetBasePanelStyle()
	panelStyle = panelStyle.Width(m.width)
	panelStyle = panelStyle.Height(m.height)

	panelStyle = panelStyle.BorderForeground(theme.GetTheme().Border.FG)
	if m.active {
		panelStyle = panelStyle.BorderForeground(theme.GetTheme().BorderActive.FG)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, m.table.View())
	titleStyle := style.Title(m.width-2, m.active)
	title := titleStyle.Render("table info")

	tw := lipgloss.Width(title)

	tabTextStyle := lipgloss.NewStyle().Background(titleStyle.GetBackground())
	var tabText string
	switch m.activeTabIndex {
	case TableInfoTabIndexColumns:
		tabText = "[cols]  inds   cons "
	case TableInfoTabIndexIndexes:
		tabText = " cols  [inds]  cons "
	case TableInfoTabIndexConstraints:
		tabText = " cols   inds  [cons]"
	}
	title = style.Title(m.width-2, m.active).Render("table info" + lipgloss.PlaceHorizontal(tw-13, lipgloss.Right, tabTextStyle.Render(tabText)))

	v := lipgloss.JoinVertical(lipgloss.Left,
		title,
		content,
		style.ShortHelp(m.width).Render(m.helpView()),
	)
	return panelStyle.Render(v)
}
