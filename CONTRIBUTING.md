# Contributing

## Running the Tests

```bash
make test
```

Runs fast unit tests against all packages; no Docker required. This is what CI runs for its unit test job.

### Integration Tests

For end-to-end verification against real databases (Postgres, MySQL, SQLite):

```bash
make integration-test
```

This starts Postgres and MySQL via Docker Compose (using the files in `test-db/`), seeds SQLite, runs all integration tests, and tears down on success. On test failure, containers are left running for debugging — run `make integration-down` to clean up.

Local ports used:
- Postgres: `50403`
- MySQL: `50306` (non-standard to avoid conflicts with locally-installed MySQL)
- SQLite: `test-db/sqlite/sqlite.db` (file-based)

Integration tests are gated by the `//go:build integration` build tag, so `go test ./...` ignores them by default. CI runs both the unit tests and integration tests on every push and PR.

## View Tests and Golden Files

The component view tests in `internal/component/` use golden files to snapshot each component's rendered output. This catches visual regressions when upgrading rendering libraries (bubbletea, lipgloss, bubbles).

Golden files are stored in `internal/component/testdata/` and must be committed alongside any change that affects component rendering.

**If you make a change that intentionally affects how a component looks**, regenerate the golden files:

```bash
go test ./internal/component/ -run "Test.*View" -update
```

Review the diff (`git diff internal/component/testdata/`) to confirm only the expected components changed, then commit the updated golden files with your change.

**When adding a new component**, create a `<componentName>_view_test.go` file alongside it, following the pattern in the existing view test files, then run the above command to generate its golden file.

## Architecture Overview

`main.go` bootstraps config and starts a Bubble Tea program. The root model lives in `internal/ui` and owns all component sub-models. Components live in `internal/component` — each is a self-contained Bubble Tea model implementing `Init`, `Update`, and `View`.

Communication is message-driven. Components return `tea.Cmd` values that resolve to typed messages. UI/UX message types are defined in `internal/commands/messages.go`; database message types in `internal/db/messages.go`. `ui.Update` receives all messages and dispatches them to the appropriate handler or sub-model.

The four main panels (tables, table info, query, results) are always rendered; `ui.Model` tracks which is active. Popups are rendered conditionally via `activePopup` — only one can be shown at a time and it captures all input while visible. Supporting packages: `internal/db` (database abstraction), `internal/keys` (key bindings), `internal/theme` / `internal/style` (theming and layout).

## Adding a New Component

1. Create `internal/component/<name>.go` — implement `Model` struct, `New<Name>Model()` constructor, `Init`, `Update`, `View`.
2. Add any new message types to `internal/commands/messages.go` or `internal/db/messages.go`.
3. Wire into `internal/ui/ui.go` — add field to `model`, instantiate in `NewModel`, call `Init` in `ui.Init`, update in `ui.Update`, render in `ui.View`.
4. Add `internal/component/<name>_view_test.go` following the existing pattern, generate the golden file with `go test ./internal/component/ -run "Test.*View" -update`, and commit it.
