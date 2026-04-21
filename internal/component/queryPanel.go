package component

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"charm.land/bubbles/v2/cursor"
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/textarea"
	"github.com/wheelibin/qrypad/internal/theme"
)

type queryPanelKeymap struct {
	execute         key.Binding
	openInEditor    key.Binding
	saveReloadQuery key.Binding
}

//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
type QueryPanelModel struct {
	active              bool
	width               int
	height              int
	queryBuffer         textarea.Model
	connectionName      string
	CurrentStatement    *Statement
	dirty               bool
	filename            string
	abbreviatedFilename string
	help                help.Model
	keymap              queryPanelKeymap
	autoSaveEnabled     bool
	autoCompleteOptions []string
	autoCompletePopup   AutoCompletePopupModel
}

func NewQueryPanelModel(connectionName string, autoSaveEnabled bool) QueryPanelModel {
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

	ac := NewAutoCompletePopupModel()

	return QueryPanelModel{
		connectionName:    connectionName,
		queryBuffer:       ta,
		help:              makeHelp(),
		autoCompletePopup: ac,
		keymap: queryPanelKeymap{
			execute:      keys.DefaultKeyMap.ExecuteQuery,
			openInEditor: keys.DefaultKeyMap.OpenInEditor,
			saveReloadQuery: key.NewBinding(
				key.WithKeys(
					fmt.Sprintf(
						"%s/%s",
						keys.DefaultKeyMap.SaveQuery.Help().Key,
						keys.DefaultKeyMap.ReloadQuery.Help().Key,
					),
					"save/reload query",
				),
				key.WithHelp(
					fmt.Sprintf(
						"%s/%s",
						keys.DefaultKeyMap.SaveQuery.Help().Key,
						keys.DefaultKeyMap.ReloadQuery.Help().Key,
					),
					"save/reload query",
				),
			),
		},
		autoSaveEnabled: autoSaveEnabled,
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

	updateQueryBuffer := true

	switch msg := msg.(type) {
	case tea.FocusMsg:
		cmds = append(cmds, m.queryBuffer.Focus())

	case commands.EditorFinishedMsg:
		cmds = append(cmds, commands.ReadOrCreateQueryFile(m.connectionName))

	case commands.AutoCompleteEntrySelectedMsg:
		m.queryBuffer = replaceFuzzyPrefixInTextarea(m.queryBuffer, string(msg))

	case commands.AutoCompleteCloseMsg:
		m.autoCompletePopup.SetActive(false)
		m.autoCompletePopup.SetFilter("")

	case tea.KeyPressMsg:
		acKeys := []string{"up", "down", "esc", "enter"}
		if slices.Contains(acKeys, msg.String()) {
			updateQueryBuffer = !m.autoCompletePopup.active
		}
	}

	// update components
	if m.active {
		if !m.queryBuffer.Focused() {
			cmds = append(cmds, m.queryBuffer.Focus())
		}

		if m.autoCompletePopup.active {
			m.autoCompletePopup, cmd = m.autoCompletePopup.Update(msg)
			cmds = append(cmds, cmd)
		}
		if updateQueryBuffer {
			m.queryBuffer, cmd = m.queryBuffer.Update(msg)
			cmds = append(cmds, cmd)
		}

		m.CurrentStatement = getStatementAtCursor(m.queryBuffer.Value(), m.queryBuffer.Line())
	} else {
		m.queryBuffer.Blur()
	}

	return m, tea.Batch(cmds...)
}

func (m QueryPanelModel) GetCurrentStatement() string {
	statementAtCursor := getStatementAtCursor(m.queryBuffer.Value(), m.queryBuffer.Line())
	if statementAtCursor != nil {
		return statementAtCursor.Text
	}
	return ""
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
	m.abbreviatedFilename = abbreviatePath(f)
}

func (m *QueryPanelModel) SetDirty(dirty bool) {
	m.dirty = dirty
}

func (m *QueryPanelModel) SetConnectionName(name string) {
	m.connectionName = name
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

func (m *QueryPanelModel) SetAutoCompleteActive(active bool) {
	m.autoCompletePopup.SetActive(active)
}

func (m QueryPanelModel) GetAutoCompleteActive() bool {
	return m.autoCompletePopup.active
}

func (m *QueryPanelModel) SetAutoCompleteFilter(filter string) {
	m.autoCompletePopup.SetFilter(filter)
}

func (m *QueryPanelModel) AutoCompleteFilterAppend(value string) {
	m.autoCompletePopup.SetFilter(m.autoCompletePopup.filter + value)
}

func (m *QueryPanelModel) AutoCompleteFilterPop() {
	f := m.autoCompletePopup.filter
	if len(f) > 0 {
		m.autoCompletePopup.SetFilter(f[:len(f)-1])
	}
}

func (m QueryPanelModel) GetWordAtCursor() string {
	return getWordAtCursor(m.queryBuffer.Value(), m.queryBuffer.Row, m.queryBuffer.Col)
}

func (m *QueryPanelModel) SetAutoCompleteOptions(opts []string) tea.Cmd {
	m.autoCompleteOptions = opts
	return m.autoCompletePopup.SetItems(opts)
}

func (m QueryPanelModel) helpView() string {
	km := []key.Binding{
		m.keymap.execute,
		m.keymap.openInEditor,
	}
	if !m.autoSaveEnabled {
		km = append(km, m.keymap.saveReloadQuery)
	}
	return m.help.ShortHelpView(km)
}

func (m QueryPanelModel) View() string {
	panelStyle := style.GetBasePanelStyle()
	panelStyle = panelStyle.Width(m.width + 2)
	panelStyle = panelStyle.Height(m.height + 2)

	panelStyle = panelStyle.BorderForeground(theme.GetTheme().Border.FG)
	if m.active {
		panelStyle = panelStyle.BorderForeground(theme.GetTheme().BorderActive.FG)
	}

	leftText := "queries"
	if m.dirty {
		leftText += " [+]"
	}

	// titleStyle drives background/foreground/bold; Width is removed so we can
	// split the bar into left+right segments manually.
	baseStyle := style.Title(m.width-2, m.active)
	bg := baseStyle.GetBackground()
	fg := baseStyle.GetForeground()

	// Render left segment without a fixed width so we know its natural width.
	leftSegStyle := lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Bold(true).
		PaddingLeft(1)
	left := leftSegStyle.Render(leftText)
	leftWidth := lipgloss.Width(left)

	// Total inner width of the title bar (Width passed to style.Title, minus the
	// MarginLeft(1) that style.Title adds outside the box).
	innerWidth := m.width - 2 // matches Width arg passed to style.Title

	rightText := m.abbreviatedFilename
	// Truncate to avoid crowding the left label (keep 2 rune padding).
	// Use rune conversion to avoid splitting multi-byte characters.
	available := innerWidth - leftWidth - 2
	if available < 0 {
		available = 0
	}
	if lipgloss.Width(rightText) > available {
		runes := []rune(rightText)
		// Trim from the left one rune at a time until the result fits.
		// "…" is 1 display column wide.
		truncated := ""
		for i := 1; i < len(runes); i++ {
			candidate := "…" + string(runes[i:])
			if lipgloss.Width(candidate) <= available {
				truncated = candidate
				break
			}
		}
		rightText = truncated // empty string if nothing fits
	}

	rightWidth := innerWidth - leftWidth
	if rightWidth < 0 {
		rightWidth = 0
	}
	// When active, match the label foreground so the path stays readable against
	// the bright active title background. When inactive, use the faded status bar colour.
	rightFG := theme.GetTheme().StatusBar.FG
	if m.active {
		rightFG = fg
	}
	right := lipgloss.NewStyle().
		Background(bg).
		Foreground(rightFG).
		Bold(m.active).
		Width(rightWidth).
		AlignHorizontal(lipgloss.Right).
		PaddingRight(1).
		Render(rightText)

	titleRow := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	title := lipgloss.NewStyle().MarginLeft(1).MarginBottom(1).Render(titleRow)
	qb := m.queryBuffer.View()

	if m.autoCompletePopup.active {
		autoComplete := m.autoCompletePopup.View()
		// Use viewport-relative row so the popup follows the cursor when scrolled
		viewRow := m.queryBuffer.CursorViewRow()
		qb = style.PlaceOverlay(m.queryBuffer.Col+5, viewRow, autoComplete, qb)
	}

	v := lipgloss.JoinVertical(lipgloss.Left,
		title,
		qb,
		style.ShortHelp(m.width).Render(m.helpView()),
	)
	return panelStyle.Render(v)
}

// abbreviatePath replaces the user's home directory prefix with "~".
// Returns path unchanged if home cannot be determined or path is empty.
func abbreviatePath(path string) string {
	if path == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+string(os.PathSeparator)) {
		return "~" + path[len(home):]
	}
	return path
}
