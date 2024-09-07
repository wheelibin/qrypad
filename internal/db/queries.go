package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/spf13/viper"
)

type Table struct {
	Name     string
	RowCount int
}

// fetches the user tables in the database
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
		query = `SELECT relname name, TO_CHAR(n_live_tup, 'FM999,999,999') rows 
          FROM pg_stat_user_tables 
        ORDER BY name;`
	}

	return ExecuteQuery(dbConn, query)
}

// fetches the column information for the specified table
func GetTableColumns(dbConn DBConn, tableName string) (*Data, error) {
	return ExecuteQuery(dbConn, fmt.Sprintf(`SELECT column_name name, data_type type, case when is_nullable = 'NO' then 'NOT NULL' else 'NULL' end nullable  
                                        FROM INFORMATION_SCHEMA.COLUMNS
                                        WHERE  TABLE_NAME = '%s';`, tableName))
}

// fetches the index information for the specified table
func GetTableIndexes(dbConn DBConn, tableName string) (*Data, error) {

	var query string
	switch dbConn.DriverName {
	case DriverNameMySQL:
		query = fmt.Sprintf(`SELECT
                        index_name 'name', 
                        GROUP_CONCAT(column_name) cols, 
                        case when non_unique = 0 then 'unique' else '' end as 'unique',
                        case when index_name = 'PRIMARY' then 'primary' else '' end as 'primary'
                      FROM
                        INFORMATION_SCHEMA.statistics
                      WHERE
                        TABLE_NAME = '%s'
                        group by index_name, non_unique
                        order by seq_in_index;`, tableName)
	case DriverNamePostgres:
		query = fmt.Sprintf(`select
                          i.relname as "name",
                          array_to_string(array_agg(a.attname), ', ') as cols,
                          ix.indisunique as "unique",
                          ix.indisprimary as "primary"
                      from
                          pg_class t,
                          pg_class i,
                          pg_index ix,
                          pg_attribute a
                      where
                          t.oid = ix.indrelid
                          and i.oid = ix.indexrelid
                          and a.attrelid = t.oid
                          and a.attnum = ANY(ix.indkey)
                          and t.relkind = 'r'
                          and t.relname like '%s'
                      group by
                          t.relname,
                          i.relname,
                          ix.indisunique,
                      ix.indisprimary
                      order by
                          t.relname,
                          i.relname;`, tableName)
	}
	return ExecuteQuery(dbConn, query)
}

// fetches n rows from the specified table
func GetTableRows(dbConn DBConn, tableName string) (*Data, error) {
	return ExecuteQuery(dbConn, fmt.Sprintf("SELECT * FROM %s limit %d;", tableName, getTableDataRowLimit()))
}

// executes a user supplied sql query or statement
func ExecuteQuery(dbConn DBConn, query string) (*Data, error) {
	timeoutSecs := getTimeoutSecs()
	queryCtx, cancel := context.WithTimeout(context.Background(), timeoutSecs*time.Second)
	defer cancel()

	// crude way to decide whether the query should returns rows or use execute
	isStatement, err := regexp.MatchString(`(?i)^\s*(UPDATE|INSERT|DELETE|DROP|TRUNCATE|CREATE|ALTER)\s+`, query)
	if err != nil {
		return nil, err
	}

	isReturning, err := regexp.MatchString(`(?i)\s*(RETURNING)\s+`, query)
	if err != nil {
		return nil, err
	}

	if isStatement && !isReturning {
		return execStatement(queryCtx, dbConn, query)
	} else {
		return fetchRows(queryCtx, dbConn, query)
	}

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

func fetchRows(ctx context.Context, dbConn DBConn, query string) (*Data, error) {
	rows, err := dbConn.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]interface{}, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	data := &Data{Columns: columns, Rows: []map[string]interface{}{}}
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, val := range values {
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
			return nil, fmt.Errorf("query timeout exceeded (%d secs)\n\n to change the timeout add or modify the 'queryTimeout` config option", getTimeoutSecs())
		}
		return nil, err
	}

	return data, nil
}

func execStatement(ctx context.Context, dbConn DBConn, query string) (*Data, error) {
	res, err := dbConn.DB.ExecContext(ctx, query)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("query timeout exceeded (%d secs)\n\n to change the timeout add or modify the 'queryTimeout` config option", getTimeoutSecs())
		}
		return nil, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}

	switch dbConn.DriverName {

	case DriverNameMySQL:
		lastInsertId, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		return &Data{
			Columns: []string{"Rows Affected", "Last Inserted ID"},
			Rows: []map[string]interface{}{{
				"Rows Affected":    rowsAffected,
				"Last Inserted ID": lastInsertId,
			}}}, nil

	default:
		return &Data{
			Columns: []string{"Rows Affected"},
			Rows: []map[string]interface{}{{
				"Rows Affected": rowsAffected,
			}}}, nil
	}

}
