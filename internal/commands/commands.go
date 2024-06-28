package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/wheelibin/dbee/internal/db"
)

func GetTableRows(dbConn db.DBConn, tableName string) tea.Cmd {
	return func() tea.Msg {
		data, err := db.GetTableRows(dbConn, tableName)
		if err != nil {
			return ErrMsg{err}
		}
		return db.DataMsg(data)
	}
}

func GetTableInfo(dbConn db.DBConn, tableName string) tea.Cmd {
	return func() tea.Msg {
		data, err := db.GetTableColumns(dbConn, tableName)
		if err != nil {
			return ErrMsg{err}
		}
		return db.TableInfoDataMsg(data)
	}
}

func GetSchemaTables(dbConn db.DBConn) tea.Cmd {
	return func() tea.Msg {
		data, err := db.GetSchemaTables(dbConn)
		if err != nil {
			return ErrMsg{Err: err}
		}
		return db.SchemaTablesMsg(data)
	}
}

func ExecuteQuery(dbConn db.DBConn, query string) tea.Cmd {
	return func() tea.Msg {
		data, err := db.ExecuteQuery(dbConn, query)
		if err != nil {
			return ErrMsg{err}
		}
		return db.DataMsg(data)
	}
}

func SetActivePanel(panelIndex int) tea.Cmd {
	return func() tea.Msg {
		return ActivePanelChangedMsg(panelIndex)
	}
}

func SetLoading() tea.Msg {
	return LoadingMsg{}
}

func TableSelectionChanged(tableName string) tea.Cmd {
	return func() tea.Msg {
		return TableSelectedMsg(tableName)
	}
}

func ReadOrCreateQueryFile(dbAlias string) tea.Cmd {
	return func() tea.Msg {

		dir, err := GetOutputDir()
		if err != nil {
			return ErrMsg{err}
		}
		filename := filepath.Join(dir, fmt.Sprintf("%s.sql", dbAlias))

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
		return QueryFileReadMsg(string(contents))
	}
}

func SaveQueryFile(dbAlias string, contents string) tea.Cmd {
	return func() tea.Msg {

		dir, err := GetOutputDir()
		if err != nil {
			return ErrMsg{err}
		}
		filename := filepath.Join(dir, fmt.Sprintf("%s.sql", dbAlias))

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
		outputDir = filepath.Join(appData, "dbee")
	case "darwin", "linux":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		outputDir = filepath.Join(homeDir, ".local", "share", "dbee")
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
