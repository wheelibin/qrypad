package autocomplete

import (
	"regexp"
	"strings"

	"github.com/wheelibin/qrypad/internal/db"
)

// CompletionKind identifies what type of autocomplete to offer.
type CompletionKind int

const (
	CompletionNone   CompletionKind = iota
	CompletionColumn                // alias. or tablename. or schema.table. -> fetch columns from DB
	CompletionTable                 // after FROM/JOIN etc -> suggest table names
	CompletionSchema                // schema. -> suggest tables within that schema (no DB hit needed)
)

// CompletionResult is returned by GetCompletions and GetCompletionsForced.
type CompletionResult struct {
	Kind     CompletionKind
	TableRef db.TableReference // for CompletionColumn: which table to fetch columns for
	Items    []string          // for CompletionTable and CompletionSchema: strings to insert
}

// tableRefPattern matches FROM/JOIN clauses and extracts table name (optionally
// schema-qualified as schema.table) + optional alias.
var tableRefPattern = regexp.MustCompile(
	`(?i)\b(?:FROM|JOIN)\s+` +
		`(\w+(?:\.\w+)?)` +
		`(?:\s+(?:AS\s+)?(\w+))?`,
)

// tableKeywordPattern matches keywords after which a table name is expected.
// The (?m) flag makes $ match end-of-line so it works when the cursor is
// on a line in the middle of a multi-statement buffer.
var tableKeywordPattern = regexp.MustCompile(
	`(?im)\b(?:FROM|JOIN|INTO|UPDATE)\s+$`,
)

// GetCompletions determines what kind of autocomplete to offer based on
// the current SQL statement, the word at cursor, and the available table references.
func GetCompletions(sql, wordAtCursor string, allRefs []db.TableReference, dbConn db.DBConn) CompletionResult {
	if sql == "" {
		return CompletionResult{Kind: CompletionNone}
	}

	// Column/schema completion: word ends with "."
	if wordAtCursor != "" && strings.HasSuffix(wordAtCursor, ".") {
		prefix := strings.TrimSuffix(wordAtCursor, ".")

		// Two-part prefix: schema.table. -> column completion
		if parts := strings.SplitN(prefix, ".", 2); len(parts) == 2 {
			return CompletionResult{
				Kind:     CompletionColumn,
				TableRef: db.TableReference{Schema: parts[0], Name: parts[1]},
			}
		}

		// One-part prefix: check if it matches a known schema name first
		schemaNames := uniqueSchemas(allRefs)
		for _, s := range schemaNames {
			if strings.EqualFold(s, prefix) {
				return CompletionResult{
					Kind:  CompletionSchema,
					Items: tablesInSchema(allRefs, s),
				}
			}
		}

		// One-part prefix: try alias/table map for column completion
		if prefix != "" {
			aliasMap := getAliasTableMap(sql)
			if tableName, ok := aliasMap[strings.ToLower(prefix)]; ok {
				return CompletionResult{
					Kind:     CompletionColumn,
					TableRef: findRef(allRefs, tableName),
				}
			}
		}
		return CompletionResult{Kind: CompletionNone}
	}

	// Table completion: cursor is after a table keyword (FROM, JOIN, INTO, UPDATE)
	textBeforeWord := getTextBeforeWord(sql, wordAtCursor)
	if tableKeywordPattern.MatchString(textBeforeWord) {
		items := filterRefInserts(allRefs, wordAtCursor, dbConn)
		if len(items) > 0 {
			return CompletionResult{Kind: CompletionTable, Items: items}
		}
	}

	return CompletionResult{Kind: CompletionNone}
}

// GetCompletionsForced is used by ctrl+space to force completion regardless of context.
// It tries context-aware completion first, then falls back to all table names.
func GetCompletionsForced(sql, wordAtCursor string, allRefs []db.TableReference, dbConn db.DBConn) CompletionResult {
	result := GetCompletions(sql, wordAtCursor, allRefs, dbConn)
	if result.Kind != CompletionNone {
		return result
	}

	items := filterRefInserts(allRefs, wordAtCursor, dbConn)
	if len(items) > 0 {
		return CompletionResult{Kind: CompletionTable, Items: items}
	}

	return CompletionResult{Kind: CompletionNone}
}

// getTextBeforeWord returns the SQL text up to (but not including) the wordAtCursor.
func getTextBeforeWord(sql, word string) string {
	if word == "" {
		return sql
	}
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

// filterRefInserts returns autocomplete insert strings for all refs whose
// AutocompleteInsert value contains the prefix (case-insensitive substring match).
func filterRefInserts(refs []db.TableReference, prefix string, dbConn db.DBConn) []string {
	lower := strings.ToLower(prefix)
	results := make([]string, 0)
	for _, ref := range refs {
		insert := ref.AutocompleteInsert(dbConn.DriverName, dbConn.ConnectedDatabase)
		if prefix == "" || strings.Contains(strings.ToLower(insert), lower) {
			results = append(results, insert)
		}
	}
	return results
}

// uniqueSchemas returns distinct non-empty schema names from refs, in order of first appearance.
func uniqueSchemas(refs []db.TableReference) []string {
	seen := make(map[string]bool)
	schemas := make([]string, 0)
	for _, ref := range refs {
		if ref.Schema != "" && !seen[ref.Schema] {
			seen[ref.Schema] = true
			schemas = append(schemas, ref.Schema)
		}
	}
	return schemas
}

// tablesInSchema returns the bare table names for refs whose Schema matches schemaName (case-insensitive).
func tablesInSchema(refs []db.TableReference, schemaName string) []string {
	lower := strings.ToLower(schemaName)
	names := make([]string, 0)
	for _, ref := range refs {
		if strings.ToLower(ref.Schema) == lower {
			names = append(names, ref.Name)
		}
	}
	return names
}

// findRef returns the first TableReference that matches tableName (case-insensitive).
// tableName may be a bare name ("users") or a qualified name ("public.users").
// If not found, returns a TableReference constructed from the name.
func findRef(refs []db.TableReference, tableName string) db.TableReference {
	// If tableName is qualified (schema.table), split and match on both fields.
	if parts := strings.SplitN(tableName, ".", 2); len(parts) == 2 {
		schemaLower := strings.ToLower(parts[0])
		nameLower := strings.ToLower(parts[1])
		for _, ref := range refs {
			if strings.ToLower(ref.Schema) == schemaLower && strings.ToLower(ref.Name) == nameLower {
				return ref
			}
		}
		// Not found in refs — construct from the qualified name.
		return db.TableReference{Schema: parts[0], Name: parts[1]}
	}

	// Bare name: match on Name field only.
	lower := strings.ToLower(tableName)
	for _, ref := range refs {
		if strings.ToLower(ref.Name) == lower {
			return ref
		}
	}
	return db.TableReference{Name: tableName}
}
