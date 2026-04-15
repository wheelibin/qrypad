package component

import (
	"fmt"
	"math"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evertras/bubble-table/table"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

const (
	TablePanelTabCount       = 2
	TablePanelTabIndexTables = 0
	TablePanelTabIndexViews  = 1
	HelpShownMinWidth        = 55
)

type tablePanelKeymap struct {
	viewData     key.Binding
	viewDataDesc key.Binding
	copy         key.Binding
}

//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
type TablePanelModel struct {
	active         bool
	width          int
	height         int
	table          table.Model
	selectedTable  string
	allNames       []string
	activeTabIndex int
	help           help.Model
	keymap         tablePanelKeymap
	showHelp       bool
}

func NewTablePanelModel() TablePanelModel {
	t := table.New([]table.Column{}).
		WithBaseStyle(style.TableColumn()).
		HeaderStyle(style.GetTableHeaderStyle()).
		HighlightStyle(style.GetTableHighlightStyle()).
		WithBorderForeground(style.GetTableBorderForeground()).
		BorderRounded().
		Filtered(true).
		Focused(true)

	return TablePanelModel{
		table:  t,
		active: true,
		help:   makeHelp(),
		keymap: tablePanelKeymap{
			viewData: key.NewBinding(
				key.WithKeys(keys.DefaultKeyMap.ViewData.Keys()...),
				key.WithHelp(keys.DefaultKeyMap.ViewData.Help().Key, "view data"),
			),
			viewDataDesc: key.NewBinding(
				key.WithKeys(keys.DefaultKeyMap.ViewDataDesc.Keys()...),
				key.WithHelp(keys.DefaultKeyMap.ViewDataDesc.Help().Key, "view data (desc)"),
			),
			copy: key.NewBinding(
				key.WithKeys(keys.DefaultKeyMap.CopyValue.Keys()...),
				key.WithHelp(keys.DefaultKeyMap.CopyValue.Help().Key, "copy name"),
			),
		},
	}
}

func (m TablePanelModel) Init() tea.Cmd {
	return nil
}

func (m TablePanelModel) Update(msg tea.Msg) (TablePanelModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case db.SchemaEntitiesFetchedMsg:
		if len(m.table.HighlightedRow().Data) > 0 {
			if name, ok := m.table.HighlightedRow().Data["name"].(string); ok {
				m.selectedTable = name
			}
		} else {
			m.selectedTable = ""
		}
		cmds = append(cmds, commands.TableSelectionChanged(m.selectedTable))

	case tea.KeyPressMsg:
		if m.active {
			switch {
			case key.Matches(msg, keys.DefaultKeyMap.NextTab):
				m.activeTabIndex = (m.activeTabIndex + 1) % TablePanelTabCount
				cmd = commands.SetActiveTablePanelTab(m.activeTabIndex)
				cmds = append(cmds, cmd)

			case key.Matches(msg, keys.DefaultKeyMap.PrevTab):
				i := m.activeTabIndex - 1
				if i < 0 {
					i = TablePanelTabCount - 1
				}
				m.activeTabIndex = i
				cmd = commands.SetActiveTablePanelTab(m.activeTabIndex)
				cmds = append(cmds, cmd)
			}
		}
	}

	if m.active {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
		for _, e := range m.table.GetLastUpdateUserEvents() {
			if _, ok := e.(table.UserEventHighlightedIndexChanged); ok {
				if len(m.table.HighlightedRow().Data) > 0 {
					if name, ok := m.table.HighlightedRow().Data["name"].(string); ok {
						m.selectedTable = name
					}
				} else {
					m.selectedTable = ""
				}
				cmds = append(cmds, commands.TableSelectionChanged(m.selectedTable))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *TablePanelModel) SetData(data *db.Data) {
	if data == nil {
		return
	}

	cols := []table.Column{}
	rows := []table.Row{}
	names := make([]string, 0, len(data.Rows))

	// get cols
	// name
	cols = append(cols, table.NewFlexColumn(data.Columns[0], data.Columns[0], 1).WithFiltered(true))

	if len(data.Columns) > 1 {
		// rows
		cols = append(cols, table.NewColumn(data.Columns[1], data.Columns[1], 12).WithFiltered(true))
	}

	for _, row := range data.Rows {
		rows = append(rows, table.Row{Data: row})
		if name, ok := row["name"]; ok {
			names = append(names, fmt.Sprintf("%v", name))
		}
	}

	m.allNames = names
	m.table = m.table.WithRows(rows)
	m.table = m.table.WithColumns(cols)
}

func (m TablePanelModel) GetSelectedTable() string {
	return m.selectedTable
}

// GetAllTableNames returns all table/view names currently loaded in the panel.
func (m TablePanelModel) GetAllTableNames() []string {
	return m.allNames
}

func (m *TablePanelModel) SetActive(active bool) {
	m.table = m.table.Focused(active)
	m.active = active
}

func (m *TablePanelModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	rowsInTable := math.Max(float64(h-9), 1)
	m.table = m.table.WithPageSize(int(rowsInTable))
	m.table = m.table.WithMinimumHeight(h - 2)
	m.table = m.table.WithTargetWidth(w)
	m.showHelp = m.width >= HelpShownMinWidth
}

func (m TablePanelModel) GetActiveTabIndex() int {
	return m.activeTabIndex
}

func (m TablePanelModel) helpView() string {
	return m.help.ShortHelpView([]key.Binding{
		m.keymap.viewData,
		m.keymap.viewDataDesc,
		m.keymap.copy,
	})
}

func (m TablePanelModel) View() string {
	panelStyle := style.GetBasePanelStyle()
	panelStyle = panelStyle.Width(m.width + 2)
	panelStyle = panelStyle.Height(m.height + 2)

	panelStyle = panelStyle.BorderForeground(theme.GetTheme().Border.FG)
	if m.active {
		panelStyle = panelStyle.BorderForeground(theme.GetTheme().BorderActive.FG)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, m.table.View())

	titleStyle := style.Title(m.width-2, m.active)

	var _tabText, _titleText string
	switch m.activeTabIndex {
	case TableInfoTabIndexColumns:
		_tabText = "[tables]  views "
		_titleText = "tables"
	case TableInfoTabIndexIndexes:
		_tabText = " tables  [views]"
		_titleText = "views"
	}

	titleText := lipgloss.NewStyle().Render(_titleText)
	tabText := lipgloss.NewStyle().
		Width((m.width - lipgloss.Width(titleText)) - 4).
		Align(lipgloss.Right).
		Render(_tabText)

	title := titleStyle.Render(lipgloss.JoinHorizontal(lipgloss.Left, titleText, tabText))
	help := ""
	if m.showHelp {
		help = style.ShortHelp(m.width).Render(m.helpView())
	}
	v := lipgloss.JoinVertical(lipgloss.Left,
		title,
		content,
		help,
	)
	return panelStyle.Render(v)
}
