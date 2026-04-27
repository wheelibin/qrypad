package textarea_test

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/wheelibin/qrypad/internal/textarea"
)

// keyPress synthesises a tea.KeyPressMsg for a single printable character.
// This mirrors what a real terminal delivers for ordinary typed input.
func keyPress(text string) tea.KeyPressMsg {
	runes := []rune(text)
	var code rune
	if len(runes) > 0 {
		code = runes[0]
	}
	return tea.KeyPressMsg(tea.Key{Text: text, Code: code})
}

// keyCtrl synthesises a tea.KeyPressMsg for ctrl+<r>. Its .String()
// comes out as "ctrl+<r>", which is what key.Matches compares against.
func keyCtrl(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Mod: tea.ModCtrl, Code: r})
}

// keyBindingCtrl builds a key.Binding for ctrl+<r>.
func keyBindingCtrl(r, desc string) key.Binding {
	return key.NewBinding(key.WithKeys("ctrl+"+r), key.WithHelp("ctrl+"+r, desc))
}

func TestModel_SetValueResetsHistory(t *testing.T) {
	m := textarea.New()
	m.Focus()

	// Simulate a mutation so history has something beyond baseline.
	textarea.ExportEnsureHistory(&m)
	textarea.ExportRecordPre(&m, textarea.ExportOpOther)
	textarea.ExportSetValueInternal(&m, "typed")
	textarea.ExportRecordPre(&m, textarea.ExportOpOther)

	if textarea.ExportHistoryLen(&m) < 2 {
		t.Fatalf("precondition: expected >=2 entries, got %d", textarea.ExportHistoryLen(&m))
	}

	// External load: SetValue resets history.
	m.SetValue("loaded")

	if textarea.ExportHistoryLen(&m) != 1 {
		t.Fatalf("expected 1 entry after SetValue, got %d", textarea.ExportHistoryLen(&m))
	}
	if textarea.ExportHistoryCursor(&m) != 0 {
		t.Fatalf("expected cursor=0, got %d", textarea.ExportHistoryCursor(&m))
	}
	if textarea.ExportHistoryEntryValue(&m, 0) != "loaded" {
		t.Fatalf("baseline mismatch: got %q", textarea.ExportHistoryEntryValue(&m, 0))
	}
	if textarea.ExportHistoryUndo(&m) {
		t.Fatal("undo immediately after SetValue should return false")
	}
}

func TestModel_UndoRedoBasicRoundTrip(t *testing.T) {
	m := textarea.New()
	m.Focus()
	// Wire Undo/Redo bindings locally so the test exercises the real
	// key-matching path without depending on the keys package.
	m.KeyMap.Undo = keyBindingCtrl("z", "undo")
	m.KeyMap.Redo = keyBindingCtrl("y", "redo")

	// Type "ab" (two opInsertChar ops — coalesce into one undo step).
	m, _ = m.Update(keyPress("a"))
	m, _ = m.Update(keyPress("b"))
	if got := m.Value(); got != "ab" {
		t.Fatalf("precondition: expected \"ab\", got %q", got)
	}

	// Undo — expect "".
	m, _ = m.Update(keyCtrl('z'))
	if got := m.Value(); got != "" {
		t.Fatalf("after undo expected \"\", got %q", got)
	}

	// Redo — expect "ab".
	m, _ = m.Update(keyCtrl('y'))
	if got := m.Value(); got != "ab" {
		t.Fatalf("after redo expected \"ab\", got %q", got)
	}
}

func TestModel_CursorMoveBreaksCoalescing(t *testing.T) {
	m := textarea.New()
	m.Focus()
	m.KeyMap.Undo = keyBindingCtrl("z", "undo")

	// Type "ab", move cursor left (ctrl+b is a default CharacterBackward
	// key in textarea.DefaultKeyMap), type "*" — expect two undo steps.
	m, _ = m.Update(keyPress("a"))
	m, _ = m.Update(keyPress("b"))
	m, _ = m.Update(keyCtrl('b')) // CharacterBackward
	m, _ = m.Update(keyPress("*"))
	if got := m.Value(); got != "a*b" {
		t.Fatalf("precondition: expected \"a*b\", got %q", got)
	}

	// First undo reverts "*" -> "ab".
	m, _ = m.Update(keyCtrl('z'))
	if got := m.Value(); got != "ab" {
		t.Fatalf("after first undo expected \"ab\", got %q", got)
	}
	// Second undo reverts "ab" -> "".
	m, _ = m.Update(keyCtrl('z'))
	if got := m.Value(); got != "" {
		t.Fatalf("after second undo expected \"\", got %q", got)
	}
}

func TestModel_SpaceBreaksCoalescing(t *testing.T) {
	m := textarea.New()
	m.Focus()
	m.KeyMap.Undo = keyBindingCtrl("z", "undo")

	// Type "a b" — space is opOther, so undo yields: "a ", "a", "".
	m, _ = m.Update(keyPress("a"))
	m, _ = m.Update(keyPress(" "))
	m, _ = m.Update(keyPress("b"))
	if got := m.Value(); got != "a b" {
		t.Fatalf("precondition: expected \"a b\", got %q", got)
	}

	m, _ = m.Update(keyCtrl('z'))
	if got := m.Value(); got != "a " {
		t.Fatalf("undo 1: expected \"a \", got %q", got)
	}
	m, _ = m.Update(keyCtrl('z'))
	if got := m.Value(); got != "a" {
		t.Fatalf("undo 2: expected \"a\", got %q", got)
	}
	m, _ = m.Update(keyCtrl('z'))
	if got := m.Value(); got != "" {
		t.Fatalf("undo 3: expected \"\", got %q", got)
	}
}

func TestModel_EditAfterUndoTruncatesRedo(t *testing.T) {
	m := textarea.New()
	m.Focus()
	m.KeyMap.Undo = keyBindingCtrl("z", "undo")
	m.KeyMap.Redo = keyBindingCtrl("y", "redo")

	m, _ = m.Update(keyPress("a"))
	m, _ = m.Update(keyPress(" ")) // break coalesce
	m, _ = m.Update(keyPress("b"))
	// Value: "a b"
	m, _ = m.Update(keyCtrl('z')) // -> "a "

	// New edit from this point.
	m, _ = m.Update(keyPress("c"))
	if got := m.Value(); got != "a c" {
		t.Fatalf("after edit expected \"a c\", got %q", got)
	}

	// Redo must do nothing now.
	m, _ = m.Update(keyCtrl('y'))
	if got := m.Value(); got != "a c" {
		t.Fatalf("after failed redo expected \"a c\", got %q", got)
	}
}

func TestModel_PasteIsSingleUndoStep(t *testing.T) {
	m := textarea.New()
	m.Focus()
	m.KeyMap.Undo = keyBindingCtrl("z", "undo")

	// Synthesise a 500-line paste.
	var sb strings.Builder
	for range 500 {
		sb.WriteString("line\n")
	}
	m, _ = m.Update(tea.PasteMsg{Content: sb.String()})

	if m.Value() == "" {
		t.Fatal("precondition: expected non-empty after paste")
	}

	m, _ = m.Update(keyCtrl('z'))
	if got := m.Value(); got != "" {
		t.Fatalf("single undo should empty buffer; got %q (len %d)", got, len(got))
	}
}

func TestModel_SetValueResetsHistoryViaKeys(t *testing.T) {
	m := textarea.New()
	m.Focus()
	m.KeyMap.Undo = keyBindingCtrl("z", "undo")

	m, _ = m.Update(keyPress("x"))
	m.SetValue("fresh")

	m, _ = m.Update(keyCtrl('z'))
	if got := m.Value(); got != "fresh" {
		t.Fatalf("undo after SetValue should be a no-op; got %q", got)
	}
}

// ReplaceValue is for programmatic full-buffer replacement (e.g. accepting an
// autocomplete suggestion) that should be reversible with Ctrl+Z. Unlike
// SetValue — which resets history because the new value is a fresh baseline
// from disk / editor — ReplaceValue records the pre-state so undo returns to
// what the user had before the replacement.
func TestModel_ReplaceValueIsUndoable(t *testing.T) {
	m := textarea.New()
	m.Focus()
	m.KeyMap.Undo = keyBindingCtrl("z", "undo")

	// Build up some user input.
	m, _ = m.Update(keyPress("u"))
	m, _ = m.Update(keyPress("s"))
	if got := m.Value(); got != "us" {
		t.Fatalf("precondition: expected \"us\", got %q", got)
	}

	// Simulate accepting an autocomplete suggestion that replaces the
	// whole buffer.
	m.ReplaceValue("users")
	if got := m.Value(); got != "users" {
		t.Fatalf("after ReplaceValue expected \"users\", got %q", got)
	}

	// Ctrl+Z should revert the autocomplete acceptance.
	m, _ = m.Update(keyCtrl('z'))
	if got := m.Value(); got != "us" {
		t.Fatalf("after undo expected \"us\", got %q", got)
	}
}
