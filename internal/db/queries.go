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

// quotedTableRef returns the quoted, driver-appropriate table reference for use in SQL.
// Postgres: "schema"."table" or just "table" (no schema)
// MySQL:    `schema`.`table` or just `table` (no schema)
// SQLite:   table (no quoting needed, schema always empty)
func quotedTableRef(dbConn DBConn, ref TableReference) string {
	switch dbConn.DriverName {
	case DriverName.Postgres:
		if ref.Schema != "" {
			return fmt.Sprintf(`"%s"."%s"`, ref.Schema, ref.Name)
		}
		return fmt.Sprintf(`"%s"`, ref.Name)
	case DriverName.MySQL:
		if ref.Schema != "" {
			return fmt.Sprintf("`%s`.`%s`", ref.Schema, ref.Name)
		}
		return fmt.Sprintf("`%s`", ref.Name)
	default:
		return ref.Name
	}
}

func GetDatabasesSQL(dbConn DBConn) string {
	var query string
	switch dbConn.DriverName {
	case DriverName.MySQL:
		query = `SELECT SCHEMA_NAME name FROM information_schema.SCHEMATA 
						WHERE SCHEMA_NAME NOT IN ('mysql', 'performance_schema', 'sys') 
						ORDER BY name;`
	case DriverName.Postgres:
		query = `SELECT datname name
						FROM pg_database
						WHERE has_database_privilege(datname, 'CONNECT')
							AND NOT datistemplate
						ORDER BY datname;`
	}

	return query
}

// GetSchemaTablesSQL fetches the user tables in the database.
func GetSchemaTablesSQL(dbConn DBConn) string {
	var query string
	switch dbConn.DriverName {
	case DriverName.MySQL:
		query = `SELECT TABLE_SCHEMA "schema", TABLE_NAME name, format(TABLE_ROWS,0) 'rows'
            FROM information_schema.TABLES
            WHERE TABLE_SCHEMA NOT IN ('mysql', 'performance_schema', 'sys')
             AND TABLE_TYPE = 'BASE TABLE'
            ORDER BY table_schema, name;`
	case DriverName.Postgres:
		query = `SELECT schemaname schema, relname name, TO_CHAR(n_live_tup, 'FM999,999,999') rows
          FROM pg_stat_user_tables
          ORDER BY schema, name;`
	case DriverName.SQLite:
		query = `SELECT name FROM sqlite_master
				WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`
	}

	return query
}

// GetSchemaViewsSQL fetches the user views in the database.
func GetSchemaViewsSQL(dbConn DBConn) string {
	var query string
	switch dbConn.DriverName {
	case DriverName.MySQL:
		query = `SELECT table_schema schema, table_name name
                        FROM information_schema.views
                        WHERE table_schema NOT IN ('mysql', 'performance_schema', 'information_schema', 'sys')
                        ORDER BY schema, table_name;`
	case DriverName.Postgres:
		query = `SELECT table_schema schema, table_name name
                        FROM information_schema.views
                        WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
                        ORDER BY schema, table_name;`
	case DriverName.SQLite:
		query = `SELECT name
                        FROM sqlite_master
                        WHERE type = 'view'
                            AND name NOT LIKE 'sqlite_%'
                        ORDER BY name;`
	}
	return query
}

// GetTableColumnsSQL fetches the column information for the specified table.
func GetTableColumnsSQL(dbConn DBConn, ref TableReference) string {
	var query string
	switch dbConn.DriverName {
	case DriverName.MySQL:
		schemaFilter := "TABLE_SCHEMA = DATABASE()"
		if ref.Schema != "" {
			schemaFilter = fmt.Sprintf("TABLE_SCHEMA = '%s'", ref.Schema)
		}
		query = fmt.Sprintf(`SELECT column_name name, data_type type, case when is_nullable = 'NO' then 'NOT NULL' else 'NULL' end nullable
                                        FROM INFORMATION_SCHEMA.COLUMNS
                                        WHERE %s AND TABLE_NAME = '%s' ORDER BY column_name;`,
			schemaFilter, ref.Name)
	case DriverName.Postgres:
		schemaFilter := "table_schema = current_schema()"
		if ref.Schema != "" {
			schemaFilter = fmt.Sprintf("table_schema = '%s'", ref.Schema)
		}
		query = fmt.Sprintf(`SELECT column_name name, data_type type, case when is_nullable = 'NO' then 'NOT NULL' else 'NULL' end nullable
                                        FROM INFORMATION_SCHEMA.COLUMNS
                                        WHERE %s AND TABLE_NAME = '%s' ORDER BY column_name;`,
			schemaFilter, ref.Name)
	case DriverName.SQLite:
		query = fmt.Sprintf(`SELECT * FROM pragma_table_info('%s');`, ref.Name)
	}
	return query
}

// GetTableIndexesSQL fetches the index information for the specified table.
func GetTableIndexesSQL(dbConn DBConn, ref TableReference) string {
	var query string
	switch dbConn.DriverName {
	case DriverName.MySQL:
		schemaFilter := "TABLE_SCHEMA = DATABASE()"
		if ref.Schema != "" {
			schemaFilter = fmt.Sprintf("TABLE_SCHEMA = '%s'", ref.Schema)
		}
		query = fmt.Sprintf(`SELECT
                        index_name 'name',
                        GROUP_CONCAT(column_name ORDER BY seq_in_index) cols,
                        case when non_unique = 0 then 'unique' else '' end as 'unique',
                        case when index_name = 'PRIMARY' then 'primary' else '' end as 'primary'
                      FROM
                        INFORMATION_SCHEMA.statistics
                      WHERE
                        %s AND TABLE_NAME = '%s'
                        group by index_name, non_unique
                        order by index_name;`, schemaFilter, ref.Name)
	case DriverName.Postgres:
		tableFilter := fmt.Sprintf("t.relname like '%s'", ref.Name)
		if ref.Schema != "" {
			tableFilter = fmt.Sprintf("t.relname like '%s' AND n.nspname = '%s'", ref.Name, ref.Schema)
		}
		query = fmt.Sprintf(`select
                          i.relname as "name",
                          array_to_string(array_agg(a.attname ORDER BY array_position(ix.indkey::int[], a.attnum::int)), ', ') as cols,
                          ix.indisunique as "unique",
                          ix.indisprimary as "primary"
                      from
                          pg_class t,
                          pg_class i,
                          pg_index ix,
                          pg_attribute a,
                          pg_namespace n
                      where
                          t.oid = ix.indrelid
                          and i.oid = ix.indexrelid
                          and a.attrelid = t.oid
                          and a.attnum = ANY(ix.indkey)
                          and t.relkind = 'r'
                          and t.relnamespace = n.oid
                          and %s
                      group by
                          t.relname,
                          i.relname,
                          ix.indisunique,
                      ix.indisprimary
                      order by
                          t.relname,
                          i.relname;`, tableFilter)
	case DriverName.SQLite:
		query = fmt.Sprintf(`select * from pragma_index_list('%s');`, ref.Name)
	}
	return query
}

func GetSQLiteTableIndexes(ctx context.Context, dbConn DBConn, ref TableReference) (*Data, error) {
	inds, err := fetchRows(ctx, dbConn, fmt.Sprintf("PRAGMA index_list('%s')", ref.Name))
	if err != nil {
		return nil, err
	}
	cols := []string{"name", "cols"}
	rows := make([]map[string]any, 0)
	for _, row := range inds.Rows {
		info, err := fetchRows(ctx, dbConn, fmt.Sprintf("PRAGMA index_info('%s')", row["name"]))
		if err != nil {
			return nil, err
		}
		indexCols := make([]string, 0)
		for _, infoRow := range info.Rows {
			if name, ok := infoRow["name"].(string); ok {
				indexCols = append(indexCols, name)
			}
		}
		rows = append(rows, map[string]any{
			"name": row["name"],
			"cols": indexCols,
		})
	}

	return &Data{
		Columns: cols,
		Rows:    rows,
	}, nil
}

// GetTableConstraintsSQL fetches the constraints information for the specified table.
func GetTableConstraintsSQL(dbConn DBConn, ref TableReference) string {
	var query string
	switch dbConn.DriverName {
	case DriverName.MySQL:
		schemaFilter := "TABLE_SCHEMA = DATABASE()"
		if ref.Schema != "" {
			schemaFilter = fmt.Sprintf("TABLE_SCHEMA = '%s'", ref.Schema)
		}
		query = fmt.Sprintf(`SELECT
                                        CONSTRAINT_NAME 'name',
                                        CONSTRAINT_TYPE 'type'
                                    FROM
                                        INFORMATION_SCHEMA.TABLE_CONSTRAINTS
                                    WHERE
                                        %s
                                        AND TABLE_NAME = '%s';`, schemaFilter, ref.Name)
	case DriverName.Postgres:
		schemaJoin := ""
		schemaWhere := ""
		if ref.Schema != "" {
			schemaJoin = "JOIN pg_namespace n ON c.connamespace = n.oid"
			schemaWhere = fmt.Sprintf("AND n.nspname = '%s'", ref.Schema)
		}
		query = fmt.Sprintf(`SELECT
                                        conname AS name,
                                        contype AS type,
                                        pg_get_constraintdef(c.oid) AS definition
                                FROM
                                        pg_constraint c
                                JOIN
                                        pg_class t ON c.conrelid = t.oid
                                %s
                                WHERE
                                        t.relname = '%s'
                                        %s;`, schemaJoin, ref.Name, schemaWhere)
	case DriverName.SQLite:
		query = fmt.Sprintf(`PRAGMA foreign_key_list('%s');`, ref.Name)
	}
	return query
}

func GetPrimaryKeyColumns(ctx context.Context, dbConn DBConn, ref TableReference) ([]string, error) {
	var query string
	switch dbConn.DriverName {
	case DriverName.MySQL:
		schemaFilter := "TABLE_SCHEMA = DATABASE()"
		if ref.Schema != "" {
			schemaFilter = fmt.Sprintf("TABLE_SCHEMA = '%s'", ref.Schema)
		}
		query = fmt.Sprintf(`SELECT COLUMN_NAME name
                                        FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
                                        WHERE %s
                                            AND TABLE_NAME = '%s'
                                            AND CONSTRAINT_NAME = 'PRIMARY'
                                        ORDER BY ORDINAL_POSITION;`, schemaFilter, ref.Name)
	case DriverName.Postgres:
		qualifiedRef := quotedTableRef(dbConn, ref)
		query = fmt.Sprintf(`SELECT a.attname name
                                        FROM pg_index i
                                        JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
                                        WHERE i.indrelid = '%s'::regclass AND i.indisprimary;`, qualifiedRef)
	case DriverName.SQLite:
		query = fmt.Sprintf(`SELECT name
                                        FROM pragma_table_info('%s')
                                        WHERE pk > 0
                                        ORDER BY pk;`, ref.Name)
	}
	data, err := fetchRows(ctx, dbConn, query)
	if err != nil {
		return nil, err
	}
	columns := make([]string, 0)
	for _, row := range data.Rows {
		if name, ok := row["name"].(string); ok {
			columns = append(columns, name)
		}
	}
	return columns, nil
}

func GetAutoCompleteColumns(ctx context.Context, dbConn DBConn, ref TableReference) ([]string, error) {
	query := GetTableColumnsSQL(dbConn, ref)
	data, err := fetchRows(ctx, dbConn, query)
	if err != nil {
		return nil, err
	}
	columns := make([]string, 0)
	for _, row := range data.Rows {
		if name, ok := row["name"].(string); ok {
			columns = append(columns, name)
		}
	}
	return columns, nil
}

// GetTableRowsSQL fetches n rows from the specified table.
func GetTableRowsSQL(dbConn DBConn, ref TableReference, primaryKeyColumns []string, sortOrder string) string {
	from := quotedTableRef(dbConn, ref)
	query := fmt.Sprintf("SELECT * FROM %s", from)

	if len(primaryKeyColumns) > 0 {
		orderClause := " ORDER BY " + strings.Join(primaryKeyColumns, ", ") + " " + sortOrder
		query += orderClause
	}

	query += fmt.Sprintf(" LIMIT %d", getTableDataRowLimit())
	return query
}

// ExecuteQuery executes a user supplied sql query or statement.
func ExecuteQuery(ctx context.Context, dbConn DBConn, query string) (*Data, error) {
	// crude way to decide whether the query should returns rows or use execute
	isStatement, err := regexp.MatchString(`(?i)^\s*(UPDATE|INSERT|DELETE|DROP|TRUNCATE|CREATE|ALTER)\s+`, query)
	if err != nil {
		return nil, fmt.Errorf("error matching statement pattern: %w", err)
	}

	isReturning, err := regexp.MatchString(`(?i)\s*(RETURNING)\s+`, query)
	if err != nil {
		return nil, fmt.Errorf("error matching returning pattern: %w", err)
	}

	if isStatement && !isReturning {
		return execStatement(ctx, dbConn, query)
	}
	return fetchRows(ctx, dbConn, query)
}

func GetTimeoutSecs() time.Duration {
	timeoutSecs := viper.GetInt(TimeoutConfigKey)
	if timeoutSecs == 0 {
		return 30 * time.Second
	}
	return time.Duration(timeoutSecs) * time.Second
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
		return nil, fmt.Errorf("error querying database: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("error getting columns: %w", err)
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
			return nil, fmt.Errorf("error scanning row: %w", err)
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
			return nil, fmt.Errorf(
				"query timeout exceeded (%d secs)\n\n to change the timeout add or modify the 'queryTimeout` config option",
				GetTimeoutSecs(),
			)
		}
		return nil, fmt.Errorf("error iterating rows: %w", err)
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
			return nil, fmt.Errorf(
				"query timeout exceeded (%d secs)\n\n to change the timeout add or modify the 'queryTimeout` config option",
				GetTimeoutSecs(),
			)
		}
		return nil, fmt.Errorf("error executing statement: %w", err)
	}
	end := time.Now()

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("error getting rows affected: %w", err)
	}

	switch dbConn.DriverName {
	case DriverName.MySQL:
		lastInsertID, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("error getting last insert ID: %w", err)
		}
		return &Data{
			Columns: []string{"Rows Affected", "Last Inserted ID"},
			Rows: []map[string]any{{
				"Rows Affected":    rowsAffected,
				"Last Inserted ID": lastInsertID,
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

	// Find the largest rune boundary i such that i <= maxBytes.
	// `range` yields i = start byte of each rune; the end of the previous
	// rune is the start of the current one.
	lastFit := 0
	for i := range s {
		if i > maxBytes {
			return s[:lastFit]
		}
		lastFit = i
	}
	// Loop completed without exceeding maxBytes; the end of the final rune
	// is len(s). If len(s) <= maxBytes we'd have returned above, so here
	// len(s) > maxBytes and lastFit is the start of the final rune.
	return s[:lastFit]
}
