package commands

type ErrMsg struct{ Err error }

func (e ErrMsg) Error() string { return e.Err.Error() }

type ActivePanelChangedMsg int

type LoadingMsg struct{}

// contains the selected table name
type TableSelectedMsg string

// the contents of the query file
type QueryFileReadMsg string

type QueryFileSavedMsg struct{}
