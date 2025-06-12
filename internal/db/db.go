package db

import (
	"database/sql"
	"fmt"
)

type DBConn struct {
	DB         *sql.DB
	DriverName string
}

type DBConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
}

func Connect(conn DBConfig) (DBConn, error) {
	var (
		connString string
		driver     string
	)
	switch conn.Driver {
	case DriverNameMySQL:
		connString = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", conn.User, conn.Password, conn.Host, conn.Port, conn.Database)
		driver = conn.Driver
	case DriverNamePostgres:
		connString = fmt.Sprintf("postgres://%s:%s@%s:%d/%s", conn.User, conn.Password, conn.Host, conn.Port, conn.Database)
		driver = "pgx"
	}
	dbConn, err := sql.Open(driver, connString)
	if err != nil {
		return DBConn{}, fmt.Errorf("error connecting to database: %w", err)
	}
	return DBConn{DB: dbConn, DriverName: conn.Driver}, nil
}
