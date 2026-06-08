package commands

import "github.com/wheelibin/qrypad/internal/db"

// ErrMsg wraps an error for use as a Bubble Tea message.
// All command errors are passed back using this.
//
//nolint:errname // ErrMsg is a Bubble Tea message type, not a standard Go error
type ErrMsg struct{ Err error }

func (e ErrMsg) Error() string { return e.Err.Error() }

// DatabaseConnectError wraps a database connection error as a Bubble Tea message.
type DatabaseConnectError struct{ Err error }

func (e DatabaseConnectError) Error() string { return e.Err.Error() }

// DatabaseConnectErrMsg is an alias for backward compatibility.
//
// Deprecated: Use DatabaseConnectError instead.
//
//nolint:errname // Alias for backward compatibility
type DatabaseConnectErrMsg = DatabaseConnectError

// ActivePanelChangedMsg is sent when the user navigates to another panel.
type ActivePanelChangedMsg int

// LoadingMsg is sent when loading has started.
type LoadingMsg struct{ Loading bool }

// DatabaseSelectedMsg contains the selected database name.
type DatabaseSelectedMsg string

// ConnectionSelectedMsg contains the selected connection name.
type ConnectionSelectedMsg string

// TableSelectedMsg contains the selected table reference.
type TableSelectedMsg db.TableReference

// QueryFileReadMsg contains the details of the query file.
type QueryFileReadMsg struct{ FileName, Contents string }

// QueryFileSavedMsg is sent when the query file has been saved.
type QueryFileSavedMsg struct{}

// TablePanelTabChangedMsg is sent when the user navigates to another tab in the table panel.
type TablePanelTabChangedMsg int

// TableInfoTabChangedMsg is sent when the user navigates to another tab in the table info panel.
type TableInfoTabChangedMsg int

// EditorFinishedMsg is fired when external editor is closed.
type EditorFinishedMsg struct{ Err error }

type (
	PasswordEnteredMsg     string
	PopupClosedMsg         struct{}
	PasswordInputNeededMsg struct{}
	PasswordSavedMsg       struct{}
	CopyValueMsg           struct {
		Value     string
		ValueDesc string
	}
	CancelQueryMsg               struct{}
	AutoCompleteEntrySelectedMsg string
	AutoCompleteCloseMsg         struct{}
)

// NoConnectionChosenMsg is sent when the user closes the connection switcher
// without choosing a connection on first launch.
type NoConnectionChosenMsg struct{}

// ExportFormat selects the serialization format for exported results.
type ExportFormat int

const (
	ExportFormatJSON ExportFormat = iota
	ExportFormatCSV
)

// ExportRequestedMsg is emitted by the export format popup when the user picks
// a format. The UI reacts by serializing the current result set and writing it
// to disk.
type ExportRequestedMsg struct {
	Format ExportFormat
}

// ExportCompletedMsg is emitted by the export command after a file has been
// successfully written.
type ExportCompletedMsg struct {
	Path     string
	RowCount int
}

// SchemaPreloadCompleteMsg is sent when background column preloading finishes.
type SchemaPreloadCompleteMsg struct{}

// SchemaPreloadErrorMsg is sent when background column preloading fails.
// This is non-fatal; the app continues with lazy loading.
type SchemaPreloadErrorMsg struct{ Err error }
