# DBee - A simple scratchpad for running ad-hoc database queries

DBee is a basic terminal application for running ad-hoc queries against a (mysql / postgres) database.

It has the following features:
- view a list of the tables in the database along with the column info for the selected table
- quickly view table data without writing sql
- keep one or more queries in the query panel and easily run the query under the cursor (queries are saved per database)

> If you want to browse the table relationships, edit columns, add indexes, or really anything other than running a query, then you need to use another tool. 

![DBee screenshot](https://github.com/wheelibin/dbee/blob/main/dbee.png?raw=true)

## Usage

`dbee [database alias]`

The database alias must match the name of a database configuration in your config file.

## Installation

`go install github.com/wheelibin/dbee@latest`

## Config

Config is read from `~/.config/dbee/dbee.toml`

### example config file
```markdown
# the timeout for all queries
queryTimeout = 60 

# the max number of rows to fetch when viewing table data (does not apply to ad-hoc queries)
tableDataRowLimit = 100

[databases]

[databases.animals]
driver = "mysql"
host = "localhost"
port = 3306
user = "root"
password = "123456"
database = "animals.0"

[databases.music]
driver = "postgres"
host = "localhost"
port = 5432
user = "postgres"
password = "123456"
database = "music-store"

```

