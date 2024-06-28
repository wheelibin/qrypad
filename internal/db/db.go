package db

import "database/sql"

type DBConn struct {
	DB         *sql.DB
	DriverName string
}
