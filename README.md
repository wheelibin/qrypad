<p align="center">
  <img src="https://github.com/wheelibin/qrypad/blob/main/header.png" style="width:600px; height:auto;" />
</p>

## Features

### 🚀 Quick Exploration

- View a snapshot of the selected table with one keypress
- Automatically display columns and indexes for the current table
- Filter result sets interactively to refine your queries

### ✍️ Query Editing

- Write and manage multiple queries in the query pad
- Run the current statement with a single keypress
- Save queries per connection for easy reuse
- Syntax-highlighted query pad for better readability

### 🔄 Connection Management

- Switch databases on the current connection
- Securely store passwords in the OS keychain

<p align="center">
  <img src="https://github.com/wheelibin/qrypad/blob/main/ss.png" style="max-width:100%; height:auto;"/>
</p>

## Installation

### Go Install

`go install github.com/wheelibin/qrypad@latest`

### Binary Install

https://github.com/wheelibin/qrypad/releases

## Usage

`qrypad [connection name]`

- `[connection name]` must match an entry in the config file (see below)
- Passwords will be prompted once, then stored securely

## Config

Config is read from `~/.config/qrypad/config.toml`

### example config file

```toml
# the timeout for all queries
queryTimeout = 60 

# the max number of rows to fetch when viewing table data (does not apply to ad-hoc queries)
tableDataRowLimit = 100

# color theme 
[theme] 
name = "catppuccin-mocha"

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

## ⌨️ Key Bindings

<details>
    <summary>click to view</summary>
### 🧭 General

- `Tab` / `Shift+Tab` - switch panels
- `F1` - show help
- `F2` - switch database
- `F3` - update stored password
- `/` - filter tables (`esc` to cancel)
- `Ctrl+T` - toggle left-side (tables/info)

### 📋 Tables Panel

- `Enter` - fetch first 100 rows
- `]` / `[` - switch tabs
- `c` - copy selected table name

### 🛠 Table Info Panel

- `]` / `[` - switch tabs
- `c` - copy selected column/index name
- `/` - filter columns (`esc` to cancel)

### 🧾 Queries Panel

- `F5` - run current query
- `Ctrl+S` - save query pad (saved per connection)
- `Ctrl+R` - reload saved query pad file
- `Ctrl+E` - open query pad in external editor

### 📊 Results Panel

- `enter` - show full row data in popup
  - `c` - copy selected value
- `c` - copy selected row as JSON
- `/` - filter results (`esc` to cancel)

</details>

## Themes

### Built-in Themes

There are currently the following built-in themes:

- `catppuccin-mocha` (the default)
- `kanagawa-wave`
- `rose-pine-moon`

You can change the theme using the following config key

```toml
[theme] 
name = "kanagawa-wave"
```

### Customising / Creating Themes

You can customise an existing theme by overriding individual colours.

```toml
[theme]
name                  = "rose-pine-moon"
borderActive          = { fg = "#ff00ff" }
```

To create a new theme, simply give it a new name and set the colours.

```toml
[theme]
name                  = "my-custom-theme"
borderActive          = { bg = "", fg = "#ff00ff" }
currentStatement      = { bg = "", fg = "" }
databaseSwitcherPopup = { bg = "", fg = "" }
error                 = { bg = "", fg = "" }
helpPopup             = { bg = "", fg = "" }
helpKey               = { bg = "", fg = "" }
helpDesc              = { bg = "", fg = "" }
panelTitle            = { bg = "", fg = "" }
panelTitleActive      = { bg = "", fg = "" }
rowDetailsPopup       = { bg = "", fg = "" }
spinner               = { bg = "", fg = "" }
statusBar             = { bg = "", fg = "" }
tableBorder           = { bg = "", fg = "" }
tableHeader           = { bg = "", fg = "" }
text                  = { bg = "", fg = "" }
titleBar              = { bg = "", fg = "" }
```
