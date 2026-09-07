package ui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/spf13/viper"

	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/component"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/querybuffer"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

// session holds in-memory runtime state for a live connection.
type session struct {
	connectionName   string
	dbConfig         db.ConnectionConfig
	selectedDatabase string
	queryText        string
	queryFilename    string
	results          *db.Data
	db               db.DBConn
	schemaCache      *db.SchemaCache
}

const (
	PanelIndexTables    = 0
	PanelIndexTableInfo = 1
	PanelIndexQuery     = 2
	PanelIndexResults   = 3

	StatusBarHeight         = 1
	TitleBarHeight          = 1
	ResultsPanelMinHeight   = 8
	TableInfoPanelMinHeight = 9
	TablePanelMinHeight     = 9
	LeftPanelSpan           = 3

	MinHeight = 24
	MinWidth  = 121
)

type PopupKindType int

//nolint:gochecknoglobals // singleton enum struct used as namespaced constants
var PopupKind = struct {
	Error              PopupKindType
	Help               PopupKindType
	Password           PopupKindType
	ResultRow          PopupKindType
	DatabaseSwitcher   PopupKindType
	Loading            PopupKindType
	ConnectionSwitcher PopupKindType
	ExportFormat       PopupKindType
	SessionList        PopupKindType
}{
	Error:              1,
	Help:               2,
	Password:           3,
	ResultRow:          4,
	DatabaseSwitcher:   5,
	Loading:            6,
	ConnectionSwitcher: 7,
	ExportFormat:       8,
	SessionList:        9,
}

type bounds struct {
	x1 int
	x2 int
	y1 int
	y2 int
}

//nolint:gochecknoglobals // package-level lipgloss style is intentional for performance
var appStyle = lipgloss.NewStyle()

//nolint:recvcheck // Bubble Tea model: Init/View/Update use value receiver per interface, mutating methods use pointer receiver
type model struct {
	// components
	tablePanel              component.TablePanelModel
	tableInfoPanel          component.TableInfoPanelModel
	queryPanel              component.QueryPanelModel
	resultsPanel            component.ResultsPanelModel
	statusBar               component.StatusBarModel
	titleBar                component.TitleBarModel
	errorPopup              component.ErrorPopupModel
	passwordPopup           component.PasswordPopupModel
	resultRowPopup          component.ResultRowPopupModel
	databaseSwitcherPopup   component.DatabaseSwitcherPopupModel
	connectionSwitcherPopup component.ConnectionSwitcherPopupModel
	helpPopup               component.HelpPopupModel
	loadingPopup            component.LoadingPopupModel
	exportFormatPopup       component.ExportFormatPopupModel
	sessionListPopup        component.SessionListPopupModel

	// state
	connectionName   string
	dbConfig         db.ConnectionConfig
	db               db.DBConn
	activePanelIndex int
	errorMessage     string
	selectedDatabase string
	currentTableRef  db.TableReference
	activePopup      PopupKindType
	cancelQuery      context.CancelFunc
	schemaCache      *db.SchemaCache
	schemaPreloading bool
	buf              *querybuffer.Buffer
	dir              string
	sessions         []session
	lastSessionName  string
	lastResultsData  *db.Data

	windowTooSmall       bool
	width                int
	height               int
	leftPanelHidden      bool
	selectablePanelCount int
	tablePanelBounds     bounds
	tableInfoPanelBounds bounds
	queryPanelBounds     bounds
	resultsPanelBounds   bounds
	autoSave             bool
}

// Model is the root Bubble Tea model for the application UI.
type Model = model

func NewModel(connectionName string, dbConfig db.ConnectionConfig, dir string) Model {
	autoSave := viper.GetBool("autoSave")

	tablePanel := component.NewTablePanelModel()
	tableInfoPanel := component.NewTableInfoPanelModel()
	queryPanel := component.NewQueryPanelModel(connectionName, dbConfig.Database, dbConfig.UseSingleQueryFile(), autoSave, dir)
	resultsPanel := component.NewResultsPanelModel()
	statusBar := component.NewStatusBarModel(connectionName)
	titleBar := component.NewTitleBarModel(connectionName, dbConfig)
	errorPopup := component.NewErrorPopupModel()
	passwordPopup := component.NewPasswordPopupModel()
	resultRowPopup := component.NewResultRowPopupModel()
	databaseSwitcherPopup := component.NewDatabaseSwitcherPopupModel()
	connectionSwitcherPopup := component.NewConnectionSwitcherPopupModel()
	sessionListPopup := component.NewSessionListPopupModel()
	helpPopup := component.NewHelpPopupModel()
	loadingPopup := component.NewLoadingPopupModel()
	exportFormatPopup := component.NewExportFormatPopupModel()

	return model{
		connectionName:          connectionName,
		dbConfig:                dbConfig,
		selectedDatabase:        dbConfig.Database,
		tablePanel:              tablePanel,
		tableInfoPanel:          tableInfoPanel,
		queryPanel:              queryPanel,
		resultsPanel:            resultsPanel,
		statusBar:               statusBar,
		titleBar:                titleBar,
		errorPopup:              errorPopup,
		passwordPopup:           passwordPopup,
		resultRowPopup:          resultRowPopup,
		databaseSwitcherPopup:   databaseSwitcherPopup,
		connectionSwitcherPopup: connectionSwitcherPopup,
		sessionListPopup:        sessionListPopup,
		helpPopup:               helpPopup,
		loadingPopup:            loadingPopup,
		exportFormatPopup:       exportFormatPopup,
		selectablePanelCount:    4,
		autoSave:                autoSave,
		schemaCache:             db.NewSchemaCache(),
		buf:                     querybuffer.New(dir),
		dir:                     dir,
	}
}

func (m model) initKeyMap() {
	isSQLite := m.dbConfig.Driver == db.DriverName.SQLite
	keys.DefaultKeyMap.SwitchDatabase.SetEnabled(!isSQLite)
	keys.DefaultKeyMap.UpdatePassword.SetEnabled(!isSQLite)
}

func (m model) Init() tea.Cmd {
	m.initKeyMap()

	// Initialize sub-models
	return tea.Batch(
		commands.ConnectToDB(m.connectionName, m.dbConfig),
		m.tablePanel.Init(),
		m.tableInfoPanel.Init(),
		m.queryPanel.Init(),
		m.resultsPanel.Init(),
		m.statusBar.Init(),
		m.titleBar.Init(),
		m.errorPopup.Init(),
		m.passwordPopup.Init(),
		m.resultRowPopup.Init(),
		m.loadingPopup.Init(),
	)
}

func (m *model) setPanelsActiveState(activePanelIndex int) {
	if activePanelIndex == -1 {
		m.tablePanel.SetActive(false)
		m.tableInfoPanel.SetActive(false)
		m.queryPanel.SetActive(false)
		m.resultsPanel.SetActive(false)
		return
	}

	m.tablePanel.SetActive(m.activePanelIndex == PanelIndexTables)
	m.tableInfoPanel.SetActive(m.activePanelIndex == PanelIndexTableInfo)
	m.queryPanel.SetActive(m.activePanelIndex == PanelIndexQuery)
	m.resultsPanel.SetActive(m.activePanelIndex == PanelIndexResults)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	if m.connectionName == "" && !m.hasActivePopup() {
		// if no connection then show connection switcher (once only)
		m.connectionSwitcherPopup.SetIsFirstConnection(true)
		m.showPopup(PopupKind.ConnectionSwitcher)
		cmds = append(cmds, commands.GetConnectionList())
	}

	// update this now so the query text value is updated and can be used below
	if !m.hasActivePopup() {
		m.queryPanel, cmd = m.queryPanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Dispatch to handlers — each handler does its own type switch and returns
	// nil for messages it doesn't own, so new message types only need to be
	// added in the relevant handler, not here.
	cmds = append(cmds, m.handleDBMessages(msg))
	cmds = append(cmds, m.handleErrorMessages(msg))
	cmds = append(cmds, m.handleCommandMessages(msg))

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.adjustSizes()

	case tea.FocusMsg:
		m.setPanelsActiveState(m.activePanelIndex)

	case tea.BlurMsg:
		m.setPanelsActiveState(-1)

	case tea.MouseClickMsg:
		cmds = append(cmds, m.handleMouseMessages(msg))

	case tea.KeyPressMsg:
		cmds = append(cmds, m.handleKeyMessages(msg))
	}

	// always update the table panel for data messages (e.g. SchemaEntitiesFetchedMsg)
	// even when a popup is active, but skip key/mouse events to avoid
	// background panels reacting to input meant for the popup
	switch msg.(type) {
	case tea.KeyPressMsg, tea.MouseMsg:
		if !m.hasActivePopup() {
			m.tablePanel, cmd = m.tablePanel.Update(msg)
			cmds = append(cmds, cmd)
		}
	default:
		m.tablePanel, cmd = m.tablePanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	// same treatment for results panel
	switch msg.(type) {
	case tea.KeyPressMsg, tea.MouseMsg:
		if !m.hasActivePopup() {
			m.resultsPanel, cmd = m.resultsPanel.Update(msg)
			cmds = append(cmds, cmd)
		}
	default:
		m.resultsPanel, cmd = m.resultsPanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.statusBar, cmd = m.statusBar.Update(msg)
	cmds = append(cmds, cmd)

	// update popup components — popups handle their own key events
	if m.hasActivePopup() {
		cmds = append(cmds, m.updateActivePopup(msg))
		return m, tea.Batch(cmds...)
	}

	if m.activePanelIndex == PanelIndexTableInfo {
		m.tableInfoPanel, cmd = m.tableInfoPanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) closePopup() {
	m.activePopup = 0
}

func (m model) hasActivePopup() bool {
	return m.activePopup > 0
}

func (m *model) showPopup(p PopupKindType) {
	m.activePopup = p
}

func (m model) popupIsActive(p PopupKindType) bool {
	return m.activePopup == p
}

func (m *model) updateActivePopup(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch m.activePopup {
	case PopupKind.ResultRow:
		m.resultRowPopup, cmd = m.resultRowPopup.Update(msg)
	case PopupKind.Help:
		m.helpPopup, cmd = m.helpPopup.Update(msg)
	case PopupKind.Password:
		m.passwordPopup, cmd = m.passwordPopup.Update(msg)
	case PopupKind.Error:
		m.errorPopup, cmd = m.errorPopup.Update(msg)
	case PopupKind.DatabaseSwitcher:
		m.databaseSwitcherPopup, cmd = m.databaseSwitcherPopup.Update(msg)
	case PopupKind.ConnectionSwitcher:
		m.connectionSwitcherPopup, cmd = m.connectionSwitcherPopup.Update(msg)
	case PopupKind.Loading:
		m.loadingPopup, cmd = m.loadingPopup.Update(msg)
	case PopupKind.ExportFormat:
		m.exportFormatPopup, cmd = m.exportFormatPopup.Update(msg)
	case PopupKind.SessionList:
		m.sessionListPopup, cmd = m.sessionListPopup.Update(msg)
	}
	return cmd
}

func (m model) activePopupView() string {
	switch m.activePopup {
	case PopupKind.Error:
		return m.errorPopup.View()
	case PopupKind.ResultRow:
		return m.resultRowPopup.View()
	case PopupKind.Help:
		return m.helpPopup.View()
	case PopupKind.Password:
		return m.passwordPopup.View()
	case PopupKind.DatabaseSwitcher:
		return m.databaseSwitcherPopup.View()
	case PopupKind.ConnectionSwitcher:
		return m.connectionSwitcherPopup.View()
	case PopupKind.Loading:
		return m.loadingPopup.View()
	case PopupKind.ExportFormat:
		return m.exportFormatPopup.View()
	case PopupKind.SessionList:
		return m.sessionListPopup.View()
	}
	return ""
}

func (m *model) adjustSizes() {
	m.windowTooSmall = false

	if m.height < MinHeight || m.width < MinWidth {
		m.windowTooSmall = true
		return
	}

	availableHeight := m.height - TitleBarHeight - StatusBarHeight
	leftWidth := style.GetSpan(LeftPanelSpan, m.width) + 1
	if m.leftPanelHidden {
		leftWidth = 3
	}
	rightWidth := m.getRightWidth(m.width) + 1

	// left
	tableInfoHeight := max(style.GetSpan(3, availableHeight), TableInfoPanelMinHeight)
	tableHeight := max(availableHeight-tableInfoHeight-4, TablePanelMinHeight)

	m.tableInfoPanel.SetSize(leftWidth, tableInfoHeight)
	m.tablePanel.SetSize(leftWidth, tableHeight)

	// right
	resultsHeight := max(style.GetSpan(6, availableHeight), ResultsPanelMinHeight)
	queryHeight := availableHeight - resultsHeight - 4
	m.resultsPanel.SetSize(rightWidth, resultsHeight)
	m.queryPanel.SetSize(rightWidth, queryHeight)

	// store the bounding boxes for the panels
	m.tablePanelBounds = bounds{x1: 1, x2: leftWidth + 1, y1: 1, y2: tableHeight + 1}
	m.tableInfoPanelBounds = bounds{x1: 1, x2: leftWidth + 1, y1: 2 + tableHeight, y2: m.height - StatusBarHeight - 1}
	m.queryPanelBounds = bounds{x1: leftWidth + 4, x2: m.width - 1, y1: 1, y2: queryHeight + 1}
	m.resultsPanelBounds = bounds{x1: leftWidth + 2, x2: m.width - 1, y1: queryHeight + 2, y2: m.height - StatusBarHeight - 1}

	// set non panel component sizes
	m.statusBar.SetSize(m.width, StatusBarHeight)
	m.titleBar.SetSize(m.width, TitleBarHeight)
	m.errorPopup.SetSize(m.width/2, 5)
	m.resultRowPopup.SetSize(m.width/2, m.height/2)
	m.databaseSwitcherPopup.SetSize(m.width/3, m.height/3)
	m.connectionSwitcherPopup.SetSize(m.width/3, m.height/3)
	m.sessionListPopup.SetSize(m.width/3, m.height/3)
	m.passwordPopup.SetSize(m.width/3, 5)
	m.helpPopup.SetSize(120, 5)
	m.exportFormatPopup.SetSize(m.width/3, 3)
}

// closeAndResetDB closes the active database connection (if any) and zeroes
// the field so that subsequent handlers (e.g. DatabaseConnectedMsg) don't
// accidentally close a session's still-live connection.
func (m *model) closeAndResetDB() {
	if m.db.DB != nil {
		_ = m.db.DB.Close()
	}
	m.db = db.DBConn{}
}

func (m *model) saveCurrentSession() {
	if m.connectionName == "" {
		return
	}
	s := session{
		connectionName:   m.connectionName,
		dbConfig:         m.dbConfig,
		selectedDatabase: m.selectedDatabase,
		queryText:        m.queryPanel.GetValue(),
		queryFilename:    m.queryPanel.GetFilename(),
		results:          m.lastResultsData,
		db:               m.db,
		schemaCache:      m.schemaCache,
	}
	if i, ok := m.sessionIndex(m.connectionName); ok {
		m.sessions[i] = s
		return
	}
	m.sessions = append(m.sessions, s)
}

func (m *model) sessionIndex(connName string) (int, bool) {
	for i, s := range m.sessions {
		if s.connectionName == connName {
			return i, true
		}
	}
	return -1, false
}

func (m *model) findSession(connName string) (session, bool) {
	if i, ok := m.sessionIndex(connName); ok {
		return m.sessions[i], true
	}
	return session{}, false
}

// popSession removes a session from the pool without closing its DB connection.
// Use this when restoring a session; the connection is still needed.
func (m *model) popSession(connName string) {
	if i, ok := m.sessionIndex(connName); ok {
		m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
	}
}

// removeSession removes a session from the pool and closes its DB connection.
// Use this when the session is being discarded entirely.
func (m *model) removeSession(connName string) {
	if i, ok := m.sessionIndex(connName); ok {
		if m.sessions[i].db.DB != nil {
			_ = m.sessions[i].db.DB.Close()
		}
		m.sessions = append(m.sessions[:i], m.sessions[i+1:]...)
	}
}

func (m *model) restoreSession(s session) tea.Cmd {
	m.connectionName = s.connectionName
	m.dbConfig = s.dbConfig
	m.selectedDatabase = s.selectedDatabase
	m.db = s.db
	m.schemaCache = s.schemaCache
	m.lastResultsData = s.results
	m.titleBar.SetConnectionName(s.connectionName)
	m.titleBar.SetConn(s.dbConfig)
	m.statusBar.SetSelectedDatabase(s.selectedDatabase)
	m.queryPanel.SetConnectionName(s.connectionName)
	m.queryPanel.SetDatabaseName(s.selectedDatabase)
	m.queryPanel.SetSingleQueryFile(s.dbConfig.UseSingleQueryFile())
	m.queryPanel.SetValue(s.queryText)
	m.queryPanel.SetFilename(s.queryFilename)
	m.queryPanel.SetDirty(false)
	m.buf.SetSaved(s.queryText)
	if s.results != nil {
		m.resultsPanel.SetData(s.results)
	} else {
		m.resultsPanel.Clear()
	}
	m.closePopup()
	m.initKeyMap()
	return commands.GetSchemaEntities(m.db, commands.TablePanelKind.Tables, m.schemaCache)
}

func (m *model) totalSessionCount() int {
	return len(m.sessions) + 1
}

func (m *model) openSessionList() {
	active := component.SessionListEntry{
		ConnName: m.connectionName,
		DBName:   m.selectedDatabase,
		IsActive: true,
	}
	entries := make([]component.SessionListEntry, 1, 1+len(m.sessions))
	entries[0] = active
	for _, s := range m.sessions {
		entries = append(entries, component.SessionListEntry{
			ConnName: s.connectionName,
			DBName:   s.selectedDatabase,
		})
	}
	m.sessionListPopup.SetEntries(entries)
	m.showPopup(PopupKind.SessionList)
}

func (m model) getRightWidth(totalWidth int) int {
	if m.leftPanelHidden {
		return style.GetSpan(12, totalWidth) - 8
	}
	return style.GetSpan(12-LeftPanelSpan, totalWidth) - 8
}

func (m model) View() tea.View {
	if m.windowTooSmall {
		content := style.WindowTooSmall(m.width, m.height).
			Render("window too small")
		v := tea.NewView(content)
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		v.ReportFocus = true
		return v
	}

	left := lipgloss.JoinVertical(lipgloss.Center,
		m.tablePanel.View(),
		m.tableInfoPanel.View(),
	)
	if m.leftPanelHidden {
		left = lipgloss.NewStyle().
			Height(m.height-StatusBarHeight-TitleBarHeight).
			Width(3).
			Background(theme.GetTheme().TitleBar.BG).
			Foreground(theme.GetTheme().TitleBar.FG).
			Align(lipgloss.Center, lipgloss.Center).
			Render("▶")
	}

	right := lipgloss.JoinVertical(lipgloss.Center,
		m.queryPanel.View(),
		m.resultsPanel.View(),
	)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Bottom,
		left+lipgloss.NewStyle().MarginRight(1).Render(),
		right,
	)

	contentView := mainContent

	if p := m.activePopupView(); p != "" {
		x := m.width/2 - lipgloss.Width(p)/2
		y := m.height/2 - 2 - lipgloss.Height(p)/2
		contentView = style.PlaceOverlay(x, y, p, mainContent)
	}

	content := appStyle.Render(lipgloss.JoinVertical(lipgloss.Center,
		m.titleBar.View(),
		contentView,
		m.statusBar.View(),
	))
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.ReportFocus = true
	return v
}
