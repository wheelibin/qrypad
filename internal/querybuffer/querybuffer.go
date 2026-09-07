package querybuffer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Buffer tracks the last-saved contents of a query file and handles writing to disk.
// dir is fixed at construction; conn/db/singleFile are passed at each call because
// they change when the user switches connections or databases.
type Buffer struct {
	saved string
	dir   string
}

func New(dir string) *Buffer {
	return &Buffer{dir: dir}
}

// SetSaved updates the saved baseline without writing to disk.
// Call this after loading a file from disk.
func (b *Buffer) SetSaved(contents string) {
	b.saved = contents
}

// IsDirty reports whether current differs from the last saved contents.
func (b *Buffer) IsDirty(current string) bool {
	return current != b.saved
}

// SaveIfChanged writes current to disk if it differs from the last saved contents.
// Returns true if a write was performed, false if contents were already in sync or
// the save was skipped (e.g. empty database name in per-database mode).
func (b *Buffer) SaveIfChanged(current, conn, db string, single bool) (bool, error) {
	if current == b.saved {
		return false, nil
	}
	if !single && db == "" {
		return false, nil
	}
	filename := filepath.Join(b.dir, QueryFileName(conn, db, single))
	if err := os.WriteFile(filename, []byte(current), 0o600); err != nil {
		return false, fmt.Errorf("writing query file %s: %w", filename, err)
	}
	b.saved = current
	return true, nil
}

// QueryFileName returns the filename for a query file.
// In per-database mode (singleFile=false), returns "<conn>.<db>.sql".
// In single-file mode (singleFile=true), returns "<conn>.sql".
func QueryFileName(conn, db string, singleFile bool) string {
	if singleFile {
		return fmt.Sprintf("%s.sql", conn)
	}
	return fmt.Sprintf("%s.%s.sql", conn, db)
}

// MigrateQueryFileIfNeeded renames a legacy <conn>.sql file to the per-database
// format if the per-database file does not already exist.
func MigrateQueryFileIfNeeded(dir, conn, db string) {
	perDBFile := filepath.Join(dir, QueryFileName(conn, db, false))
	legacyFile := filepath.Join(dir, QueryFileName(conn, "", true))

	if _, err := os.Stat(perDBFile); err == nil {
		return
	}
	if _, err := os.Stat(legacyFile); err == nil {
		_ = os.Rename(legacyFile, perDBFile)
	}
}

// Load reads the query file for the given connection/database from dir, creating
// it if it does not exist. Returns empty strings without error when databaseName
// is empty in per-database mode (the caller should retry once the database is known).
func Load(dir, conn, db string, singleFile bool) (string, string, error) {
	if !singleFile && db == "" {
		return "", "", nil
	}

	if !singleFile {
		MigrateQueryFileIfNeeded(dir, conn, db)
	}

	filename := filepath.Join(dir, QueryFileName(conn, db, singleFile))

	if _, statErr := os.Stat(filename); errors.Is(statErr, os.ErrNotExist) {
		if _, createErr := os.Create(filename); createErr != nil {
			return "", "", fmt.Errorf("creating query file %s: %w", filename, createErr)
		}
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return "", "", fmt.Errorf("reading query file %s: %w", filename, err)
	}
	return string(data), filename, nil
}
