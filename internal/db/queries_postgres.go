package db

import (
	"context"
	"fmt"
)

// postgresQueries implements QueryProvider for PostgreSQL.
type postgresQueries struct{}

func (postgresQueries) Databases() string {
	return `SELECT datname name
						FROM pg_database
						WHERE has_database_privilege(datname, 'CONNECT')
							AND NOT datistemplate
						ORDER BY datname;`
}

func (postgresQueries) SchemaTables() string {
	return `SELECT schemaname schema, relname name, TO_CHAR(n_live_tup, 'FM999,999,999') rows
          FROM pg_stat_user_tables
          ORDER BY schema, name;`
}

func (postgresQueries) SchemaViews() string {
	return `SELECT table_schema schema, table_name name
                        FROM information_schema.views
                        WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
                        ORDER BY schema, table_name;`
}

func (postgresQueries) TableColumns(ref TableReference) string {
	schemaFilter := "table_schema = current_schema()"
	if ref.Schema != "" {
		schemaFilter = fmt.Sprintf("table_schema = '%s'", ref.Schema)
	}
	return fmt.Sprintf(`SELECT column_name name, data_type type, case when is_nullable = 'NO' then 'NOT NULL' else 'NULL' end nullable
                                        FROM INFORMATION_SCHEMA.COLUMNS
                                        WHERE %s AND TABLE_NAME = '%s' ORDER BY column_name;`,
		schemaFilter, ref.Name)
}

func (postgresQueries) TableConstraints(ref TableReference) string {
	schemaJoin := ""
	schemaWhere := ""
	if ref.Schema != "" {
		schemaJoin = "JOIN pg_namespace n ON c.connamespace = n.oid"
		schemaWhere = fmt.Sprintf("AND n.nspname = '%s'", ref.Schema)
	}
	return fmt.Sprintf(`SELECT
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
}

func (postgresQueries) TableRows(ref TableReference, primaryKeyColumns []string, sortOrder string) string {
	return buildTableRowsSQL(quotedTableRefForDriver(DriverName.Postgres, ref), primaryKeyColumns, sortOrder)
}

func (p postgresQueries) TableIndexes(ctx context.Context, dbConn DBConn, ref TableReference) (*Data, error) {
	return fetchRows(ctx, dbConn, p.tableIndexesSQL(ref))
}

func (p postgresQueries) PrimaryKeyColumns(ctx context.Context, dbConn DBConn, ref TableReference) ([]string, error) {
	return fetchPrimaryKeyColumns(ctx, dbConn, p.primaryKeyColumnsSQL(ref))
}

func (postgresQueries) tableIndexesSQL(ref TableReference) string {
	tableFilter := fmt.Sprintf("t.relname like '%s'", ref.Name)
	if ref.Schema != "" {
		tableFilter = fmt.Sprintf("t.relname like '%s' AND n.nspname = '%s'", ref.Name, ref.Schema)
	}
	return fmt.Sprintf(`select
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
}

func (postgresQueries) primaryKeyColumnsSQL(ref TableReference) string {
	qualifiedRef := quotedTableRefForDriver(DriverName.Postgres, ref)
	return fmt.Sprintf(`SELECT a.attname name
                                        FROM pg_index i
                                        JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
                                        WHERE i.indrelid = '%s'::regclass AND i.indisprimary;`, qualifiedRef)
}

var _ QueryProvider = postgresQueries{}
