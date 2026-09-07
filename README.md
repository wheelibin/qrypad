<p align="center">
  <img src="assets/header.png" style="width:600px; height:auto;" />
</p>

<p align="center">A terminal SQL client for Postgres, MySQL and SQLite.</p>

<p align="center">
  <img src="assets/ss.png" style="max-width:100%; height:auto;"/>
</p>

## Features

**Write queries**

- Autocomplete tables and columns in the query pad
- SQL syntax highlighting
- Run only the statement under the cursor — no selection needed
- Save and reload a query pad per connection, or open it in `$EDITOR`

**Explore your schema**

- Browse tables, views, columns, indexes, and constraints
- View any table's data with a single keypress
- Filter tables and columns instantly

**Work with results**

- Filter result sets on the fly
- Inspect any row in a popup, with long values wrapped across lines
- Export to JSON or CSV

**Manage connections**

- Switch connections without restarting
- Switch databases on the current connection (Postgres / MySQL)
- Keep multiple connections open as sessions and flip between them instantly
- Passwords stored securely in the OS keychain

**Customise**

- Configurable key bindings
- Built-in themes (kanagawa, catppuccin, rose-pine) with full colour customisation

## Installation

### Binary

https://github.com/wheelibin/qrypad/releases

### Go install

```
go install github.com/wheelibin/qrypad@latest
```

### Nix

Run without installing:

```bash
nix run github:wheelibin/qrypad
```

Install into your profile:

```bash
nix profile install github:wheelibin/qrypad
```

Or add to your `flake.nix`:

```nix
inputs.qrypad.url = "github:wheelibin/qrypad";

# then reference the package as:
inputs.qrypad.packages.${system}.default
```

## Usage

```
qrypad
qrypad --connection <connection name>
```

`--connection` must match an entry in the config file. If omitted, you will be prompted to choose a connection on startup.

If the connection requires a password you will be prompted on first use; it is then stored in the OS keychain.

You can also switch connections from inside the app with `Ctrl+K`.

## Config

Config is read from `~/.config/qrypad/config.toml`.

### Example

```toml
# query timeout (seconds)
queryTimeout = 60

# max rows fetched when viewing table data (does not apply to ad-hoc queries)
tableDataRowLimit = 100

# show horizontal borders between table rows
rowBorders = true

[theme]
name = "catppuccin"

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

[connections.orders]
driver = "sqlite"
database = "db/orders.db"

[connections.warehouse]
driver = "postgres"
host = "localhost"
port = 5432
user = "postgres"
database = "warehouse"
# share a single query pad across every database on this connection,
# instead of one query pad per database (SQLite connections always
# behave this way, regardless of this setting)
singleQueryFile = true
```

## Sessions

Sessions let you keep multiple database connections open at the same time and switch between them without losing your place. Each session remembers the connection, selected database, query text, and last results.

Sessions are created automatically -- every time you switch to a different connection, the current state is saved as a session in the background. When you switch back, everything is restored exactly as you left it, including the live database connection (no reconnect needed).

- `Ctrl+L` -- open the session list (type to filter, `Enter` to switch, `x` to close a session)
- `Ctrl+T` -- toggle back to the previous session (like Alt-Tab for connections)
- `Ctrl+K` -- open a new connection (the current one becomes a background session)

The status bar shows `[N sessions]` when more than one connection is open.

## Key bindings

<details>
    <summary>Default key bindings</summary>

### General

- `Tab` / `Shift+Tab` — switch panels
- `?` / `F1` — show help
- `Ctrl+C` — quit
- `Ctrl+D` — switch database
- `Ctrl+K` — switch connection
- `Ctrl+L` — open session list
- `Ctrl+T` — toggle to previous session
- `Ctrl+P` — update stored password
- `Ctrl+B` — toggle the left (tables / info) panel
- `R` — refresh schema
- `/` — filter tables (`esc` to cancel)

### Tables panel

- `Enter` — fetch first N rows (`tableDataRowLimit`)
- `D` — fetch first N rows, descending
- `]` / `[` — switch tabs
- `y` — copy the selected table name

### Table info panel

- `]` / `[` — switch tabs
- `y` — copy the selected column / index name
- `/` — filter columns (`esc` to cancel)

### Query panel

- `F5` — run the statement at the cursor
- `Ctrl+Space` — autocomplete table / column
- `Esc` — cancel a running query
- `Ctrl+S` — save the query pad (per connection)
- `Ctrl+R` — reload the saved query pad from disk
- `Ctrl+E` — open the query pad in `$EDITOR`
- `Ctrl+Z` — undo
- `Ctrl+Y` — redo

### Results panel

- `Enter` — open the selected row in a popup
  - `y` — copy the selected value
- `y` — copy the selected row as JSON
- `Y` — copy all results as JSON
- `Ctrl+X` — export results to a file
- `/` — filter results (`esc` to cancel)

</details>

### Overriding key bindings

Any of the keys below can be overridden in the config file.

```toml
[keys]
AutoComplete     = ""
CancelQuery      = ""
CopyResults      = ""
CopyValue        = ""
ExecuteQuery     = ""
ExportResults    = ""
FilterTable      = ""
Help             = ""
NextPanel        = ""
NextTab          = ""
OpenInEditor     = ""
PrevPanel        = ""
PrevTab          = ""
Quit             = ""
Redo             = ""
RefreshSchema    = ""
ReloadQuery      = ""
SaveQuery        = ""
SessionList      = ""
SessionToggle    = ""
SwitchConnection = ""
SwitchDatabase   = ""
ToggleLeftPanel  = ""
Undo             = ""
UpdatePassword   = ""
ViewData         = ""
ViewDataDesc     = ""
```

## Themes

### Built-in themes

- `kanagawa` (default)
- `catppuccin`
- `rose-pine`

Set the active theme in the config:

```toml
[theme]
name = "catppuccin"
```

### Customising themes

Override individual colours on top of an existing theme:

```toml
[theme]
name         = "catppuccin"
borderActive = { fg = "#ff00ff" }
```

Or define a new theme from scratch by giving it a new name and setting the colours:

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
titleBarAlt           = { bg = "", fg = "" }
syntaxKeyword         = { fg = "" }
syntaxString          = { fg = "" }
syntaxNumber          = { fg = "" }
syntaxComment         = { fg = "" }
syntaxOperator        = { fg = "" }
syntaxName            = { fg = "" }
syntaxLiteral         = { fg = "" }
syntaxPunctuation     = { fg = "" }
```

The `syntax*` keys control SQL syntax highlighting colours. If omitted, they fall back to colours derived from the UI theme.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Released under the [MIT License](LICENSE).
