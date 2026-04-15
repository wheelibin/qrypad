package style

import (
	"bytes"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/lipgloss"
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

// ExtractStyledChar returns the character at the given printable-character
// offset in an ANSI-styled string, together with the ANSI escape sequences
// that were active at that position (i.e. the syntax-highlight color from
// Chroma). The returned string is self-contained: it starts with any active
// ANSI codes, contains exactly one printable character, and ends with a reset.
//
// If offset is out of range, an empty string is returned.
func ExtractStyledChar(styled string, offset int) string {
	var (
		pos    int
		isAnsi bool
		ab     bytes.Buffer // ANSI state accumulated before the target char
		b      bytes.Buffer // final output
	)
	for _, c := range styled {
		if c == ansi.Marker || isAnsi {
			isAnsi = true
			ab.WriteRune(c)
			if ansi.IsTerminator(c) {
				isAnsi = false
				// Discard accumulated state on a full reset
				if bytes.HasSuffix(ab.Bytes(), []byte("[0m")) {
					ab.Reset()
				}
			}
			continue
		}

		if pos == offset {
			// Write the accumulated ANSI state, the character, then reset
			b.Write(ab.Bytes())
			b.WriteRune(c)
			b.WriteString("\033[0m")
			return b.String()
		}
		pos++
	}
	return ""
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

// fgColorPattern matches ANSI 256-color and truecolor foreground escape sequences.
//   - 256-color: \x1b[38;5;Nm  → group 1 = "N"
//   - Truecolor: \x1b[38;2;R;G;Bm → group 2 = "R", group 3 = "G", group 4 = "B"
var fgColorPattern = regexp.MustCompile(`\x1b\[38;(?:5;(\d+)|2;(\d+);(\d+);(\d+))m`)

// ParseFgColor extracts the foreground color from an ANSI-styled string
// (such as the output of ExtractStyledChar) and returns it as a lipgloss.Color.
// Returns an empty Color if no foreground color escape is found.
//
// Supports Chroma's terminal256 output (\x1b[38;5;Nm) and truecolor
// (\x1b[38;2;R;G;Bm). This is used to read the syntax-highlight colour for a
// character so it can be applied to the cursor block rendering.
func ParseFgColor(styled string) lipgloss.Color {
	m := fgColorPattern.FindStringSubmatch(styled)
	if m == nil {
		return lipgloss.Color("")
	}
	if m[1] != "" {
		// 256-color: return the numeric index as a lipgloss color string
		return lipgloss.Color(m[1])
	}
	// Truecolor: convert R,G,B to #RRGGBB hex
	r, _ := strconv.Atoi(m[2])
	g, _ := strconv.Atoi(m[3])
	b, _ := strconv.Atoi(m[4])
	return lipgloss.Color(fmt.Sprintf("#%02X%02X%02X", r, g, b))
}
