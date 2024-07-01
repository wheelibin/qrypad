package component

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/dbee/internal/colour"
	"github.com/wheelibin/dbee/internal/commands"
	"github.com/wheelibin/dbee/internal/keys"
	"github.com/wheelibin/dbee/internal/style"
)

type QueryPanelModel struct {
	active           bool
	width            int
	height           int
	queryBuffer      textarea.Model
	dbAlias          string
	CurrentStatement string
	dirty            bool
}

func NewQueryPanelModel(dbAlias string) QueryPanelModel {
	ta := textarea.New()
	ta.Placeholder = "sql statement(s)..."
	ta.Prompt = "┃ "
	ta.Cursor.SetMode(cursor.CursorStatic)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	return QueryPanelModel{dbAlias: dbAlias, queryBuffer: ta}
}

func (m QueryPanelModel) Init() tea.Cmd {
	return tea.Batch(commands.ReadOrCreateQueryFile(m.dbAlias))
}

func (m QueryPanelModel) Update(msg tea.Msg) (QueryPanelModel, tea.Cmd) {
	// log.Println("queryPanel.model::Update", msg)
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {

	case commands.QueryFileReadMsg:
		m.queryBuffer.SetValue(string(msg))

	}

	// update components
	if m.active {
		if !m.queryBuffer.Focused() {
			cmds = append(cmds, m.queryBuffer.Focus())
		}
		m.queryBuffer, cmd = m.queryBuffer.Update(msg)
		cmds = append(cmds, cmd)

		m.CurrentStatement = getStatementAtCursor(m.queryBuffer.Value(), m.queryBuffer.Line())

	} else {
		m.queryBuffer.Blur()
	}

	return m, tea.Batch(cmds...)
}

func (m QueryPanelModel) GetCurrentStatement() string {
	return getStatementAtCursor(m.queryBuffer.Value(), m.queryBuffer.Line())
}

func (m QueryPanelModel) GetValue() string {
	return m.queryBuffer.Value()
}

func (m *QueryPanelModel) SetDirty(dirty bool) {
	m.dirty = dirty
}

func (m *QueryPanelModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.queryBuffer.SetWidth(m.width)
	m.queryBuffer.SetHeight(m.height - style.CurrentStatementHeight - style.TitleHeight - style.Margin - 1)
}

func (m *QueryPanelModel) SetActive(active bool) {
	m.active = active
}

func (m QueryPanelModel) View() string {
	var panelStyle = style.BasePanelStyle
	panelStyle = panelStyle.Width(m.width)
	panelStyle = panelStyle.Height(m.height)

	panelStyle = panelStyle.BorderForeground(colour.Border)
	if m.active {
		panelStyle = panelStyle.BorderForeground(colour.BorderActive)
	}

	currentStatementStyle := lipgloss.NewStyle().
		Background(colour.CurrentStatementBG).
		Foreground(colour.CurrentStatementFG).
		MarginLeft(1).
		MarginTop(1)

	currentStatement := currentStatementStyle.Render("")

	if len(m.CurrentStatement) > 0 && m.active {
		currentStatement = currentStatementStyle.Render(fmt.Sprintf("(%s) execute: %s", keys.DefaultKeyMap.ExecuteQuery.Keys()[0], strings.ReplaceAll(m.CurrentStatement, "\n", " ")))
	}

	text := "queries"
	if m.dirty {
		text = text + " [+]"
	}
	title := style.Title(m.width-2, m.active).MarginBottom(1).Render(text)

	v := lipgloss.JoinVertical(lipgloss.Left, title, m.queryBuffer.View(), currentStatement)
	return panelStyle.Render(v)
}
