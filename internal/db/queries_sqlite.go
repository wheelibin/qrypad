package db

import (
	"context"
	"fmt"
)

// sqliteQueries implements QueryProvider for SQLite.
type sqliteQueries struct{}

func (sqliteQueries) Databases() string {
	// SQLite is a single-file database; there are no schemas to list.
	return ""
}

func (sqliteQueries) SchemaTables() string {
	return `SELECT name FROM sqlite_master
				WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`
}

func (sqliteQueries) SchemaViews() string {
	return `SELECT name
                        FROM sqlite_master
                        WHERE type = 'view'
                            AND name NOT LIKE 'sqlite_%'
                        ORDER BY name;`
}

func (sqliteQueries) TableColumns(ref TableReference) string {
	return fmt.Sprintf(`SELECT * FROM pragma_table_info('%s');`, ref.Name)
}

func (sqliteQueries) TableConstraints(ref TableReference) string {
	return fmt.Sprintf(`PRAGMA foreign_key_list('%s');`, ref.Name)
}

func (sqliteQueries) TableRows(ref TableReference, primaryKeyColumns []string, sortOrder string) string {
	return buildTableRowsSQL(quotedTableRefForDriver(DriverName.SQLite, ref), primaryKeyColumns, sortOrder)
}

func (sqliteQueries) TableIndexes(ctx context.Context, dbConn DBConn, ref TableReference) (*Data, error) {
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

func (s sqliteQueries) PrimaryKeyColumns(ctx context.Context, dbConn DBConn, ref TableReference) ([]string, error) {
	return fetchPrimaryKeyColumns(ctx, dbConn, s.primaryKeyColumnsSQL(ref))
}

func (sqliteQueries) primaryKeyColumnsSQL(ref TableReference) string {
	return fmt.Sprintf(`SELECT name
                                        FROM pragma_table_info('%s')
                                        WHERE pk > 0
                                        ORDER BY pk;`, ref.Name)
}

var _ QueryProvider = sqliteQueries{}
