package textarea

import "testing"

// fakeBuffer is a test double for bufferState.
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
	h := newHistory(b)
	if got := len(h.entries); got != 1 {
		t.Fatalf("expected 1 baseline entry, got %d", got)
	}
	if h.cursor != 0 {
		t.Fatalf("expected cursor=0, got %d", h.cursor)
	}
	if h.undo(b) {
		t.Fatalf("undo at baseline should return false")
	}
	if h.redo(b) {
		t.Fatalf("redo at baseline should return false")
	}
}

func TestHistory_RecordPre_SameOpCoalesces(t *testing.T) {
	b := &fakeBuffer{}
	h := newHistory(b)

	// recordPre is called BEFORE the mutation (per its contract). Simulate
	// typing "s", "e", "l" — each keystroke snapshots the pre-state and
	// then the mutation lands in the fake buffer.
	h.recordPre(b, opInsertChar)
	b.value = "s"
	h.recordPre(b, opInsertChar)
	b.value = "se"
	h.recordPre(b, opInsertChar)
	b.value = "sel"

	// First call transitioned from opNone->opInsertChar: captured pre-state "",
	// which dedups against the baseline. Subsequent calls hit the hot path
	// (op == lastOp && op != opOther) and return without touching entries.
	if len(h.entries) != 1 {
		t.Fatalf("expected 1 entry (coalesced), got %d", len(h.entries))
	}
}

func TestHistory_RecordPre_OpTransitionPushes(t *testing.T) {
	b := &fakeBuffer{}
	h := newHistory(b)

	// First keystroke: opNone -> opInsertChar transition. Captures pre-state "",
	// dedups against baseline tip. entries stays at len 1.
	h.recordPre(b, opInsertChar)
	b.value = "s"
	if len(h.entries) != 1 {
		t.Fatalf("after first keystroke expected 1 entry (dedup vs baseline), got %d", len(h.entries))
	}

	// Transition to opOther (e.g. a paste): recordPre captures the current
	// live value "s" as pre-state, then the caller applies the paste.
	h.recordPre(b, opOther)
	b.value = "spasted"

	if len(h.entries) != 2 {
		t.Fatalf("expected 2 entries after transition, got %d", len(h.entries))
	}
	if h.entries[1].value != "s" {
		t.Fatalf("expected second entry value 's', got %q", h.entries[1].value)
	}
	if h.cursor != 1 {
		t.Fatalf("expected cursor=1, got %d", h.cursor)
	}
}

func TestHistory_RecordPre_OpOtherAlwaysPushes(t *testing.T) {
	b := &fakeBuffer{}
	h := newHistory(b)

	// opOther must never coalesce, even when lastOp is also opOther. Two
	// consecutive pastes should each produce a pre-state snapshot (with
	// baseline dedup for the first one since pre-state "" matches).
	h.recordPre(b, opOther)
	b.value = "aaa"
	h.recordPre(b, opOther)
	b.value = "aaabbb"

	// entries: ["", "aaa"]. The baseline is the pre-state of paste 1;
	// "aaa" is the pre-state of paste 2. The live "aaabbb" is not yet
	// snapshotted — syncCurrent handles that on undo (Task 3).
	if len(h.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(h.entries))
	}
	if h.entries[1].value != "aaa" {
		t.Fatalf("expected entries[1]=\"aaa\", got %q", h.entries[1].value)
	}

	// A third opOther with no mutation between must still push if the
	// candidate differs from the tip; otherwise dedup skips.
	h.recordPre(b, opOther)
	// candidate = "aaabbb" (live), tip = "aaa". Pushes. entries=[...3].
	if len(h.entries) != 3 {
		t.Fatalf("expected 3 entries after third opOther, got %d", len(h.entries))
	}
	if h.entries[2].value != "aaabbb" {
		t.Fatalf("expected entries[2]=\"aaabbb\", got %q", h.entries[2].value)
	}
}

func TestHistory_RecordPre_SkipsDuplicateValue(t *testing.T) {
	b := &fakeBuffer{}
	h := newHistory(b)

	// No real mutation happened between two opOther transitions.
	h.recordPre(b, opOther)
	h.recordPre(b, opOther)

	if len(h.entries) != 1 {
		t.Fatalf("expected 1 entry (duplicates skipped), got %d", len(h.entries))
	}
	// lastOp must still be updated even when the push is skipped.
	if h.lastOp != opOther {
		t.Fatalf("expected lastOp==opOther after dedup, got %v", h.lastOp)
	}
}

func TestHistory_RecordPre_TruncatesRedoFuture(t *testing.T) {
	b := &fakeBuffer{}
	h := newHistory(b)

	// Build entries ["", "a", "ab"] with cursor at tip and live "abc".
	h.recordPre(b, opOther)
	b.value = "a"
	h.recordPre(b, opOther)
	b.value = "ab"
	h.recordPre(b, opOther)
	b.value = "abc"
	if len(h.entries) != 3 {
		t.Fatalf("precondition: expected 3 entries, got %d", len(h.entries))
	}

	// Undo moves cursor back. syncCurrent appends the live "abc" first,
	// so after the undo entries are ["", "a", "ab", "abc"] with cursor=2,
	// and b.value is restored to entries[2]="ab".
	h.undo(b)
	if b.value != "ab" {
		t.Fatalf("after undo expected live buffer \"ab\", got %q", b.value)
	}
	tipBefore := len(h.entries)
	cursorBefore := h.cursor
	if cursorBefore >= tipBefore-1 {
		t.Fatalf("precondition: expected cursor < tip after undo, got cursor=%d tip=%d",
			cursorBefore, tipBefore)
	}

	// New mutation: recordPre must discard the redo future beyond cursor.
	h.recordPre(b, opOther)
	b.value = "aZ"

	// The redo future entries (indices > cursorBefore) must have been dropped.
	if len(h.entries) != cursorBefore+1 {
		t.Fatalf("expected redo future truncated to len %d, got %d",
			cursorBefore+1, len(h.entries))
	}
	// Pre-state "ab" captured here dedups against tip entries[cursorBefore]="ab",
	// so no push from this call — verified by the truncation-only length above.

	// Now a second opOther with a different live value should push.
	h.recordPre(b, opOther)
	b.value = "aZZ"
	if h.entries[h.cursor].value != "aZ" {
		t.Fatalf("expected tip value \"aZ\" after second push, got %q",
			h.entries[h.cursor].value)
	}
}

func TestHistory_EntryCapEvictsOldest(t *testing.T) {
	b := &fakeBuffer{}
	h := newHistory(b)

	// Push 250 distinct opOther entries. Cap is 200. Per the recordPre
	// contract we snapshot pre-state, then "apply" the mutation.
	for i := 0; i < 250; i++ {
		h.recordPre(b, opOther)
		b.value = repeat("v", i+1) // each one distinct
	}

	if len(h.entries) != maxHistoryEntries {
		t.Fatalf("expected %d entries after eviction, got %d",
			maxHistoryEntries, len(h.entries))
	}
	// cursor must still point at the tip.
	if h.cursor != len(h.entries)-1 {
		t.Fatalf("expected cursor at tip (%d), got %d", len(h.entries)-1, h.cursor)
	}
}

// repeat returns s concatenated n times. Used to generate distinct
// entries in eviction-cap tests.
func repeat(s string, n int) string {
	out := make([]byte, 0, n*len(s))
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}

func TestHistory_ByteCapEvictsOldest(t *testing.T) {
	b := &fakeBuffer{}
	h := newHistory(b)

	// Push a handful of 20 MB entries; after 3 the byte cap (50 MB) must kick in.
	big := make([]byte, 20*1024*1024)
	for i := range big {
		big[i] = 'x'
	}
	for i := 0; i < 4; i++ {
		h.recordPre(b, opOther)
		// make each value distinct so dedup doesn't skip it
		b.value = string(big) + string(rune('a'+i))
	}

	if h.totalBytes > maxHistoryBytes {
		t.Fatalf("totalBytes=%d exceeds cap %d", h.totalBytes, maxHistoryBytes)
	}
}

func TestHistory_UndoAfterCoalescedRunThenRedo(t *testing.T) {
	b := &fakeBuffer{}
	h := newHistory(b)

	// Simulate typing "sel" as a coalesced run. recordPre is called
	// BEFORE each mutation (contract). The first recordPre dedups against
	// the baseline; subsequent recordPre calls hit the hot path.
	h.recordPre(b, opInsertChar)
	b.value = "s"
	h.recordPre(b, opInsertChar)
	b.value = "se"
	h.recordPre(b, opInsertChar)
	b.value = "sel"

	// At this point entries == [""], cursor=0, live buffer = "sel".
	if len(h.entries) != 1 {
		t.Fatalf("precondition: expected 1 entry, got %d", len(h.entries))
	}

	// Undo should revert live buffer to "". syncCurrent must first
	// append the live "sel" so redo has a target.
	if !h.undo(b) {
		t.Fatal("undo should have succeeded")
	}
	if b.value != "" {
		t.Fatalf("after undo expected value \"\", got %q", b.value)
	}

	// Redo must return to "sel".
	if !h.redo(b) {
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
	h := newHistory(b)

	// Build entries=["", "a"] with cursor=1 and live buffer "a".
	h.recordPre(b, opOther) // dedups pre-state "" vs baseline, lastOp=opOther
	b.value = "a"
	h.recordPre(b, opOther) // pushes pre-state "a", entries=["","a"], cursor=1
	// b.value is still "a" — live == tip.

	if len(h.entries) != 2 {
		t.Fatalf("precondition: expected 2 entries, got %d", len(h.entries))
	}

	// Live buffer equals tip, so syncCurrent should be a no-op during undo.
	// Undo decrements cursor to 0 and restores "".
	if !h.undo(b) {
		t.Fatal("undo should have succeeded")
	}
	if b.value != "" {
		t.Fatalf("after undo expected \"\", got %q", b.value)
	}
	// len(entries) must remain 2 — no duplicate was appended.
	if len(h.entries) != 2 {
		t.Fatalf("expected entries unchanged at len 2, got %d", len(h.entries))
	}
}
