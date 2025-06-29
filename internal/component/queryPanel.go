package component

import (
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/textarea"
	"github.com/wheelibin/qrypad/internal/theme"
)

type queryPanelKeymap struct {
	execute key.Binding
}

type QueryPanelModel struct {
	active           bool
	width            int
	height           int
	queryBuffer      textarea.Model
	connectionName   string
	CurrentStatement *Statement
	dirty            bool
	filename         string
	help             help.Model
	keymap           queryPanelKeymap
}

func NewQueryPanelModel(connectionName string) QueryPanelModel {
	ta := textarea.New()
	ta.Placeholder = "sql statement(s)..."
	ta.Cursor.SetMode(cursor.CursorBlink)
	ta.Cursor.Style = lipgloss.NewStyle().Foreground(theme.GetTheme().Text.FG)
	ta.CharLimit = 0
	ta.FocusedStyle.Prompt = ta.FocusedStyle.Prompt.Foreground(theme.GetTheme().Text.FG)

	// Remove cursor line styling
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle = ta.FocusedStyle
	ta.ShowLineNumbers = false

	return QueryPanelModel{
		connectionName: connectionName,
		queryBuffer:    ta,
		help:           makeHelp(),
		keymap: queryPanelKeymap{
			execute: keys.DefaultKeyMap.ExecuteQuery,
		},
	}
}

func (m QueryPanelModel) Init() tea.Cmd {
	return tea.Batch(commands.ReadOrCreateQueryFile(m.connectionName))
}

func (m QueryPanelModel) Update(msg tea.Msg) (QueryPanelModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// set the prompt to highlight the current statement
	m.queryBuffer.SetPromptFunc(2, func(lineIdx int) string {
		if m.CurrentStatement == nil {
			return ""
		}
		if lineIdx >= m.CurrentStatement.StartLine && lineIdx <= m.CurrentStatement.EndLine {
			return "┃ "
		}
		return ""
	})

	switch msg.(type) {
	case tea.FocusMsg:
		cmds = append(cmds, m.queryBuffer.Focus())

	case commands.EditorFinishedMsg:
		cmds = append(cmds, commands.ReadOrCreateQueryFile(m.connectionName))
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
	statementAtCursor := getStatementAtCursor(m.queryBuffer.Value(), m.queryBuffer.Line())
	return statementAtCursor.Text
}

func (m QueryPanelModel) GetValue() string {
	return m.queryBuffer.Value()
}

func (m QueryPanelModel) GetFilename() string {
	return m.filename
}

func (m *QueryPanelModel) SetValue(value string) {
	m.queryBuffer.SetValue(value)
}

func (m *QueryPanelModel) SetFilename(f string) {
	m.filename = f
}

func (m *QueryPanelModel) SetDirty(dirty bool) {
	m.dirty = dirty
}

func (m *QueryPanelModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.queryBuffer.SetWidth(m.width)
	m.queryBuffer.SetHeight(m.height - style.CurrentStatementHeight - style.TitleHeight - style.Margin)
}

func (m *QueryPanelModel) SetActive(active bool) {
	m.active = active
}

func (m QueryPanelModel) helpView() string {
	return m.help.ShortHelpView([]key.Binding{
		m.keymap.execute,
	})
}

func (m QueryPanelModel) View() string {
	panelStyle := style.GetBasePanelStyle()
	panelStyle = panelStyle.Width(m.width)
	panelStyle = panelStyle.Height(m.height)

	panelStyle = panelStyle.BorderForeground(theme.GetTheme().Border.FG)
	if m.active {
		panelStyle = panelStyle.BorderForeground(theme.GetTheme().BorderActive.FG)
	}

	text := "queries"
	if m.dirty {
		text = text + " [+]"
	}
	title := style.Title(m.width-2, m.active).MarginBottom(1).Render(text)

	v := lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.queryBuffer.View(),
		style.ShortHelp(m.width).Render(m.helpView()),
	)
	return panelStyle.Render(v)
}
