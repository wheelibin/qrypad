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
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

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
}{
	Error:              1,
	Help:               2,
	Password:           3,
	ResultRow:          4,
	DatabaseSwitcher:   5,
	Loading:            6,
	ConnectionSwitcher: 7,
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

	windowTooSmall         bool
	width                  int
	height                 int
	leftPanelHidden        bool
	selectablePanelCount   int
	lastSavedQueryContents string
	tablePanelBounds       bounds
	tableInfoPanelBounds   bounds
	queryPanelBounds       bounds
	resultsPanelBounds     bounds
	autoSave               bool
}

// Model is the root Bubble Tea model for the application UI.
type Model = model

func NewModel(connectionName string, dbConfig db.ConnectionConfig) Model {
	autoSave := viper.GetBool("autoSave")

	tablePanel := component.NewTablePanelModel()
	tableInfoPanel := component.NewTableInfoPanelModel()
	queryPanel := component.NewQueryPanelModel(connectionName, autoSave)
	resultsPanel := component.NewResultsPanelModel()
	statusBar := component.NewStatusBarModel(connectionName)
	titleBar := component.NewTitleBarModel(connectionName, dbConfig)
	errorPopup := component.NewErrorPopupModel()
	passwordPopup := component.NewPasswordPopupModel()
	resultRowPopup := component.NewResultRowPopupModel()
	databaseSwitcherPopup := component.NewDatabaseSwitcherPopupModel()
	connectionSwitcherPopup := component.NewConnectionSwitcherPopupModel()
	helpPopup := component.NewHelpPopupModel()
	loadingPopup := component.NewLoadingPopupModel()

	return model{
		connectionName:          connectionName,
		dbConfig:                dbConfig,
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
		helpPopup:               helpPopup,
		loadingPopup:            loadingPopup,
		selectablePanelCount:    4,
		autoSave:                autoSave,
		schemaCache:             db.NewSchemaCache(),
	}
}

func (m model) initKeyMap() {
	if m.dbConfig.Driver == db.DriverName.SQLite {
		keys.DefaultKeyMap.SwitchDatabase.SetEnabled(false)
		keys.DefaultKeyMap.UpdatePassword.SetEnabled(false)
	}
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

	// update this now so the query text value is updated and can be used below
	if !m.hasActivePopup() {
		m.queryPanel, cmd = m.queryPanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.adjustSizes()

	case tea.FocusMsg:
		m.setPanelsActiveState(m.activePanelIndex)
	case tea.BlurMsg:
		m.setPanelsActiveState(-1)

	case db.AutoCompleteDataFetchedMsg,
		db.DataFetchedMsg,
		db.DatabaseConnectedMsg,
		db.DatabaseListFetchedMsg,
		db.ConnectionListFetchedMsg,
		db.SchemaEntitiesFetchedMsg,
		db.TableInfoDataFetchedMsg,
		db.QueryControlMsg:
		cmds = append(cmds, m.handleDBMessages(msg))

	case commands.DatabaseConnectError,
		commands.ErrMsg:
		cmds = append(cmds, m.handleErrorMessages(msg))

	case commands.ActivePanelChangedMsg,
		commands.CopyValueMsg,
		commands.DatabaseSelectedMsg,
		commands.ConnectionSelectedMsg,
		commands.PasswordEnteredMsg,
		commands.PasswordInputNeededMsg,
		commands.PopupClosedMsg,
		commands.PasswordSavedMsg,
		commands.TableInfoTabChangedMsg,
		commands.TablePanelTabChangedMsg,
		commands.TableSelectedMsg,
		commands.QueryFileReadMsg,
		commands.LoadingMsg,
		commands.CancelQueryMsg:
		cmds = append(cmds, m.handleCommandMessages(msg))

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
	m.passwordPopup.SetSize(m.width/3, 5)
	m.helpPopup.SetSize(120, 5)
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
