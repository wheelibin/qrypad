package component_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
	"github.com/wheelibin/qrypad/internal/db"
)

func TestIsWordChar(t *testing.T) {
	tests := []struct {
		name string
		char byte
		want bool
	}{
		{"lowercase letter", 'a', true},
		{"uppercase letter", 'Z', true},
		{"digit", '5', true},
		{"underscore", '_', true},
		{"dot", '.', true},
		{"space", ' ', false},
		{"hyphen", '-', false},
		{"semicolon", ';', false},
		{"comma", ',', false},
		{"open paren", '(', false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := component.IsWordChar(tt.char)
			if got != tt.want {
				t.Errorf("isWordChar(%q) = %v, want %v", tt.char, got, tt.want)
			}
		})
	}
}

func TestGetStatementAtCursor(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		cursorLine int
		wantNil    bool
		wantText   string
		wantStart  int
		wantEnd    int
	}{
		{
			name:       "single statement with semicolon",
			text:       "SELECT * FROM users;",
			cursorLine: 0,
			wantText:   "SELECT * FROM users;\n",
			wantStart:  0,
			wantEnd:    0,
		},
		{
			name:       "two statements cursor on first",
			text:       "SELECT * FROM users;\n\nSELECT * FROM orders;",
			cursorLine: 0,
			wantText:   "SELECT * FROM users;\n",
			wantStart:  0,
			wantEnd:    0,
		},
		{
			name:       "two statements cursor on second",
			text:       "SELECT * FROM users;\n\nSELECT * FROM orders;",
			cursorLine: 2,
			wantText:   "SELECT * FROM orders;\n",
			wantStart:  2,
			wantEnd:    2,
		},
		{
			name:       "cursor on blank line between statements returns nil",
			text:       "SELECT * FROM users;\n\nSELECT * FROM orders;",
			cursorLine: 1, // blank line
			wantNil:    true,
		},
		{
			name:       "trailing statement without semicolon",
			text:       "SELECT * FROM users",
			cursorLine: 0,
			wantText:   "SELECT * FROM users\n",
			wantStart:  0,
			wantEnd:    0,
		},
		{
			name:       "multiline statement cursor on first line",
			text:       "SELECT *\nFROM users\nWHERE id = 1;",
			cursorLine: 0,
			wantText:   "SELECT *\nFROM users\nWHERE id = 1;\n",
			wantStart:  0,
			wantEnd:    2,
		},
		{
			name:       "multiline statement cursor on last line",
			text:       "SELECT *\nFROM users\nWHERE id = 1;",
			cursorLine: 2,
			wantText:   "SELECT *\nFROM users\nWHERE id = 1;\n",
			wantStart:  0,
			wantEnd:    2,
		},
		{
			name:       "empty text returns nil",
			text:       "",
			cursorLine: 0,
			wantNil:    true,
		},
		{
			name:       "cursorLine out of bounds returns nil",
			text:       "SELECT 1;",
			cursorLine: 99,
			wantNil:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := component.GetStatementAtCursor(tt.text, tt.cursorLine)
			if tt.wantNil {
				if got != nil {
					t.Errorf("getStatementAtCursor: expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("getStatementAtCursor: expected non-nil result, got nil")
				return
			}
			if got.Text != tt.wantText {
				t.Errorf("Text: expected %q, got %q", tt.wantText, got.Text)
			}
			if got.StartLine != tt.wantStart {
				t.Errorf("StartLine: expected %d, got %d", tt.wantStart, got.StartLine)
			}
			if got.EndLine != tt.wantEnd {
				t.Errorf("EndLine: expected %d, got %d", tt.wantEnd, got.EndLine)
			}
		})
	}
}

func TestGetWordAtCursor(t *testing.T) {
	tests := []struct {
		name string
		text string
		row  int
		col  int
		want string
	}{
		{
			name: "cursor mid-word",
			text: "SELECT users FROM",
			row:  0,
			col:  9, // inside "users"
			want: "users",
		},
		{
			name: "cursor at start of word",
			text: "SELECT users FROM",
			row:  0,
			col:  7, // at 'u' of "users"
			want: "users",
		},
		{
			name: "cursor at end of word",
			text: "SELECT users FROM",
			row:  0,
			col:  12, // just after "users"
			want: "users",
		},
		{
			name: "cursor at space between words",
			text: "SELECT users FROM",
			row:  0,
			col:  6, // space after "SELECT" — function returns word to the left
			want: "SELECT",
		},
		{
			name: "cursor on second line",
			text: "SELECT *\nFROM users",
			row:  1,
			col:  5, // inside "users"
			want: "users",
		},
		{
			name: "row out of bounds returns empty",
			text: "SELECT users",
			row:  5,
			col:  0,
			want: "",
		},
		{
			name: "col beyond line length uses end of line",
			text: "SELECT users",
			row:  0,
			col:  9999,
			want: "users",
		},
		{
			name: "word with dot (alias.column)",
			text: "SELECT u.name",
			row:  0,
			col:  9, // inside "u.name"
			want: "u.name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := component.GetWordAtCursor(tt.text, tt.row, tt.col)
			if got != tt.want {
				t.Errorf("getWordAtCursor(text, %d, %d) = %q, want %q", tt.row, tt.col, got, tt.want)
			}
		})
	}
}

func TestAbbreviatePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "empty path",
			path: "",
			want: "",
		},
		{
			name: "path equals home exactly",
			path: home,
			want: "~",
		},
		{
			name: "path under home",
			path: filepath.Join(home, ".local", "share", "qrypad", "prod.sql"),
			want: "~" + string(os.PathSeparator) + filepath.Join(".local", "share", "qrypad", "prod.sql"),
		},
		{
			name: "path that shares prefix but is not under home (collision case)",
			path: home + "bob" + string(os.PathSeparator) + "file",
			want: home + "bob" + string(os.PathSeparator) + "file",
		},
		{
			name: "unrelated path",
			path: "/tmp/something.sql",
			want: "/tmp/something.sql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := component.AbbreviatePath(tt.path)
			if got != tt.want {
				t.Errorf("AbbreviatePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestGetColumnWidth(t *testing.T) {
	tests := []struct {
		name     string
		col      string
		data     db.Data
		maxWidth int
		want     int
	}{
		{
			name: "no rows - uses longest column header",
			col:  "name",
			data: db.Data{
				Columns: []string{"id", "name", "email"},
				Rows:    []map[string]any{},
			},
			maxWidth: 50,
			want:     6, // len("email") = 5 + padding 1
		},
		{
			name: "row value longer than header",
			col:  "name",
			data: db.Data{
				Columns: []string{"id", "name"},
				Rows: []map[string]any{
					{"id": "1", "name": "a_very_long_name"},
				},
			},
			maxWidth: 50,
			want:     17, // len("a_very_long_name") = 16 + padding 1
		},
		{
			name: "maxWidth caps result",
			col:  "name",
			data: db.Data{
				Columns: []string{"name"},
				Rows: []map[string]any{
					{"name": "this_is_extremely_long_value"},
				},
			},
			maxWidth: 10,
			want:     11, // capped at maxWidth=10 + padding 1
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := component.GetColumnWidth(tt.col, tt.data, tt.maxWidth)
			if got != tt.want {
				t.Errorf("getColumnWidth: expected %d, got %d", tt.want, got)
			}
		})
	}
}
