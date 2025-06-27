package style

import (
	"log"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/wheelibin/qrypad/internal/colour"
)

// Render syntax-highlighted code to a string using Chroma
func HighlightText(txt string) string {
	themeName := colour.GetTheme().ThemeName

	if themeName == "kanagawa-wave" {
		themeName = "nordic"
	}

	var sb strings.Builder
	err := quick.Highlight(&sb, txt, "sql", "terminal256", themeName)
	if err != nil {
		log.Println("error highlighting text", err)
	}

	// Return the highlighted code instead of the default textarea
	return sb.String()
}
