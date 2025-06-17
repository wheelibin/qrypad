package ui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/component"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/keys"
	"golang.design/x/clipboard"
)

func waitForResult(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m *model) handleDBMessages(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case db.DatabaseConnectedMsg:
		if m.db.DB != nil {
			m.db.DB.Close()
		}
		m.db = db.DBConn(msg)
		m.closePopup()
		m.statusBar.SetSelectedDatabase(m.db.ConnectedDatabase)
		switch m.tablePanel.GetActiveTabIndex() {
		case component.TablePanelTabIndexTables:
			return commands.GetSchemaEntities(m.db, commands.TablePanelKind.Tables)
		case component.TableInfoTabIndexIndexes:
			return commands.GetSchemaEntities(m.db, commands.TablePanelKind.Views)
		}

	case db.QueryControlMsg:
		m.cancelQuery = msg.Cancel
		return tea.Batch(
			commands.SetLoading(true),
			waitForResult(msg.ResultChan),
		)

	case db.DataFetchedMsg:
		m.resultsPanel.SetData(msg.Data)
		return commands.SetLoading(false)

	case db.TableInfoDataFetchedMsg:
		m.tableInfoPanel.SetData(msg.Data)
		m.adjustSizes()
		return commands.SetLoading(false)

	case db.SchemaEntitiesFetchedMsg:
		m.tablePanel.SetData(msg.Data)
		m.adjustSizes()
		return commands.SetLoading(false)

	case db.DatabaseListFetchedMsg:
		m.databaseSwitcherPopup.SetData(msg.Data)
		m.adjustSizes()
		return commands.SetLoading(false)

	}

	return nil
}

func (m *model) handleErrorMessages(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case commands.ErrMsg:
		m.errorMessage = msg.Error()
		m.errorPopup.SetText(m.errorMessage)
		m.showPopup(PopupKind.Error)
		return commands.SetLoading(false)

	case commands.DatabaseConnectErrMsg:
		m.closePopup()
		m.errorMessage = msg.Error()
		m.errorPopup.SetText(m.errorMessage)
		m.errorPopup.SetIsConnectionError(true)
		m.showPopup(PopupKind.Error)
		return commands.SetLoading(false)

	}

	return nil
}

func (m *model) handleCommandMessages(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case commands.ActivePanelChangedMsg:
		m.activePanelIndex = int(msg)
		m.setPanelsActiveState(m.activePanelIndex)
		if m.activePanelIndex == PanelIndexQuery {
			var cmd tea.Cmd
			m.queryPanel, cmd = m.queryPanel.Update(msg)
			return cmd
		}

	case commands.TableSelectedMsg:
		switch m.tableInfoPanel.GetActiveTabIndex() {
		case component.TableInfoTabIndexColumns:
			return commands.GetTableInfo(m.db, m.tablePanel.GetSelectedTable(), commands.TableInfoKind.Columns)
		case component.TableInfoTabIndexIndexes:
			return commands.GetTableInfo(m.db, m.tablePanel.GetSelectedTable(), commands.TableInfoKind.Indexes)
		}

	case commands.TablePanelTabChangedMsg:
		switch msg {
		case component.TablePanelTabIndexTables:
			return commands.GetSchemaEntities(m.db, commands.TablePanelKind.Tables)
		case component.TableInfoTabIndexIndexes:
			return commands.GetSchemaEntities(m.db, commands.TablePanelKind.Views)
		}

	case commands.TableInfoTabChangedMsg:
		switch msg {
		case component.TableInfoTabIndexColumns:
			return commands.GetTableInfo(m.db, m.tablePanel.GetSelectedTable(), commands.TableInfoKind.Columns)
		case component.TableInfoTabIndexIndexes:
			return commands.GetTableInfo(m.db, m.tablePanel.GetSelectedTable(), commands.TableInfoKind.Indexes)
		}

	case commands.DatabaseSelectedMsg:
		m.selectedDatabase = string(msg)
		m.dbConfig.Database = string(msg)
		return commands.ConnectToDB(m.connectionName, m.dbConfig)

	case commands.PasswordEnteredMsg:
		return commands.SavePassword(m.connectionName, string(msg))

	case commands.PasswordSavedMsg:
		return commands.ConnectToDB(m.connectionName, m.dbConfig)

	case commands.PopupClosedMsg:
		m.closePopup()

	case commands.PasswordInputNeededMsg:
		m.passwordPopup.Clear()
		m.showPopup(PopupKind.Password)

	case commands.CopyValueMsg:
		if msg.ValueDesc != "" {
			m.statusBar.SetCopiedTextInfo(msg.ValueDesc)
		} else {
			m.statusBar.SetCopiedTextInfo(msg.Value)
		}

		clipboard.Write(clipboard.FmtText, []byte(msg.Value))

	case commands.QueryFileReadMsg:
		m.queryPanel.SetValue(msg.Contents)
		m.queryPanel.SetFilename(msg.FileName)
		m.lastSavedQueryContents = msg.Contents
	}

	return nil
}

func (m *model) handleMouseMessages(msg tea.MouseMsg) tea.Cmd {
	if tea.MouseEvent(msg).Button == tea.MouseButtonLeft {
		if isInBounds(msg.X, msg.Y, m.tablePanelBounds) {
			if m.activePanelIndex != PanelIndexTables {
				return commands.SetActivePanel(PanelIndexTables)
			}
		} else if isInBounds(msg.X, msg.Y, m.tableInfoPanelBounds) {
			if m.activePanelIndex != PanelIndexTableInfo {
				return commands.SetActivePanel(PanelIndexTableInfo)
			}
		} else if isInBounds(msg.X, msg.Y, m.queryPanelBounds) {
			if m.activePanelIndex != PanelIndexQuery {
				return commands.SetActivePanel(PanelIndexQuery)
			}
		} else if isInBounds(msg.X, msg.Y, m.resultsPanelBounds) {
			if m.activePanelIndex != PanelIndexResults {
				return commands.SetActivePanel(PanelIndexResults)
			}
		}
	}

	return nil
}

func (m *model) handleKeyMessages(msg tea.KeyMsg) tea.Cmd {
	if key.Matches(msg, keys.DefaultKeyMap.Quit) {
		if m.db.DB != nil {
			m.db.DB.Close()
		}
		return tea.Quit
	}

	// popups handle their own key events and emit messages
	if m.hasActivePopup() {
		return nil
	}

	switch {

	case key.Matches(msg, keys.DefaultKeyMap.CancelQuery):
		m.cancelQuery()

	case key.Matches(msg, keys.DefaultKeyMap.NextPanel):
		return commands.SetActivePanel((m.activePanelIndex + 1) % m.selectablePanelCount)

	case key.Matches(msg, keys.DefaultKeyMap.PrevPanel):
		i := m.activePanelIndex - 1
		if i < 0 {
			i = m.selectablePanelCount - 1
		}
		return commands.SetActivePanel(i)

	case key.Matches(msg, keys.DefaultKeyMap.ViewData):
		switch m.activePanelIndex {
		case PanelIndexTables:
			return commands.GetTableRows(m.db, m.tablePanel.GetSelectedTable())
		case PanelIndexResults:
			if !m.popupIsActive(PopupKind.ResultRow) {
				m.resultRowPopup.SetData(m.resultsPanel.GetSelectedRow())
				m.showPopup(PopupKind.ResultRow)
			}
		case PanelIndexTableInfo:
			if !m.popupIsActive(PopupKind.ResultRow) {
				m.resultRowPopup.SetData(m.tableInfoPanel.GetSelectedRow())
				m.showPopup(PopupKind.ResultRow)
			}
		}

	case key.Matches(msg, keys.DefaultKeyMap.ExecuteQuery):
		if m.activePanelIndex == PanelIndexQuery {
			return commands.ExecuteQuery(m.db, m.queryPanel.GetCurrentStatement(), commands.QueryResultBuilder)
		}

	case key.Matches(msg, keys.DefaultKeyMap.ToggleLeftPanel):
		m.leftPanelHidden = !m.leftPanelHidden
		if m.leftPanelHidden {
			m.selectablePanelCount = 2
			m.adjustSizes()
			if m.activePanelIndex < 2 {
				return commands.SetActivePanel(PanelIndexQuery)
			}
		} else {
			m.selectablePanelCount = 4
			m.adjustSizes()
		}

	case key.Matches(msg, keys.DefaultKeyMap.SaveQuery):
		if m.activePanelIndex == PanelIndexQuery {
			m.queryPanel.SetDirty(false)
			m.lastSavedQueryContents = m.queryPanel.GetValue()
			return commands.SaveQueryFile(m.connectionName, m.queryPanel.GetValue())
		}

	case key.Matches(msg, keys.DefaultKeyMap.ReloadQuery):
		if m.activePanelIndex == PanelIndexQuery {
			m.queryPanel.SetDirty(false)
			return commands.ReadOrCreateQueryFile(m.connectionName)
		}

	case key.Matches(msg, keys.DefaultKeyMap.ClosePopup):
		m.closePopup()

	case key.Matches(msg, keys.DefaultKeyMap.Help):
		if !m.popupIsActive(PopupKind.Help) {
			m.showPopup(PopupKind.Help)
		} else {
			m.closePopup()
		}

	case key.Matches(msg, keys.DefaultKeyMap.SwitchDatabase):
		if !m.popupIsActive(PopupKind.DatabaseSwitcher) {
			m.showPopup(PopupKind.DatabaseSwitcher)
			return commands.GetDatabases(m.db)
		}

	case key.Matches(msg, keys.DefaultKeyMap.OpenInEditor):
		return commands.OpenEditor(m.queryPanel.GetFilename())

	case key.Matches(msg, keys.DefaultKeyMap.CopyValue):
		var valueToCopy, copiedTextInfo string
		switch m.activePanelIndex {
		case PanelIndexTables:
			valueToCopy = m.tablePanel.GetSelectedTable()
		case PanelIndexTableInfo:
			valueToCopy = m.tableInfoPanel.GetSelectedRow()["name"].(string)
		case PanelIndexResults:
			valueToCopy = m.resultsPanel.GetSelectedRowJSON()
			copiedTextInfo = "<row as json>"
		}
		if valueToCopy != "" {
			return commands.CopyValue(valueToCopy, copiedTextInfo)
		}

	case key.Matches(msg, keys.DefaultKeyMap.UpdatePassword):
		m.showPopup(PopupKind.Password)

	default:
		// any other key
		if m.activePanelIndex == PanelIndexQuery {
			if m.lastSavedQueryContents != m.queryPanel.GetValue() {
				m.queryPanel.SetDirty(true)
			}
		}
	}

	return nil
}
