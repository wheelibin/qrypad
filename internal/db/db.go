package db

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
)

//nolint:revive // DBConn is intentionally named with DB prefix for clarity
type DBConn struct {
	DB                *sql.DB
	DriverName        DriverNameType
	ConnectedDatabase string
	Queries           QueryProvider
}

type ConnectionConfig struct {
	Driver           DriverNameType `mapstructure:"driver"`
	Host             string         `mapstructure:"host"`
	Port             int            `mapstructure:"port"`
	User             string         `mapstructure:"user"`
	InsecurePassword string         `mapstructure:"insecurePassword"`
	Database         string         `mapstructure:"database"`
	SingleQueryFile  bool           `mapstructure:"singleQueryFile"`
}

// UseSingleQueryFile returns true if this connection should use a single query
// file rather than per-database files. SQLite always uses a single file.
func (c ConnectionConfig) UseSingleQueryFile() bool {
	if c.Driver == DriverName.SQLite {
		return true
	}
	return c.SingleQueryFile
}

func Connect(conn ConnectionConfig, password string) (DBConn, error) {
	var (
		connString string
		driver     string
	)
	switch conn.Driver {
	case DriverName.MySQL:
		connString = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", conn.User, password, conn.Host, conn.Port, conn.Database)
		driver = string(conn.Driver)
	case DriverName.Postgres:
		host := net.JoinHostPort(conn.Host, strconv.Itoa(conn.Port))
		connString = fmt.Sprintf("postgres://%s:%s@%s/%s", conn.User, password, host, conn.Database)
		driver = "pgx"
	case DriverName.SQLite:
		connString = conn.Database
		driver = "sqlite3"
	}
	dbConn, err := sql.Open(driver, connString)
	if err != nil {
		return DBConn{}, fmt.Errorf("error opening connection to database: %w", err)
	}
	if err := dbConn.PingContext(context.Background()); err != nil {
		return DBConn{}, fmt.Errorf("error connecting to database: %w", err)
	}

	connectedDB := conn.Database
	if connectedDB == "" {
		// no database specified in the connection, so read it
		var sqlQuery, dbName string
		switch conn.Driver {
		case DriverName.MySQL:
			sqlQuery = "SELECT DATABASE()"
		case DriverName.Postgres:
			sqlQuery = "SELECT current_database()"
		}
		row := dbConn.QueryRowContext(context.Background(), sqlQuery)
		if err := row.Scan(&dbName); err != nil {
			return DBConn{}, fmt.Errorf("error connecting to database: %w", err)
		}
		connectedDB = dbName
	}

	var queries QueryProvider
	switch conn.Driver {
	case DriverName.MySQL:
		queries = mysqlQueries{}
	case DriverName.Postgres:
		queries = postgresQueries{}
	case DriverName.SQLite:
		queries = sqliteQueries{}
	}

	return DBConn{
		DB:                dbConn,
		DriverName:        conn.Driver,
		ConnectedDatabase: connectedDB,
		Queries:           queries,
	}, nil
}
