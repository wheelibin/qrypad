package keys

import (
	"charm.land/bubbles/v2/key"
)

type keyMap struct {
	AutoComplete     key.Binding
	CancelQuery      key.Binding
	ClosePopup       key.Binding
	CopyValue        key.Binding
	ExecuteQuery     key.Binding
	ExportResults    key.Binding
	FilterTable      key.Binding
	Help             key.Binding
	NextPanel        key.Binding
	NextTab          key.Binding
	OpenInEditor     key.Binding
	PrevPanel        key.Binding
	PrevTab          key.Binding
	Quit             key.Binding
	RefreshSchema    key.Binding
	ReloadQuery      key.Binding
	SaveQuery        key.Binding
	SwitchDatabase   key.Binding
	SwitchConnection key.Binding
	ToggleLeftPanel  key.Binding
	UpdatePassword   key.Binding
	ViewData         key.Binding
	ViewDataDesc     key.Binding
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
		{k.NextPanel, k.PrevPanel, k.Help, k.SwitchDatabase, k.SwitchConnection, k.UpdatePassword, k.FilterTable, k.ToggleLeftPanel, k.RefreshSchema},
		{k.ExecuteQuery, k.SaveQuery, k.ReloadQuery, k.OpenInEditor},
		{k.ViewData, k.NextTab, k.PrevTab, k.CopyValue, k.Quit},
	}
}

//nolint:gochecknoglobals // global keymap is required for Bubble Tea key handling and is modified at startup
var DefaultKeyMap = keyMap{
	AutoComplete: key.NewBinding(
		key.WithKeys("ctrl+space"),
		key.WithHelp("ctrl+space", "autocomplete"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?", "f1"),
		key.WithHelp("?/F1", "show help"),
	),
	SwitchDatabase: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "switch database"),
	),
	UpdatePassword: key.NewBinding(
		key.WithKeys("ctrl+p"),
		key.WithHelp("ctrl+p", "update password"),
	),
	SwitchConnection: key.NewBinding(
		key.WithKeys("ctrl+k"),
		key.WithHelp("ctrl+k", "switch connection"),
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
		key.WithHelp("F5", "execute statement at cursor"),
	),
	ViewData: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "view table data / view result row"),
	),
	ViewDataDesc: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "view table data (desc)"),
	),
	ToggleLeftPanel: key.NewBinding(
		key.WithKeys("ctrl+b"),
		key.WithHelp("ctrl+b", "toggle left panel"),
	),
	SaveQuery: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save query"),
	),
	ReloadQuery: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "reload query"),
	),
	RefreshSchema: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "refresh schema"),
	),
	CancelQuery: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel query"),
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
		key.WithKeys("y"),
		key.WithHelp("y", "copy value"),
	),
	FilterTable: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter table"),
	),
	ExportResults: key.NewBinding(
		key.WithKeys("ctrl+x"),
		key.WithHelp("ctrl+x", "export results"),
	),
}
