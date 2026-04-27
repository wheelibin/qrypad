package db

import (
	"context"
	"fmt"
)

// mysqlDefaultSchemaFilter is the SQL predicate used when no explicit schema is specified.
const mysqlDefaultSchemaFilter = "TABLE_SCHEMA = DATABASE()"

// mysqlQueries implements QueryProvider for MySQL.
type mysqlQueries struct{}

func (mysqlQueries) Databases() string {
	return `SELECT SCHEMA_NAME name FROM information_schema.SCHEMATA 
						WHERE SCHEMA_NAME NOT IN ('mysql', 'performance_schema', 'sys') 
						ORDER BY name;`
}

func (mysqlQueries) SchemaTables() string {
	return `SELECT TABLE_SCHEMA ` + "`schema`" + `, TABLE_NAME name, format(TABLE_ROWS,0) 'rows'
            FROM information_schema.TABLES
            WHERE TABLE_SCHEMA NOT IN ('mysql', 'performance_schema', 'sys')
             AND TABLE_TYPE = 'BASE TABLE'
            ORDER BY table_schema, name;`
}

func (mysqlQueries) SchemaViews() string {
	return `SELECT table_schema ` + "`schema`" + `, table_name name
                        FROM information_schema.views
                        WHERE table_schema NOT IN ('mysql', 'performance_schema', 'information_schema', 'sys')
                        ORDER BY table_schema, table_name;`
}

func (mysqlQueries) TableColumns(ref TableReference) string {
	schemaFilter := mysqlDefaultSchemaFilter
	if ref.Schema != "" {
		schemaFilter = fmt.Sprintf("TABLE_SCHEMA = '%s'", ref.Schema)
	}
	return fmt.Sprintf(`SELECT column_name name, data_type type, case when is_nullable = 'NO' then 'NOT NULL' else 'NULL' end nullable
                                        FROM INFORMATION_SCHEMA.COLUMNS
                                        WHERE %s AND TABLE_NAME = '%s' ORDER BY column_name;`,
		schemaFilter, ref.Name)
}

func (mysqlQueries) TableConstraints(ref TableReference) string {
	schemaFilter := mysqlDefaultSchemaFilter
	if ref.Schema != "" {
		schemaFilter = fmt.Sprintf("TABLE_SCHEMA = '%s'", ref.Schema)
	}
	return fmt.Sprintf(`SELECT
                                        CONSTRAINT_NAME 'name',
                                        CONSTRAINT_TYPE 'type'
                                    FROM
                                        INFORMATION_SCHEMA.TABLE_CONSTRAINTS
                                    WHERE
                                        %s
                                        AND TABLE_NAME = '%s';`, schemaFilter, ref.Name)
}

func (mysqlQueries) TableRows(ref TableReference, primaryKeyColumns []string, sortOrder string) string {
	return buildTableRowsSQL(quotedTableRefForDriver(DriverName.MySQL, ref), primaryKeyColumns, sortOrder)
}

func (m mysqlQueries) TableIndexes(ctx context.Context, dbConn DBConn, ref TableReference) (*Data, error) {
	return fetchRows(ctx, dbConn, m.tableIndexesSQL(ref))
}

func (m mysqlQueries) PrimaryKeyColumns(ctx context.Context, dbConn DBConn, ref TableReference) ([]string, error) {
	return fetchPrimaryKeyColumns(ctx, dbConn, m.primaryKeyColumnsSQL(ref))
}

func (mysqlQueries) tableIndexesSQL(ref TableReference) string {
	schemaFilter := mysqlDefaultSchemaFilter
	if ref.Schema != "" {
		schemaFilter = fmt.Sprintf("TABLE_SCHEMA = '%s'", ref.Schema)
	}
	return fmt.Sprintf(`SELECT
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
}

func (mysqlQueries) primaryKeyColumnsSQL(ref TableReference) string {
	schemaFilter := mysqlDefaultSchemaFilter
	if ref.Schema != "" {
		schemaFilter = fmt.Sprintf("TABLE_SCHEMA = '%s'", ref.Schema)
	}
	return fmt.Sprintf(`SELECT COLUMN_NAME name
                                        FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
                                        WHERE %s
                                            AND TABLE_NAME = '%s'
                                            AND CONSTRAINT_NAME = 'PRIMARY'
                                        ORDER BY ORDINAL_POSITION;`, schemaFilter, ref.Name)
}

var _ QueryProvider = mysqlQueries{}
