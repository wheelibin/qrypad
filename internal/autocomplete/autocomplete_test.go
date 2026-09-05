package autocomplete_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/autocomplete"
	"github.com/wheelibin/qrypad/internal/db"
)

func makeRefs(names ...string) []db.TableReference {
	refs := make([]db.TableReference, len(names))
	for i, n := range names {
		refs[i] = db.TableReference{Schema: "public", Name: n}
	}
	return refs
}

func makeMixedRefs() []db.TableReference {
	return []db.TableReference{
		{Schema: "public", Name: "users"},
		{Schema: "public", Name: "orders"},
		{Schema: "myschema", Name: "products"},
	}
}

func postgresConn() autocomplete.ConnCtx {
	return autocomplete.ConnCtx{Driver: db.DriverName.Postgres, ConnectedDB: "mydb"}
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
			name:     "FROM with qualified schema.table and alias",
			sql:      "SELECT * FROM public.users u",
			expected: map[string]string{"public.users": "public.users", "u": "public.users"},
		},
		{
			name:     "FROM with qualified schema.table no alias",
			sql:      "SELECT * FROM myschema.orders",
			expected: map[string]string{"myschema.orders": "myschema.orders"},
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
	conn := postgresConn()

	tests := []struct {
		name         string
		sql          string
		wordAtCursor string
		allRefs      []db.TableReference
		wantKind     autocomplete.CompletionKind
		wantTableRef db.TableReference
		wantItems    []string
	}{
		{
			name:         "empty SQL returns None",
			sql:          "",
			wordAtCursor: "",
			allRefs:      makeRefs("users", "orders"),
			wantKind:     autocomplete.CompletionNone,
		},
		{
			name:         "word ending with dot triggers column completion via alias",
			sql:          "SELECT u. FROM users u",
			wordAtCursor: "u.",
			allRefs:      makeRefs("users", "orders"),
			wantKind:     autocomplete.CompletionColumn,
			wantTableRef: db.TableReference{Schema: "public", Name: "users"},
		},
		{
			name:         "dot with unknown alias returns None",
			sql:          "SELECT x. FROM users u",
			wordAtCursor: "x.",
			allRefs:      makeRefs("users"),
			wantKind:     autocomplete.CompletionNone,
		},
		{
			name:         "alias of schema.table triggers column completion",
			sql:          "select * from public.users t where t.",
			wordAtCursor: "t.",
			allRefs:      makeMixedRefs(),
			wantKind:     autocomplete.CompletionColumn,
			wantTableRef: db.TableReference{Schema: "public", Name: "users"},
		},
		{
			name:         "schema dot triggers schema completion",
			sql:          "SELECT * FROM public.",
			wordAtCursor: "public.",
			allRefs:      makeMixedRefs(),
			wantKind:     autocomplete.CompletionSchema,
			wantItems:    []string{"users", "orders"},
		},
		{
			name:         "schema.table. triggers column completion",
			sql:          "SELECT * FROM public.users",
			wordAtCursor: "public.users.",
			allRefs:      makeMixedRefs(),
			wantKind:     autocomplete.CompletionColumn,
			wantTableRef: db.TableReference{Schema: "public", Name: "users"},
		},
		{
			name:         "after FROM triggers table completion with autocomplete insert strings",
			sql:          "SELECT * FROM ",
			wordAtCursor: "",
			allRefs:      makeMixedRefs(),
			wantKind:     autocomplete.CompletionTable,
			// public.users and public.orders insert as "users"/"orders" (default schema for postgres)
			// myschema.products inserts as "myschema.products"
			wantItems: []string{"users", "orders", "myschema.products"},
		},
		{
			name:         "after FROM with partial name filters results",
			sql:          "SELECT * FROM us",
			wordAtCursor: "us",
			allRefs:      makeRefs("users", "orders", "products"),
			wantKind:     autocomplete.CompletionTable,
			wantItems:    []string{"users"},
		},
		{
			name:         "after FROM with no matching names returns None",
			sql:          "SELECT * FROM xyz",
			wordAtCursor: "xyz",
			allRefs:      makeRefs("users", "orders"),
			wantKind:     autocomplete.CompletionNone,
		},
		{
			name:         "after FROM in middle of buffer triggers table completion",
			sql:          "SELECT * FROM \nSELECT 1;",
			wordAtCursor: "",
			allRefs:      makeRefs("users", "orders"),
			wantKind:     autocomplete.CompletionTable,
			wantItems:    []string{"users", "orders"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := autocomplete.GetCompletions(tt.sql, tt.wordAtCursor, tt.allRefs, conn)
			if got.Kind != tt.wantKind {
				t.Errorf("GetCompletions kind: expected %d, got %d", tt.wantKind, got.Kind)
			}
			if (tt.wantTableRef != db.TableReference{}) {
				if got.TableRef != tt.wantTableRef {
					t.Errorf("GetCompletions TableRef: expected %+v, got %+v", tt.wantTableRef, got.TableRef)
				}
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
	conn := postgresConn()
	allRefs := makeRefs("users", "orders")

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
			wantItems:    []string{"users", "orders"},
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
			wantItems:    []string{"users", "orders"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := autocomplete.GetCompletionsForced(tt.sql, tt.wordAtCursor, allRefs, conn)
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
