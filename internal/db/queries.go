package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Table struct {
	Name     string
	RowCount int
}

func GetSchemaTables(dbConn DBConn) (*Data, error) {

	var query string
	switch dbConn.DriverName {
	case DriverNameMySQL:
		query = `SELECT TABLE_NAME name, format(TABLE_ROWS,0) 'rows' 
            FROM information_schema.TABLES 
            WHERE TABLE_SCHEMA not in ('mysql', 'performance_schema', 'sys') 
             AND TABLE_TYPE LIKE 'BASE_TABLE'
            ORDER BY name;`
	case DriverNamePostgres:
		query = `SELECT relname name, TO_CHAR(n_live_tup, 'FM9,999,999') rows 
          FROM pg_stat_user_tables 
        ORDER BY name;`
	}

	return ExecuteQuery(dbConn, query)

}

func GetTableColumns(dbConn DBConn, tableName string) (*Data, error) {
	return ExecuteQuery(dbConn, fmt.Sprintf(`SELECT column_name name, data_type type, case when is_nullable = 'NO' then 'NOT NULL' else 'NULL' end nullable  
                                        FROM INFORMATION_SCHEMA.COLUMNS
                                        WHERE  TABLE_NAME = '%s';`, tableName))
}

func GetTableRows(dbConn DBConn, tableName string) (*Data, error) {
	return ExecuteQuery(dbConn, fmt.Sprintf("SELECT * FROM %s limit %d;", tableName, getTableDataRowLimit()))
}

func ExecuteQuery(dbConn DBConn, query string) (*Data, error) {
	var data Data

	timeoutSecs := getTimeoutSecs()
	queryCtx, cancel := context.WithTimeout(context.Background(), timeoutSecs*time.Second)
	defer cancel()

	// Execute the query
	rows, err := dbConn.DB.QueryContext(queryCtx, query)
	if err != nil {

		return nil, err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	data.Columns = columns

	// Make a slice for the values
	values := make([]sql.RawBytes, len(columns))

	// rows.Scan wants '[]interface{}' as an argument, so we must copy the
	// references into such a slice
	// See http://code.google.com/p/go-wiki/wiki/InterfaceSlice for details
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	// Fetch rows
	for rows.Next() {
		// get RawBytes from data
		err = rows.Scan(scanArgs...)
		if err != nil {

			return nil, err
		}

		// Now do something with the data.
		// Here we just print each column as a string.
		row := make(map[string]any, 0)
		for i, val := range values {
			// Here we can check if the value is nil (NULL value)
			if val == nil {
				row[columns[i]] = "NULL"
			} else {
				row[columns[i]] = string(val)
			}
		}
		data.Rows = append(data.Rows, row)
	}
	if err = rows.Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("query timeout exceeded (%d secs)\n\n to change the timeout add or modify the 'queryTimeout` config option", timeoutSecs)
		}
		return nil, err
	}

	return &data, nil
}

func getTimeoutSecs() time.Duration {
	timeoutSecs := viper.GetInt(TimeoutConfigKey)
	if timeoutSecs == 0 {
		return 30
	}
	return time.Duration(timeoutSecs)
}

func getTableDataRowLimit() int {
	rowLimit := viper.GetInt(TableDataRowLimitConfigKey)
	if rowLimit == 0 {
		return 100
	}
	return rowLimit
}
