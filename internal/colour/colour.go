package colour

import "github.com/charmbracelet/lipgloss"

var (
	background = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#24273a"}
	lightGrey  = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#b8c0e0"}
	darkGrey   = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#494d64"}
	black      = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#181926"}

	green  = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#a6da95"}
	teal   = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#8bd5ca"}
	blue   = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#91d7e3"}
	orange = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#f5a97f"}
	yellow = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#eed49f"}
	red    = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#ed8796"}

	Border             = darkGrey
	BorderActive       = green
	PanelTitleActiveBG = green
	PanelTitleBG       = darkGrey
	PanelTitleActiveFG = background
	// ListItemTitleFG = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#a6da95"}
	ListItemDescFG          = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#8087a2"}
	ListItemSelectedTitleFG = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#eed49f"}
	ListItemSelectedDescFG  = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#eed49f"}
	CurrentStatementBG      = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#eed49f"}
	CurrentStatementFG      = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#24273a"}
	Spinner                 = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#f5bde6"}
	ResultsTableBorder      = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#494d64"}

	StatusBarBG = darkGrey
	StatusBarFG = blue
	TitleBarBG  = darkGrey
	TitleBarFG  = blue
	Error       = red

	ResultRowPopupTitleBG = yellow
	HelpBorder            = orange
	HelpKey               = orange
	HelpDesc              = lipgloss.NoColor{}
)
