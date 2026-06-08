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

const (
	columnTypeUnknown  = "unknown"
	columnTypeNumber   = "number"
	columnTypeString   = "string"
	columnTypeDatetime = "datetime"
	columnTypeBinary   = "binary"
	columnTypeBoolean  = "boolean"
	columnTypeJSON     = "json"

	colRowsAffected   = "Rows Affected"
	colLastInsertedID = "Last Inserted ID"
)

type Table struct {
	Name     string
	RowCount int
}

// QueryProvider produces driver-specific SQL strings and executes
// driver-specific queries. A single implementation exists per driver;
// the correct one is attached to DBConn.Queries at connection time.
type QueryProvider interface {
	// Databases returns SQL listing databases (or schemas) visible to the
	// current connection. Returns "" for SQLite (single-file databases).
	Databases() string
	// SchemaTables returns SQL listing user tables in the current database.
	// Result columns: schema (omitted for SQLite), name, rows (optional).
	SchemaTables() string
	// SchemaViews returns SQL listing user views. Result columns: schema
	// (omitted for SQLite), name.
	SchemaViews() string
	// TableColumns returns SQL listing the columns of the given table.
	// Result columns: name, type, nullable (MySQL/Postgres); SQLite returns
	// pragma_table_info shape (cid, name, type, notnull, dflt_value, pk).
	TableColumns(ref TableReference) string
	// TableConstraints returns SQL listing the constraints of the given table.
	// SQLite returns only foreign keys (pragma_foreign_key_list shape).
	TableConstraints(ref TableReference) string
	// AllTableColumns returns SQL that fetches columns for ALL user tables
	// in a single query. Returns "" if the driver doesn't support bulk column
	// fetching (e.g. SQLite). Result columns must include: table_name, name,
	// type, nullable. Rows are ordered by table_name, then column_name.
	AllTableColumns() string
	// TableRows returns SQL selecting rows from the given table, ordered
	// by primaryKeyColumns (if any) in the given sortOrder ("ASC"/"DESC"),
	// limited to a configured row count.
	TableRows(ref TableReference, primaryKeyColumns []string, sortOrder string) string
	// TableIndexes fetches the indexes of the given table. Result columns:
	// name, cols, and (MySQL/Postgres only) unique, primary. SQLite performs
	// a two-pragma walk and returns cols as a []string per row.
	TableIndexes(ctx context.Context, dbConn DBConn, ref TableReference) (*Data, error)
	// PrimaryKeyColumns returns the list of primary-key column names for
	// the given table, ordered by position within the primary key.
	PrimaryKeyColumns(ctx context.Context, dbConn DBConn, ref TableReference) ([]string, error)
}

// quotedTableRefForDriver returns the quoted, driver-appropriate table
// reference for use in SQL, given the driver name directly.
// Postgres: "schema"."table" or just "table" (no schema)
// MySQL:    `schema`.`table` or just `table` (no schema)
// SQLite:   table (no quoting needed, schema always empty)
func quotedTableRefForDriver(driver DriverNameType, ref TableReference) string {
	switch driver {
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

// buildTableRowsSQL builds a SELECT * ... [ORDER BY pks sortOrder] LIMIT N
// query. The caller is responsible for quoting `from` appropriately for
// its driver (typically via quotedTableRefForDriver).
func buildTableRowsSQL(from string, primaryKeyColumns []string, sortOrder string) string {
	query := fmt.Sprintf("SELECT * FROM %s", from)

	if len(primaryKeyColumns) > 0 {
		orderClause := " ORDER BY " + strings.Join(primaryKeyColumns, ", ") + " " + sortOrder
		query += orderClause
	}

	query += fmt.Sprintf(" LIMIT %d", getTableDataRowLimit())
	return query
}

func GetAutoCompleteColumns(ctx context.Context, dbConn DBConn, ref TableReference) ([]string, error) {
	query := dbConn.Queries.TableColumns(ref)
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

	// Extract column types for result cell styling
	columnTypes := make([]string, len(columns))
	if cts, err := rows.ColumnTypes(); err == nil {
		for i, ct := range cts {
			columnTypes[i] = normalizeColumnType(ct.DatabaseTypeName())
		}
	} else {
		for i := range columns {
			columnTypes[i] = columnTypeUnknown
		}
	}

	values := make([]sql.RawBytes, len(columns))
	scanArgs := make([]any, len(values))
	for i := range values {
		scanArgs[i] = &values[i]
	}

	data := &Data{Columns: columns, ColumnTypes: columnTypes, Rows: []map[string]any{}}
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
			Columns:     []string{colRowsAffected, colLastInsertedID},
			ColumnTypes: []string{columnTypeNumber, columnTypeNumber},
			Rows: []map[string]any{{
				colRowsAffected:   rowsAffected,
				colLastInsertedID: lastInsertID,
			}},
			QueryTime: end.Sub(start),
		}, nil

	default:
		return &Data{
			Columns:     []string{colRowsAffected},
			ColumnTypes: []string{columnTypeNumber},
			Rows: []map[string]any{{
				colRowsAffected: rowsAffected,
			}},
			QueryTime: end.Sub(start),
		}, nil
	}
}

// fetchPrimaryKeyColumns runs the given SQL and reshapes the result into
// a slice of column-name strings. Shared implementation used by every
// driver's PrimaryKeyColumns method.
func fetchPrimaryKeyColumns(ctx context.Context, dbConn DBConn, query string) ([]string, error) {
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

// columnTypeExact maps exact (uppercased) driver type names to abstract categories.
//
//nolint:gochecknoglobals // immutable lookup table
var columnTypeExact = map[string]string{
	// Numbers
	"INT": columnTypeNumber, "INT2": columnTypeNumber, "INT4": columnTypeNumber, "INT8": columnTypeNumber,
	"TINYINT": columnTypeNumber, "SMALLINT": columnTypeNumber, "MEDIUMINT": columnTypeNumber, "BIGINT": columnTypeNumber,
	"INTEGER": columnTypeNumber, "REAL": columnTypeNumber,
	"FLOAT": columnTypeNumber, "FLOAT4": columnTypeNumber, "FLOAT8": columnTypeNumber,
	"DOUBLE": columnTypeNumber, "DECIMAL": columnTypeNumber, "NUMERIC": columnTypeNumber,
	"BIT": columnTypeNumber,
	// Strings
	"TEXT": columnTypeString, "VARCHAR": columnTypeString, "CHAR": columnTypeString, "BPCHAR": columnTypeString, "NAME": columnTypeString,
	"TINYTEXT": columnTypeString, "MEDIUMTEXT": columnTypeString, "LONGTEXT": columnTypeString,
	"ENUM": columnTypeString, "SET": columnTypeString,
	// Booleans
	"BOOL": columnTypeBoolean, "BOOLEAN": columnTypeBoolean,
	// JSON
	"JSON": columnTypeJSON, "JSONB": columnTypeJSON,
	// Date/time
	"DATE": columnTypeDatetime, "TIME": columnTypeDatetime, "TIMESTAMP": columnTypeDatetime,
	"TIMESTAMPTZ": columnTypeDatetime, "DATETIME": columnTypeDatetime, "INTERVAL": columnTypeDatetime, "YEAR": columnTypeDatetime,
	// Binary
	"BYTEA": columnTypeBinary, "BINARY": columnTypeBinary, "VARBINARY": columnTypeBinary,
	"BLOB": columnTypeBinary, "TINYBLOB": columnTypeBinary, "MEDIUMBLOB": columnTypeBinary, "LONGBLOB": columnTypeBinary,
}

// columnTypePrefixes maps type-name prefixes to abstract categories.
// Used for MySQL UNSIGNED variants and parameterised types (e.g. TIMESTAMP(6)).
//
//nolint:gochecknoglobals // immutable lookup table
var columnTypePrefixes = []struct {
	prefix   string
	category string
}{
	{"UNSIGNED", columnTypeNumber},
	{"INT", columnTypeNumber},
	{"FLOAT", columnTypeNumber},
	{"TIMESTAMP", columnTypeDatetime},
}

// normalizeColumnType maps a driver-specific DatabaseTypeName() string to an
// abstract category used for result cell styling.
func normalizeColumnType(name string) string {
	upper := strings.ToUpper(strings.TrimSpace(name))
	if upper == "" {
		return columnTypeUnknown
	}

	if cat, ok := columnTypeExact[upper]; ok {
		return cat
	}

	for _, p := range columnTypePrefixes {
		if strings.HasPrefix(upper, p.prefix) {
			return p.category
		}
	}

	return columnTypeUnknown
}
