package autocomplete

// Exported wrappers for unexported functions, used only in tests.

var (
	FilterNames       = filterNames       //nolint:gochecknoglobals
	GetAliasTableMap  = getAliasTableMap  //nolint:gochecknoglobals
	GetTextBeforeWord = getTextBeforeWord //nolint:gochecknoglobals
)
