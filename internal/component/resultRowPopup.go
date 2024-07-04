package component

import (
	"math"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/wheelibin/dbee/internal/colour"
	"github.com/wheelibin/dbee/internal/style"
)

type ResultRowPopupModel struct {
	width  int
	height int
	table  table.Model
}

func NewResultRowPopupModel() ResultRowPopupModel {
	t := table.New([]table.Column{}).
		WithBaseStyle(
			lipgloss.NewStyle().
				BorderForeground(colour.ResultsTableBorder).
				// Foreground(lipgloss.Color("#a7a")).
				Align(lipgloss.Left),
		).
		WithHeaderVisibility(false).
		Filtered(true).
		Focused(true)

	return ResultRowPopupModel{table: t}
}

func (m ResultRowPopupModel) Init() tea.Cmd {
	return nil
}

func (m ResultRowPopupModel) Update(msg tea.Msg) (ResultRowPopupModel, tea.Cmd) {
	// log.Println("resultRowPopup.model::Update", msg)
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *ResultRowPopupModel) SetData(data map[string]any) {
	if data == nil {
		return
	}

	cols := []table.Column{
		table.NewFlexColumn("field", "field", 1).WithFiltered(true),
		table.NewFlexColumn("value", "value", 3).WithFiltered(true),
	}
	rows := []table.Row{}

	for k, v := range data {
		rows = append(rows, table.Row{Data: map[string]any{"field": k, "value": v}})
	}

	m.table = m.table.WithRows(rows)
	m.table = m.table.WithColumns(cols)
	m.table = m.table.SortByAsc("field")
}

func (m *ResultRowPopupModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	rowsInTable := math.Max(float64(h-6), 1)
	m.table = m.table.WithPageSize(int(rowsInTable))
	m.table = m.table.WithMinimumHeight(h - 1)
	m.table = m.table.WithTargetWidth(w)
}

func (m ResultRowPopupModel) View() string {
	var panelStyle = style.BasePanelStyle
	panelStyle = panelStyle.Width(m.width)
	panelStyle = panelStyle.Height(m.height)

	panelStyle = panelStyle.BorderForeground(colour.ResultRowPopupTitleBG)

	content := lipgloss.JoinVertical(lipgloss.Left, m.table.View())

	title := style.Title(m.width-2, false).
		Background(colour.ResultRowPopupTitleBG).
		Foreground(colour.PanelTitleActiveFG).
		Align(lipgloss.Center).
		Render("record details")

	v := lipgloss.JoinVertical(lipgloss.Left, title, content)
	return panelStyle.Render(v)

}
