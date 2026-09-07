package component

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/evertras/bubble-table/table"

	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

type sessionListKeymap struct {
	switchTo key.Binding
	close    key.Binding
	cancel   key.Binding
}

//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
type SessionListPopupModel struct {
	width  int
	height int
	table  table.Model
	keymap sessionListKeymap
	help   help.Model
}

func NewSessionListPopupModel() SessionListPopupModel {
	t := table.New([]table.Column{}).
		WithBaseStyle(style.TableColumn()).
		HighlightStyle(style.GetTableHighlightStyle()).
		WithBorderForeground(style.GetTableBorderForeground()).
		BorderRounded().
		WithHeaderVisibility(false).
		Filtered(true).
		Focused(true)

	return SessionListPopupModel{
		table: t,
		help:  makeHelp(),
		keymap: sessionListKeymap{
			switchTo: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "switch")),
			close:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "close")),
			cancel:   key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		},
	}
}

func (m *SessionListPopupModel) SetEntries(entries []SessionListEntry) {
	cols := []table.Column{
		table.NewFlexColumn("label", "label", 1).WithFiltered(true),
	}

	rows := make([]table.Row, 0, len(entries))
	for _, e := range entries {
		db := e.DBName
		if db == "" {
			db = "—"
		}
		indicator := "  "
		if e.IsActive {
			indicator = "● "
		}
		label := indicator + e.ConnName + " · " + db
		rows = append(rows, table.Row{
			Data: map[string]any{
				"label":    label,
				"connName": e.ConnName,
				"isActive": e.IsActive,
			},
		})
	}

	m.table = m.table.WithRows(rows).WithColumns(cols)
	m.SetSize(m.width, m.height)
}

// SessionListEntry is one row in the session list popup.
type SessionListEntry struct {
	ConnName string
	DBName   string
	IsActive bool
}

func (m *SessionListPopupModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	tableHeight := max(h-1, 1)
	m.table = m.table.WithTargetHeight(tableHeight)
	m.table = m.table.WithMinimumHeight(tableHeight)
	m.table = m.table.WithTargetWidth(w)
}

func (m SessionListPopupModel) Init() tea.Cmd { return nil }

func (m SessionListPopupModel) Update(msg tea.Msg) (SessionListPopupModel, tea.Cmd) {
	var cmd tea.Cmd
	cmds := make([]tea.Cmd, 0, 1)

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	if kMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(kMsg, m.keymap.switchTo):
			row := m.table.HighlightedRow()
			if len(row.Data) == 0 {
				break
			}
			isActive, _ := row.Data["isActive"].(bool)
			if isActive {
				return m, commands.ClosePopup()
			}
			connName, _ := row.Data["connName"].(string)
			return m, commands.ConnectionSelectionChanged(connName)

		case key.Matches(kMsg, m.keymap.close):
			row := m.table.HighlightedRow()
			if len(row.Data) == 0 {
				break
			}
			isActive, _ := row.Data["isActive"].(bool)
			connName, _ := row.Data["connName"].(string)
			if isActive && len(m.table.GetVisibleRows()) == 1 {
				break // can't close the only session
			}
			return m, commands.SessionClose(connName)

		case key.Matches(kMsg, m.keymap.cancel):
			return m, commands.ClosePopup()
		}
	}

	return m, tea.Batch(cmds...)
}

func (m SessionListPopupModel) View() string {
	t := theme.GetTheme()
	panelStyle := style.GetBasePanelStyle().
		Width(m.width + 2).
		Height(m.height + 2).
		BorderForeground(t.ConnectionSwitcherPopup.BG)

	title := style.Title(m.width-2, false).
		Background(t.ConnectionSwitcherPopup.BG).
		Foreground(t.ConnectionSwitcherPopup.FG).
		Align(lipgloss.Center).
		Render("sessions")

	content := lipgloss.JoinVertical(lipgloss.Left, m.table.View())

	helpView := "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.switchTo,
		m.keymap.close,
		m.keymap.cancel,
	})

	return panelStyle.Render(lipgloss.JoinVertical(lipgloss.Left,
		title,
		content,
		style.ShortHelp(m.width).Render(helpView),
	))
}
