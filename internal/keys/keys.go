package keys

import (
	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	Quit            key.Binding
	Up              key.Binding
	Down            key.Binding
	Left            key.Binding
	Right           key.Binding
	NextPanel       key.Binding
	PrevPanel       key.Binding
	ExecuteQuery    key.Binding
	ViewData        key.Binding
	ToggleLeftPanel key.Binding
	SaveQuery       key.Binding
	ReloadQuery     key.Binding
	ClosePopup      key.Binding
	Help            key.Binding
	NextTab         key.Binding
	PrevTab         key.Binding
	OpenInEditor    key.Binding
	SwitchDatabase  key.Binding
	CopyValue       key.Binding
	UpdatePassword  key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view. It's part
// of the key.Map interface.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view. It's part of the
// key.Map interface.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.SwitchDatabase, k.NextPanel, k.PrevPanel, k.ToggleLeftPanel},
		{k.ExecuteQuery, k.ViewData, k.SaveQuery, k.ReloadQuery, k.OpenInEditor},
		{k.Help, k.ClosePopup, k.Quit},
	}
}

var DefaultKeyMap = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("f1"),
		key.WithHelp("F1", "toggle help"),
	),
	SwitchDatabase: key.NewBinding(
		key.WithKeys("f2"),
		key.WithHelp("F2", "switch database"),
	),
	UpdatePassword: key.NewBinding(
		key.WithKeys("f3"),
		key.WithHelp("F3", "update password"),
	),
	NextPanel: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next panel"),
	),
	PrevPanel: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous panel"),
	),
	ExecuteQuery: key.NewBinding(
		key.WithKeys("f5"),
		key.WithHelp("F5", "execute query under cursor"),
	),
	ViewData: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "view table data / view result row"),
	),
	ToggleLeftPanel: key.NewBinding(
		key.WithKeys("ctrl+t"),
		key.WithHelp("ctrl+t", "toggle table panel"),
	),
	SaveQuery: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save query"),
	),
	ReloadQuery: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "reload query"),
	),
	ClosePopup: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "close popup"),
	),
	NextTab: key.NewBinding(
		key.WithKeys("]"),
		key.WithHelp("]", "next tab"),
	),
	PrevTab: key.NewBinding(
		key.WithKeys("["),
		key.WithHelp("[", "previous tab"),
	),
	OpenInEditor: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "open query in editor"),
	),
	CopyValue: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy value"),
	),
}
