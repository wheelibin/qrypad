package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/password"
)

type TablePanelKindType string

var TablePanelKind = struct {
	Tables TablePanelKindType
	Views  TablePanelKindType
}{
	Tables: "tables",
	Views:  "views",
}

type TableInfoKindType string

var TableInfoKind = struct {
	Columns TableInfoKindType
	Indexes TableInfoKindType
}{
	Columns: "cols",
	Indexes: "inds",
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
	return tea.Sequence(SetLoading(true), func() tea.Msg {
		var pass string
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

		dbConn, err := db.Connect(dbConfig, pass)
		if err != nil {
			if errors.Is(err, password.ErrPasswordNotSaved) {
				return PasswordInputNeededMsg{}
			}
			return DatabaseConnectErrMsg{err}
		}
		return db.DatabaseConnectedMsg(dbConn)
	})
}

var QueryResultBuilder = func(d *db.Data, err error) tea.Msg {
	return db.DataFetchedMsg{Data: d, Err: err}
}

func GetTableRows(dbConn db.DBConn, tableName string) tea.Cmd {
	primaryKeyColumns, err := db.GetPrimaryKeyColumns(context.Background(), dbConn, tableName)
	if err != nil {
		return func() tea.Msg {
			return ErrMsg{Err: err}
		}
	}

	return ExecuteQuery(dbConn, db.GetTableRowsSQL(tableName, primaryKeyColumns), QueryResultBuilder)
}

func GetTableInfo(dbConn db.DBConn, tableName string, kind TableInfoKindType) tea.Cmd {
	switch kind {
	case TableInfoKind.Columns:
		return ExecuteQuery(dbConn, db.GetTableColumnsSQL(tableName), func(d *db.Data, err error) tea.Msg {
			return db.TableInfoDataFetchedMsg{Data: d, Err: err}
		})
	case TableInfoKind.Indexes:
		return ExecuteQuery(dbConn, db.GetTableIndexesSQL(dbConn, tableName), func(d *db.Data, err error) tea.Msg {
			return db.TableInfoDataFetchedMsg{Data: d, Err: err}
		})
	}
	return nil
}

func GetDatabases(dbConn db.DBConn) tea.Cmd {
	return ExecuteQuery(dbConn, db.GetDatabasesSQL(dbConn), func(d *db.Data, err error) tea.Msg {
		return db.DatabaseListFetchedMsg{Data: d, Err: err}
	})
}

func GetSchemaEntities(dbConn db.DBConn, kind TablePanelKindType) tea.Cmd {
	switch kind {
	case TablePanelKind.Tables:
		return ExecuteQuery(dbConn, db.GetSchemaTablesSQL(dbConn), func(d *db.Data, err error) tea.Msg {
			return db.SchemaEntitiesFetchedMsg{Data: d, Err: err}
		})
	case TablePanelKind.Views:
		return ExecuteQuery(dbConn, db.GetSchemaViewsSQL(dbConn), func(d *db.Data, err error) tea.Msg {
			return db.SchemaEntitiesFetchedMsg{Data: d, Err: err}
		})
	}
	return nil
}

type queryResultBuilder func(*db.Data, error) tea.Msg

func ExecuteQuery(dbConn db.DBConn, query string, resultBuilder queryResultBuilder) tea.Cmd {
	timeout := db.GetTimeoutSecs()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)

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

func TableSelectionChanged(tableName string) tea.Cmd {
	return func() tea.Msg {
		return TableSelectedMsg(tableName)
	}
}

func DatabaseSelectionChanged(name string) tea.Cmd {
	return func() tea.Msg {
		return DatabaseSelectedMsg(name)
	}
}

func ClosePopup() tea.Cmd {
	return func() tea.Msg {
		return PopupClosedMsg{}
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

func ReadOrCreateQueryFile(connectionName string) tea.Cmd {
	return func() tea.Msg {
		dir, err := GetOutputDir()
		if err != nil {
			return ErrMsg{err}
		}
		filename := filepath.Join(dir, fmt.Sprintf("%s.sql", connectionName))

		if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
			_, err := os.Create(filename)
			if err != nil {
				return ErrMsg{err}
			}
		}

		contents, err := os.ReadFile(filename)
		if err != nil {
			return ErrMsg{err}
		}
		return QueryFileReadMsg{Contents: string(contents), FileName: filename}
	}
}

func SaveQueryFile(connectionName string, contents string) tea.Cmd {
	return func() tea.Msg {
		dir, err := GetOutputDir()
		if err != nil {
			return ErrMsg{err}
		}
		filename := filepath.Join(dir, fmt.Sprintf("%s.sql", connectionName))

		if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
			_, err := os.Create(filename)
			if err != nil {
				return ErrMsg{err}
			}
		}

		f, err := os.Create(filename)
		if err != nil {
			return ErrMsg{err}
		}

		_, err = f.WriteString(contents)
		if err != nil {
			return ErrMsg{err}
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
			return "", err
		}
		outputDir = filepath.Join(homeDir, ".local", "share", "qrypad")
	default:
		return "", fmt.Errorf("error determining folder to hold query files, unsupported OS")
	}

	// Ensure the directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		err = os.MkdirAll(outputDir, 0755)
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
	c := exec.Command(editor, file) //nolint:gosec
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return EditorFinishedMsg{err}
	})
}
