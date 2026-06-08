package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"

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
		m.schemaCache.Invalidate()
		m.closePopup()
		m.statusBar.SetSelectedDatabase(m.db.ConnectedDatabase)

		var cmds []tea.Cmd

		// If selectedDatabase was empty (not specified in config), this is the
		// first time we know the actual database name. Load the query file now.
		if m.selectedDatabase == "" && m.db.ConnectedDatabase != "" {
			m.selectedDatabase = m.db.ConnectedDatabase
			m.queryPanel.SetDatabaseName(m.db.ConnectedDatabase)
			cmds = append(cmds, commands.ReadOrCreateQueryFile(
				m.connectionName, m.selectedDatabase, m.dbConfig.UseSingleQueryFile(),
			))
		}

		switch m.tablePanel.GetActiveTabIndex() {
		case component.TablePanelTabIndexTables:
			cmds = append(cmds, commands.GetSchemaEntities(m.db, commands.TablePanelKind.Tables, m.schemaCache))
		case component.TableInfoTabIndexIndexes:
			cmds = append(cmds, commands.GetSchemaEntities(m.db, commands.TablePanelKind.Views, m.schemaCache))
		}

		if len(cmds) > 0 {
			return tea.Batch(cmds...)
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

			// Trigger eager column preload if table count is within threshold
			refs := m.tablePanel.GetAllTableRefs()
			if len(refs) > 0 && len(refs) <= commands.PreloadTableThreshold {
				m.schemaPreloading = true
				m.statusBar.SetStatusInfo("loading schema...")
				return tea.Batch(
					commands.SetLoading(false),
					commands.PreloadColumns(m.db, refs, m.schemaCache),
				)
			}
		}
		return commands.SetLoading(false)

	case commands.SchemaPreloadCompleteMsg:
		m.schemaPreloading = false
		m.statusBar.SetStatusInfo("")
		return nil

	case commands.SchemaPreloadErrorMsg:
		m.schemaPreloading = false
		m.statusBar.SetStatusInfo("")
		return nil

	case db.DatabaseListFetchedMsg:
		if msg.Err != nil {
			m.handleError(msg.Err)
		} else {
			m.databaseSwitcherPopup.SetData(msg.Data)
			m.adjustSizes()
		}
		return commands.SetLoading(false)

	case db.ConnectionListFetchedMsg:
		if msg.Err != nil {
			m.handleError(msg.Err)
		} else {
			m.connectionSwitcherPopup.SetData(msg.Data)
			m.adjustSizes()
		}

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
	if cmd := m.handlePopupMessages(msg); cmd != nil {
		return cmd
	}
	if cmd := m.handleNavigationMessages(msg); cmd != nil {
		return cmd
	}
	if cmd := m.handleQueryMessages(msg); cmd != nil {
		return cmd
	}
	return nil
}

func (m *model) handlePopupMessages(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case commands.LoadingMsg:
		if m.popupIsActive(PopupKind.DatabaseSwitcher) || m.popupIsActive(PopupKind.ConnectionSwitcher) {
			return nil
		}
		if msg.Loading && !m.popupIsActive(PopupKind.Error) {
			m.showPopup(PopupKind.Loading)
		} else if !m.popupIsActive(PopupKind.Error) {
			// loading finished
			m.closePopup()
		}
		return nil

	case commands.PopupClosedMsg:
		m.closePopup()
		return nil

	case commands.NoConnectionChosenMsg:
		return tea.Quit

	case commands.PasswordInputNeededMsg:
		m.passwordPopup.Clear()
		m.showPopup(PopupKind.Password)
		return nil

	case commands.CopyValueMsg:
		if msg.ValueDesc != "" {
			m.statusBar.SetCopiedTextInfo(msg.ValueDesc)
		} else {
			m.statusBar.SetCopiedTextInfo(msg.Value)
		}
		_ = clipboard.WriteAll(msg.Value)
		return nil

	case commands.ExportCompletedMsg:
		m.statusBar.SetStatusInfo(fmt.Sprintf("exported %d rows to %s", msg.RowCount, filepath.Base(msg.Path)))
		return nil
	}

	return nil
}

func (m *model) handleNavigationMessages(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case commands.CancelQueryMsg:
		if m.cancelQuery != nil {
			m.cancelQuery()
		}
		return nil

	case commands.ActivePanelChangedMsg:
		m.activePanelIndex = int(msg)
		m.setPanelsActiveState(m.activePanelIndex)
		if m.activePanelIndex == PanelIndexQuery {
			var cmd tea.Cmd
			m.queryPanel, cmd = m.queryPanel.Update(msg)
			return cmd
		}
		return nil

	case commands.TablePanelTabChangedMsg:
		switch msg {
		case component.TablePanelTabIndexTables:
			return commands.GetSchemaEntities(m.db, commands.TablePanelKind.Tables, m.schemaCache)
		case component.TableInfoTabIndexIndexes:
			return commands.GetSchemaEntities(m.db, commands.TablePanelKind.Views, m.schemaCache)
		}
		return nil

	case commands.TableInfoTabChangedMsg:
		switch msg {
		case component.TableInfoTabIndexColumns:
			return commands.GetTableInfo(m.db, m.currentTableRef, commands.TableInfoKind.Columns, m.schemaCache)
		case component.TableInfoTabIndexIndexes:
			return commands.GetTableInfo(m.db, m.currentTableRef, commands.TableInfoKind.Indexes, m.schemaCache)
		case component.TableInfoTabIndexConstraints:
			return commands.GetTableInfo(m.db, m.currentTableRef, commands.TableInfoKind.Constraints, m.schemaCache)
		}
		return nil

	case commands.TableSelectedMsg:
		m.currentTableRef = db.TableReference(msg)
		switch m.tableInfoPanel.GetActiveTabIndex() {
		case component.TableInfoTabIndexColumns:
			return commands.GetTableInfo(m.db, m.currentTableRef, commands.TableInfoKind.Columns, m.schemaCache)
		case component.TableInfoTabIndexIndexes:
			return commands.GetTableInfo(m.db, m.currentTableRef, commands.TableInfoKind.Indexes, m.schemaCache)
		case component.TableInfoTabIndexConstraints:
			return commands.GetTableInfo(m.db, m.currentTableRef, commands.TableInfoKind.Constraints, m.schemaCache)
		}
		return nil
	}

	return nil
}

func (m *model) handleQueryMessages(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case commands.DatabaseSelectedMsg:
		newDB := string(msg)

		// Save current buffer before switching (if in per-database mode)
		if !m.dbConfig.UseSingleQueryFile() {
			if m.queryPanel.GetValue() != m.lastSavedQueryContents {
				if saveErr := commands.SaveQueryFileToDisk(
					m.connectionName, m.selectedDatabase, m.queryPanel.GetValue(), false,
				); saveErr != nil {
					m.handleError(saveErr)
					return nil
				}
				m.lastSavedQueryContents = m.queryPanel.GetValue()
				m.queryPanel.SetDirty(false)
			}
		}

		m.selectedDatabase = newDB
		m.dbConfig.Database = newDB
		m.titleBar.SetConn(m.dbConfig)
		m.queryPanel.SetDatabaseName(newDB)

		cmds := []tea.Cmd{commands.ConnectToDB(m.connectionName, m.dbConfig)}

		// Load new database's query file (if in per-database mode)
		if !m.dbConfig.UseSingleQueryFile() {
			cmds = append(cmds, commands.ReadOrCreateQueryFile(
				m.connectionName, newDB, false,
			))
		}

		return tea.Batch(cmds...)

	case commands.ConnectionSelectedMsg:
		connName := string(msg)
		conns, err := db.GetConnections()
		if err != nil {
			m.handleError(err)
			return nil
		}
		conn, ok := conns[connName]
		if !ok {
			m.handleError(errors.New("connection not found"))
			return nil
		}

		// Save current buffer to old connection's file before switching (bypass debouncer)
		if m.queryPanel.GetValue() != m.lastSavedQueryContents {
			if saveErr := commands.SaveQueryFileToDisk(
				m.connectionName, m.selectedDatabase, m.queryPanel.GetValue(), m.dbConfig.UseSingleQueryFile(),
			); saveErr != nil {
				m.handleError(saveErr)
				return nil
			}
			m.lastSavedQueryContents = m.queryPanel.GetValue()
			m.queryPanel.SetDirty(false)
		}

		m.queryPanel.SetConnectionName(connName)
		m.queryPanel.SetDatabaseName(conn.Database)
		m.queryPanel.SetSingleQueryFile(conn.UseSingleQueryFile())
		m.connectionName = connName
		m.dbConfig = conn
		m.selectedDatabase = conn.Database
		m.titleBar.SetConnectionName(connName)
		m.titleBar.SetConn(conn)
		// re-evaluate driver-dependent key bindings for the new connection
		m.initKeyMap()
		return tea.Batch(
			commands.ConnectToDB(m.connectionName, m.dbConfig),
			commands.ReadOrCreateQueryFile(connName, conn.Database, conn.UseSingleQueryFile()),
		)

	case commands.PasswordEnteredMsg:
		return commands.SavePassword(m.connectionName, string(msg))

	case commands.PasswordSavedMsg:
		return commands.ConnectToDB(m.connectionName, m.dbConfig)

	case commands.QueryFileReadMsg:
		m.queryPanel.SetValue(msg.Contents)
		m.queryPanel.SetFilename(msg.FileName)
		m.lastSavedQueryContents = msg.Contents
		m.queryPanel.SetDirty(false)
		return nil

	case commands.ExportRequestedMsg:
		rows := m.resultsPanel.GetExportRows()
		cols := m.resultsPanel.GetColumns()
		cwd, err := os.Getwd()
		if err != nil {
			m.handleError(err)
			return nil
		}
		m.closePopup()
		return commands.ExportResults(rows, cols, msg.Format, cwd)
	}

	return nil
}

func (m *model) handleMouseMessages(msg tea.MouseClickMsg) tea.Cmd {
	if msg.Button == tea.MouseLeft {
		mouse := msg.Mouse()
		switch {
		case isInBounds(mouse.X, mouse.Y, m.tablePanelBounds):
			if m.activePanelIndex != PanelIndexTables {
				return commands.SetActivePanel(PanelIndexTables)
			}
		case isInBounds(mouse.X, mouse.Y, m.tableInfoPanelBounds):
			if m.activePanelIndex != PanelIndexTableInfo {
				return commands.SetActivePanel(PanelIndexTableInfo)
			}
		case isInBounds(mouse.X, mouse.Y, m.queryPanelBounds):
			if m.activePanelIndex != PanelIndexQuery {
				return commands.SetActivePanel(PanelIndexQuery)
			}
		case isInBounds(mouse.X, mouse.Y, m.resultsPanelBounds):
			if m.activePanelIndex != PanelIndexResults {
				return commands.SetActivePanel(PanelIndexResults)
			}
		}
	}

	return nil
}

//nolint:gocyclo,cyclop // Bubble Tea key handler inherently requires complex switch statements
func (m *model) handleKeyMessages(msg tea.KeyPressMsg) tea.Cmd {
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
			if m.tablePanel.GetSelectedTable().Name != "" {
				return commands.GetTableRows(m.db, m.tablePanel.GetSelectedTable(), "asc")
			}
		case PanelIndexResults:
			if !m.popupIsActive(PopupKind.ResultRow) {
				m.resultRowPopup.SetData(
					m.resultsPanel.GetSelectedRow(),
					m.resultsPanel.GetColumns(),
					m.resultsPanel.GetColumnTypes(),
				)
				m.showPopup(PopupKind.ResultRow)
			}
		case PanelIndexTableInfo:
			if !m.popupIsActive(PopupKind.ResultRow) {
				row := m.tableInfoPanel.GetSelectedRow()
				// Table info panel has no column types — pass nil
				m.resultRowPopup.SetData(row, nil, nil)
				m.showPopup(PopupKind.ResultRow)
			}
		}

	case key.Matches(msg, keys.DefaultKeyMap.ViewDataDesc):
		if m.activePanelIndex == PanelIndexTables {
			if m.tablePanel.GetSelectedTable().Name != "" {
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
			debouncer.Trigger("save-query", commands.SaveQueryFile(
				m.connectionName, m.selectedDatabase, m.queryPanel.GetValue(), m.dbConfig.UseSingleQueryFile(),
			))
			return waitForResult(debouncedMsgs)
		}

	case key.Matches(msg, keys.DefaultKeyMap.ReloadQuery):
		if m.activePanelIndex == PanelIndexQuery {
			m.queryPanel.SetDirty(false)
			return commands.ReadOrCreateQueryFile(m.connectionName, m.selectedDatabase, m.dbConfig.UseSingleQueryFile())
		}

	case key.Matches(msg, keys.DefaultKeyMap.ClosePopup):
		m.closePopup()

	case key.Matches(msg, keys.DefaultKeyMap.Help):
		// When the query panel is focused, "?" is a valid SQL character
		// (parameter placeholder / JSONB operator) so only treat it as help
		// from other panels. "f1" always opens help regardless of focus.
		if m.activePanelIndex == PanelIndexQuery && msg.String() == "?" {
			break
		}
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

	case key.Matches(msg, keys.DefaultKeyMap.SwitchConnection):
		if !m.popupIsActive(PopupKind.ConnectionSwitcher) {
			m.connectionSwitcherPopup.SetIsFirstConnection(false)
			m.showPopup(PopupKind.ConnectionSwitcher)
			return commands.GetConnectionList()
		}

	case key.Matches(msg, keys.DefaultKeyMap.OpenInEditor):
		if m.activePanelIndex == PanelIndexQuery {
			return commands.OpenEditor(m.queryPanel.GetFilename())
		}

	case key.Matches(msg, keys.DefaultKeyMap.CopyValue):
		var valueToCopy, copiedTextInfo string
		switch m.activePanelIndex {
		case PanelIndexTables:
			valueToCopy = m.tablePanel.GetSelectedTable().QualifiedName()
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

	case key.Matches(msg, keys.DefaultKeyMap.ExportResults):
		if len(m.resultsPanel.GetExportRows()) == 0 {
			m.statusBar.SetStatusInfo("nothing to export")
			break
		}
		m.showPopup(PopupKind.ExportFormat)

	case key.Matches(msg, keys.DefaultKeyMap.UpdatePassword):
		m.showPopup(PopupKind.Password)

	case key.Matches(msg, key.NewBinding(key.WithKeys("."))):
		if m.activePanelIndex == PanelIndexQuery {
			result := autocomplete.GetCompletions(
				m.queryPanel.GetCurrentStatement(),
				m.queryPanel.GetWordAtCursor(),
				m.tablePanel.GetAllTableRefs(),
				m.db,
			)
			return m.handleCompletionResult(result)
		}

	case key.Matches(msg, keys.DefaultKeyMap.AutoComplete):
		if m.activePanelIndex == PanelIndexQuery {
			result := autocomplete.GetCompletionsForced(
				m.queryPanel.GetCurrentStatement(),
				m.queryPanel.GetWordAtCursor(),
				m.tablePanel.GetAllTableRefs(),
				m.db,
			)
			return m.handleCompletionResult(result)
		}

	case key.Matches(msg, keys.DefaultKeyMap.RefreshSchema):
		if m.activePanelIndex == PanelIndexTables || m.activePanelIndex == PanelIndexTableInfo {
			m.schemaCache.Invalidate()
			m.statusBar.SetStatusInfo("schema refreshed")
			var cmds []tea.Cmd
			switch m.tablePanel.GetActiveTabIndex() {
			case component.TablePanelTabIndexTables:
				cmds = append(cmds, commands.GetSchemaEntities(m.db, commands.TablePanelKind.Tables, m.schemaCache))
			case component.TablePanelTabIndexViews:
				cmds = append(cmds, commands.GetSchemaEntities(m.db, commands.TablePanelKind.Views, m.schemaCache))
			}
			if m.currentTableRef.Name != "" {
				switch m.tableInfoPanel.GetActiveTabIndex() {
				case component.TableInfoTabIndexColumns:
					cmds = append(cmds, commands.GetTableInfo(m.db, m.currentTableRef, commands.TableInfoKind.Columns, m.schemaCache))
				case component.TableInfoTabIndexIndexes:
					cmds = append(cmds, commands.GetTableInfo(m.db, m.currentTableRef, commands.TableInfoKind.Indexes, m.schemaCache))
				case component.TableInfoTabIndexConstraints:
					cmds = append(cmds, commands.GetTableInfo(m.db, m.currentTableRef, commands.TableInfoKind.Constraints, m.schemaCache))
				}
			}
			return tea.Batch(cmds...)
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
			// queryPanel.Update is called before handleKeyMessages in ui.Update,
			// so the textarea has already processed the space — read current state directly.
			if msg.String() == "space" && !m.queryPanel.GetAutoCompleteActive() {
				result := autocomplete.GetCompletions(
					m.queryPanel.GetCurrentStatement(),
					m.queryPanel.GetWordAtCursor(),
					m.tablePanel.GetAllTableRefs(),
					m.db,
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
					debouncer.Trigger("save-query", commands.SaveQueryFile(
						m.connectionName, m.selectedDatabase, m.queryPanel.GetValue(), m.dbConfig.UseSingleQueryFile(),
					))
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
		return commands.GetAutocompleteData(m.db, result.TableRef, m.schemaCache)
	case autocomplete.CompletionTable:
		m.queryPanel.SetAutoCompleteActive(true)
		return m.queryPanel.SetAutoCompleteOptions(result.Items)
	case autocomplete.CompletionSchema:
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
