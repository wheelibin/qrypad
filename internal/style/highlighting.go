package style

import (
	"log"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
)

func HighlightText(txt string) string {
	// Render syntax-highlighted code to a string using Chroma
	var sb strings.Builder
	err := quick.Highlight(&sb, txt, "sql", "terminal256", "catppuccin-mocha")
	if err != nil {
		log.Println("error highlighting text", err)
	}

	// Return the highlighted code instead of the default textarea
	return sb.String()
}
