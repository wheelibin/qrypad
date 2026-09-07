package component

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/viper"

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
		currentStatement.WriteString(line)
		currentStatement.WriteByte('\n')
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
	// Start with the longest column header as a minimum width.
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
		// Short-circuit: already at max, no need to check more rows.
		if maxNeededLen >= maxWidth {
			return maxWidth + 1 // +1 for padding
		}
	}
	padding := 1
	return maxNeededLen + padding
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

	// ReplaceValue (not SetValue) so accepting an autocomplete suggestion
	// remains undoable with Ctrl+Z.
	t.ReplaceValue(strings.Join(lines, "\n"))

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

// jsonCellData holds a pre-highlighted JSON string for display while keeping
// the raw JSON string for copy/export operations.
//
// bubble-table renders StyledCell.Data via fmt.Sprintf("%v", data), so
// implementing fmt.Stringer here causes the highlighted string to be shown in
// the table, while Raw is returned by unwrapCellData for data consumers.
type jsonCellData struct {
	Raw         string
	highlighted string
}

func (j jsonCellData) String() string { return j.highlighted }

// unwrapCellData extracts the underlying data from a StyledCell, or returns
// the value as-is if it is not a StyledCell. Used by export/copy/popup
// functions that need plain values without styling.
func unwrapCellData(v any) any {
	if sc, ok := v.(table.StyledCell); ok {
		if jc, ok := sc.Data.(jsonCellData); ok {
			return jc.Raw
		}
		return sc.Data
	}
	return v
}

// withRowBorders applies the rowBorders config option to a table model.
func withRowBorders(t table.Model) table.Model {
	if viper.GetBool("rowBorders") {
		return t.WithRowBorder(true)
	}
	return t
}
