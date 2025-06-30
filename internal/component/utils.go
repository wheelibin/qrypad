package component

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/qrypad/internal/db"
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
			statements = append(statements, Statement{startLine, l, currentStatement.String()})
			currentStatement.Reset()
		}
	}

	// If there's a remaining statement without a semicolon, add it
	if currentStatement.Len() > 0 {
		statements = append(statements, Statement{startLine, len(lines), currentStatement.String()})
	}

	lineCounter := 0
	for _, stmt := range statements {
		stmtLines := strings.Split(stmt.Text, "\n")
		if cursorLine >= lineCounter && cursorLine < lineCounter+len(stmtLines) {
			return &stmt
		}
		lineCounter += len(stmtLines)
	}

	return nil // statement not found
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
		if len(fmt.Sprintf("%v", r[col])) > maxNeededLen {
			maxNeededLen = len(r[col].(string))
		}
	}
	padding := 1
	return int(math.Min(float64(maxNeededLen), float64(maxWidth))) + padding
}
