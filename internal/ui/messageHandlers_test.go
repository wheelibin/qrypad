package ui //nolint:testpackage // whitebox: sets up unexported model fields (db, sessions) to reproduce connection lifecycle bugs

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"

	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/db"
)

func openInMemoryDB(t *testing.T) db.DBConn {
	t.Helper()
	conn, err := db.Connect(db.ConnectionConfig{Driver: db.DriverName.SQLite, Database: ":memory:"}, "")
	if err != nil {
		t.Fatalf("db.Connect: %v", err)
	}
	t.Cleanup(func() { _ = conn.DB.Close() })
	return conn
}

// newModelWithParkedSession returns a model with connection "a" active and
// connection "b" parked in the session pool, plus their raw *sql.DB handles
// for asserting open/closed state after the test exercises a handler.
func newModelWithParkedSession(t *testing.T) (Model, *sql.DB, *sql.DB) {
	t.Helper()
	m := NewModel("a", db.ConnectionConfig{}, t.TempDir())
	m.db = openInMemoryDB(t)
	m.sessions = []session{{connectionName: "b", db: openInMemoryDB(t), schemaCache: db.NewSchemaCache()}}
	return m, m.db.DB, m.sessions[0].db.DB
}

// Regression test for: switching to a second session and back closed the
// connection ("sql: database is closed"). ConnectionSelectedMsg must hand the
// outgoing connection off to the session pool without closing it.
func TestConnectionSelectedMsg_DoesNotCloseSwitchedAwaySession(t *testing.T) {
	m, _, _ := newModelWithParkedSession(t)

	// Switch a -> b
	if cmd := m.handleQueryMessages(commands.ConnectionSelectedMsg("b")); cmd == nil {
		t.Fatalf("expected a command switching to session b")
	}
	if m.connectionName != "b" {
		t.Fatalf("expected active connection to be b, got %s", m.connectionName)
	}

	// Switch b -> a
	if cmd := m.handleQueryMessages(commands.ConnectionSelectedMsg("a")); cmd == nil {
		t.Fatalf("expected a command switching back to session a")
	}
	if m.connectionName != "a" {
		t.Fatalf("expected active connection to be a, got %s", m.connectionName)
	}

	if err := m.db.DB.PingContext(context.Background()); err != nil {
		t.Fatalf("restored session a's connection should still be open: %v", err)
	}
	sessB, ok := m.findSession("b")
	if !ok {
		t.Fatalf("expected session b to still be in the pool")
	}
	if err := sessB.db.DB.PingContext(context.Background()); err != nil {
		t.Fatalf("parked session b's connection should still be open: %v", err)
	}
}

// Regression test for: quitting via a popup (e.g. dismissing a connection
// error) bypassed cleanup and left other parked sessions' connections open.
// QuitRequestedMsg must close the active connection and every parked session.
func TestQuitRequestedMsg_ClosesAllConnections(t *testing.T) {
	m, activeDB, parkedDB := newModelWithParkedSession(t)

	cmd := m.handleCommandMessages(commands.QuitRequestedMsg{})
	if cmd == nil {
		t.Fatalf("expected a quit command")
	}

	if err := activeDB.PingContext(context.Background()); err == nil {
		t.Fatalf("expected active connection to be closed")
	}
	if err := parkedDB.PingContext(context.Background()); err == nil {
		t.Fatalf("expected parked session's connection to be closed")
	}
}
