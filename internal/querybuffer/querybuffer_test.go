package querybuffer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wheelibin/qrypad/internal/querybuffer"
)

func TestQueryFileName(t *testing.T) {
	tests := []struct {
		name       string
		conn       string
		db         string
		singleFile bool
		want       string
	}{
		{"per-database mode includes database name", "animals_pg", "qrypad", false, "animals_pg.qrypad.sql"},
		{"single file mode uses connection name only", "animals_pg", "qrypad", true, "animals_pg.sql"},
		{"per-database with different database", "my_server", "analytics", false, "my_server.analytics.sql"},
		{"single file ignores database name", "animals_sqlite", "main", true, "animals_sqlite.sql"},
		{"empty database name in single file mode is fine", "myconn", "", true, "myconn.sql"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := querybuffer.QueryFileName(tt.conn, tt.db, tt.singleFile)
			if got != tt.want {
				t.Errorf("QueryFileName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMigrateQueryFileIfNeeded(t *testing.T) {
	t.Run("renames legacy file to per-database name", func(t *testing.T) {
		dir := t.TempDir()
		legacyFile := filepath.Join(dir, "myconn.sql")
		if err := os.WriteFile(legacyFile, []byte("SELECT 1;"), 0o600); err != nil {
			t.Fatal(err)
		}

		querybuffer.MigrateQueryFileIfNeeded(dir, "myconn", "mydb")

		if _, err := os.Stat(legacyFile); !os.IsNotExist(err) {
			t.Error("expected legacy file to be removed")
		}
		contents, err := os.ReadFile(filepath.Join(dir, "myconn.mydb.sql"))
		if err != nil {
			t.Fatalf("expected new file to exist: %v", err)
		}
		if string(contents) != "SELECT 1;" {
			t.Errorf("contents = %q, want %q", string(contents), "SELECT 1;")
		}
	})

	t.Run("does nothing if per-database file already exists", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "myconn.sql"), []byte("old query"), 0o600); err != nil {
			t.Fatal(err)
		}
		newFile := filepath.Join(dir, "myconn.mydb.sql")
		if err := os.WriteFile(newFile, []byte("new query"), 0o600); err != nil {
			t.Fatal(err)
		}

		querybuffer.MigrateQueryFileIfNeeded(dir, "myconn", "mydb")

		contents, err := os.ReadFile(newFile)
		if err != nil {
			t.Fatalf("expected new file to exist: %v", err)
		}
		if string(contents) != "new query" {
			t.Errorf("contents = %q, want %q", string(contents), "new query")
		}
	})

	t.Run("does nothing if no legacy file exists", func(t *testing.T) {
		dir := t.TempDir()
		querybuffer.MigrateQueryFileIfNeeded(dir, "myconn", "mydb")
		if _, err := os.Stat(filepath.Join(dir, "myconn.mydb.sql")); !os.IsNotExist(err) {
			t.Error("expected no file to be created")
		}
	})
}

func TestLoad(t *testing.T) {
	t.Run("returns empty strings when per-database mode and db name is empty", func(t *testing.T) {
		dir := t.TempDir()
		contents, filename, err := querybuffer.Load(dir, "myconn", "", false)
		if err != nil {
			t.Fatal(err)
		}
		if contents != "" || filename != "" {
			t.Errorf("expected empty results, got (%q, %q)", contents, filename)
		}
	})

	t.Run("creates file and returns empty contents when file does not exist", func(t *testing.T) {
		dir := t.TempDir()
		contents, filename, err := querybuffer.Load(dir, "myconn", "mydb", false)
		if err != nil {
			t.Fatal(err)
		}
		if contents != "" {
			t.Errorf("expected empty contents for new file, got %q", contents)
		}
		if filename == "" {
			t.Error("expected non-empty filename")
		}
		if _, err := os.Stat(filename); err != nil {
			t.Errorf("expected file to exist: %v", err)
		}
	})

	t.Run("reads existing file contents", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "myconn.mydb.sql")
		if err := os.WriteFile(path, []byte("SELECT 1;"), 0o600); err != nil {
			t.Fatal(err)
		}
		contents, filename, err := querybuffer.Load(dir, "myconn", "mydb", false)
		if err != nil {
			t.Fatal(err)
		}
		if contents != "SELECT 1;" {
			t.Errorf("contents = %q, want %q", contents, "SELECT 1;")
		}
		if filename != path {
			t.Errorf("filename = %q, want %q", filename, path)
		}
	})

	t.Run("single file mode works with empty database name", func(t *testing.T) {
		dir := t.TempDir()
		contents, filename, err := querybuffer.Load(dir, "myconn", "", true)
		if err != nil {
			t.Fatal(err)
		}
		if filename == "" {
			t.Error("expected non-empty filename")
		}
		_ = contents
	})
}

func TestBuffer_SaveIfChanged(t *testing.T) {
	t.Run("no-op when contents unchanged", func(t *testing.T) {
		dir := t.TempDir()
		buf := querybuffer.New(dir)
		buf.SetSaved("SELECT 1;")
		saved, err := buf.SaveIfChanged("SELECT 1;", "myconn", "mydb", false)
		if err != nil {
			t.Fatal(err)
		}
		if saved {
			t.Error("expected no save when contents unchanged")
		}
	})

	t.Run("no-op when per-database mode and db name is empty", func(t *testing.T) {
		dir := t.TempDir()
		buf := querybuffer.New(dir)
		saved, err := buf.SaveIfChanged("SELECT 1;", "myconn", "", false)
		if err != nil {
			t.Fatal(err)
		}
		if saved {
			t.Error("expected no save when db name is empty in per-database mode")
		}
	})

	t.Run("writes file and updates saved baseline when changed", func(t *testing.T) {
		dir := t.TempDir()
		buf := querybuffer.New(dir)
		buf.SetSaved("old")

		saved, err := buf.SaveIfChanged("new", "myconn", "mydb", false)
		if err != nil {
			t.Fatal(err)
		}
		if !saved {
			t.Error("expected save to occur")
		}
		if buf.IsDirty("new") {
			t.Error("expected IsDirty to return false after save")
		}

		data, err := os.ReadFile(filepath.Join(dir, "myconn.mydb.sql"))
		if err != nil {
			t.Fatalf("expected file to exist: %v", err)
		}
		if string(data) != "new" {
			t.Errorf("file contents = %q, want %q", string(data), "new")
		}
	})
}
