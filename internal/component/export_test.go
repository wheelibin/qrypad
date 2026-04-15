package component

// Exported wrappers for unexported functions, used only in tests.

var (
	IsWordChar           = isWordChar           //nolint:gochecknoglobals
	GetStatementAtCursor = getStatementAtCursor //nolint:gochecknoglobals
	GetWordAtCursor      = getWordAtCursor      //nolint:gochecknoglobals
	GetColumnWidth       = getColumnWidth       //nolint:gochecknoglobals
)

func (m *TablePanelModel) SetActiveTabIndex(i int) {
	m.activeTabIndex = i
}

func (m *TableInfoPanelModel) SetActiveTabIndex(i int) {
	m.activeTabIndex = i
}
