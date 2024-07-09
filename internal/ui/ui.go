package ui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/wheelibin/dbee/internal/colour"
	"github.com/wheelibin/dbee/internal/commands"
	"github.com/wheelibin/dbee/internal/component"
	"github.com/wheelibin/dbee/internal/db"
	"github.com/wheelibin/dbee/internal/keys"
	"github.com/wheelibin/dbee/internal/style"
)

const (
	PanelIndexTables    = 0
	PanelIndexTableInfo = 1
	PanelIndexQuery     = 2
	PanelIndexResults   = 3

	StatusBarHeight         = 1
	TitleBarHeight          = 0
	ResultsPanelMinHeight   = 8
	QueryPanelMinHeight     = 5
	TableInfoPanelMinHeight = 8
	TablePanelMinHeight     = 10
)

type bounds struct {
	x1 int
	x2 int
	y1 int
	y2 int
}

var appStyle = lipgloss.NewStyle()

type model struct {
	// components
	tablePanel     component.TablePanelModel
	tableInfoPanel component.TableInfoPanelModel
	queryPanel     component.QueryPanelModel
	resultsPanel   component.ResultsPanelModel
	statusBar      component.StatusBarModel
	titleBar       component.TitlBarModel
	errorPopup     component.ErrorPopupModel
	resultRowPopup component.ResultRowPopupModel
	help           help.Model

	// state
	dbAlias                string
	db                     db.DBConn
	activePanelIndex       int
	errorMessage           string
	loading                bool
	windowTooSmall         bool
	width                  int
	height                 int
	leftPanelHidden        bool
	selectablePanelCount   int
	lastSavedQueryContents string
	showResultRowPopup     bool
	showHelpPopup          bool
	tablePanelBounds       bounds
	tableInfoPanelBounds   bounds
	queryPanelBounds       bounds
	resultsPanelBounds     bounds
}

func NewModel(dbAlias string, db db.DBConn) model {
	tablePanel := component.NewTablePanelModel()
	tableInfoPanel := component.NewTableInfoPanelModel()
	queryPanel := component.NewQueryPanelModel(dbAlias)
	resultsPanel := component.NewResultsPanelModel()
	statusBar := component.NewStatusBarModel(dbAlias)
	titleBar := component.NewTitlBarModel()
	errorPopup := component.NewErrorPopupModel()
	resultRowPopup := component.NewResultRowPopupModel()

	help := help.New()
	help.Styles.FullKey = lipgloss.NewStyle().Foreground(colour.HelpKey)
	help.Styles.FullDesc = lipgloss.NewStyle().Foreground(colour.HelpDesc)

	return model{
		dbAlias:              dbAlias,
		db:                   db,
		tablePanel:           tablePanel,
		tableInfoPanel:       tableInfoPanel,
		queryPanel:           queryPanel,
		resultsPanel:         resultsPanel,
		statusBar:            statusBar,
		titleBar:             titleBar,
		errorPopup:           errorPopup,
		resultRowPopup:       resultRowPopup,
		help:                 help,
		selectablePanelCount: 4,
	}
}

func (m model) Init() tea.Cmd {
	// Initialize sub-models
	return tea.Batch(
		m.tablePanel.Init(m.db),
		m.tableInfoPanel.Init(),
		m.queryPanel.Init(),
		m.resultsPanel.Init(),
		m.statusBar.Init(),
		m.titleBar.Init(),
		m.errorPopup.Init(),
		m.resultRowPopup.Init(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// log.Println("ui.model::Update", msg)
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// update this now so the query text value is updated and can be used below
	if len(m.errorMessage) == 0 {
		m.queryPanel, cmd = m.queryPanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.adjustSizes()

	case db.DataMsg:
		m.loading = false
		m.resultsPanel.SetData(msg)

	case db.TableInfoDataMsg:
		m.loading = false
		m.tableInfoPanel.SetData(msg)
		m.adjustSizes()

	case db.SchemaTablesMsg:
		m.loading = false
		m.tablePanel.SetData(msg)
		m.adjustSizes()

	case commands.ErrMsg:
		m.loading = false
		m.resultsPanel.SetLoading(false)
		m.errorMessage = msg.Error()
		m.errorPopup.SetText(m.errorMessage)

	case commands.ActivePanelChangedMsg:
		m.activePanelIndex = int(msg)
		m.tablePanel.SetActive(m.activePanelIndex == PanelIndexTables)
		m.tableInfoPanel.SetActive(m.activePanelIndex == PanelIndexTableInfo)
		m.queryPanel.SetActive(m.activePanelIndex == PanelIndexQuery)
		m.resultsPanel.SetActive(m.activePanelIndex == PanelIndexResults)
		if m.activePanelIndex == PanelIndexQuery {
			m.queryPanel, cmd = m.queryPanel.Update(msg)
			cmds = append(cmds, cmd)
		}

	case commands.TableSelectedMsg:
		switch m.tableInfoPanel.GetActiveTabIndex() {
		case component.TableInfoTabIndexColumns:
			cmd = commands.GetTableInfo(m.db, m.tablePanel.GetSelectedTable(), commands.TableInfoKind.Columns)
		case component.TableInfoTabIndexIndexes:
			cmd = commands.GetTableInfo(m.db, m.tablePanel.GetSelectedTable(), commands.TableInfoKind.Indexes)
		}
		cmds = append(cmds, cmd)

	case commands.TableInfoTabChangedMsg:
		switch msg {
		case component.TableInfoTabIndexColumns:
			cmd = commands.GetTableInfo(m.db, m.tablePanel.GetSelectedTable(), commands.TableInfoKind.Columns)
		case component.TableInfoTabIndexIndexes:
			cmd = commands.GetTableInfo(m.db, m.tablePanel.GetSelectedTable(), commands.TableInfoKind.Indexes)
		}
		cmds = append(cmds, cmd)

	case tea.MouseMsg:
		if tea.MouseEvent(msg).Button == tea.MouseButtonLeft {

			if isInBounds(msg.X, msg.Y, m.tablePanelBounds) {
				if m.activePanelIndex != PanelIndexTables {
					cmd = commands.SetActivePanel(PanelIndexTables)
					cmds = append(cmds, cmd)
				}
			} else if isInBounds(msg.X, msg.Y, m.tableInfoPanelBounds) {
				if m.activePanelIndex != PanelIndexTableInfo {
					cmd = commands.SetActivePanel(PanelIndexTableInfo)
					cmds = append(cmds, cmd)
				}
			} else if isInBounds(msg.X, msg.Y, m.queryPanelBounds) {
				if m.activePanelIndex != PanelIndexQuery {
					cmd = commands.SetActivePanel(PanelIndexQuery)
					cmds = append(cmds, cmd)
				}
			} else if isInBounds(msg.X, msg.Y, m.resultsPanelBounds) {
				if m.activePanelIndex != PanelIndexResults {
					cmd = commands.SetActivePanel(PanelIndexResults)
					cmds = append(cmds, cmd)
				}
			}

		}

	case tea.KeyMsg:

		if len(m.errorMessage) > 0 {
			// the error popup is shown, so any key should remove it
			m.errorMessage = ""
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.DefaultKeyMap.NextPanel):
			cmd = commands.SetActivePanel((m.activePanelIndex + 1) % m.selectablePanelCount)
			cmds = append(cmds, cmd)

		case key.Matches(msg, keys.DefaultKeyMap.PrevPanel):
			i := m.activePanelIndex - 1
			if i < 0 {
				i = m.selectablePanelCount - 1
			}
			cmd = commands.SetActivePanel(i)
			cmds = append(cmds, cmd)

		case key.Matches(msg, keys.DefaultKeyMap.ViewData):
			switch m.activePanelIndex {
			case PanelIndexTables:
				m.setLoading()
				cmd = commands.GetTableRows(m.db, m.tablePanel.GetSelectedTable())
				cmds = append(cmds, cmd)
			case PanelIndexResults:
				if !m.showResultRowPopup {
					m.resultRowPopup.SetData(m.resultsPanel.GetSelectedRow())
					m.showResultRowPopup = true
				}
			}

		case key.Matches(msg, keys.DefaultKeyMap.ExecuteQuery):
			if m.activePanelIndex == PanelIndexQuery {
				cmds = append(cmds, commands.SetLoading)
				m.setLoading()
				cmd = commands.ExecuteQuery(m.db, m.queryPanel.GetCurrentStatement())
				cmds = append(cmds, cmd)
			}

		case key.Matches(msg, keys.DefaultKeyMap.ToggleLeftPanel):
			m.leftPanelHidden = !m.leftPanelHidden
			if m.leftPanelHidden {
				m.selectablePanelCount = 2
				m.adjustSizes()
				if m.activePanelIndex < 2 {
					return m, commands.SetActivePanel(PanelIndexQuery)
				}
			} else {
				m.selectablePanelCount = 4
				m.adjustSizes()
			}

		case key.Matches(msg, keys.DefaultKeyMap.SaveQuery):
			if m.activePanelIndex == PanelIndexQuery {
				m.queryPanel.SetDirty(false)
				m.lastSavedQueryContents = m.queryPanel.GetValue()
				cmds = append(cmds, commands.SaveQueryFile(m.dbAlias, m.queryPanel.GetValue()))
			}

		case key.Matches(msg, keys.DefaultKeyMap.ReloadQuery):
			if m.activePanelIndex == PanelIndexQuery {
				m.queryPanel.SetDirty(false)
				cmds = append(cmds, commands.ReadOrCreateQueryFile(m.dbAlias))
			}

		case key.Matches(msg, keys.DefaultKeyMap.CloseResultRowPopup):
			m.showResultRowPopup = false

		case key.Matches(msg, keys.DefaultKeyMap.Help):
			m.help.ShowAll = true
			m.showHelpPopup = !m.showHelpPopup

		case key.Matches(msg, keys.DefaultKeyMap.Quit):
			return m, tea.Quit

		default:
			// any other key
			if m.activePanelIndex == PanelIndexQuery {
				if m.lastSavedQueryContents != m.queryPanel.GetValue() {
					m.queryPanel.SetDirty(true)
				}
			}
		}
	}

	// update components
	if m.showResultRowPopup {
		m.resultRowPopup, cmd = m.resultRowPopup.Update(msg)
		cmds = append(cmds, cmd)
		// skip other component updates if popup is shown
		return m, tea.Batch(cmds...)
	}
	if m.showHelpPopup {
		// skip other component updates if popup is shown
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

	if m.activePanelIndex == PanelIndexResults || m.loading {
		m.resultsPanel, cmd = m.resultsPanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) adjustSizes() {
	m.windowTooSmall = false

	availableHeight := m.height - TitleBarHeight - StatusBarHeight
	leftWidth := style.GetSpan(3, m.width)
	rightWidth := m.getRightWidth(m.width)

	//left
	tableInfoHeight := style.GetSpan(3, availableHeight)
	if tableInfoHeight < TableInfoPanelMinHeight {
		tableInfoHeight = TableInfoPanelMinHeight
	}
	tableHeight := availableHeight - tableInfoHeight - 4
	if tableHeight < TablePanelMinHeight {
		m.windowTooSmall = true
	}
	m.tableInfoPanel.SetSize(leftWidth, tableInfoHeight)
	m.tablePanel.SetSize(leftWidth, tableHeight)

	// right
	resultsHeight := style.GetSpan(6, availableHeight)
	if resultsHeight < ResultsPanelMinHeight {
		resultsHeight = ResultsPanelMinHeight
	}
	queryHeight := availableHeight - resultsHeight - 4
	if queryHeight < QueryPanelMinHeight {
		m.windowTooSmall = true
	}
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

	m.help.Width = m.width
}

func (m *model) setLoading() {
	m.errorMessage = ""
	m.errorPopup.SetText(m.errorMessage)
	m.resultsPanel.SetLoading(true)
	m.loading = true
}

func (m model) getRightWidth(totalWidth int) int {
	if m.leftPanelHidden {
		return style.GetSpan(12, totalWidth) - 6
	} else {
		return style.GetSpan(9, totalWidth) - 8
	}
}

func (m model) View() string {

	if m.windowTooSmall {
		return appStyle.Render("window too small")
	}

	left := lipgloss.JoinVertical(lipgloss.Center,
		m.tablePanel.View(),
		m.tableInfoPanel.View(),
	)
	if m.leftPanelHidden {
		left = ""
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
	if len(m.errorMessage) > 0 {
		p := m.errorPopup.View()
		x := m.width/2 - lipgloss.Width(p)/2
		y := m.height/2 - 2 - lipgloss.Height(p)/2
		contentView = style.PlaceOverlay(x, y, p, mainContent)
	}
	if m.showResultRowPopup {
		p := m.resultRowPopup.View()
		x := m.width/2 - lipgloss.Width(p)/2
		y := m.height/2 - 2 - lipgloss.Height(p)/2
		contentView = style.PlaceOverlay(x, y, p, mainContent)
	}
	if m.showHelpPopup {
		p := m.help.View(keys.DefaultKeyMap)
		x := m.width/2 - lipgloss.Width(p)/2
		y := m.height/2 - 2 - lipgloss.Height(p)/2
		helpStyle := style.BasePanelStyle.BorderForeground(colour.HelpBorder)
		contentView = style.PlaceOverlay(x, y, helpStyle.Render(p), mainContent)
	}

	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Center,
		m.titleBar.View(),
		contentView,
		m.statusBar.View(),
	))
}
