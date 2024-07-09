package keys

import (
	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	Quit                key.Binding
	Up                  key.Binding
	Down                key.Binding
	Left                key.Binding
	Right               key.Binding
	NextPanel           key.Binding
	PrevPanel           key.Binding
	ExecuteQuery        key.Binding
	ViewData            key.Binding
	ToggleLeftPanel     key.Binding
	SaveQuery           key.Binding
	ReloadQuery         key.Binding
	CloseResultRowPopup key.Binding
	Help                key.Binding
	NextTab             key.Binding
	PrevTab             key.Binding
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
		{k.NextPanel, k.PrevPanel, k.ToggleLeftPanel},
		{k.ExecuteQuery, k.ViewData, k.SaveQuery, k.ReloadQuery},
		{k.Help, k.CloseResultRowPopup, k.Quit},
	}
}

var DefaultKeyMap = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	// Up: key.NewBinding(
	// 	key.WithKeys("k", "up"),        // actual keybindings
	// 	key.WithHelp("↑/k", "move up"), // corresponding help text
	// ),
	// Down: key.NewBinding(
	// 	key.WithKeys("j", "down"),
	// 	key.WithHelp("↓/j", "move down"),
	// ),
	// Left: key.NewBinding(
	// 	key.WithKeys("h", "left"),
	// 	key.WithHelp("↓/h", "move left"),
	// ),
	// Right: key.NewBinding(
	// 	key.WithKeys("l", "right"),
	// 	key.WithHelp("↓/l", "move right"),
	// ),
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
		key.WithHelp("f5", "execute query"),
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
	CloseResultRowPopup: key.NewBinding(
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
}
