package component

import (
	"fmt"
	"math"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

type resultRowPopupKeymap struct {
	copy  key.Binding
	close key.Binding
}

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
		WithHeaderVisibility(false).
		Filtered(true).
		Focused(true)

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

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.copy):
			var valDesc string
			val := m.GetSelectedValue()
			maxLength := 30
			if len(val) > maxLength {
				valDesc = val[0:maxLength-3] + "..."
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
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
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
	panelStyle := style.GetBasePanelStyle()
	panelStyle = panelStyle.Width(m.width)
	panelStyle = panelStyle.Height(m.height)

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
