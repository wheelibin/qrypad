package autocomplete_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/autocomplete"
)

func TestFilterNames(t *testing.T) {
	tests := []struct {
		name     string
		names    []string
		prefix   string
		expected []string
	}{
		{
			name:     "empty prefix returns all names",
			names:    []string{"users", "orders", "products"},
			prefix:   "",
			expected: []string{"users", "orders", "products"},
		},
		{
			name:     "case-insensitive substring match",
			names:    []string{"Users", "orders", "UserProfiles"},
			prefix:   "user",
			expected: []string{"Users", "UserProfiles"},
		},
		{
			name:     "no match returns empty slice",
			names:    []string{"users", "orders"},
			prefix:   "xyz",
			expected: []string{},
		},
		{
			name:     "empty names list returns empty slice",
			names:    []string{},
			prefix:   "us",
			expected: []string{},
		},
		{
			name:     "exact match",
			names:    []string{"users", "orders"},
			prefix:   "users",
			expected: []string{"users"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := autocomplete.FilterNames(tt.names, tt.prefix)
			if len(got) != len(tt.expected) {
				t.Fatalf("filterNames(%v, %q): expected %v (len %d), got %v (len %d)",
					tt.names, tt.prefix, tt.expected, len(tt.expected), got, len(got))
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("filterNames(%v, %q): item %d: expected %q, got %q",
						tt.names, tt.prefix, i, tt.expected[i], got[i])
				}
			}
		})
	}
}

func TestGetAliasTableMap(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected map[string]string
	}{
		{
			name:     "FROM without alias",
			sql:      "SELECT * FROM users",
			expected: map[string]string{"users": "users"},
		},
		{
			name:     "FROM with explicit alias",
			sql:      "SELECT * FROM users u",
			expected: map[string]string{"users": "users", "u": "users"},
		},
		{
			name:     "FROM with AS alias",
			sql:      "SELECT * FROM users AS u",
			expected: map[string]string{"users": "users", "u": "users"},
		},
		{
			name:     "JOIN with alias",
			sql:      "SELECT * FROM orders o JOIN users u ON o.user_id = u.id",
			expected: map[string]string{"orders": "orders", "o": "orders", "users": "users", "u": "users"},
		},
		{
			name:     "case insensitive FROM keyword",
			sql:      "select * from Products p",
			expected: map[string]string{"products": "Products", "p": "Products"},
		},
		{
			name:     "empty SQL",
			sql:      "",
			expected: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := autocomplete.GetAliasTableMap(tt.sql)
			if len(got) != len(tt.expected) {
				t.Fatalf("getAliasTableMap(%q): expected %v (len %d), got %v (len %d)",
					tt.sql, tt.expected, len(tt.expected), got, len(got))
			}
			for k, v := range tt.expected {
				if got[k] != v {
					t.Errorf("getAliasTableMap(%q): key %q: expected %q, got %q",
						tt.sql, k, v, got[k])
				}
			}
		})
	}
}

func TestGetTextBeforeWord(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		word     string
		expected string
	}{
		{
			name:     "empty word returns full SQL",
			sql:      "SELECT * FROM users",
			word:     "",
			expected: "SELECT * FROM users",
		},
		{
			name:     "word present returns text before it",
			sql:      "SELECT * FROM users",
			word:     "users",
			expected: "SELECT * FROM ",
		},
		{
			name:     "word absent returns full SQL",
			sql:      "SELECT * FROM users",
			word:     "products",
			expected: "SELECT * FROM users",
		},
		{
			name:     "uses last occurrence of word",
			sql:      "SELECT name FROM users WHERE name LIKE",
			word:     "name",
			expected: "SELECT name FROM users WHERE ",
		},
		{
			name:     "case insensitive match",
			sql:      "SELECT * FROM Users",
			word:     "users",
			expected: "SELECT * FROM ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := autocomplete.GetTextBeforeWord(tt.sql, tt.word)
			if got != tt.expected {
				t.Errorf("getTextBeforeWord(%q, %q): expected %q, got %q",
					tt.sql, tt.word, tt.expected, got)
			}
		})
	}
}

func TestGetCompletions(t *testing.T) {
	tableNames := []string{"users", "orders", "products"}

	tests := []struct {
		name         string
		sql          string
		wordAtCursor string
		tableNames   []string
		wantKind     autocomplete.CompletionKind
		wantTable    string
		wantItems    []string
	}{
		{
			name:         "empty SQL returns None",
			sql:          "",
			wordAtCursor: "",
			tableNames:   tableNames,
			wantKind:     autocomplete.CompletionNone,
		},
		{
			name:         "word ending with dot triggers column completion",
			sql:          "SELECT u. FROM users u",
			wordAtCursor: "u.",
			tableNames:   tableNames,
			wantKind:     autocomplete.CompletionColumn,
			wantTable:    "users",
		},
		{
			name:         "dot with unknown alias returns None",
			sql:          "SELECT x. FROM users u",
			wordAtCursor: "x.",
			tableNames:   tableNames,
			wantKind:     autocomplete.CompletionNone,
		},
		{
			name:         "after FROM triggers table completion",
			sql:          "SELECT * FROM ",
			wordAtCursor: "",
			tableNames:   tableNames,
			wantKind:     autocomplete.CompletionTable,
			wantItems:    tableNames,
		},
		{
			name:         "after FROM with partial name filters results",
			sql:          "SELECT * FROM us",
			wordAtCursor: "us",
			tableNames:   tableNames,
			wantKind:     autocomplete.CompletionTable,
			wantItems:    []string{"users"},
		},
		{
			name:         "after JOIN triggers table completion",
			sql:          "SELECT * FROM users JOIN ",
			wordAtCursor: "",
			tableNames:   tableNames,
			wantKind:     autocomplete.CompletionTable,
			wantItems:    tableNames,
		},
		{
			name:         "after FROM with no matching names returns None",
			sql:          "SELECT * FROM xyz",
			wordAtCursor: "xyz",
			tableNames:   tableNames,
			wantKind:     autocomplete.CompletionNone,
		},
		{
			name:         "in the middle of a SELECT with no context returns None",
			sql:          "SELECT name",
			wordAtCursor: "name",
			tableNames:   tableNames,
			wantKind:     autocomplete.CompletionNone,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := autocomplete.GetCompletions(tt.sql, tt.wordAtCursor, tt.tableNames)
			if got.Kind != tt.wantKind {
				t.Errorf("GetCompletions kind: expected %d, got %d", tt.wantKind, got.Kind)
			}
			if tt.wantTable != "" && got.TableName != tt.wantTable {
				t.Errorf("GetCompletions TableName: expected %q, got %q", tt.wantTable, got.TableName)
			}
			if tt.wantItems != nil {
				if len(got.Items) != len(tt.wantItems) {
					t.Fatalf("GetCompletions Items len: expected %d, got %d (%v)", len(tt.wantItems), len(got.Items), got.Items)
				}
				for i := range tt.wantItems {
					if got.Items[i] != tt.wantItems[i] {
						t.Errorf("GetCompletions Items[%d]: expected %q, got %q", i, tt.wantItems[i], got.Items[i])
					}
				}
			}
		})
	}
}

func TestGetCompletionsForced(t *testing.T) {
	tableNames := []string{"users", "orders"}

	tests := []struct {
		name         string
		sql          string
		wordAtCursor string
		wantKind     autocomplete.CompletionKind
		wantItems    []string
	}{
		{
			name:         "falls back to all table names when no context",
			sql:          "SELECT name",
			wordAtCursor: "",
			wantKind:     autocomplete.CompletionTable,
			wantItems:    tableNames,
		},
		{
			name:         "falls back filtered by word at cursor",
			sql:          "SELECT name",
			wordAtCursor: "us",
			wantKind:     autocomplete.CompletionTable,
			wantItems:    []string{"users"},
		},
		{
			name:         "returns None when forced but nothing matches",
			sql:          "SELECT name",
			wordAtCursor: "xyz",
			wantKind:     autocomplete.CompletionNone,
		},
		{
			name:         "uses context-aware completion when available",
			sql:          "SELECT * FROM ",
			wordAtCursor: "",
			wantKind:     autocomplete.CompletionTable,
			wantItems:    tableNames,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := autocomplete.GetCompletionsForced(tt.sql, tt.wordAtCursor, tableNames)
			if got.Kind != tt.wantKind {
				t.Errorf("GetCompletionsForced kind: expected %d, got %d", tt.wantKind, got.Kind)
			}
			if tt.wantItems != nil {
				if len(got.Items) != len(tt.wantItems) {
					t.Fatalf("GetCompletionsForced Items len: expected %d, got %d (%v)", len(tt.wantItems), len(got.Items), got.Items)
				}
				for i := range tt.wantItems {
					if got.Items[i] != tt.wantItems[i] {
						t.Errorf("GetCompletionsForced Items[%d]: expected %q, got %q", i, tt.wantItems[i], got.Items[i])
					}
				}
			}
		})
	}
}
