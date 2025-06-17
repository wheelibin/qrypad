package db

import (
	"database/sql"
	"fmt"
)

type DBConn struct {
	DB                *sql.DB
	DriverName        string
	ConnectedDatabase string
}

type ConnectionConfig struct {
	Driver           string `mapstructure:"driver"`
	Host             string `mapstructure:"host"`
	Port             int    `mapstructure:"port"`
	User             string `mapstructure:"user"`
	InsecurePassword string `mapstructure:"insecurePassword"`
	Database         string `mapstructure:"database"`
}

func Connect(conn ConnectionConfig, password string) (DBConn, error) {
	var (
		connString string
		driver     string
	)
	switch conn.Driver {
	case DriverNameMySQL:
		connString = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", conn.User, password, conn.Host, conn.Port, conn.Database)
		driver = conn.Driver
	case DriverNamePostgres:
		connString = fmt.Sprintf("postgres://%s:%s@%s:%d/%s", conn.User, password, conn.Host, conn.Port, conn.Database)
		driver = "pgx"
	}
	dbConn, err := sql.Open(driver, connString)
	if err != nil {
		return DBConn{}, fmt.Errorf("error opening connection to database: %w", err)
	}
	if err := dbConn.Ping(); err != nil {
		return DBConn{}, fmt.Errorf("error connecting to database: %w", err)
	}

	connectedDB := conn.Database
	if connectedDB == "" {
		// no database specified in the connection, so read it
		var sql, dbName string
		switch conn.Driver {
		case DriverNameMySQL:
			sql = "SELECT DATABASE()"
		case DriverNamePostgres:
			sql = "SELECT current_database()"
		}
		row := dbConn.QueryRow(sql)
		if err := row.Scan(&dbName); err != nil {
			return DBConn{}, fmt.Errorf("error connecting to database: %w", err)
		}
		connectedDB = dbName
	}

	return DBConn{DB: dbConn, DriverName: conn.Driver, ConnectedDatabase: connectedDB}, nil
}
