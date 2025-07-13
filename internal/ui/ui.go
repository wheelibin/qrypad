package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	PopupError     = 1
	PopupHelp      = 2
	PopupPassword  = 3
	PopupResultRow = 4

	MinHeight = 24
	MinWidth  = 121
)

type PopupKindType int

var PopupKind = struct {
	Error            PopupKindType
	Help             PopupKindType
	Password         PopupKindType
	ResultRow        PopupKindType
	DatabaseSwitcher PopupKindType
	LoadingPopup     PopupKindType
}{
	Error:            1,
	Help:             2,
	Password:         3,
	ResultRow:        4,
	DatabaseSwitcher: 5,
	LoadingPopup:     6,
}

type bounds struct {
	x1 int
	x2 int
	y1 int
	y2 int
}

var appStyle = lipgloss.NewStyle()

type model struct {
	// components
	tablePanel            component.TablePanelModel
	tableInfoPanel        component.TableInfoPanelModel
	queryPanel            component.QueryPanelModel
	resultsPanel          component.ResultsPanelModel
	statusBar             component.StatusBarModel
	titleBar              component.TitlBarModel
	errorPopup            component.ErrorPopupModel
	passwordPopup         component.PasswordPopupModel
	resultRowPopup        component.ResultRowPopupModel
	databaseSwitcherPopup component.DatabaseSwitcherPopupModel
	helpPopup             component.HelpPopupModel
	loadingPopup          component.LoadingPopupModel

	// state
	connectionName   string
	dbConfig         db.ConnectionConfig
	db               db.DBConn
	activePanelIndex int
	errorMessage     string
	selectedDatabase string
	activePopup      PopupKindType
	cancelQuery      context.CancelFunc

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

func NewModel(connectionName string, dbConfig db.ConnectionConfig) model {
	tablePanel := component.NewTablePanelModel()
	tableInfoPanel := component.NewTableInfoPanelModel()
	queryPanel := component.NewQueryPanelModel(connectionName)
	resultsPanel := component.NewResultsPanelModel()
	statusBar := component.NewStatusBarModel(connectionName)
	titleBar := component.NewTitlBarModel(connectionName, dbConfig)
	errorPopup := component.NewErrorPopupModel()
	passwordPopup := component.NewPasswordPopupModel()
	resultRowPopup := component.NewResultRowPopupModel()
	databaseSwitcherPopup := component.NewDatabaseSwitcherPopupModel()
	helpPopup := component.NewHelpPopupModel()
	loadingPopup := component.NewLoadingPopupModel()

	autoSave := viper.GetBool("autoSave")

	return model{
		connectionName:        connectionName,
		dbConfig:              dbConfig,
		tablePanel:            tablePanel,
		tableInfoPanel:        tableInfoPanel,
		queryPanel:            queryPanel,
		resultsPanel:          resultsPanel,
		statusBar:             statusBar,
		titleBar:              titleBar,
		errorPopup:            errorPopup,
		passwordPopup:         passwordPopup,
		resultRowPopup:        resultRowPopup,
		databaseSwitcherPopup: databaseSwitcherPopup,
		helpPopup:             helpPopup,
		loadingPopup:          loadingPopup,
		selectablePanelCount:  4,
		autoSave:              autoSave,
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

	case db.DataFetchedMsg,
		db.DatabaseConnectedMsg,
		db.DatabaseListFetchedMsg,
		db.SchemaEntitiesFetchedMsg,
		db.TableInfoDataFetchedMsg,
		db.QueryControlMsg:
		cmds = append(cmds, m.handleDBMessages(msg))

	case commands.DatabaseConnectErrMsg,
		commands.ErrMsg:
		cmds = append(cmds, m.handleErrorMessages(msg))

	case commands.ActivePanelChangedMsg,
		commands.CopyValueMsg,
		commands.DatabaseSelectedMsg,
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

	case tea.MouseMsg:
		cmds = append(cmds, m.handleMouseMessages(msg))

	case tea.KeyMsg:
		cmds = append(cmds, m.handleKeyMessages(msg))

	}

	// update components
	switch m.activePopup {
	case PopupKind.ResultRow:
		m.resultRowPopup, cmd = m.resultRowPopup.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case PopupKind.Help:
		m.helpPopup, cmd = m.helpPopup.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case PopupKind.Password:
		m.passwordPopup, cmd = m.passwordPopup.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case PopupKind.Error:
		m.errorPopup, cmd = m.errorPopup.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case PopupKind.DatabaseSwitcher:
		m.databaseSwitcherPopup, cmd = m.databaseSwitcherPopup.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)

	case PopupKind.LoadingPopup:
		m.loadingPopup, cmd = m.loadingPopup.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	if m.activePanelIndex == PanelIndexTables {
		m.tablePanel, cmd = m.tablePanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.activePanelIndex == PanelIndexTableInfo {
		m.tableInfoPanel, cmd = m.tableInfoPanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	// always update the results panel so it can listen for loading messages
	m.resultsPanel, cmd = m.resultsPanel.Update(msg)
	cmds = append(cmds, cmd)

	// always update the status bar
	m.statusBar, cmd = m.statusBar.Update(msg)
	cmds = append(cmds, cmd)

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
	rightWidth := m.getRightWidth(m.width)

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
	m.passwordPopup.SetSize(m.width/3, 5)
	m.helpPopup.SetSize(120, 5)
}

func (m model) getRightWidth(totalWidth int) int {
	if m.leftPanelHidden {
		return style.GetSpan(12, totalWidth) - 8
	} else {
		return style.GetSpan(12-LeftPanelSpan, totalWidth) - 8
	}
}

func (m model) View() string {
	if m.windowTooSmall {
		return style.WindowTooSmall(m.width, m.height).
			Render("window too small")
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

	switch m.activePopup {
	case PopupKind.Error:
		p := m.errorPopup.View()
		x := m.width/2 - lipgloss.Width(p)/2
		y := m.height/2 - 2 - lipgloss.Height(p)/2
		contentView = style.PlaceOverlay(x, y, p, mainContent)

	case PopupKind.ResultRow:
		p := m.resultRowPopup.View()
		x := m.width/2 - lipgloss.Width(p)/2
		y := m.height/2 - 2 - lipgloss.Height(p)/2
		contentView = style.PlaceOverlay(x, y, p, mainContent)
	case PopupKind.Help:
		p := m.helpPopup.View()
		x := m.width/2 - lipgloss.Width(p)/2
		y := m.height/2 - 2 - lipgloss.Height(p)/2
		contentView = style.PlaceOverlay(x, y, p, mainContent)
	case PopupKind.Password:
		p := m.passwordPopup.View()
		x := m.width/2 - lipgloss.Width(p)/2
		y := m.height/2 - 2 - lipgloss.Height(p)/2
		contentView = style.PlaceOverlay(x, y, p, mainContent)
	case PopupKind.DatabaseSwitcher:
		p := m.databaseSwitcherPopup.View()
		x := m.width/2 - lipgloss.Width(p)/2
		y := m.height/2 - 2 - lipgloss.Height(p)/2
		contentView = style.PlaceOverlay(x, y, p, mainContent)
	case PopupKind.LoadingPopup:
		p := m.loadingPopup.View()
		x := m.width/2 - lipgloss.Width(p)/2
		y := m.height/2 - 2 - lipgloss.Height(p)/2
		contentView = style.PlaceOverlay(x, y, p, mainContent)
	}

	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Center,
		m.titleBar.View(),
		contentView,
		m.statusBar.View(),
	))
}
