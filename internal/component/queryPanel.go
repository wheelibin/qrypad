package component

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/colour"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/textarea"
)

type QueryPanelModel struct {
	active           bool
	width            int
	height           int
	queryBuffer      textarea.Model
	dbAlias          string
	CurrentStatement string
	dirty            bool
	filename         string
}

func NewQueryPanelModel(dbAlias string) QueryPanelModel {
	ta := textarea.New()
	ta.Placeholder = "sql statement(s)..."
	ta.Prompt = "┃ "
	ta.Cursor.SetMode(cursor.CursorBlink)
	ta.CharLimit = 0

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle = ta.FocusedStyle
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
		m.filename = msg.FileName
		m.queryBuffer.SetValue(string(msg.Contents))

	case commands.EditorFinishedMsg:
		cmds = append(cmds, commands.ReadOrCreateQueryFile(m.dbAlias))
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

func (m QueryPanelModel) GetFilename() string {
	return m.filename
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
	panelStyle := style.BasePanelStyle
	panelStyle = panelStyle.Width(m.width)
	panelStyle = panelStyle.Height(m.height)

	panelStyle = panelStyle.BorderForeground(colour.GetTheme().Border.FG)
	if m.active {
		panelStyle = panelStyle.BorderForeground(colour.GetTheme().BorderActive.FG)
	}

	currentStatementStyle := lipgloss.NewStyle().
		Background(colour.GetTheme().CurrentStatement.BG).
		Foreground(colour.GetTheme().CurrentStatement.FG).
		MarginLeft(1).
		MarginTop(1)

	currentStatement := currentStatementStyle.Render("")

	if len(m.CurrentStatement) > 0 && m.active {
		var truncated string
		if len(m.CurrentStatement) > m.width-14 {
			truncated = m.CurrentStatement[:m.width-17] + "..."
		} else {
			truncated = m.CurrentStatement
		}
		s := strings.ReplaceAll(strings.ReplaceAll(truncated, "\n", " "), "  ", " ")
		currentStatement = currentStatementStyle.Render(fmt.Sprintf("(%s) execute: %s", keys.DefaultKeyMap.ExecuteQuery.Help().Key, s))
	}

	text := "queries"
	if m.dirty {
		text = text + " [+]"
	}
	title := style.Title(m.width-2, m.active).MarginBottom(1).Render(text)

	v := lipgloss.JoinVertical(lipgloss.Left, title, m.queryBuffer.View(), currentStatement)
	return panelStyle.Render(v)
}
