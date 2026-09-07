package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/password"
	"github.com/wheelibin/qrypad/internal/querybuffer"
)

type TablePanelKindType string

//nolint:gochecknoglobals // singleton-like enum structs used as namespaced constants
var TablePanelKind = struct {
	Tables TablePanelKindType
	Views  TablePanelKindType
}{
	Tables: "tables",
	Views:  "views",
}

type TableInfoKindType string

//nolint:gochecknoglobals // singleton-like enum struct used as namespaced constants
var TableInfoKind = struct {
	Columns     TableInfoKindType
	Indexes     TableInfoKindType
	Constraints TableInfoKindType
}{
	Columns:     "cols",
	Indexes:     "inds",
	Constraints: "constr",
}

func SavePassword(connectionName, pass string) tea.Cmd {
	return func() tea.Msg {
		err := password.SetPassword(connectionName, pass)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return PasswordSavedMsg{}
	}
}

func ConnectToDB(connectionName string, dbConfig db.ConnectionConfig) tea.Cmd {
	if connectionName == "" {
		return nil
	}
	return tea.Sequence(SetLoading(true), func() tea.Msg {
		var pass string
		if dbConfig.Driver != db.DriverName.SQLite {
			if len(dbConfig.InsecurePassword) == 0 {
				keyringPass, err := password.GetPassword(connectionName)
				if err != nil {
					if errors.Is(err, password.ErrPasswordNotSaved) {
						return PasswordInputNeededMsg{}
					}
				}
				pass = keyringPass
			} else {
				pass = dbConfig.InsecurePassword
			}
		}

		dbConn, err := db.Connect(dbConfig, pass)
		if err != nil {
			if errors.Is(err, password.ErrPasswordNotSaved) {
				return PasswordInputNeededMsg{}
			}
			return DatabaseConnectError{err}
		}
		return db.DatabaseConnectedMsg(dbConn)
	})
}

//nolint:gochecknoglobals // function variable for dependency injection in tests
var QueryResultBuilder = func(d *db.Data, err error) tea.Msg {
	return db.DataFetchedMsg{Data: d, Err: err}
}

func GetTableRows(dbConn db.DBConn, ref db.TableReference, sortOrder string) tea.Cmd {
	return func() tea.Msg {
		primaryKeyColumns, err := dbConn.Queries.PrimaryKeyColumns(context.Background(), dbConn, ref)
		if err != nil {
			return ErrMsg{Err: err}
		}

		query := dbConn.Queries.TableRows(ref, primaryKeyColumns, sortOrder)
		timeout := db.GetTimeoutSecs()
		ctx, cancel := context.WithTimeout(context.Background(), timeout) //nolint:gosec // cancel is returned via QueryControlMsg

		resultCh := make(chan tea.Msg, 1)
		go func() {
			data, err := db.ExecuteQuery(ctx, dbConn, query)
			resultCh <- QueryResultBuilder(data, err)
		}()

		return db.QueryControlMsg{Cancel: cancel, ResultChan: resultCh}
	}
}

func GetAutocompleteData(dbConn db.DBConn, ref db.TableReference, cache *db.SchemaCache) tea.Cmd {
	return func() tea.Msg {
		// Reuse cached column data if available (populated when tableInfoPanel fetches cols).
		if cache != nil {
			if cached, ok := cache.GetTableInfo(ref, string(TableInfoKind.Columns)); ok {
				cols := make([]string, 0, len(cached.Rows))
				for _, row := range cached.Rows {
					if name, ok := row["name"].(string); ok {
						cols = append(cols, name)
					}
				}
				return db.AutoCompleteDataFetchedMsg(cols)
			}
		}

		cols, err := db.GetAutoCompleteColumns(context.Background(), dbConn, ref)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return db.AutoCompleteDataFetchedMsg(cols)
	}
}

func GetTableInfo(dbConn db.DBConn, ref db.TableReference, kind TableInfoKindType, cache *db.SchemaCache) tea.Cmd {
	cacheKey := string(kind)

	// Cache hit: return immediately without a DB round-trip.
	if cache != nil {
		if cached, ok := cache.GetTableInfo(ref, cacheKey); ok {
			return func() tea.Msg {
				return db.TableInfoDataFetchedMsg{Data: cached, Err: nil}
			}
		}
	}

	// Cache miss: fetch from DB, populate cache on success.
	storeResult := func(d *db.Data, err error) tea.Msg {
		if err == nil && cache != nil {
			cache.SetTableInfo(ref, cacheKey, d)
		}
		return db.TableInfoDataFetchedMsg{Data: d, Err: err}
	}

	switch kind {
	case TableInfoKind.Columns:
		return ExecuteQuery(dbConn, dbConn.Queries.TableColumns(ref), storeResult)
	case TableInfoKind.Indexes:
		return func() tea.Msg {
			d, err := dbConn.Queries.TableIndexes(context.Background(), dbConn, ref)
			return storeResult(d, err)
		}
	case TableInfoKind.Constraints:
		return ExecuteQuery(dbConn, dbConn.Queries.TableConstraints(ref), storeResult)
	}
	return nil
}

func GetDatabases(dbConn db.DBConn) tea.Cmd {
	return ExecuteQuery(dbConn, dbConn.Queries.Databases(), func(d *db.Data, err error) tea.Msg {
		return db.DatabaseListFetchedMsg{Data: d, Err: err}
	})
}

func GetConnectionList() tea.Cmd {
	return func() tea.Msg {
		conns, err := db.GetConnections()
		if err != nil {
			return db.ConnectionListFetchedMsg{Data: nil, Err: err}
		}
		return db.ConnectionListFetchedMsg{Data: connectionsToDataResponse(conns), Err: nil}
	}
}

func GetSchemaEntities(dbConn db.DBConn, kind TablePanelKindType, cache *db.SchemaCache) tea.Cmd {
	cacheKey := string(kind)

	if cache != nil {
		if cached, ok := cache.GetEntities(cacheKey); ok {
			return func() tea.Msg {
				return db.SchemaEntitiesFetchedMsg{Data: cached, Err: nil}
			}
		}
	}

	storeResult := func(d *db.Data, err error) tea.Msg {
		if err == nil && cache != nil {
			cache.SetEntities(cacheKey, d)
		}
		return db.SchemaEntitiesFetchedMsg{Data: d, Err: err}
	}

	switch kind {
	case TablePanelKind.Tables:
		return ExecuteQuery(dbConn, dbConn.Queries.SchemaTables(), storeResult)
	case TablePanelKind.Views:
		return ExecuteQuery(dbConn, dbConn.Queries.SchemaViews(), storeResult)
	}
	return nil
}

type queryResultBuilder func(*db.Data, error) tea.Msg

func ExecuteQuery(dbConn db.DBConn, query string, resultBuilder queryResultBuilder) tea.Cmd {
	timeout := db.GetTimeoutSecs()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout) //nolint:gosec // cancel is returned via QueryControlMsg

		// Send cancel control back
		resultCh := make(chan tea.Msg, 1)

		go func() {
			data, err := db.ExecuteQuery(ctx, dbConn, query)
			resultCh <- resultBuilder(data, err)
		}()

		// Return control message now, and separately return result later
		return db.QueryControlMsg{Cancel: cancel, ResultChan: resultCh}
	}
}

func SetActivePanel(panelIndex int) tea.Cmd {
	return func() tea.Msg {
		return ActivePanelChangedMsg(panelIndex)
	}
}

func SetActiveTablePanelTab(tabIndex int) tea.Cmd {
	return func() tea.Msg {
		return TablePanelTabChangedMsg(tabIndex)
	}
}

func SetActiveTableInfoTab(tabIndex int) tea.Cmd {
	return func() tea.Msg {
		return TableInfoTabChangedMsg(tabIndex)
	}
}

func SetLoading(loading bool) tea.Cmd {
	return func() tea.Msg {
		return LoadingMsg{Loading: loading}
	}
}

func CancelQuery() tea.Cmd {
	return func() tea.Msg {
		return CancelQueryMsg{}
	}
}

func TableSelectionChanged(ref db.TableReference) tea.Cmd {
	return func() tea.Msg {
		return TableSelectedMsg(ref)
	}
}

func DatabaseSelectionChanged(name string) tea.Cmd {
	return func() tea.Msg {
		return DatabaseSelectedMsg(name)
	}
}

func ConnectionSelectionChanged(name string) tea.Cmd {
	return func() tea.Msg {
		return ConnectionSelectedMsg(name)
	}
}

func ClosePopup() tea.Cmd {
	return func() tea.Msg {
		return PopupClosedMsg{}
	}
}

// QuitNoConnection is sent when the user closes the connection switcher on
// first launch without choosing a connection.
func QuitNoConnection() tea.Cmd {
	return func() tea.Msg {
		return NoConnectionChosenMsg{}
	}
}

func CopyValue(value string, valueDesc string) tea.Cmd {
	return func() tea.Msg {
		return CopyValueMsg{Value: value, ValueDesc: valueDesc}
	}
}

func RequestPasswordInput() tea.Cmd {
	return func() tea.Msg {
		return PasswordInputNeededMsg{}
	}
}

func PasswordEntered(pwd string) tea.Cmd {
	return func() tea.Msg {
		return PasswordEnteredMsg(pwd)
	}
}

// ReadOrCreateQueryFile returns a tea.Cmd that loads the query file for the given
// connection/database from dir, creating it if it does not exist.
func ReadOrCreateQueryFile(dir, connectionName, databaseName string, singleFile bool) tea.Cmd {
	return func() tea.Msg {
		contents, filename, err := querybuffer.Load(dir, connectionName, databaseName, singleFile)
		if err != nil {
			return ErrMsg{err}
		}
		if filename == "" {
			return nil // per-database mode, database not yet known
		}
		return QueryFileReadMsg{Contents: contents, FileName: filename}
	}
}

// SaveQueryFile returns a tea.Cmd that calls buf.SaveIfChanged with the given contents.
func SaveQueryFile(buf *querybuffer.Buffer, conn, db, current string, single bool) tea.Cmd {
	return func() tea.Msg {
		if _, err := buf.SaveIfChanged(current, conn, db, single); err != nil {
			return ErrMsg{Err: err}
		}
		return QueryFileSavedMsg{}
	}
}

func GetOutputDir() (string, error) {
	var outputDir string

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("AppData")
		outputDir = filepath.Join(appData, "qrypad")
	case "darwin", "linux":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("error getting home directory: %w", err)
		}
		outputDir = filepath.Join(homeDir, ".local", "share", "qrypad")
	default:
		return "", errors.New("error determining folder to hold query files, unsupported OS")
	}

	// Ensure the directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) { //nolint:gosec // path constructed from trusted sources
		err = os.MkdirAll(outputDir, 0o750) //nolint:gosec // path constructed from trusted sources
		if err != nil {
			return "", fmt.Errorf("error creating folder to hold query files: %w", err)
		}
	}

	return outputDir, nil
}

func OpenEditor(file string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}
	c := exec.CommandContext(context.Background(), editor, file) //nolint:gosec // editor from $EDITOR env var is intentional
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{Err: err}
	})
}

func AutoCompleteEntrySelect(entry string) tea.Cmd {
	return func() tea.Msg {
		return AutoCompleteEntrySelectedMsg(entry)
	}
}

func AutoCompleteClose() tea.Cmd {
	return func() tea.Msg {
		return AutoCompleteCloseMsg{}
	}
}

func connectionsToDataResponse(conns map[string]db.ConnectionConfig) *db.Data {
	d := db.Data{
		Columns: []string{"name", "driver", "host"}, //nolint:goconst
		Rows:    make([]map[string]any, 0, len(conns)),
	}
	keys := make([]string, 0, len(conns))
	for name := range conns {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		c := conns[name]
		d.Rows = append(d.Rows, map[string]any{"name": name, "driver": c.Driver, "host": c.Host})
	}
	return &d
}

// PreloadTableThreshold is the max number of tables for which eager column
// preloading is performed.
const PreloadTableThreshold = 100

// PreloadColumns fetches column metadata for all tables in bulk and populates
// the schema cache. For Postgres/MySQL this uses a single bulk query; for
// SQLite it iterates per-table PRAGMAs. Returns SchemaPreloadCompleteMsg on
// success or SchemaPreloadErrorMsg on failure (non-fatal).
func PreloadColumns(dbConn db.DBConn, refs []db.TableReference, cache *db.SchemaCache) tea.Cmd {
	if len(refs) == 0 || len(refs) > PreloadTableThreshold || cache == nil {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		bulkSQL := dbConn.Queries.AllTableColumns()
		if bulkSQL != "" {
			return preloadColumnsBulk(ctx, dbConn, refs, cache, bulkSQL)
		}
		return preloadColumnsSequential(ctx, dbConn, refs, cache)
	}
}

func preloadColumnsBulk(ctx context.Context, dbConn db.DBConn, refs []db.TableReference, cache *db.SchemaCache, query string) tea.Msg {
	data, err := db.ExecuteQuery(ctx, dbConn, query)
	if err != nil {
		return SchemaPreloadErrorMsg{Err: err}
	}

	// Build a lookup of expected refs keyed by "schema.name"
	refLookup := make(map[string]db.TableReference, len(refs))
	for _, ref := range refs {
		key := ref.Schema + "." + ref.Name
		refLookup[key] = ref
	}

	// Partition rows by table_name
	grouped := make(map[string][]map[string]any)
	for _, row := range data.Rows {
		tableName, _ := row["table_name"].(string)
		grouped[tableName] = append(grouped[tableName], row)
	}

	// Store each table's columns in cache
	for tableName, rows := range grouped {
		ref, ok := refLookup[tableName]
		if !ok {
			continue
		}
		colData := &db.Data{
			Columns: []string{"name", "type", "nullable"},
			Rows:    rows,
		}
		cache.SetTableInfo(ref, string(TableInfoKind.Columns), colData)
	}

	return SchemaPreloadCompleteMsg{}
}

func preloadColumnsSequential(ctx context.Context, dbConn db.DBConn, refs []db.TableReference, cache *db.SchemaCache) tea.Msg {
	for _, ref := range refs {
		// Skip if already cached (e.g. user navigated before preload reached this table)
		if _, ok := cache.GetTableInfo(ref, string(TableInfoKind.Columns)); ok {
			continue
		}
		query := dbConn.Queries.TableColumns(ref)
		data, err := db.ExecuteQuery(ctx, dbConn, query)
		if err != nil {
			return SchemaPreloadErrorMsg{Err: err}
		}
		cache.SetTableInfo(ref, string(TableInfoKind.Columns), data)
	}
	return SchemaPreloadCompleteMsg{}
}
