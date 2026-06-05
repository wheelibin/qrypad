package component

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

type autoCompleteItem string

func (i autoCompleteItem) FilterValue() string { return string(i) }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(autoCompleteItem)
	if !ok {
		return
	}

	str := string(i)

	itemStyle := lipgloss.NewStyle().Foreground(theme.GetTheme().Text.FG)
	selectedItemStyle := lipgloss.NewStyle().
		Background(theme.GetTheme().PanelTitleActive.BG).
		Foreground(theme.GetTheme().PanelTitleActive.FG)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = selectedItemStyle.Render
	}

	fmt.Fprint(w, fn(str))
}

//nolint:recvcheck // Bubble Tea model: Init/View use value receiver, mutating methods use pointer receiver
type AutoCompletePopupModel struct {
	list     list.Model
	active   bool
	filter   string
	allItems []string
}

func NewAutoCompletePopupModel() AutoCompletePopupModel {
	l := list.New([]list.Item{}, itemDelegate{}, 20, 7)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowTitle(false)
	l.SetShowPagination(false)
	// Disable default key bindings -- we handle keys ourselves
	l.KeyMap.Quit.SetEnabled(false)
	l.KeyMap.ForceQuit.SetEnabled(false)
	l.KeyMap.Filter.SetEnabled(false)
	l.KeyMap.ClearFilter.SetEnabled(false)
	l.KeyMap.ShowFullHelp.SetEnabled(false)
	l.KeyMap.CloseFullHelp.SetEnabled(false)
	return AutoCompletePopupModel{
		list: l,
	}
}

func (m AutoCompletePopupModel) Init() tea.Cmd {
	return nil
}

func (m *AutoCompletePopupModel) SetActive(active bool) {
	m.active = active
}

func (m *AutoCompletePopupModel) SetFilter(filter string) {
	m.filter = filter
	m.applyFilter()
}

func (m *AutoCompletePopupModel) applyFilter() {
	filtered := make([]list.Item, 0)
	lower := strings.ToLower(m.filter)
	for _, item := range m.allItems {
		if strings.Contains(strings.ToLower(item), lower) {
			filtered = append(filtered, autoCompleteItem(item))
		}
	}
	m.list.SetItems(filtered)
}

func (m AutoCompletePopupModel) Update(msg tea.Msg) (AutoCompletePopupModel, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			return m, commands.AutoCompleteClose()
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			cmds = append(cmds,
				commands.AutoCompleteEntrySelect(fmt.Sprintf("%s", m.list.SelectedItem())),
				commands.AutoCompleteClose(),
			)
			return m, tea.Batch(cmds...)
		}
	}

	if m.active {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *AutoCompletePopupModel) SetItems(items []string) tea.Cmd {
	m.allItems = items
	m.filter = ""
	listItems := make([]list.Item, 0, len(items))
	for _, item := range items {
		listItems = append(listItems, autoCompleteItem(item))
	}
	return m.list.SetItems(listItems)
}

func (m AutoCompletePopupModel) View() string {
	theme := theme.GetTheme()
	return style.GetBasePanelStyle().
		Foreground(theme.Text.FG).
		Padding(0, 1).
		Render(m.list.View())
}
