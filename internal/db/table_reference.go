package db

// TableReference identifies a table or view by its optional schema and name.
// Schema is always empty for SQLite (which has no schema concept).
type TableReference struct {
	Schema string
	Name   string
}

// QualifiedName returns "schema.name" when Schema is non-empty, otherwise "name".
func (t TableReference) QualifiedName() string {
	if t.Schema == "" {
		return t.Name
	}
	return t.Schema + "." + t.Name
}

// IsDefaultSchema reports whether Schema is the driver's default schema.
//
// Rules:
//   - Postgres: default is "public"
//   - MySQL: default is the connected database name
//   - SQLite: always true (Schema is always "")
func (t TableReference) IsDefaultSchema(driver DriverNameType, connectedDB string) bool {
	switch driver {
	case DriverName.Postgres:
		return t.Schema == "public"
	case DriverName.MySQL:
		return t.Schema == connectedDB
	default: // SQLite and unknown drivers
		return true
	}
}

// AutocompleteInsert returns the string to insert into the query buffer.
// It returns just Name when IsDefaultSchema is true, or QualifiedName otherwise.
func (t TableReference) AutocompleteInsert(driver DriverNameType, connectedDB string) string {
	if t.IsDefaultSchema(driver, connectedDB) {
		return t.Name
	}
	return t.QualifiedName()
}
