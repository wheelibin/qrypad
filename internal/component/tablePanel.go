package component

import (
	"math"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/wheelibin/qrypad/internal/colour"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
)

const (
	TablePanelTabCount       = 2
	TablePanelTabIndexTables = 0
	TablePanelTabIndexViews  = 1
)

type TablePanelModel struct {
	active         bool
	width          int
	height         int
	table          table.Model
	selectedTable  string
	activeTabIndex int
}

func NewTablePanelModel() TablePanelModel {
	t := table.New([]table.Column{}).
		WithBaseStyle(style.TableColumn()).
		HeaderStyle(style.GetTableHeaderStyle()).
		Filtered(true).
		Focused(true)

	return TablePanelModel{table: t, active: true}
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
			m.selectedTable = m.table.HighlightedRow().Data["name"].(string)
		} else {
			m.selectedTable = ""
		}
		cmds = append(cmds, commands.TableSelectionChanged(m.selectedTable))

	case tea.KeyMsg:
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

	if m.active {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
		for _, e := range m.table.GetLastUpdateUserEvents() {
			switch e.(type) {
			case table.UserEventHighlightedIndexChanged:
				if len(m.table.HighlightedRow().Data) > 0 {
					m.selectedTable = m.table.HighlightedRow().Data["name"].(string)
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

	// get cols
	// name
	cols = append(cols, table.NewFlexColumn(data.Columns[0], data.Columns[0], 1).WithFiltered(true))

	if len(data.Columns) > 1 {
		// rows
		cols = append(cols, table.NewColumn(data.Columns[1], data.Columns[1], 12).WithFiltered(true))
	}

	for _, row := range data.Rows {
		rows = append(rows, table.Row{Data: row})
	}

	m.table = m.table.WithRows(rows)
	m.table = m.table.WithColumns(cols)
}

func (m TablePanelModel) GetSelectedTable() string {
	return m.selectedTable
}

func (m *TablePanelModel) SetActive(active bool) {
	m.table = m.table.Focused(active)
	m.active = active
}

func (m *TablePanelModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	rowsInTable := math.Max(float64(h-7), 1)
	m.table = m.table.WithPageSize(int(rowsInTable))
	m.table = m.table.WithMinimumHeight(h - 1)
	m.table = m.table.WithTargetWidth(w)
}

func (m TablePanelModel) GetActiveTabIndex() int {
	return m.activeTabIndex
}

func (m TablePanelModel) View() string {
	panelStyle := style.GetBasePanelStyle()
	panelStyle = panelStyle.Width(m.width)
	panelStyle = panelStyle.Height(m.height)

	panelStyle = panelStyle.BorderForeground(colour.GetTheme().Border.FG)
	if m.active {
		panelStyle = panelStyle.BorderForeground(colour.GetTheme().BorderActive.FG)
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
	v := lipgloss.JoinVertical(lipgloss.Left, title, content)
	return panelStyle.Render(v)
}
