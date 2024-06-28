package component

import (
	"strings"
)

func getStatementAtCursor(text string, cursorLine int) string {
	lines := strings.Split(text, "\n")
	if cursorLine < 0 || cursorLine >= len(lines) {
		return "" // cursorLine out of bounds
	}

	var currentStatement strings.Builder
	var statements []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		currentStatement.WriteString(line + "\n")
		if strings.Contains(line, ";") {
			statements = append(statements, currentStatement.String())
			currentStatement.Reset()
		}
	}

	// If there's a remaining statement without a semicolon, add it
	if currentStatement.Len() > 0 {
		statements = append(statements, currentStatement.String())
	}
	lineCounter := 0
	for _, stmt := range statements {
		stmtLines := strings.Split(stmt, "\n")
		if cursorLine >= lineCounter && cursorLine < lineCounter+len(stmtLines) {
			return stmt
		}
		lineCounter += len(stmtLines)
	}

	return "" // statement not found
}
