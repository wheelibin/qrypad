package theme

import "sync"

// ResetThemeOnce resets the theme singleton so the next call to GetTheme()
// reloads the theme from scratch. Intended for use in tests only.
func ResetThemeOnce() {
	themeOnce = sync.Once{}
	theme = Theme{}
	errTheme = nil
}
