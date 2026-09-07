package commands_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/db"
)

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
