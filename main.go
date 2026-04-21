package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/spf13/viper"
	"github.com/wheelibin/qrypad/internal/commands"
	"github.com/wheelibin/qrypad/internal/constants"
	"github.com/wheelibin/qrypad/internal/db"
	"github.com/wheelibin/qrypad/internal/keys"
	"github.com/wheelibin/qrypad/internal/ui"
)

func main() {
	viper.SetConfigName("config")               // name of config file (without extension)
	viper.AddConfigPath("$HOME/.config/qrypad") // call multiple times to add many search paths
	viper.AddConfigPath(".")                    // optionally look for config in the working directory
	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			exitWithError("no config found\n(see https://github.com/wheelibin/qrypad/blob/main/README.md)\n\n", nil)
		} else {
			exitWithError("error reading config\n(for proper format see https://github.com/wheelibin/qrypad/blob/main/README.md)\n\n", nil)
		}
	}

	conns, err := db.GetConnections()
	if err != nil {
		exitWithError("error unmarshalling config\n(for config format see https://github.com/wheelibin/qrypad/blob/main/README.md)\n\n", err)
	}

	if len(os.Args[1:]) == 0 {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"\nUsage:  qrypad [connection]\n\n%s\n\n    [connection]  The name of a database connection defined in your config\n\n",
			constants.AppDesc,
		)
		os.Exit(1)
	}

	connectionName := os.Args[1]
	conn, ok := conns[connectionName]
	if !ok {
		exitWithError("no config found for the specified database\n(see https://github.com/wheelibin/qrypad/blob/main/README.md)\n\n", nil)
	}

	dir, err := commands.GetOutputDir()
	if err != nil {
		exitWithError("unexpected error\n\n", err)
	}
	filename := filepath.Join(dir, "debug.log")
	f, err := tea.LogToFile(filename, "debug")
	if err != nil {
		exitWithError("unexpected error\n\n", err)
	}
	defer f.Close()

	keys.MapCustomKeys()
	m := ui.NewModel(connectionName, conn)

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		exitWithError("unexpected error\n\n", err)
	}
}

func exitWithError(msg string, err error) {
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: %v", msg, err)
	} else {
		_, _ = fmt.Fprint(os.Stderr, msg)
	}
	os.Exit(1)
}
