package commands_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/db"
)

func TestQueryFileName(t *testing.T) {
	tests := []struct {
		name           string
		connectionName string
		databaseName   string
		singleFile     bool
		want           string
	}{
		{
			name:           "per-database mode includes database name",
			connectionName: "animals_pg",
			databaseName:   "qrypad",
			singleFile:     false,
			want:           "animals_pg.qrypad.sql",
		},
		{
			name:           "single file mode uses connection name only",
			connectionName: "animals_pg",
			databaseName:   "qrypad",
			singleFile:     true,
			want:           "animals_pg.sql",
		},
		{
			name:           "per-database with different database",
			connectionName: "my_server",
			databaseName:   "analytics",
			singleFile:     false,
			want:           "my_server.analytics.sql",
		},
		{
			name:           "single file ignores database name",
			connectionName: "animals_sqlite",
			databaseName:   "main",
			singleFile:     true,
			want:           "animals_sqlite.sql",
		},
		{
			name:           "empty database name in single file mode is fine",
			connectionName: "myconn",
			databaseName:   "",
			singleFile:     true,
			want:           "myconn.sql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commands.QueryFileName(tt.connectionName, tt.databaseName, tt.singleFile)
			if got != tt.want {
				t.Errorf("QueryFileName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadOrCreateQueryFile_EmptyDatabaseName(t *testing.T) {
	t.Run("returns nil msg when per-database mode and database name is empty", func(t *testing.T) {
		cmd := commands.ReadOrCreateQueryFile("myconn", "", false)
		if cmd == nil {
			t.Fatal("expected a non-nil tea.Cmd (the function returns a closure)")
		}
		msg := cmd()
		if msg != nil {
			t.Errorf("expected nil msg when databaseName is empty in per-database mode, got %T", msg)
		}
	})

	t.Run("proceeds normally in single file mode even with empty database name", func(t *testing.T) {
		// This should not return nil — it should produce a QueryFileReadMsg
		// (though it will use GetOutputDir which may create files in the real data dir).
		// We just verify the cmd doesn't short-circuit to nil.
		cmd := commands.ReadOrCreateQueryFile("test_read_single", "", true)
		if cmd == nil {
			t.Fatal("expected a non-nil tea.Cmd")
		}
		msg := cmd()
		if msg == nil {
			t.Error("expected non-nil msg in single file mode")
		}
		// Clean up the file that may have been created
		if readMsg, ok := msg.(commands.QueryFileReadMsg); ok {
			_ = os.Remove(readMsg.FileName)
		}
	})
}

func TestSaveQueryFileToDisk_EmptyDatabaseName(t *testing.T) {
	t.Run("no-ops when per-database mode and database name is empty", func(t *testing.T) {
		err := commands.SaveQueryFileToDisk("myconn", "", "SELECT 1;", false)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
		// Verify no file was created with double-dot name
		dir, _ := commands.GetOutputDir()
		badFile := filepath.Join(dir, "myconn..sql")
		if _, err := os.Stat(badFile); err == nil {
			_ = os.Remove(badFile)
			t.Error("file with double-dot name was created, but should not have been")
		}
	})

	t.Run("proceeds normally in single file mode with empty database name", func(t *testing.T) {
		dir := t.TempDir()
		// We can't easily override GetOutputDir, so test the guard logic only.
		// The single-file guard check (singleFile=true) means it won't short-circuit.
		err := commands.SaveQueryFileToDisk("test_save_single", "", "SELECT 1;", true)
		if err != nil {
			t.Errorf("expected nil error in single file mode, got %v", err)
		}
		// Clean up
		outputDir, _ := commands.GetOutputDir()
		_ = os.Remove(filepath.Join(outputDir, "test_save_single.sql"))
		_ = dir // suppress unused
	})
}

func TestMigrateQueryFileIfNeeded(t *testing.T) {
	t.Run("renames legacy file to per-database name", func(t *testing.T) {
		dir := t.TempDir()
		legacyFile := filepath.Join(dir, "myconn.sql")
		if err := os.WriteFile(legacyFile, []byte("SELECT 1;"), 0o600); err != nil {
			t.Fatal(err)
		}

		newFile := filepath.Join(dir, "myconn.mydb.sql")

		commands.MigrateQueryFileIfNeeded(dir, "myconn", "mydb")

		// Legacy file should no longer exist
		if _, err := os.Stat(legacyFile); !os.IsNotExist(err) {
			t.Errorf("expected legacy file to be removed, but it still exists")
		}
		// New file should exist with same contents
		contents, err := os.ReadFile(newFile)
		if err != nil {
			t.Fatalf("expected new file to exist: %v", err)
		}
		if string(contents) != "SELECT 1;" {
			t.Errorf("contents = %q, want %q", string(contents), "SELECT 1;")
		}
	})

	t.Run("does nothing if per-database file already exists", func(t *testing.T) {
		dir := t.TempDir()
		legacyFile := filepath.Join(dir, "myconn.sql")
		if err := os.WriteFile(legacyFile, []byte("old query"), 0o600); err != nil {
			t.Fatal(err)
		}

		newFile := filepath.Join(dir, "myconn.mydb.sql")
		if err := os.WriteFile(newFile, []byte("new query"), 0o600); err != nil {
			t.Fatal(err)
		}

		commands.MigrateQueryFileIfNeeded(dir, "myconn", "mydb")

		// Per-database file should be unchanged
		contents, err := os.ReadFile(newFile)
		if err != nil {
			t.Fatalf("expected new file to exist: %v", err)
		}
		if string(contents) != "new query" {
			t.Errorf("contents = %q, want %q", string(contents), "new query")
		}
		// Legacy file should still exist (not deleted since per-db file already existed)
		if _, err := os.Stat(legacyFile); os.IsNotExist(err) {
			t.Errorf("expected legacy file to still exist")
		}
	})

	t.Run("does nothing if no legacy file exists", func(t *testing.T) {
		dir := t.TempDir()

		commands.MigrateQueryFileIfNeeded(dir, "myconn", "mydb")

		newFile := filepath.Join(dir, "myconn.mydb.sql")
		if _, err := os.Stat(newFile); !os.IsNotExist(err) {
			t.Errorf("expected no file to be created")
		}
	})
}

func TestPreloadColumns_NilOnEmptyRefs(t *testing.T) {
	cache := db.NewSchemaCache()
	cmd := commands.PreloadColumns(db.DBConn{}, nil, cache)
	if cmd != nil {
		t.Error("expected nil cmd for empty refs")
	}
}

func TestPreloadColumns_NilOverThreshold(t *testing.T) {
	cache := db.NewSchemaCache()
	refs := make([]db.TableReference, commands.PreloadTableThreshold+1)
	cmd := commands.PreloadColumns(db.DBConn{}, refs, cache)
	if cmd != nil {
		t.Error("expected nil cmd when refs exceed threshold")
	}
}

func TestPreloadColumns_NilOnNilCache(t *testing.T) {
	refs := []db.TableReference{{Name: "test"}}
	cmd := commands.PreloadColumns(db.DBConn{}, refs, nil)
	if cmd != nil {
		t.Error("expected nil cmd for nil cache")
	}
}
