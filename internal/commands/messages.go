package commands

// all command errors are passed back using this
type ErrMsg struct{ Err error }

func (e ErrMsg) Error() string { return e.Err.Error() }

// sent when the user navigates to another panel
type ActivePanelChangedMsg int

// sent when loading has started
type LoadingMsg struct{ Loading bool }

// contains the selected table name
type TableSelectedMsg string

// the details of the query file
type QueryFileReadMsg struct{ FileName, Contents string }

// sent when the query file has been saved
type QueryFileSavedMsg struct{}

// sent when the user navigates to another tab in the table info panel
type TableInfoTabChangedMsg int

// fired when external editor is closed
type EditorFinishedMsg struct{ err error }
