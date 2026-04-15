package style

import (
	"bytes"
	"log/slog"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/muesli/ansi"
	"github.com/wheelibin/qrypad/internal/theme"
)

// highlightCache caches Chroma output keyed by raw line text.
// Since the same text always produces the same highlighting,
// this avoids re-running Chroma on unchanged lines.
//
//nolint:gochecknoglobals // performance cache - global by design
var highlightCache = make(map[string]string)

// HighlightText returns syntax-highlighted ANSI text for the given input.
// Results are cached per unique input string.
func HighlightText(txt string) string {
	if txt == "" {
		return ""
	}

	if cached, ok := highlightCache[txt]; ok {
		return cached
	}

	themeName := theme.GetTheme().ThemeName
	var sb strings.Builder
	err := quick.Highlight(&sb, txt, "sql", "terminal256", themeName)
	if err != nil {
		slog.Error("error highlighting text", "error", err)
		return txt
	}

	result := strings.TrimRight(sb.String(), "\n")
	highlightCache[txt] = result
	return result
}

// ClearHighlightCache clears the highlighting cache.
// Call this when the theme changes or as periodic cleanup.
func ClearHighlightCache() {
	highlightCache = make(map[string]string)
}

// SplitStyledLine takes an ANSI-styled string and a character offset
// (in printable/visible characters), and returns the styled text
// before and after the offset. Both halves preserve ANSI state.
//
// The character at the offset itself is excluded from both halves.
// This is used to split a highlighted line around the cursor position.
func SplitStyledLine(styled string, offset int) (string, string) {
	if offset <= 0 {
		// cursor is at the start: before is empty, after skips char 0
		after := cutStyledLeft(styled, 1)
		return "", after
	}

	before := truncateStyled(styled, offset)
	after := cutStyledLeft(styled, offset+1)
	return before, after
}

// truncateStyled returns the first `width` printable characters of an
// ANSI-styled string, preserving all ANSI escape sequences encountered
// along the way and appending a reset at the end.
func truncateStyled(s string, width int) string {
	var (
		pos    int
		isAnsi bool
		b      strings.Builder
	)
	for _, c := range s {
		if c == ansi.Marker || isAnsi {
			isAnsi = true
			b.WriteRune(c)
			if ansi.IsTerminator(c) {
				isAnsi = false
			}
			continue
		}

		if pos >= width {
			break
		}

		b.WriteRune(c)
		pos++
	}
	b.WriteString("\033[0m")
	return b.String()
}

// cutStyledLeft removes the first `cutWidth` printable characters from an
// ANSI-styled string, keeping all ANSI state so the remainder renders
// correctly. Based on the cutLeft function in overlay.go.
func cutStyledLeft(s string, cutWidth int) string {
	var (
		pos    int
		isAnsi bool
		ab     bytes.Buffer // accumulates ANSI sequences before the cut point
		b      bytes.Buffer // accumulates output after the cut point
		past   bool         // true once we've passed the cut point
	)
	for _, c := range s {
		if c == ansi.Marker || isAnsi {
			isAnsi = true
			if past {
				b.WriteRune(c)
			} else {
				ab.WriteRune(c)
			}
			if ansi.IsTerminator(c) {
				isAnsi = false
				if !past {
					// Keep track of ANSI state but reset if we see a full reset
					if bytes.HasSuffix(ab.Bytes(), []byte("[0m")) {
						ab.Reset()
					}
				}
			}
			continue
		}

		if pos >= cutWidth {
			if !past {
				past = true
				// Prepend accumulated ANSI state
				if ab.Len() > 0 {
					b.Write(ab.Bytes())
				}
			}
			b.WriteRune(c)
		}
		pos++
	}
	return b.String()
}
