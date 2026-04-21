package autocomplete

// Exported wrappers for unexported functions, used only in tests.

var (
	GetAliasTableMap  = getAliasTableMap  //nolint:gochecknoglobals
	GetTextBeforeWord = getTextBeforeWord //nolint:gochecknoglobals
)
