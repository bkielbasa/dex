# dex

A vim-inspired TUI database explorer. Like a Pokedex -- it catalogs everything.

## Installation

```bash
go install github.com/bklimczak/dex@latest
```

## Configuration

Create `~/.dex/connections.yaml`:

```yaml
default: local-pg

connections:
  - name: local-pg
    engine: postgres
    host: localhost
    port: 5432
    user: postgres
    password: ${POSTGRES_PASSWORD}
    database: mydb

  - name: local-mysql
    engine: mysql
    host: localhost
    port: 3306
    user: root
    password: ${MYSQL_PASSWORD}
    database: mydb
```

- `default` — auto-connect on startup
- Passwords support `${ENV_VAR}` substitution

## Features

**Navigation** — three-pane layout (sidebar, results, query bar) with vim motions throughout.

**Connection management** — multiple connections, `:conn` picker, `:reload` to re-read config, `Ctrl+n` to add new, `1-9` quick-switch.

**Query execution** — inline query bar (`:`), full SQL editor (`Ctrl+e`), query history (`Ctrl+p/n`).

**Saved queries** — `:save <name>` to save current query per connection, `:queries` to browse with fuzzy search, `Ctrl+d` to delete.

**Results grid** — scrollable table with cell cursor, NULL highlighting, column width auto-sizing (capped at 40 chars).

**Inline cell editing** — `i`/`a` to edit a cell, `Enter` to save, `Esc` to cancel. Supports strings, integers, booleans, and NULL (type `NULL` or leave empty).

**Row detail view** — `Enter` opens a vertical column:value modal for the selected row. `j`/`k` to scroll, `Esc` to close.

**Column sorting** — `s` sorts by current column (server-side `ORDER BY`). Press again to toggle ascending/descending. Sort indicator shown in header.

**Server-side pagination** — table browsing uses `LIMIT 100 OFFSET N`. `n`/`p` to page. Status bar shows `page 1/12 | 1-100 of 1,150`.

**Column hide/show** — `c` hides current column, `C` shows all. View-only; export includes all columns.

**Export** — `:export csv [path]` or `:export json [path]`. Exports current result set. Defaults to `./tablename.csv`.

**Schema inspection** — `S` for CREATE TABLE, `D` for DESCRIBE/column info.

## Keybindings

### Global

| Key | Action |
|-----|--------|
| `Ctrl+w` then `h/j/k/l` | Focus left/down/up/right pane |
| `/` | Search/filter sidebar |
| `:` | Command bar |
| `Ctrl+e` | Full SQL editor |
| `Ctrl+n` | New connection form |
| `Ctrl+p/n` | Query history prev/next |
| `1-9` | Quick-switch connections |
| `q` | Quit |

### Results Grid

| Key | Action |
|-----|--------|
| `j/k` | Move cursor up/down |
| `h/l` | Move cursor left/right |
| `g/G` | Jump to first/last row |
| `Enter` | Open row detail view |
| `i` or `a` | Edit cell |
| `s` | Sort by current column |
| `n/p` | Next/previous page |
| `c` | Hide current column |
| `C` | Show all columns |
| `S` | View table schema |
| `D` | Describe table |

### Commands

| Command | Action |
|---------|--------|
| `:conn` | Open connection picker |
| `:reload` | Re-read config file |
| `:save <name>` | Save current query |
| `:queries` | Browse saved queries |
| `:export csv [path]` | Export as CSV |
| `:export json [path]` | Export as JSON |
| Any SQL | Execute directly |

## Supported Engines

- PostgreSQL
- MySQL

## License

MIT
