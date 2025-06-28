package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Table struct {
	Name     string
	RowCount int
}

func GetDatabasesSQL(dbConn DBConn) string {
	var query string
	switch dbConn.DriverName {
	case DriverNameMySQL:
		query = `SELECT SCHEMA_NAME name FROM information_schema.SCHEMATA 
						WHERE SCHEMA_NAME NOT IN ('mysql', 'performance_schema', 'sys') 
						ORDER BY name;`
	case DriverNamePostgres:
		query = `SELECT datname name
						FROM pg_database
						WHERE has_database_privilege(datname, 'CONNECT')
							AND NOT datistemplate
						ORDER BY datname;`
	}

	return query
}

// fetches the user tables in the database
func GetSchemaTablesSQL(dbConn DBConn) string {
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

	return query
}

// fetches the user views in the database
func GetSchemaViewsSQL(dbConn DBConn) string {
	query := `SELECT table_name name
						FROM information_schema.views
						WHERE table_schema NOT IN ('mysql', 'performance_schema', 'information_schema', 'sys', 'pg_catalog')
						ORDER BY table_name;`
	return query
}

// fetches the column information for the specified table
func GetTableColumnsSQL(tableName string) string {
	return fmt.Sprintf(`SELECT column_name name, data_type type, case when is_nullable = 'NO' then 'NOT NULL' else 'NULL' end nullable  
                                        FROM INFORMATION_SCHEMA.COLUMNS
                                        WHERE  TABLE_NAME = '%s';`, tableName)
}

// fetches the index information for the specified table
func GetTableIndexesSQL(dbConn DBConn, tableName string) string {
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
	return query
}

func GetPrimaryKeyColumns(ctx context.Context, dbConn DBConn, tableName string) ([]string, error) {
	var query string
	switch dbConn.DriverName {
	case DriverNameMySQL:
		query = fmt.Sprintf(`SELECT COLUMN_NAME name
												FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
												WHERE TABLE_SCHEMA = DATABASE()
													AND TABLE_NAME = '%s'
													AND CONSTRAINT_NAME = 'PRIMARY'
												ORDER BY ORDINAL_POSITION;`, tableName)
	case DriverNamePostgres:
		query = fmt.Sprintf(`SELECT a.attname name
												FROM pg_index i
												JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
												WHERE i.indrelid = '%s'::regclass AND i.indisprimary;`, tableName)
	}
	data, err := fetchRows(ctx, dbConn, query)
	if err != nil {
		return nil, err
	}
	columns := make([]string, 0)
	for _, row := range data.Rows {
		columns = append(columns, row["name"].(string))
	}
	return columns, nil
}

// fetches n rows from the specified table
func GetTableRowsSQL(tableName string, primaryKeyColumns []string, sortOrder string) string {
	query := fmt.Sprintf("SELECT * FROM %s", tableName)

	if len(primaryKeyColumns) > 0 {
		orderClause := " ORDER BY " + strings.Join(primaryKeyColumns, ", ") + " " + sortOrder
		query += orderClause
	}

	query += fmt.Sprintf(" LIMIT %d", getTableDataRowLimit())
	return query
}

// executes a user supplied sql query or statement
func ExecuteQuery(ctx context.Context, dbConn DBConn, query string) (*Data, error) {
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
		return execStatement(ctx, dbConn, query)
	} else {
		return fetchRows(ctx, dbConn, query)
	}
}

func GetTimeoutSecs() time.Duration {
	timeoutSecs := viper.GetInt(TimeoutConfigKey)
	if timeoutSecs == 0 {
		return time.Duration(30 * time.Second)
	}
	return time.Duration(timeoutSecs * int(time.Second))
}

func getTableDataRowLimit() int {
	rowLimit := viper.GetInt(TableDataRowLimitConfigKey)
	if rowLimit == 0 {
		return 100
	}
	return rowLimit
}

func fetchRows(ctx context.Context, dbConn DBConn, query string) (*Data, error) {
	start := time.Now()
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
	scanArgs := make([]any, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	data := &Data{Columns: columns, Rows: []map[string]any{}}
	for rows.Next() {
		err = rows.Scan(scanArgs...)
		if err != nil {
			return nil, err
		}

		row := make(map[string]any)
		for i, val := range values {
			if val == nil {
				row[columns[i]] = "NULL"
			} else {
				row[columns[i]] = truncateToSize(string(val), 100_000) // max 100k to not break the UI
			}
		}
		data.Rows = append(data.Rows, row)
	}

	if err = rows.Err(); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("query timeout exceeded (%d secs)\n\n to change the timeout add or modify the 'queryTimeout` config option", GetTimeoutSecs())
		}
		return nil, err
	}
	end := time.Now()
	data.QueryTime = end.Sub(start)

	return data, nil
}

func execStatement(ctx context.Context, dbConn DBConn, query string) (*Data, error) {
	start := time.Now()
	res, err := dbConn.DB.ExecContext(ctx, query)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("query timeout exceeded (%d secs)\n\n to change the timeout add or modify the 'queryTimeout` config option", GetTimeoutSecs())
		}
		return nil, err
	}
	end := time.Now()

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
			Rows: []map[string]any{{
				"Rows Affected":    rowsAffected,
				"Last Inserted ID": lastInsertId,
			}},
			QueryTime: end.Sub(start),
		}, nil

	default:
		return &Data{
			Columns: []string{"Rows Affected"},
			Rows: []map[string]any{{
				"Rows Affected": rowsAffected,
			}},
			QueryTime: end.Sub(start),
		}, nil
	}
}

func truncateToSize(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}

	var size int
	for i := range s {
		if i > maxBytes {
			return s[:size]
		}
		size = i
	}
	return s // all runes fit exactly
}
