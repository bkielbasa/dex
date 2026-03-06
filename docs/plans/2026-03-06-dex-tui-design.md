# Dex - Database Explorer TUI

A vim-inspired TUI for browsing, querying, and inspecting databases. Like a Pokedex — it catalogs everything.

## Layout

Three-pane layout with modal overlays:

```
+-----------+------------------------------+
| Sidebar   |  Results / Data View         |
|           |                              |
| [conns]   |  (scrollable table grid)     |
|  +- db1   |                              |
|    +- tbl |                              |
|  +- db2   |                              |
|           |                              |
|           +------------------------------+
|           |  Query Bar (single line)     |
+-----------+------------------------------+
```

- **Sidebar:** Tree view of connections > tables. Expand/collapse with Enter.
- **Results pane:** Query output or table data preview (SELECT * LIMIT 100).
- **Query bar:** Single-line SQL input, focused with `:`.

### Modal Overlays

- Full SQL editor: `Ctrl+e` (multi-line, query history)
- Connection manager: `Ctrl+n` (add), `Ctrl+d` (manage)
- Table schema: `S` on selected table

## Navigation

| Key              | Action                          |
|------------------|---------------------------------|
| Tab / Shift+Tab  | Cycle pane focus                |
| Ctrl+h/l         | Focus left/right pane           |
| j/k              | Move up/down in lists/results   |
| g/G              | Top/bottom of list              |
| /                | Search/filter in current pane   |
| :                | Focus query bar                 |
| Ctrl+e           | Open full SQL editor            |
| Enter            | Expand tree node / execute query|
| Esc              | Close modal / unfocus           |
| q                | Quit                            |
| 1-9              | Quick-switch between connections|
| S                | Schema modal for selected table |
| D                | Describe table                  |

## Connection Management

Config file at `~/.dex/connections.yaml`:

```yaml
connections:
  - name: local-pg
    engine: postgres
    host: localhost
    port: 5432
    database: myapp_dev
    user: postgres
    password: ${PG_PASSWORD}
    ssl: false
```

- Env var expansion for passwords (`${VAR}` syntax)
- TUI form for add/edit/test/delete
- Warn if password stored as plaintext

## Data Viewing

- Select table in sidebar -> auto `SELECT * FROM table LIMIT 100`
- Scrollable grid with pinned column headers (h/j/k/l)
- Page through with n/p
- Errors shown inline in red

## Query Execution

- Query bar (`:`) -> single line, Enter to run
- Full editor (`Ctrl+e`) -> multi-line, Ctrl+Enter to run
- History: Ctrl+p / Ctrl+n to cycle (in-memory, per session)

## Architecture

```
dex/
+-- main.go
+-- go.mod
+-- internal/
|   +-- app/app.go            # root bubbletea model
|   +-- ui/
|   |   +-- sidebar/          # tree view
|   |   +-- results/          # data grid
|   |   +-- querybar/         # single-line input
|   |   +-- editor/           # full SQL editor modal
|   |   +-- connform/         # connection form modal
|   |   +-- schema/           # table schema modal
|   |   +-- styles/           # shared lipgloss styles
|   +-- db/
|   |   +-- engine.go         # Engine interface
|   |   +-- postgres.go       # postgres impl
|   |   +-- mysql.go          # mysql impl
|   |   +-- registry.go       # active connections
|   +-- config/config.go      # YAML load/save, env var expansion
|   +-- keymap/keymap.go      # centralized key bindings
```

### DB Engine Interface

```go
type Engine interface {
    Connect(cfg ConnectionConfig) error
    Close() error
    Tables(database string) ([]string, error)
    Databases() ([]string, error)
    Schema(table string) (*TableSchema, error)
    Execute(query string) (*QueryResult, error)
}
```

### Key Dependencies

- charmbracelet/bubbletea, bubbles, lipgloss
- lib/pq (Postgres), go-sql-driver/mysql
- database/sql (stdlib)
- gopkg.in/yaml.v3

### State Flow

- App model holds focused pane, active connection, child model states
- Pane models communicate via bubbletea messages (TableSelectedMsg, QueryResultMsg, etc.)
- Modals layered on top — app intercepts keys when modal is open
- All DB operations run async via bubbletea commands (non-blocking UI)

## MVP Scope (v0.1)

**Included:**
- Three-pane layout with sidebar, results, query bar
- Config file loading (~/.dex/connections.yaml)
- Connection form modal (add/edit/test/delete)
- Postgres + MySQL engines
- Sidebar tree: connections > tables
- Table select -> auto-preview
- Query bar + full editor modal
- Results grid with vim-style scrolling, pinned headers
- Schema modal
- All keybindings from nav table
- Query history (in-memory, per session)

**Deferred:**
- Persistent query history
- CSV/JSON export
- Sort/filter in results grid
- Multi-database browsing per connection
- Syntax highlighting in editor
- Password keyring integration
- Additional engines (SQLite, etc.)
