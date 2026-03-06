# dex

A vim-inspired TUI database explorer. Like a Pokedex -- it catalogs everything.

## Installation

```bash
go install github.com/bklimczak/dex@latest
```

## Configuration

Create `~/.dex/connections.yaml`:

```yaml
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

Passwords support `${ENV_VAR}` substitution so you can avoid storing secrets in plain text.

## Keybindings

| Key | Action |
|-----|--------|
| Tab / Shift+Tab | Cycle pane focus |
| Ctrl+h/l | Focus left/right pane |
| j/k | Move up/down in lists/results |
| h/l | Scroll columns in results |
| g/G | Top/bottom of list |
| / | Search/filter in sidebar |
| : | Focus query bar |
| Ctrl+e | Open full SQL editor |
| Ctrl+n | Add new connection |
| Enter | Expand tree node / execute query |
| Esc | Close modal / unfocus |
| S | View table schema |
| D | Describe table |
| 1-9 | Quick-switch between connections |
| Ctrl+p/n | Cycle query history |
| q | Quit |

## Supported Engines

- PostgreSQL
- MySQL

## License

MIT
