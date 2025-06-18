# QryPad - A simple client for quick, ad-hoc database exploration 

<!-- <p align="center"> -->
<!--   <img src="https://github.com/wheelibin/qrypad/blob/main/icon.png" height="100" /> -->
<!-- </p> -->

QryPad is a simple client for quick, ad-hoc database exploration of mysql and postgres databases.

It has the following features:
- view a list of the tables in the database along with the column info for the selected table
- quickly view table data without writing sql
- keep one or more queries in the query panel and easily run the query under the cursor (queries are saved per database)

> If you want to browse the table relationships, edit columns, add indexes, or really anything other than running a query, then you need to use another tool. 



<p align="center">
  <img src="https://github.com/wheelibin/qrypad/blob/main/ss.png" />
</p>

## Usage

`qrypad [connection name]`

The database alias must match the name of a database configuration in your config file.

## Installation

`go install github.com/wheelibin/qrypad@latest`

## Config

Config is read from `~/.config/qrypad/config.toml`

### example config file
```markdown
# the timeout for all queries
queryTimeout = 60 

# the max number of rows to fetch when viewing table data (does not apply to ad-hoc queries)
tableDataRowLimit = 100

[connections]

[connections.animals]
driver = "mysql"
host = "localhost"
port = 3306
user = "root"
database = "animals.0"

[connections.music]
driver = "postgres"
host = "localhost"
port = 5432
user = "postgres"
database = "music-store"

```

## Keys

Configurable key map is coming soon, but for now the default keys are:

### general
- `tab` / `shift+tab` to navigate between panels
- `ctrl+t` toggle tables
- `/` to filter in the tables, table info, and results panel (`esc` to cancel) 

### table panel
- `enter` to fetch the first 100 rows of the selected table

### query panel
- `F5` to run the query under the cursor
- `ctrl+s` to save the query (buffer is saved per db)
- `ctrl+r` to reload the query file from disk




