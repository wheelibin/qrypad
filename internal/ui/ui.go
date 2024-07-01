package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	TableInfoPanelMinHeight = 6
	TablePanelMinHeight     = 10
)

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
}

func NewModel(dbAlias string, db db.DBConn) model {
	tablePanel := component.NewTablePanelModel()
	tableInfoPanel := component.NewTableInfoPanelModel()
	queryPanel := component.NewQueryPanelModel(dbAlias)
	resultsPanel := component.NewResultsPanelModel()
	statusBar := component.NewStatusBarModel(dbAlias)
	titleBar := component.NewTitlBarModel()
	errorPopup := component.NewErrorPopupModel()

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
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// log.Println("ui.model::Update", msg)
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// update this now so the query text value is updated and can be used below
	m.queryPanel, cmd = m.queryPanel.Update(msg)
	cmds = append(cmds, cmd)

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
		cmd = commands.GetTableInfo(m.db, string(msg))
		cmds = append(cmds, cmd)

	case tea.KeyMsg:

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

		case key.Matches(msg, keys.DefaultKeyMap.ViewTableData):
			if m.activePanelIndex == PanelIndexTables {
				m.setLoading()
				cmd = commands.GetTableRows(m.db, m.tablePanel.GetSelectedTable())
				cmds = append(cmds, cmd)
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

		case key.Matches(msg, keys.DefaultKeyMap.Quit):
			return m, tea.Quit

		default:
			// any other key
			if len(m.errorMessage) > 0 {
				// the error popup is shown, so any key should remove it
				m.errorMessage = ""
				return m, nil
			}
			if m.activePanelIndex == PanelIndexQuery {
				if m.lastSavedQueryContents != m.queryPanel.GetValue() {
					m.queryPanel.SetDirty(true)
				}
			}
		}
	}

	// update components
	if m.activePanelIndex == PanelIndexTables {
		m.tablePanel, cmd = m.tablePanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.activePanelIndex == PanelIndexTableInfo {
		m.tableInfoPanel, cmd = m.tableInfoPanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	if m.activePanelIndex == PanelIndexResults {
		m.resultsPanel, cmd = m.resultsPanel.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) adjustSizes() {
	m.windowTooSmall = false

	availableHeight := m.height - TitleBarHeight - StatusBarHeight

	//left
	tableInfoHeight := style.GetSpan(3, availableHeight)
	if tableInfoHeight < TableInfoPanelMinHeight {
		tableInfoHeight = TableInfoPanelMinHeight
	}
	tableHeight := availableHeight - tableInfoHeight - 4
	if tableHeight < TablePanelMinHeight {
		m.windowTooSmall = true
	}
	m.tableInfoPanel.SetSize(style.GetSpan(3, m.width), tableInfoHeight)
	m.tablePanel.SetSize(style.GetSpan(3, m.width), tableHeight)

	// right
	resultsHeight := style.GetSpan(6, availableHeight)
	if resultsHeight < ResultsPanelMinHeight {
		resultsHeight = ResultsPanelMinHeight
	}
	queryHeight := availableHeight - resultsHeight - 4
	if queryHeight < QueryPanelMinHeight {
		m.windowTooSmall = true
	}
	m.resultsPanel.SetSize(m.getRightWidth(m.width), resultsHeight)
	m.queryPanel.SetSize(m.getRightWidth(m.width), queryHeight)

	m.statusBar.SetSize(m.width, StatusBarHeight)
	m.titleBar.SetSize(m.width, TitleBarHeight)
	m.errorPopup.SetSize(m.width/2, 5)
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

	finalView := mainContent
	if len(m.errorMessage) > 0 {
		ep := m.errorPopup.View()
		epX := m.width/2 - lipgloss.Width(ep)/2
		epY := m.height/2 - 2 - lipgloss.Height(ep)/2
		finalView = style.PlaceOverlay(epX, epY, m.errorPopup.View(), mainContent)
	}

	return appStyle.Render(lipgloss.JoinVertical(lipgloss.Center,
		m.titleBar.View(),
		finalView,
		m.statusBar.View(),
	))
}
