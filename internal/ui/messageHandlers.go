package ui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/wheelibin/qrypad/internal/autocomplete"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/component"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/keys"
)

var (
	debouncedMsgs = make(chan tea.Msg, 10)                                     //nolint:gochecknoglobals // debouncer requires global channel
	debouncer     = commands.NewDebouncer(500*time.Millisecond, debouncedMsgs) //nolint:gochecknoglobals // debouncer is a package-level singleton
)

func waitForResult(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m *model) handleError(err error) {
	if errors.Is(err, context.Canceled) {
		return
	}
	m.errorMessage = err.Error()
	m.errorPopup.SetText(m.errorMessage)
	m.showPopup(PopupKind.Error)
}

func (m *model) handleDBMessages(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case db.DatabaseConnectedMsg:
		if m.db.DB != nil {
			_ = m.db.DB.Close()
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
		if msg.Err != nil {
			m.handleError(msg.Err)
		} else {
			m.resultsPanel.SetData(msg.Data)
		}
		return commands.SetLoading(false)

	case db.TableInfoDataFetchedMsg:
		if msg.Err != nil {
			m.handleError(msg.Err)
		} else {
			m.tableInfoPanel.SetData(msg.Data)
			m.adjustSizes()
		}
		return commands.SetLoading(false)

	case db.SchemaEntitiesFetchedMsg:
		if msg.Err != nil {
			m.handleError(msg.Err)
		} else {
			m.tablePanel.SetData(msg.Data)
			m.adjustSizes()
		}
		return commands.SetLoading(false)

	case db.DatabaseListFetchedMsg:
		if msg.Err != nil {
			m.handleError(msg.Err)
		} else {
			m.databaseSwitcherPopup.SetData(msg.Data)
			m.adjustSizes()
		}
		return commands.SetLoading(false)

	case db.AutoCompleteDataFetchedMsg:
		if len(msg) > 0 {
			m.queryPanel.SetAutoCompleteActive(true)
			return m.queryPanel.SetAutoCompleteOptions(msg)
		}
	}

	return nil
}

func (m *model) handleErrorMessages(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case commands.ErrMsg:
		m.handleError(msg.Err)
		return commands.SetLoading(false)

	case commands.DatabaseConnectError:
		m.errorPopup.SetIsConnectionError(true)
		m.handleError(msg.Err)
		return commands.SetLoading(false)
	}

	return nil
}

func (m *model) handleCommandMessages(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case commands.LoadingMsg:
		if m.popupIsActive(PopupKind.DatabaseSwitcher) {
			return nil
		}
		if msg.Loading && !m.popupIsActive(PopupKind.Error) {
			m.showPopup(PopupKind.Loading)
		} else if !m.popupIsActive(PopupKind.Error) {
			// loading finished
			m.closePopup()
		}

	case commands.CancelQueryMsg:
		if m.cancelQuery != nil {
			m.cancelQuery()
		}

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
		case component.TableInfoTabIndexConstraints:
			return commands.GetTableInfo(m.db, m.tablePanel.GetSelectedTable(), commands.TableInfoKind.Constraints)
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
		case component.TableInfoTabIndexConstraints:
			return commands.GetTableInfo(m.db, m.tablePanel.GetSelectedTable(), commands.TableInfoKind.Constraints)
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

		_ = clipboard.WriteAll(msg.Value)

	case commands.QueryFileReadMsg:
		m.queryPanel.SetValue(msg.Contents)
		m.queryPanel.SetFilename(msg.FileName)
		m.lastSavedQueryContents = msg.Contents
	}

	return nil
}

func (m *model) handleMouseMessages(msg tea.MouseMsg) tea.Cmd {
	if tea.MouseEvent(msg).Button == tea.MouseButtonLeft {
		switch {
		case isInBounds(msg.X, msg.Y, m.tablePanelBounds):
			if m.activePanelIndex != PanelIndexTables {
				return commands.SetActivePanel(PanelIndexTables)
			}
		case isInBounds(msg.X, msg.Y, m.tableInfoPanelBounds):
			if m.activePanelIndex != PanelIndexTableInfo {
				return commands.SetActivePanel(PanelIndexTableInfo)
			}
		case isInBounds(msg.X, msg.Y, m.queryPanelBounds):
			if m.activePanelIndex != PanelIndexQuery {
				return commands.SetActivePanel(PanelIndexQuery)
			}
		case isInBounds(msg.X, msg.Y, m.resultsPanelBounds):
			if m.activePanelIndex != PanelIndexResults {
				return commands.SetActivePanel(PanelIndexResults)
			}
		}
	}

	return nil
}

//nolint:gocyclo,cyclop // Bubble Tea key handler inherently requires complex switch statements
func (m *model) handleKeyMessages(msg tea.KeyMsg) tea.Cmd {
	if key.Matches(msg, keys.DefaultKeyMap.Quit) {
		if m.db.DB != nil {
			_ = m.db.DB.Close()
		}
		return tea.Quit
	}

	// popups handle their own key events and emit messages
	if m.hasActivePopup() {
		return nil
	}

	switch {
	case key.Matches(msg, keys.DefaultKeyMap.NextPanel):
		nextPanelIndex := (m.activePanelIndex + 1) % m.selectablePanelCount
		if m.leftPanelHidden {
			return commands.SetActivePanel(nextPanelIndex + 2)
		}
		return commands.SetActivePanel(nextPanelIndex)

	case key.Matches(msg, keys.DefaultKeyMap.PrevPanel):
		prevPanelIndex := (m.activePanelIndex - 1 + m.selectablePanelCount) % m.selectablePanelCount
		if m.leftPanelHidden {
			return commands.SetActivePanel(prevPanelIndex + 2)
		}
		return commands.SetActivePanel(prevPanelIndex)

	case key.Matches(msg, keys.DefaultKeyMap.ViewData):
		switch m.activePanelIndex {
		case PanelIndexTables:
			if m.tablePanel.GetSelectedTable() != "" {
				return commands.GetTableRows(m.db, m.tablePanel.GetSelectedTable(), "asc")
			}
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

	case key.Matches(msg, keys.DefaultKeyMap.ViewDataDesc):
		if m.activePanelIndex == PanelIndexTables {
			if m.tablePanel.GetSelectedTable() != "" {
				return commands.GetTableRows(m.db, m.tablePanel.GetSelectedTable(), "desc")
			}
		}

	case key.Matches(msg, keys.DefaultKeyMap.ExecuteQuery):
		if m.activePanelIndex == PanelIndexQuery {
			if sql := m.queryPanel.GetCurrentStatement(); sql != "" {
				return commands.ExecuteQuery(m.db, sql, commands.QueryResultBuilder)
			}
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
			if m.lastSavedQueryContents == m.queryPanel.GetValue() {
				return nil
			}

			m.queryPanel.SetDirty(false)
			m.lastSavedQueryContents = m.queryPanel.GetValue()
			debouncer.Trigger("save-query", commands.SaveQueryFile(m.connectionName, m.queryPanel.GetValue()))
			return waitForResult(debouncedMsgs)
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
		if m.activePanelIndex == PanelIndexQuery {
			return commands.OpenEditor(m.queryPanel.GetFilename())
		}

	case key.Matches(msg, keys.DefaultKeyMap.CopyValue):
		var valueToCopy, copiedTextInfo string
		switch m.activePanelIndex {
		case PanelIndexTables:
			valueToCopy = m.tablePanel.GetSelectedTable()
		case PanelIndexTableInfo:
			row := m.tableInfoPanel.GetSelectedRow()
			if name, ok := row["name"]; ok {
				valueToCopy = fmt.Sprintf("%v", name)
			}
		case PanelIndexResults:
			valueToCopy = m.resultsPanel.GetSelectedRowJSON()
			copiedTextInfo = "<row as json>"
		}
		if valueToCopy != "" {
			return commands.CopyValue(valueToCopy, copiedTextInfo)
		}

	case key.Matches(msg, keys.DefaultKeyMap.UpdatePassword):
		m.showPopup(PopupKind.Password)

	case key.Matches(msg, key.NewBinding(key.WithKeys("."))):
		if m.activePanelIndex == PanelIndexQuery {
			result := autocomplete.GetCompletions(
				m.queryPanel.GetCurrentStatement(),
				m.queryPanel.GetWordAtCursor(),
				m.tablePanel.GetAllTableNames(),
			)
			return m.handleCompletionResult(result)
		}

	case key.Matches(msg, keys.DefaultKeyMap.AutoComplete):
		if m.activePanelIndex == PanelIndexQuery {
			result := autocomplete.GetCompletionsForced(
				m.queryPanel.GetCurrentStatement(),
				m.queryPanel.GetWordAtCursor(),
				m.tablePanel.GetAllTableNames(),
			)
			return m.handleCompletionResult(result)
		}

	default:
		// any other key
		if m.activePanelIndex == PanelIndexQuery {
			if m.queryPanel.GetAutoCompleteActive() {
				key := msg.String()
				if !isAutoCompleteKey(key) {
					if isPrintableKey(key) {
						m.queryPanel.AutoCompleteFilterAppend(key)
					} else if key == "backspace" {
						m.queryPanel.AutoCompleteFilterPop()
					}
				}
			}

			// Check for table name completion after space (e.g. "FROM ", "JOIN ")
			if msg.String() == " " && !m.queryPanel.GetAutoCompleteActive() {
				result := autocomplete.GetCompletions(
					m.queryPanel.GetCurrentStatement(),
					m.queryPanel.GetWordAtCursor(),
					m.tablePanel.GetAllTableNames(),
				)
				if result.Kind == autocomplete.CompletionTable {
					return m.handleCompletionResult(result)
				}
			}

			// buffer file operations
			if m.lastSavedQueryContents != m.queryPanel.GetValue() {
				// buffer has changed
				if m.autoSave {
					m.queryPanel.SetDirty(false)
					m.lastSavedQueryContents = m.queryPanel.GetValue()
					debouncer.Trigger("save-query", commands.SaveQueryFile(m.connectionName, m.queryPanel.GetValue()))
					return waitForResult(debouncedMsgs)
				}
				m.queryPanel.SetDirty(true)
			}
		}
	}

	return nil
}

func (m *model) handleCompletionResult(result autocomplete.CompletionResult) tea.Cmd {
	switch result.Kind {
	case autocomplete.CompletionColumn:
		return commands.GetAutocompleteData(m.db, result.TableName)
	case autocomplete.CompletionTable:
		m.queryPanel.SetAutoCompleteActive(true)
		return m.queryPanel.SetAutoCompleteOptions(result.Items)
	case autocomplete.CompletionNone:
		// nothing to complete
	}
	return nil
}

func isAutoCompleteKey(key string) bool {
	acKeys := []string{"up", "down", "esc", "enter"}
	return slices.Contains(acKeys, key)
}

func isPrintableKey(key string) bool {
	return len(key) == 1 && strconv.IsPrint(rune(key[0]))
}
