package component

import (
	"fmt"
	"slices"

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

	colNameName   = "name"
	colNameSchema = "schema"
)

type tablePanelKeymap struct {
	viewData      key.Binding
	viewDataDesc  key.Binding
	copy          key.Binding
	refreshSchema key.Binding
}

//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
type TablePanelModel struct {
	active         bool
	width          int
	height         int
	table          table.Model
	selectedTable  db.TableReference
	allRefs        []db.TableReference
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
	t = withRowBorders(t)

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
			refreshSchema: key.NewBinding(
				key.WithKeys(keys.DefaultKeyMap.RefreshSchema.Keys()...),
				key.WithHelp(keys.DefaultKeyMap.RefreshSchema.Help().Key, "refresh schema"),
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
		m.selectedTable = m.highlightedRef()
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
				m.selectedTable = m.highlightedRef()
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
	refs := make([]db.TableReference, 0, len(data.Rows))

	// Check whether this dataset includes a schema column
	hasSchema := slices.Contains(data.Columns, colNameSchema)

	// Build columns: schema (fixed 16 chars, if present), name (flex), other columns (fixed 12)
	if hasSchema {
		cols = append(cols, table.NewColumn(colNameSchema, colNameSchema, 16).WithFiltered(true))
	}
	cols = append(cols, table.NewFlexColumn(colNameName, colNameName, 1).WithFiltered(true))
	for _, col := range data.Columns {
		if col != colNameSchema && col != colNameName {
			cols = append(cols, table.NewColumn(col, col, 12).WithFiltered(true))
		}
	}

	for _, row := range data.Rows {
		styledRow := make(map[string]any, len(row))

		for _, colName := range data.Columns {
			val := row[colName]
			switch colName {
			case colNameSchema:
				styledRow[colName] = table.NewStyledCell(val, style.ResultCellStyle("binary"))
			case colNameName:
				styledRow[colName] = table.NewStyledCell(val, style.ResultCellStyle("string"))
			case "rows":
				styledRow[colName] = table.NewStyledCell(val, style.ResultCellStyle("number"))
			}
		}

		rows = append(rows, table.Row{Data: styledRow})

		name := ""
		if n, ok := row[colNameName]; ok {
			name = fmt.Sprintf("%v", n)
		}
		schema := ""
		if hasSchema {
			if s, ok := row[colNameSchema]; ok {
				schema = fmt.Sprintf("%v", s)
			}
		}
		refs = append(refs, db.TableReference{Schema: schema, Name: name})
	}

	m.allRefs = refs
	m.table = m.table.WithRows(rows)
	m.table = m.table.WithColumns(cols)
}

func (m TablePanelModel) GetSelectedTable() db.TableReference {
	return m.selectedTable
}

// GetAllTableRefs returns all table/view references currently loaded in the panel.
func (m TablePanelModel) GetAllTableRefs() []db.TableReference {
	return m.allRefs
}

func (m TablePanelModel) highlightedRef() db.TableReference {
	raw := m.table.HighlightedRow().Data

	if len(raw) == 0 {
		return db.TableReference{}
	}
	schema, _ := unwrapCellData(raw[colNameSchema]).(string)
	name, _ := unwrapCellData(raw[colNameName]).(string)
	return db.TableReference{Schema: schema, Name: name}
}

func (m *TablePanelModel) SetActive(active bool) {
	m.table = m.table.Focused(active)
	m.active = active
}

func (m *TablePanelModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	tableHeight := max(h-2, 1)
	m.table = m.table.WithTargetHeight(tableHeight)
	m.table = m.table.WithMinimumHeight(tableHeight)
	m.table = m.table.WithTargetWidth(w)
	m.showHelp = m.width >= HelpShownMinWidth
}

func (m TablePanelModel) GetActiveTabIndex() int {
	return m.activeTabIndex
}

func (m TablePanelModel) helpView() string {
	bindings := []key.Binding{
		m.keymap.viewData,
		m.keymap.viewDataDesc,
		m.keymap.copy,
	}
	return m.help.ShortHelpView(bindings)
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
