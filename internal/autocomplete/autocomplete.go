package autocomplete

import (
	"regexp"
	"strings"
)

type CompletionKind int

const (
	CompletionNone   CompletionKind = iota
	CompletionColumn                // alias. or tablename. -> fetch columns from DB
	CompletionTable                 // after FROM/JOIN etc -> suggest table names
)

type CompletionResult struct {
	Kind      CompletionKind
	TableName string   // for CompletionColumn: which table to fetch columns for
	Items     []string // for CompletionTable: matching table names
}

// tableRefPattern matches FROM/JOIN clauses and extracts table name + optional alias.
var tableRefPattern = regexp.MustCompile(
	`(?i)\b(?:FROM|JOIN)\s+` +
		`(\w+)` +
		`(?:\s+(?:AS\s+)?(\w+))?`,
)

// tableKeywordPattern matches keywords after which a table name is expected.
// It checks if the text immediately before the cursor ends with one of these.
var tableKeywordPattern = regexp.MustCompile(
	`(?i)\b(?:FROM|JOIN|INTO|UPDATE)\s+$`,
)

// GetCompletions determines what kind of autocomplete to offer based on
// the current SQL statement, the word at cursor, and the available table names.
func GetCompletions(sql, wordAtCursor string, tableNames []string) CompletionResult {
	if sql == "" {
		return CompletionResult{Kind: CompletionNone}
	}

	// Column completion: word ends with "."
	if wordAtCursor != "" && strings.HasSuffix(wordAtCursor, ".") {
		prefix := strings.TrimSuffix(wordAtCursor, ".")
		if prefix != "" {
			aliasMap := getAliasTableMap(sql)
			if tableName, ok := aliasMap[strings.ToLower(prefix)]; ok {
				return CompletionResult{Kind: CompletionColumn, TableName: tableName}
			}
		}
		return CompletionResult{Kind: CompletionNone}
	}

	// Table completion: cursor is after a table keyword (FROM, JOIN, INTO, UPDATE)
	// We need the text up to the cursor to check what precedes the current word.
	// The wordAtCursor is what's been typed so far after the keyword (could be empty
	// or a partial table name for ctrl+space re-trigger).
	textBeforeWord := getTextBeforeWord(sql, wordAtCursor)
	if tableKeywordPattern.MatchString(textBeforeWord) {
		filtered := filterNames(tableNames, wordAtCursor)
		if len(filtered) > 0 {
			return CompletionResult{Kind: CompletionTable, Items: filtered}
		}
	}

	return CompletionResult{Kind: CompletionNone}
}

// GetCompletionsForced is used by ctrl+space to force completion regardless of context.
// It tries context-aware completion first, then falls back to table names.
func GetCompletionsForced(sql, wordAtCursor string, tableNames []string) CompletionResult {
	// Try context-aware first
	result := GetCompletions(sql, wordAtCursor, tableNames)
	if result.Kind != CompletionNone {
		return result
	}

	// Fallback: suggest table names filtered by whatever word is at cursor
	filtered := filterNames(tableNames, wordAtCursor)
	if len(filtered) > 0 {
		return CompletionResult{Kind: CompletionTable, Items: filtered}
	}

	return CompletionResult{Kind: CompletionNone}
}

// getTextBeforeWord returns the SQL text up to (but not including) the wordAtCursor.
// This is used to check what keyword precedes the current typing position.
func getTextBeforeWord(sql, word string) string {
	if word == "" {
		return sql
	}
	// Find the last occurrence of the word in the SQL
	idx := strings.LastIndex(strings.ToLower(sql), strings.ToLower(word))
	if idx < 0 {
		return sql
	}
	return sql[:idx]
}

func getAliasTableMap(sql string) map[string]string {
	aliasMap := make(map[string]string)

	matches := tableRefPattern.FindAllStringSubmatch(sql, -1)
	for _, match := range matches {
		tableName := match[1]
		alias := match[2]

		if alias != "" {
			aliasMap[strings.ToLower(alias)] = tableName
		}
		aliasMap[strings.ToLower(tableName)] = tableName
	}

	return aliasMap
}

func filterNames(names []string, prefix string) []string {
	if prefix == "" {
		return names
	}
	lower := strings.ToLower(prefix)
	filtered := make([]string, 0)
	for _, name := range names {
		if strings.Contains(strings.ToLower(name), lower) {
			filtered = append(filtered, name)
		}
	}
	return filtered
}
