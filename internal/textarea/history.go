package textarea

// opKind classifies a buffer mutation so consecutive mutations of the
// same kind can be coalesced into a single undo step.
type opKind int

const (
	opNone       opKind = iota
	opInsertChar        // single printable rune insert (not whitespace)
	opDeleteChar        // single-character delete (backspace or delete)
	opOther             // pastes, newlines, word ops, case changes, merges, transpose
)

const (
	maxHistoryEntries = 200
	maxHistoryBytes   = 50 * 1024 * 1024 // 50 MB
)

// bufferState is the minimum surface area the history needs on the
// textarea Model. This indirection keeps history unit tests free of a
// real Model and enforces that restore goes through SetValueRaw (never
// SetValue — which would reset history).
type bufferState interface {
	Value() string
	SetValueRaw(s string)
	CursorPos() (row, col int)
	SetCursorPos(row, col int)
}

type historyEntry struct {
	value string
	row   int
	col   int
}

type history struct {
	entries    []historyEntry
	cursor     int
	lastOp     opKind
	totalBytes int
}

// newHistory creates a history with the buffer's current value pushed as
// the baseline entry. After this, len(entries) == 1 and cursor == 0.
func newHistory(b bufferState) *history {
	row, col := b.CursorPos()
	e := historyEntry{value: b.Value(), row: row, col: col}
	return &history{
		entries:    []historyEntry{e},
		cursor:     0,
		lastOp:     opNone,
		totalBytes: len(e.value),
	}
}

func (h *history) undo(b bufferState) bool {
	// syncCurrent first: if we're mid-coalesced-run, the live buffer is
	// ahead of the tip snapshot. Append that live state so redo has a
	// target and so the cursor has somewhere to step back FROM.
	h.syncCurrent(b)
	if h.cursor == 0 {
		return false
	}
	h.cursor--
	h.restore(b, h.entries[h.cursor])
	h.lastOp = opNone
	return true
}

func (h *history) redo(b bufferState) bool {
	if h.cursor >= len(h.entries)-1 {
		return false
	}
	h.cursor++
	h.restore(b, h.entries[h.cursor])
	h.lastOp = opNone
	return true
}

func (h *history) restore(b bufferState, e historyEntry) {
	b.SetValueRaw(e.value)
	b.SetCursorPos(e.row, e.col)
}

// syncCurrent plugs the coalescing gap: if the live buffer differs from
// the current tip entry (because we're mid-way through a coalesced typing
// run that has not yet crossed an op-kind transition), append the live
// state so redo has a target to jump forward to.
//
// Called by undo() before decrementing cursor.
func (h *history) syncCurrent(b bufferState) {
	row, col := b.CursorPos()
	live := historyEntry{value: b.Value(), row: row, col: col}
	if live.value == h.entries[h.cursor].value {
		return
	}
	// Truncate redo future if any (same as recordPre).
	if h.cursor < len(h.entries)-1 {
		for i := h.cursor + 1; i < len(h.entries); i++ {
			h.totalBytes -= len(h.entries[i].value)
		}
		h.entries = h.entries[:h.cursor+1]
	}
	h.entries = append(h.entries, live)
	h.totalBytes += len(live.value)
	h.cursor = len(h.entries) - 1
	h.evict()
}

// recordPre snapshots the CURRENT live buffer state as a pre-edit
// checkpoint, but ONLY when the op kind transitions (or when op is
// opOther, which never coalesces). During a coalesced run of the same
// op kind, this function is a no-op — the existing top-of-stack
// snapshot already represents the state we want to return to.
//
// Caller contract: call recordPre BEFORE applying the mutation.
func (h *history) recordPre(b bufferState, op opKind) {
	// Hot path: coalesce consecutive identical ops (except opOther).
	if op == h.lastOp && op != opOther {
		return
	}

	row, col := b.CursorPos()
	candidate := historyEntry{value: b.Value(), row: row, col: col}

	// Truncate any redo future — we're about to diverge from it.
	if h.cursor < len(h.entries)-1 {
		for i := h.cursor + 1; i < len(h.entries); i++ {
			h.totalBytes -= len(h.entries[i].value)
		}
		h.entries = h.entries[:h.cursor+1]
	}

	// Skip duplicates: if the candidate equals the current tip, don't
	// push (still update lastOp below).
	if candidate.value != h.entries[h.cursor].value {
		h.entries = append(h.entries, candidate)
		h.totalBytes += len(candidate.value)
		h.cursor = len(h.entries) - 1
		h.evict()
	}

	h.lastOp = op
}

// evict drops the oldest entries until both caps are satisfied.
// Eviction removes index 0, decrements cursor, and subtracts bytes.
func (h *history) evict() {
	for len(h.entries) > maxHistoryEntries || h.totalBytes > maxHistoryBytes {
		if len(h.entries) <= 1 {
			// Never evict the last remaining entry.
			return
		}
		h.totalBytes -= len(h.entries[0].value)
		h.entries = h.entries[1:]
		if h.cursor > 0 {
			h.cursor--
		}
	}
}
