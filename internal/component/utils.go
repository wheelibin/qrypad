package component

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/textarea"
	"github.com/wheelibin/qrypad/internal/theme"
)

type Statement struct {
	StartLine int
	EndLine   int
	Text      string
}

func getStatementAtCursor(text string, cursorLine int) *Statement {
	lines := strings.Split(text, "\n")
	if cursorLine < 0 || cursorLine >= len(lines) {
		return nil // cursorLine out of bounds
	}

	var currentStatement strings.Builder
	var startLine int
	var statements []Statement

	for l, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if currentStatement.Len() == 0 {
			startLine = l
		}
		currentStatement.WriteString(line + "\n")
		if strings.Contains(line, ";") {
			statements = append(statements, Statement{
				StartLine: startLine,
				EndLine:   l,
				Text:      currentStatement.String(),
			})
			currentStatement.Reset()
		}
	}

	// If there's a remaining statement without a semicolon, treat it as trailing
	if currentStatement.Len() > 0 {
		statements = append(statements, Statement{
			StartLine: startLine,
			EndLine:   len(lines) - 1,
			Text:      currentStatement.String(),
		})
	}

	for _, stmt := range statements {
		if cursorLine >= stmt.StartLine && cursorLine <= stmt.EndLine {
			return &stmt
		}
	}

	return nil // no statement under cursor
}

func makeHelp() help.Model {
	help := help.New()
	help.ShowAll = true
	help.Styles.ShortKey = lipgloss.NewStyle().Foreground(theme.GetTheme().HelpKey.FG)
	help.Styles.ShortDesc = lipgloss.NewStyle().Foreground(theme.GetTheme().HelpDesc.FG)
	help.Styles.FullKey = lipgloss.NewStyle().Foreground(theme.GetTheme().HelpKey.FG)
	help.Styles.FullDesc = lipgloss.NewStyle().Foreground(theme.GetTheme().HelpDesc.FG)
	return help
}

func getColumnWidth(col string, data db.Data, maxWidth int) int {
	maxNeededLen := 0
	for _, c := range data.Columns {
		if len(c) > maxNeededLen {
			maxNeededLen = len(c)
		}
	}
	for _, r := range data.Rows {
		l := len(fmt.Sprintf("%v", r[col]))
		if l > maxNeededLen {
			maxNeededLen = l
		}
	}
	padding := 1
	return int(math.Min(float64(maxNeededLen), float64(maxWidth))) + padding
}

func getWordAtCursor(text string, row, col int) string {
	lines := strings.Split(text, "\n")
	if row < 0 || row >= len(lines) {
		return ""
	}
	line := lines[row]
	if col > len(line) {
		col = len(line)
	}

	start := col
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}
	end := col
	for end < len(line) && isWordChar(line[end]) {
		end++
	}

	return line[start:end]
}

func isWordChar(r byte) bool {
	return r == '_' || r == '.' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func replaceFuzzyPrefixInTextarea(t textarea.Model, selected string) textarea.Model {
	lines := strings.Split(t.Value(), "\n")
	targetRow := t.Row
	targetCol := t.Col

	if targetRow >= len(lines) {
		return t
	}

	line := lines[targetRow]
	if targetCol > len(line) {
		targetCol = len(line)
	}

	before := line[:targetCol]
	after := line[targetCol:]

	// Find the start of the replaceable region:
	// If there's a '.' (column completion), replace after the last dot.
	// Otherwise (table completion), replace from the start of the current word.
	lastDot := strings.LastIndex(before, ".")
	var startCol int
	if lastDot != -1 {
		startCol = lastDot + 1
	} else {
		// Walk backward to find word start
		startCol = targetCol
		for startCol > 0 && isWordChar(before[startCol-1]) {
			startCol--
		}
	}

	// Build new line
	newLine := line[:startCol] + selected + after
	lines[targetRow] = newLine

	// Set updated content
	t.SetValue(strings.Join(lines, "\n"))

	// Reset cursor row by navigating
	for t.Row > targetRow {
		t.CursorUp()
	}
	for t.Row < targetRow {
		t.CursorDown()
	}

	// Set cursor column
	t.SetCursor(startCol + len(selected))

	return t
}
