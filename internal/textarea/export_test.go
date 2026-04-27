// export_test.go – compiled only during tests; provides white-box access to
// unexported identifiers for the textarea_test package.
package textarea

// Re-export op-kind constants.
const (
	ExportOpInsertChar      = opInsertChar
	ExportOpOther           = opOther
	ExportMaxHistoryEntries = maxHistoryEntries
	ExportMaxHistoryBytes   = maxHistoryBytes
)

// ExportBufferState is the exported alias for the unexported bufferState
// interface, so tests in textarea_test can declare types that implement it.
type ExportBufferState = bufferState

// ExportHistory wraps *history and exposes its fields and methods for
// white-box testing from the textarea_test package.
type ExportHistory struct {
	h *history
}

// NewExportHistory wraps newHistory for use from textarea_test.
func NewExportHistory(b ExportBufferState) *ExportHistory {
	return &ExportHistory{h: newHistory(b)}
}

func (eh *ExportHistory) Entries() int            { return len(eh.h.entries) }
func (eh *ExportHistory) Cursor() int             { return eh.h.cursor }
func (eh *ExportHistory) LastOp() ExportOpKind    { return eh.h.lastOp }
func (eh *ExportHistory) TotalBytes() int         { return eh.h.totalBytes }
func (eh *ExportHistory) EntryValue(i int) string { return eh.h.entries[i].value }

func (eh *ExportHistory) RecordPre(b ExportBufferState, op ExportOpKind) {
	eh.h.recordPre(b, op)
}
func (eh *ExportHistory) Undo(b ExportBufferState) bool { return eh.h.undo(b) }
func (eh *ExportHistory) Redo(b ExportBufferState) bool { return eh.h.redo(b) }

// ExportOpKind is the exported type alias for opKind.
type ExportOpKind = opKind

// Model-level exports for textarea_undo tests.

// ExportHistoryLen returns len(m.history.entries).
func ExportHistoryLen(m *Model) int {
	return len(m.history.entries)
}

// ExportHistoryCursor returns m.history.cursor.
func ExportHistoryCursor(m *Model) int {
	return m.history.cursor
}

// ExportHistoryEntryValue returns m.history.entries[i].value.
func ExportHistoryEntryValue(m *Model, i int) string {
	return m.history.entries[i].value
}

// ExportHistoryUndo calls m.history.undo(m).
func ExportHistoryUndo(m *Model) bool {
	return m.history.undo(m)
}

// ExportEnsureHistory calls m.ensureHistory().
func ExportEnsureHistory(m *Model) {
	m.ensureHistory()
}

// ExportRecordPre calls m.history.recordPre(m, op).
func ExportRecordPre(m *Model, op ExportOpKind) {
	m.history.recordPre(m, op)
}

// ExportSetValueInternal calls m.setValueInternal(s).
func ExportSetValueInternal(m *Model, s string) {
	m.setValueInternal(s)
}

// ExportHistoryEntriesLen returns len(m.history.entries).
// (Alias provided for the undo test to get the count after recordPre setup.)
func ExportHistoryEntriesLen(m *Model) int {
	return len(m.history.entries)
}
