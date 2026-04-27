package textarea_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/textarea"
)

// fakeBuffer is a test double for textarea.ExportBufferState.
type fakeBuffer struct {
	value string
	row   int
	col   int
}

func (f *fakeBuffer) Value() string             { return f.value }
func (f *fakeBuffer) SetValueRaw(s string)      { f.value = s }
func (f *fakeBuffer) CursorPos() (int, int)     { return f.row, f.col }
func (f *fakeBuffer) SetCursorPos(row, col int) { f.row, f.col = row, col }

func TestHistory_NewBaseline(t *testing.T) {
	b := &fakeBuffer{}
	h := textarea.NewExportHistory(b)
	if got := h.Entries(); got != 1 {
		t.Fatalf("expected 1 baseline entry, got %d", got)
	}
	if h.Cursor() != 0 {
		t.Fatalf("expected cursor=0, got %d", h.Cursor())
	}
	if h.Undo(b) {
		t.Fatalf("undo at baseline should return false")
	}
	if h.Redo(b) {
		t.Fatalf("redo at baseline should return false")
	}
}

func TestHistory_RecordPre_SameOpCoalesces(t *testing.T) {
	b := &fakeBuffer{}
	h := textarea.NewExportHistory(b)

	// recordPre is called BEFORE the mutation (per its contract). Simulate
	// typing "s", "e", "l" — each keystroke snapshots the pre-state and
	// then the mutation lands in the fake buffer.
	h.RecordPre(b, textarea.ExportOpInsertChar)
	b.value = "s"
	h.RecordPre(b, textarea.ExportOpInsertChar)
	b.value = "se"
	h.RecordPre(b, textarea.ExportOpInsertChar)
	b.value = "sel"

	// First call transitioned from opNone->opInsertChar: captured pre-state "",
	// which dedups against the baseline. Subsequent calls hit the hot path
	// (op == lastOp && op != opOther) and return without touching entries.
	if h.Entries() != 1 {
		t.Fatalf("expected 1 entry (coalesced), got %d", h.Entries())
	}
}

func TestHistory_RecordPre_OpTransitionPushes(t *testing.T) {
	b := &fakeBuffer{}
	h := textarea.NewExportHistory(b)

	// First keystroke: opNone -> opInsertChar transition. Captures pre-state "",
	// dedups against baseline tip. entries stays at len 1.
	h.RecordPre(b, textarea.ExportOpInsertChar)
	b.value = "s"
	if h.Entries() != 1 {
		t.Fatalf("after first keystroke expected 1 entry (dedup vs baseline), got %d", h.Entries())
	}

	// Transition to opOther (e.g. a paste): recordPre captures the current
	// live value "s" as pre-state, then the caller applies the paste.
	h.RecordPre(b, textarea.ExportOpOther)
	b.value = "spasted"

	if h.Entries() != 2 {
		t.Fatalf("expected 2 entries after transition, got %d", h.Entries())
	}
	if h.EntryValue(1) != "s" {
		t.Fatalf("expected second entry value 's', got %q", h.EntryValue(1))
	}
	if h.Cursor() != 1 {
		t.Fatalf("expected cursor=1, got %d", h.Cursor())
	}
}

func TestHistory_RecordPre_OpOtherAlwaysPushes(t *testing.T) {
	b := &fakeBuffer{}
	h := textarea.NewExportHistory(b)

	// opOther must never coalesce, even when lastOp is also opOther. Two
	// consecutive pastes should each produce a pre-state snapshot (with
	// baseline dedup for the first one since pre-state "" matches).
	h.RecordPre(b, textarea.ExportOpOther)
	b.value = "aaa"
	h.RecordPre(b, textarea.ExportOpOther)
	b.value = "aaabbb"

	// entries: ["", "aaa"]. The baseline is the pre-state of paste 1;
	// "aaa" is the pre-state of paste 2. The live "aaabbb" is not yet
	// snapshotted — syncCurrent handles that on undo (Task 3).
	if h.Entries() != 2 {
		t.Fatalf("expected 2 entries, got %d", h.Entries())
	}
	if h.EntryValue(1) != "aaa" {
		t.Fatalf("expected entries[1]=\"aaa\", got %q", h.EntryValue(1))
	}

	// A third opOther with no mutation between must still push if the
	// candidate differs from the tip; otherwise dedup skips.
	h.RecordPre(b, textarea.ExportOpOther)
	// candidate = "aaabbb" (live), tip = "aaa". Pushes. entries=[...3].
	if h.Entries() != 3 {
		t.Fatalf("expected 3 entries after third opOther, got %d", h.Entries())
	}
	if h.EntryValue(2) != "aaabbb" {
		t.Fatalf("expected entries[2]=\"aaabbb\", got %q", h.EntryValue(2))
	}
}

func TestHistory_RecordPre_SkipsDuplicateValue(t *testing.T) {
	b := &fakeBuffer{}
	h := textarea.NewExportHistory(b)

	// No real mutation happened between two opOther transitions.
	h.RecordPre(b, textarea.ExportOpOther)
	h.RecordPre(b, textarea.ExportOpOther)

	if h.Entries() != 1 {
		t.Fatalf("expected 1 entry (duplicates skipped), got %d", h.Entries())
	}
	// lastOp must still be updated even when the push is skipped.
	if h.LastOp() != textarea.ExportOpOther {
		t.Fatalf("expected lastOp==opOther after dedup, got %v", h.LastOp())
	}
}

func TestHistory_RecordPre_TruncatesRedoFuture(t *testing.T) {
	b := &fakeBuffer{}
	h := textarea.NewExportHistory(b)

	// Build entries ["", "a", "ab"] with cursor at tip and live "abc".
	h.RecordPre(b, textarea.ExportOpOther)
	b.value = "a"
	h.RecordPre(b, textarea.ExportOpOther)
	b.value = "ab"
	h.RecordPre(b, textarea.ExportOpOther)
	b.value = "abc"
	if h.Entries() != 3 {
		t.Fatalf("precondition: expected 3 entries, got %d", h.Entries())
	}

	// Undo moves cursor back. syncCurrent appends the live "abc" first,
	// so after the undo entries are ["", "a", "ab", "abc"] with cursor=2,
	// and b.value is restored to entries[2]="ab".
	h.Undo(b)
	if b.value != "ab" {
		t.Fatalf("after undo expected live buffer \"ab\", got %q", b.value)
	}
	tipBefore := h.Entries()
	cursorBefore := h.Cursor()
	if cursorBefore >= tipBefore-1 {
		t.Fatalf("precondition: expected cursor < tip after undo, got cursor=%d tip=%d",
			cursorBefore, tipBefore)
	}

	// New mutation: recordPre must discard the redo future beyond cursor.
	h.RecordPre(b, textarea.ExportOpOther)
	b.value = "aZ"

	// The redo future entries (indices > cursorBefore) must have been dropped.
	if h.Entries() != cursorBefore+1 {
		t.Fatalf("expected redo future truncated to len %d, got %d",
			cursorBefore+1, h.Entries())
	}
	// Pre-state "ab" captured here dedups against tip entries[cursorBefore]="ab",
	// so no push from this call — verified by the truncation-only length above.

	// Now a second opOther with a different live value should push.
	h.RecordPre(b, textarea.ExportOpOther)
	b.value = "aZZ"
	if h.EntryValue(h.Cursor()) != "aZ" {
		t.Fatalf("expected tip value \"aZ\" after second push, got %q",
			h.EntryValue(h.Cursor()))
	}
}

func TestHistory_EntryCapEvictsOldest(t *testing.T) {
	b := &fakeBuffer{}
	h := textarea.NewExportHistory(b)

	// Push 250 distinct opOther entries. Cap is 200. Per the recordPre
	// contract we snapshot pre-state, then "apply" the mutation.
	for i := range 250 {
		h.RecordPre(b, textarea.ExportOpOther)
		b.value = repeat("v", i+1) // each one distinct
	}

	if h.Entries() != textarea.ExportMaxHistoryEntries {
		t.Fatalf("expected %d entries after eviction, got %d",
			textarea.ExportMaxHistoryEntries, h.Entries())
	}
	// cursor must still point at the tip.
	if h.Cursor() != h.Entries()-1 {
		t.Fatalf("expected cursor at tip (%d), got %d", h.Entries()-1, h.Cursor())
	}
}

// repeat returns s concatenated n times. Used to generate distinct
// entries in eviction-cap tests.
func repeat(s string, n int) string {
	out := make([]byte, 0, n*len(s))
	for range n {
		out = append(out, s...)
	}
	return string(out)
}

func TestHistory_ByteCapEvictsOldest(t *testing.T) {
	b := &fakeBuffer{}
	h := textarea.NewExportHistory(b)

	// Push a handful of 20 MB entries; after 3 the byte cap (50 MB) must kick in.
	big := make([]byte, 20*1024*1024)
	for i := range big {
		big[i] = 'x'
	}
	for i := range 4 {
		h.RecordPre(b, textarea.ExportOpOther)
		// make each value distinct so dedup doesn't skip it
		b.value = string(big) + string(rune('a'+i))
	}

	if h.TotalBytes() > textarea.ExportMaxHistoryBytes {
		t.Fatalf("totalBytes=%d exceeds cap %d", h.TotalBytes(), textarea.ExportMaxHistoryBytes)
	}
}

func TestHistory_UndoAfterCoalescedRunThenRedo(t *testing.T) {
	b := &fakeBuffer{}
	h := textarea.NewExportHistory(b)

	// Simulate typing "sel" as a coalesced run. recordPre is called
	// BEFORE each mutation (contract). The first recordPre dedups against
	// the baseline; subsequent recordPre calls hit the hot path.
	h.RecordPre(b, textarea.ExportOpInsertChar)
	b.value = "s"
	h.RecordPre(b, textarea.ExportOpInsertChar)
	b.value = "se"
	h.RecordPre(b, textarea.ExportOpInsertChar)
	b.value = "sel"

	// At this point entries == [""], cursor=0, live buffer = "sel".
	if h.Entries() != 1 {
		t.Fatalf("precondition: expected 1 entry, got %d", h.Entries())
	}

	// Undo should revert live buffer to "". syncCurrent must first
	// append the live "sel" so redo has a target.
	if !h.Undo(b) {
		t.Fatal("undo should have succeeded")
	}
	if b.value != "" {
		t.Fatalf("after undo expected value \"\", got %q", b.value)
	}

	// Redo must return to "sel".
	if !h.Redo(b) {
		t.Fatal("redo should have succeeded")
	}
	if b.value != "sel" {
		t.Fatalf("after redo expected value \"sel\", got %q", b.value)
	}
}

func TestHistory_UndoWithoutCoalescedGap(t *testing.T) {
	// If the live buffer already equals the tip entry, syncCurrent must
	// not append a duplicate.
	b := &fakeBuffer{}
	h := textarea.NewExportHistory(b)

	// Build entries=["", "a"] with cursor=1 and live buffer "a".
	h.RecordPre(b, textarea.ExportOpOther) // dedups pre-state "" vs baseline, lastOp=opOther
	b.value = "a"
	h.RecordPre(b, textarea.ExportOpOther) // pushes pre-state "a", entries=["","a"], cursor=1
	// b.value is still "a" — live == tip.

	if h.Entries() != 2 {
		t.Fatalf("precondition: expected 2 entries, got %d", h.Entries())
	}

	// Live buffer equals tip, so syncCurrent should be a no-op during undo.
	// Undo decrements cursor to 0 and restores "".
	if !h.Undo(b) {
		t.Fatal("undo should have succeeded")
	}
	if b.value != "" {
		t.Fatalf("after undo expected \"\", got %q", b.value)
	}
	// len(entries) must remain 2 — no duplicate was appended.
	if h.Entries() != 2 {
		t.Fatalf("expected entries unchanged at len 2, got %d", h.Entries())
	}
}
