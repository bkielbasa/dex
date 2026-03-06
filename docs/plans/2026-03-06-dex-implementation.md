# Dex TUI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a vim-inspired TUI database explorer in Go with bubbletea, supporting Postgres and MySQL.

**Architecture:** Three-pane layout (sidebar tree, results grid, query bar) with modal overlays for connection management, SQL editor, and schema inspection. DB engines implement a common interface; all DB ops run async via bubbletea commands.

**Tech Stack:** Go, charmbracelet/bubbletea+bubbles+lipgloss, database/sql with lib/pq and go-sql-driver/mysql, gopkg.in/yaml.v3

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/app/app.go`

**Step 1: Initialize Go module and install dependencies**

Run:
```bash
cd /Users/bklimczak/Projects/dex
go mod init github.com/bklimczak/dex
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/bubbles@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/lib/pq@latest
go get github.com/go-sql-driver/mysql@latest
go get gopkg.in/yaml.v3@latest
```

**Step 2: Create minimal main.go**

```go
package main

import (
	"fmt"
	"os"

	"github.com/bklimczak/dex/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(app.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 3: Create minimal app model**

```go
// internal/app/app.go
package app

import tea "github.com/charmbracelet/bubbletea"

type Model struct{}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	return "dex - database explorer\n\nPress q to quit."
}
```

**Step 4: Verify it compiles and runs**

Run: `go build -o dex . && echo "Build OK"`
Expected: `Build OK`

**Step 5: Commit**

```bash
git add go.mod go.sum main.go internal/app/app.go
git commit -m "feat: project scaffolding with minimal bubbletea app"
```

---

### Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write failing tests**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.yaml")
	content := `connections:
  - name: test-pg
    engine: postgres
    host: localhost
    port: 5432
    database: testdb
    user: testuser
    password: testpass
    ssl: false
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(cfg.Connections))
	}
	c := cfg.Connections[0]
	if c.Name != "test-pg" {
		t.Errorf("expected name 'test-pg', got '%s'", c.Name)
	}
	if c.Engine != "postgres" {
		t.Errorf("expected engine 'postgres', got '%s'", c.Engine)
	}
	if c.Port != 5432 {
		t.Errorf("expected port 5432, got %d", c.Port)
	}
}

func TestEnvVarExpansion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.yaml")
	os.Setenv("DEX_TEST_PASS", "secret123")
	defer os.Unsetenv("DEX_TEST_PASS")

	content := `connections:
  - name: test-pg
    engine: postgres
    host: localhost
    port: 5432
    database: testdb
    user: testuser
    password: ${DEX_TEST_PASS}
    ssl: false
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Connections[0].Password != "secret123" {
		t.Errorf("expected password 'secret123', got '%s'", cfg.Connections[0].Password)
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connections.yaml")

	cfg := &Config{
		Connections: []Connection{
			{
				Name:     "new-db",
				Engine:   "mysql",
				Host:     "localhost",
				Port:     3306,
				Database: "mydb",
				User:     "root",
				Password: "",
				SSL:      false,
			},
		},
	}

	err := Save(path, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error loading saved config: %v", err)
	}
	if len(loaded.Connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(loaded.Connections))
	}
	if loaded.Connections[0].Name != "new-db" {
		t.Errorf("expected name 'new-db', got '%s'", loaded.Connections[0].Name)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path.yaml")
	if err != nil {
		t.Fatalf("missing file should return empty config, got error: %v", err)
	}
	if len(cfg.Connections) != 0 {
		t.Errorf("expected 0 connections, got %d", len(cfg.Connections))
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -v`
Expected: FAIL (package doesn't exist yet)

**Step 3: Implement config package**

```go
// internal/config/config.go
package config

import (
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type Connection struct {
	Name     string `yaml:"name"`
	Engine   string `yaml:"engine"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSL      bool   `yaml:"ssl"`
}

type Config struct {
	Connections []Connection `yaml:"connections"`
}

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

func expandEnvVars(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		varName := envVarPattern.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return match
	})
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	expanded := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return home + "/.dex/connections.yaml"
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -v`
Expected: All 4 tests PASS

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: config package with YAML load/save and env var expansion"
```

---

### Task 3: DB Engine Interface & Types

**Files:**
- Create: `internal/db/engine.go`

**Step 1: Define the engine interface and shared types**

```go
// internal/db/engine.go
package db

import "database/sql"

type ConnectionConfig struct {
	Name     string
	Engine   string
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSL      bool
}

type ColumnInfo struct {
	Name     string
	Type     string
	Nullable bool
}

type IndexInfo struct {
	Name    string
	Columns []string
	Unique  bool
}

type ForeignKey struct {
	Column         string
	RefTable       string
	RefColumn      string
	ConstraintName string
}

type TableSchema struct {
	Name        string
	Columns     []ColumnInfo
	Indexes     []IndexInfo
	ForeignKeys []ForeignKey
}

type QueryResult struct {
	Columns []string
	Rows    [][]string
	RowCount int
	Error   string
}

type Engine interface {
	Connect(cfg ConnectionConfig) error
	Close() error
	DB() *sql.DB
	Tables() ([]string, error)
	Schema(table string) (*TableSchema, error)
	Execute(query string) (*QueryResult, error)
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/db/`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/db/engine.go
git commit -m "feat: db engine interface and shared types"
```

---

### Task 4: Postgres Engine

**Files:**
- Create: `internal/db/postgres.go`
- Create: `internal/db/postgres_test.go`

Note: Tests in this task use a real Postgres connection. They'll be skipped if `DEX_TEST_POSTGRES_DSN` is not set. For local testing: `export DEX_TEST_POSTGRES_DSN="postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"`

**Step 1: Write integration tests (skip if no DB)**

```go
// internal/db/postgres_test.go
package db

import (
	"os"
	"testing"
)

func getPostgresTestConfig(t *testing.T) ConnectionConfig {
	dsn := os.Getenv("DEX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("DEX_TEST_POSTGRES_DSN not set, skipping postgres integration test")
	}
	// Parse from DSN or use defaults for testing
	return ConnectionConfig{
		Engine:   "postgres",
		Host:     "localhost",
		Port:     5432,
		Database: "postgres",
		User:     "postgres",
		Password: "postgres",
		SSL:      false,
	}
}

func TestPostgresConnect(t *testing.T) {
	cfg := getPostgresTestConfig(t)
	pg := &Postgres{}
	if err := pg.Connect(cfg); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pg.Close()
}

func TestPostgresTables(t *testing.T) {
	cfg := getPostgresTestConfig(t)
	pg := &Postgres{}
	if err := pg.Connect(cfg); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pg.Close()

	_, err := pg.Tables()
	if err != nil {
		t.Fatalf("failed to list tables: %v", err)
	}
}

func TestPostgresExecute(t *testing.T) {
	cfg := getPostgresTestConfig(t)
	pg := &Postgres{}
	if err := pg.Connect(cfg); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer pg.Close()

	result, err := pg.Execute("SELECT 1 AS num")
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}
	if len(result.Columns) != 1 || result.Columns[0] != "num" {
		t.Errorf("expected column 'num', got %v", result.Columns)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "1" {
		t.Errorf("expected row ['1'], got %v", result.Rows)
	}
}
```

**Step 2: Run tests to verify they fail (or skip)**

Run: `go test ./internal/db/ -v -run TestPostgres`
Expected: FAIL (Postgres type doesn't exist) or tests compile but skip

**Step 3: Implement Postgres engine**

```go
// internal/db/postgres.go
package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type Postgres struct {
	db *sql.DB
}

func (p *Postgres) Connect(cfg ConnectionConfig) error {
	sslmode := "disable"
	if cfg.SSL {
		sslmode = "require"
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, sslmode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}
	p.db = db
	return nil
}

func (p *Postgres) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *Postgres) DB() *sql.DB {
	return p.db
}

func (p *Postgres) Tables() ([]string, error) {
	rows, err := p.db.Query(`
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public'
		ORDER BY table_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func (p *Postgres) Schema(table string) (*TableSchema, error) {
	schema := &TableSchema{Name: table}

	// Columns
	colRows, err := p.db.Query(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position
	`, table)
	if err != nil {
		return nil, err
	}
	defer colRows.Close()

	for colRows.Next() {
		var col ColumnInfo
		var nullable string
		if err := colRows.Scan(&col.Name, &col.Type, &nullable); err != nil {
			return nil, err
		}
		col.Nullable = nullable == "YES"
		schema.Columns = append(schema.Columns, col)
	}
	if err := colRows.Err(); err != nil {
		return nil, err
	}

	// Indexes
	idxRows, err := p.db.Query(`
		SELECT i.relname, array_agg(a.attname ORDER BY k.n), ix.indisunique
		FROM pg_index ix
		JOIN pg_class t ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_namespace ns ON ns.oid = t.relnamespace
		JOIN LATERAL unnest(ix.indkey) WITH ORDINALITY AS k(attnum, n) ON true
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = k.attnum
		WHERE ns.nspname = 'public' AND t.relname = $1
		GROUP BY i.relname, ix.indisunique
		ORDER BY i.relname
	`, table)
	if err != nil {
		return nil, err
	}
	defer idxRows.Close()

	for idxRows.Next() {
		var idx IndexInfo
		var cols []string
		if err := idxRows.Scan(&idx.Name, (*StringArray)(&cols), &idx.Unique); err != nil {
			return nil, err
		}
		idx.Columns = cols
		schema.Indexes = append(schema.Indexes, idx)
	}
	if err := idxRows.Err(); err != nil {
		return nil, err
	}

	// Foreign keys
	fkRows, err := p.db.Query(`
		SELECT kcu.column_name, ccu.table_name, ccu.column_name, tc.constraint_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name AND ccu.table_schema = tc.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = 'public'
			AND tc.table_name = $1
	`, table)
	if err != nil {
		return nil, err
	}
	defer fkRows.Close()

	for fkRows.Next() {
		var fk ForeignKey
		if err := fkRows.Scan(&fk.Column, &fk.RefTable, &fk.RefColumn, &fk.ConstraintName); err != nil {
			return nil, err
		}
		schema.ForeignKeys = append(schema.ForeignKeys, fk)
	}

	return schema, fkRows.Err()
}

func (p *Postgres) Execute(query string) (*QueryResult, error) {
	rows, err := p.db.Query(query)
	if err != nil {
		return &QueryResult{Error: err.Error()}, nil
	}
	defer rows.Close()

	return scanRows(rows)
}
```

**Step 4: Create shared helpers (scanRows, StringArray)**

```go
// internal/db/helpers.go
package db

import (
	"database/sql"
	"fmt"
	"strings"
)

func scanRows(rows *sql.Rows) (*QueryResult, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := &QueryResult{Columns: cols}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make([]string, len(cols))
		for i, v := range values {
			if v == nil {
				row[i] = "NULL"
			} else {
				row[i] = fmt.Sprintf("%v", v)
			}
		}
		result.Rows = append(result.Rows, row)
	}
	result.RowCount = len(result.Rows)
	return result, rows.Err()
}

// StringArray implements sql.Scanner for postgres text[] columns
type StringArray []string

func (a *StringArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}
	switch v := src.(type) {
	case []byte:
		*a = parsePostgresArray(string(v))
		return nil
	case string:
		*a = parsePostgresArray(v)
		return nil
	}
	return fmt.Errorf("cannot scan %T into StringArray", src)
}

func parsePostgresArray(s string) []string {
	s = strings.Trim(s, "{}")
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}
```

**Step 5: Verify it compiles**

Run: `go build ./internal/db/`
Expected: No errors

**Step 6: Commit**

```bash
git add internal/db/postgres.go internal/db/postgres_test.go internal/db/helpers.go
git commit -m "feat: postgres engine implementation"
```

---

### Task 5: MySQL Engine

**Files:**
- Create: `internal/db/mysql.go`
- Create: `internal/db/mysql_test.go`

**Step 1: Write integration tests (skip if no DB)**

```go
// internal/db/mysql_test.go
package db

import (
	"os"
	"testing"
)

func getMySQLTestConfig(t *testing.T) ConnectionConfig {
	dsn := os.Getenv("DEX_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("DEX_TEST_MYSQL_DSN not set, skipping mysql integration test")
	}
	return ConnectionConfig{
		Engine:   "mysql",
		Host:     "localhost",
		Port:     3306,
		Database: "mysql",
		User:     "root",
		Password: "root",
		SSL:      false,
	}
}

func TestMySQLConnect(t *testing.T) {
	cfg := getMySQLTestConfig(t)
	m := &MySQL{}
	if err := m.Connect(cfg); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer m.Close()
}

func TestMySQLTables(t *testing.T) {
	cfg := getMySQLTestConfig(t)
	m := &MySQL{}
	if err := m.Connect(cfg); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer m.Close()

	tables, err := m.Tables()
	if err != nil {
		t.Fatalf("failed to list tables: %v", err)
	}
	if len(tables) == 0 {
		t.Error("expected at least one table in mysql database")
	}
}

func TestMySQLExecute(t *testing.T) {
	cfg := getMySQLTestConfig(t)
	m := &MySQL{}
	if err := m.Connect(cfg); err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer m.Close()

	result, err := m.Execute("SELECT 1 AS num")
	if err != nil {
		t.Fatalf("failed to execute: %v", err)
	}
	if len(result.Columns) != 1 || result.Columns[0] != "num" {
		t.Errorf("expected column 'num', got %v", result.Columns)
	}
	if len(result.Rows) != 1 || result.Rows[0][0] != "1" {
		t.Errorf("expected row ['1'], got %v", result.Rows)
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/db/ -v -run TestMySQL`
Expected: FAIL (MySQL type doesn't exist)

**Step 3: Implement MySQL engine**

```go
// internal/db/mysql.go
package db

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type MySQL struct {
	db       *sql.DB
	database string
}

func (m *MySQL) Connect(cfg ConnectionConfig) error {
	tls := "false"
	if cfg.SSL {
		tls = "true"
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?tls=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, tls)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return err
	}
	m.db = db
	m.database = cfg.Database
	return nil
}

func (m *MySQL) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *MySQL) DB() *sql.DB {
	return m.db
}

func (m *MySQL) Tables() ([]string, error) {
	rows, err := m.db.Query(`
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = ?
		ORDER BY table_name
	`, m.database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func (m *MySQL) Schema(table string) (*TableSchema, error) {
	schema := &TableSchema{Name: table}

	colRows, err := m.db.Query(`
		SELECT column_name, column_type, is_nullable
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?
		ORDER BY ordinal_position
	`, m.database, table)
	if err != nil {
		return nil, err
	}
	defer colRows.Close()

	for colRows.Next() {
		var col ColumnInfo
		var nullable string
		if err := colRows.Scan(&col.Name, &col.Type, &nullable); err != nil {
			return nil, err
		}
		col.Nullable = nullable == "YES"
		schema.Columns = append(schema.Columns, col)
	}
	if err := colRows.Err(); err != nil {
		return nil, err
	}

	idxRows, err := m.db.Query(`
		SELECT index_name, GROUP_CONCAT(column_name ORDER BY seq_in_index), NOT non_unique
		FROM information_schema.statistics
		WHERE table_schema = ? AND table_name = ?
		GROUP BY index_name, non_unique
		ORDER BY index_name
	`, m.database, table)
	if err != nil {
		return nil, err
	}
	defer idxRows.Close()

	for idxRows.Next() {
		var idx IndexInfo
		var colStr string
		if err := idxRows.Scan(&idx.Name, &colStr, &idx.Unique); err != nil {
			return nil, err
		}
		idx.Columns = strings.Split(colStr, ",")
		schema.Indexes = append(schema.Indexes, idx)
	}
	if err := idxRows.Err(); err != nil {
		return nil, err
	}

	fkRows, err := m.db.Query(`
		SELECT column_name, referenced_table_name, referenced_column_name, constraint_name
		FROM information_schema.key_column_usage
		WHERE table_schema = ? AND table_name = ? AND referenced_table_name IS NOT NULL
	`, m.database, table)
	if err != nil {
		return nil, err
	}
	defer fkRows.Close()

	for fkRows.Next() {
		var fk ForeignKey
		if err := fkRows.Scan(&fk.Column, &fk.RefTable, &fk.RefColumn, &fk.ConstraintName); err != nil {
			return nil, err
		}
		schema.ForeignKeys = append(schema.ForeignKeys, fk)
	}

	return schema, fkRows.Err()
}

func (m *MySQL) Execute(query string) (*QueryResult, error) {
	rows, err := m.db.Query(query)
	if err != nil {
		return &QueryResult{Error: err.Error()}, nil
	}
	defer rows.Close()

	return scanRows(rows)
}
```

Note: add `"strings"` to imports.

**Step 4: Verify it compiles**

Run: `go build ./internal/db/`
Expected: No errors

**Step 5: Commit**

```bash
git add internal/db/mysql.go internal/db/mysql_test.go
git commit -m "feat: mysql engine implementation"
```

---

### Task 6: DB Registry

**Files:**
- Create: `internal/db/registry.go`
- Create: `internal/db/registry_test.go`

**Step 1: Write tests**

```go
// internal/db/registry_test.go
package db

import "testing"

func TestRegistryNewEngine(t *testing.T) {
	_, err := NewEngine("postgres")
	if err != nil {
		t.Errorf("expected no error for postgres, got %v", err)
	}
	_, err = NewEngine("mysql")
	if err != nil {
		t.Errorf("expected no error for mysql, got %v", err)
	}
	_, err = NewEngine("sqlite")
	if err == nil {
		t.Error("expected error for unsupported engine")
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	if r.Active() != nil {
		t.Error("expected nil active engine on new registry")
	}
	if len(r.Names()) != 0 {
		t.Error("expected empty names on new registry")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/db/ -v -run "TestRegistry"`
Expected: FAIL

**Step 3: Implement registry**

```go
// internal/db/registry.go
package db

import "fmt"

func NewEngine(engine string) (Engine, error) {
	switch engine {
	case "postgres":
		return &Postgres{}, nil
	case "mysql":
		return &MySQL{}, nil
	default:
		return nil, fmt.Errorf("unsupported engine: %s", engine)
	}
}

type Registry struct {
	engines map[string]Engine
	configs map[string]ConnectionConfig
	active  string
	order   []string
}

func NewRegistry() *Registry {
	return &Registry{
		engines: make(map[string]Engine),
		configs: make(map[string]ConnectionConfig),
	}
}

func (r *Registry) Add(name string, engine Engine, cfg ConnectionConfig) {
	r.engines[name] = engine
	r.configs[name] = cfg
	r.order = append(r.order, name)
	if r.active == "" {
		r.active = name
	}
}

func (r *Registry) Remove(name string) {
	delete(r.engines, name)
	delete(r.configs, name)
	for i, n := range r.order {
		if n == name {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
	if r.active == name {
		if len(r.order) > 0 {
			r.active = r.order[0]
		} else {
			r.active = ""
		}
	}
}

func (r *Registry) Get(name string) Engine {
	return r.engines[name]
}

func (r *Registry) GetConfig(name string) ConnectionConfig {
	return r.configs[name]
}

func (r *Registry) SetActive(name string) {
	if _, ok := r.engines[name]; ok {
		r.active = name
	}
}

func (r *Registry) Active() Engine {
	if r.active == "" {
		return nil
	}
	return r.engines[r.active]
}

func (r *Registry) ActiveName() string {
	return r.active
}

func (r *Registry) Names() []string {
	return r.order
}

func (r *Registry) CloseAll() {
	for _, e := range r.engines {
		e.Close()
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/db/ -v -run "TestRegistry"`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/db/registry.go internal/db/registry_test.go
git commit -m "feat: db registry for managing multiple connections"
```

---

### Task 7: Styles & Keymap

**Files:**
- Create: `internal/ui/styles/styles.go`
- Create: `internal/keymap/keymap.go`

**Step 1: Create shared styles**

```go
// internal/ui/styles/styles.go
package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Pane borders
	ActiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62"))

	InactiveBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	// Sidebar
	SelectedItem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	NormalItem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	TreeIndent = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	// Results
	HeaderCell = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62")).
			Padding(0, 1)

	DataCell = lipgloss.NewStyle().
			Padding(0, 1)

	NullCell = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Padding(0, 1)

	// Status
	ErrorText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	StatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	// Modal
	ModalOverlay = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	ModalTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))
)
```

**Step 2: Create centralized keymap**

```go
// internal/keymap/keymap.go
package keymap

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	Quit          key.Binding
	FocusNext     key.Binding
	FocusPrev     key.Binding
	FocusLeft     key.Binding
	FocusRight    key.Binding
	Up            key.Binding
	Down          key.Binding
	Top           key.Binding
	Bottom        key.Binding
	Enter         key.Binding
	Escape        key.Binding
	QueryBar      key.Binding
	Editor        key.Binding
	NewConn       key.Binding
	ManageConns   key.Binding
	SchemaView    key.Binding
	Describe      key.Binding
	Search        key.Binding
	NextPage      key.Binding
	PrevPage      key.Binding
	HistoryPrev   key.Binding
	HistoryNext   key.Binding
	ExecuteEditor key.Binding
	Conn1         key.Binding
	Conn2         key.Binding
	Conn3         key.Binding
	Conn4         key.Binding
	Conn5         key.Binding
	Conn6         key.Binding
	Conn7         key.Binding
	Conn8         key.Binding
	Conn9         key.Binding
}

var Default = KeyMap{
	Quit:          key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	FocusNext:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next pane")),
	FocusPrev:     key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev pane")),
	FocusLeft:     key.NewBinding(key.WithKeys("ctrl+h"), key.WithHelp("ctrl+h", "focus left")),
	FocusRight:    key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "focus right")),
	Up:            key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "up")),
	Down:          key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "down")),
	Top:           key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "top")),
	Bottom:        key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "bottom")),
	Enter:         key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select/execute")),
	Escape:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back/close")),
	QueryBar:      key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "query bar")),
	Editor:        key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "sql editor")),
	NewConn:       key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "new connection")),
	ManageConns:   key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "manage connections")),
	SchemaView:    key.NewBinding(key.WithKeys("S"), key.WithHelp("S", "schema")),
	Describe:      key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "describe")),
	Search:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
	NextPage:      key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next page")),
	PrevPage:      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "prev page")),
	HistoryPrev:   key.NewBinding(key.WithKeys("ctrl+p"), key.WithHelp("ctrl+p", "prev query")),
	HistoryNext:   key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "next query")),
	ExecuteEditor: key.NewBinding(key.WithKeys("ctrl+enter"), key.WithHelp("ctrl+enter", "run query")),
	Conn1:         key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "connection 1")),
	Conn2:         key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "connection 2")),
	Conn3:         key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "connection 3")),
	Conn4:         key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "connection 4")),
	Conn5:         key.NewBinding(key.WithKeys("5"), key.WithHelp("5", "connection 5")),
	Conn6:         key.NewBinding(key.WithKeys("6"), key.WithHelp("6", "connection 6")),
	Conn7:         key.NewBinding(key.WithKeys("7"), key.WithHelp("7", "connection 7")),
	Conn8:         key.NewBinding(key.WithKeys("8"), key.WithHelp("8", "connection 8")),
	Conn9:         key.NewBinding(key.WithKeys("9"), key.WithHelp("9", "connection 9")),
}

func ConnBinding(n int) key.Binding {
	bindings := []key.Binding{
		Default.Conn1, Default.Conn2, Default.Conn3,
		Default.Conn4, Default.Conn5, Default.Conn6,
		Default.Conn7, Default.Conn8, Default.Conn9,
	}
	if n >= 1 && n <= 9 {
		return bindings[n-1]
	}
	return key.Binding{}
}
```

**Step 3: Verify both compile**

Run: `go build ./internal/ui/styles/ && go build ./internal/keymap/`
Expected: No errors

**Step 4: Commit**

```bash
git add internal/ui/styles/styles.go internal/keymap/keymap.go
git commit -m "feat: shared styles and centralized keymap"
```

---

### Task 8: Sidebar Tree View

**Files:**
- Create: `internal/ui/sidebar/sidebar.go`

**Step 1: Implement sidebar model**

```go
// internal/ui/sidebar/sidebar.go
package sidebar

import (
	"fmt"
	"strings"

	"github.com/bklimczak/dex/internal/ui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Messages
type TableSelectedMsg struct {
	Connection string
	Table      string
}

type ConnectionSelectedMsg struct {
	Connection string
}

// Tree node
type Node struct {
	Name     string
	Children []Node
	Expanded bool
}

type Model struct {
	nodes    []Node
	cursor   int
	focused  bool
	width    int
	height   int
	filter   string
	filtered []flatNode
}

type flatNode struct {
	name       string
	depth      int
	nodeIdx    int
	childIdx   int
	isConn     bool
	connName   string
}

func New() Model {
	return Model{}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetFocused(f bool) {
	m.focused = f
}

func (m *Model) Focused() bool {
	return m.focused
}

func (m *Model) SetConnections(names []string, tables map[string][]string) {
	m.nodes = make([]Node, len(names))
	for i, name := range names {
		children := make([]Node, len(tables[name]))
		for j, t := range tables[name] {
			children[j] = Node{Name: t}
		}
		m.nodes[i] = Node{
			Name:     name,
			Children: children,
			Expanded: i == 0,
		}
	}
	m.rebuildFlat()
}

func (m *Model) UpdateTables(connName string, tables []string) {
	for i := range m.nodes {
		if m.nodes[i].Name == connName {
			children := make([]Node, len(tables))
			for j, t := range tables {
				children[j] = Node{Name: t}
			}
			m.nodes[i].Children = children
			break
		}
	}
	m.rebuildFlat()
}

func (m *Model) rebuildFlat() {
	m.filtered = nil
	for i, node := range m.nodes {
		m.filtered = append(m.filtered, flatNode{
			name:     node.Name,
			depth:    0,
			nodeIdx:  i,
			childIdx: -1,
			isConn:   true,
			connName: node.Name,
		})
		if node.Expanded {
			for j, child := range node.Children {
				if m.filter == "" || strings.Contains(strings.ToLower(child.Name), strings.ToLower(m.filter)) {
					m.filtered = append(m.filtered, flatNode{
						name:     child.Name,
						depth:    1,
						nodeIdx:  i,
						childIdx: j,
						isConn:   false,
						connName: node.Name,
					})
				}
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m *Model) SetFilter(f string) {
	m.filter = f
	m.rebuildFlat()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused {
			return m, nil
		}
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
			m.cursor = 0
		case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
			m.cursor = max(0, len(m.filtered)-1)
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.cursor < len(m.filtered) {
				node := m.filtered[m.cursor]
				if node.isConn {
					m.nodes[node.nodeIdx].Expanded = !m.nodes[node.nodeIdx].Expanded
					m.rebuildFlat()
					return m, func() tea.Msg {
						return ConnectionSelectedMsg{Connection: node.connName}
					}
				}
				return m, func() tea.Msg {
					return TableSelectedMsg{
						Connection: node.connName,
						Table:      node.name,
					}
				}
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if len(m.filtered) == 0 {
		return styles.NormalItem.Render("No connections.\nPress Ctrl+n to add one.")
	}

	var b strings.Builder
	visibleHeight := m.height - 2
	if visibleHeight < 1 {
		visibleHeight = 10
	}

	start := 0
	if m.cursor >= visibleHeight {
		start = m.cursor - visibleHeight + 1
	}

	for i := start; i < len(m.filtered) && i < start+visibleHeight; i++ {
		node := m.filtered[i]
		prefix := ""
		if node.isConn {
			if m.nodes[node.nodeIdx].Expanded {
				prefix = "▼ "
			} else {
				prefix = "▶ "
			}
		} else {
			prefix = "  └─ "
		}

		line := fmt.Sprintf("%s%s", prefix, node.name)

		if i == m.cursor && m.focused {
			line = styles.SelectedItem.Render(line)
		} else if node.isConn {
			line = styles.NormalItem.Bold(true).Render(line)
		} else {
			line = styles.NormalItem.Render(line)
		}

		// Truncate to width
		if m.width > 0 && lipgloss.Width(line) > m.width-2 {
			line = line[:m.width-2]
		}

		b.WriteString(line)
		if i < start+visibleHeight-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/ui/sidebar/`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/ui/sidebar/sidebar.go
git commit -m "feat: sidebar tree view component"
```

---

### Task 9: Results Grid

**Files:**
- Create: `internal/ui/results/results.go`

**Step 1: Implement results grid model**

```go
// internal/ui/results/results.go
package results

import (
	"fmt"
	"strings"

	"github.com/bklimczak/dex/internal/db"
	"github.com/bklimczak/dex/internal/ui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	result    *db.QueryResult
	cursorRow int
	cursorCol int
	scrollX   int
	scrollY   int
	focused   bool
	width     int
	height    int
	colWidths []int
	err       string
}

func New() Model {
	return Model{}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *Model) SetFocused(f bool) {
	m.focused = f
}

func (m *Model) Focused() bool {
	return m.focused
}

func (m *Model) SetResult(r *db.QueryResult) {
	m.result = r
	m.cursorRow = 0
	m.cursorCol = 0
	m.scrollX = 0
	m.scrollY = 0
	m.err = ""
	if r != nil && r.Error != "" {
		m.err = r.Error
		return
	}
	m.calculateColWidths()
}

func (m *Model) SetError(e string) {
	m.err = e
	m.result = nil
}

func (m *Model) calculateColWidths() {
	if m.result == nil {
		return
	}
	m.colWidths = make([]int, len(m.result.Columns))
	for i, col := range m.result.Columns {
		m.colWidths[i] = len(col)
	}
	for _, row := range m.result.Rows {
		for i, cell := range row {
			if len(cell) > m.colWidths[i] {
				m.colWidths[i] = len(cell)
			}
		}
	}
	// Cap column width
	for i := range m.colWidths {
		if m.colWidths[i] > 40 {
			m.colWidths[i] = 40
		}
		if m.colWidths[i] < 4 {
			m.colWidths[i] = 4
		}
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.focused || m.result == nil {
			return m, nil
		}
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if m.cursorRow < len(m.result.Rows)-1 {
				m.cursorRow++
			}
			m.ensureVisible()
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if m.cursorRow > 0 {
				m.cursorRow--
			}
			m.ensureVisible()
		case key.Matches(msg, key.NewBinding(key.WithKeys("h", "left"))):
			if m.cursorCol > 0 {
				m.cursorCol--
			}
			m.ensureVisibleX()
		case key.Matches(msg, key.NewBinding(key.WithKeys("l", "right"))):
			if m.cursorCol < len(m.result.Columns)-1 {
				m.cursorCol++
			}
			m.ensureVisibleX()
		case key.Matches(msg, key.NewBinding(key.WithKeys("g"))):
			m.cursorRow = 0
			m.ensureVisible()
		case key.Matches(msg, key.NewBinding(key.WithKeys("G"))):
			m.cursorRow = max(0, len(m.result.Rows)-1)
			m.ensureVisible()
		}
	}
	return m, nil
}

func (m *Model) ensureVisible() {
	visibleRows := m.height - 4 // header + border + status
	if visibleRows < 1 {
		visibleRows = 1
	}
	if m.cursorRow < m.scrollY {
		m.scrollY = m.cursorRow
	}
	if m.cursorRow >= m.scrollY+visibleRows {
		m.scrollY = m.cursorRow - visibleRows + 1
	}
}

func (m *Model) ensureVisibleX() {
	if m.cursorCol < m.scrollX {
		m.scrollX = m.cursorCol
	}
	// Simple: keep at most a few columns ahead
	if m.cursorCol > m.scrollX+3 {
		m.scrollX = m.cursorCol - 3
	}
}

func (m Model) View() string {
	if m.err != "" {
		return styles.ErrorText.Render("Error: " + m.err)
	}
	if m.result == nil {
		return styles.NormalItem.Render("No data. Select a table or run a query.")
	}
	if len(m.result.Rows) == 0 {
		return styles.NormalItem.Render(fmt.Sprintf("Query returned 0 rows. Columns: %s",
			strings.Join(m.result.Columns, ", ")))
	}

	var b strings.Builder

	// Determine visible columns
	endCol := len(m.result.Columns)
	startCol := m.scrollX
	if startCol >= endCol {
		startCol = 0
	}

	// Header
	var headerParts []string
	for i := startCol; i < endCol; i++ {
		w := m.colWidths[i]
		cell := truncPad(m.result.Columns[i], w)
		headerParts = append(headerParts, styles.HeaderCell.Width(w+2).Render(cell))
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, headerParts...))
	b.WriteString("\n")

	// Separator
	for i := startCol; i < endCol; i++ {
		b.WriteString(strings.Repeat("─", m.colWidths[i]+2))
		if i < endCol-1 {
			b.WriteString("┼")
		}
	}
	b.WriteString("\n")

	// Rows
	visibleRows := m.height - 4
	if visibleRows < 1 {
		visibleRows = 10
	}
	endRow := min(m.scrollY+visibleRows, len(m.result.Rows))

	for r := m.scrollY; r < endRow; r++ {
		var rowParts []string
		for i := startCol; i < endCol; i++ {
			w := m.colWidths[i]
			cell := ""
			if i < len(m.result.Rows[r]) {
				cell = m.result.Rows[r][i]
			}
			cellStr := truncPad(cell, w)
			style := styles.DataCell.Width(w + 2)
			if cell == "NULL" {
				style = styles.NullCell.Width(w + 2)
			}
			if r == m.cursorRow && i == m.cursorCol && m.focused {
				style = style.Background(lipgloss.Color("237"))
			} else if r == m.cursorRow && m.focused {
				style = style.Background(lipgloss.Color("235"))
			}
			rowParts = append(rowParts, style.Render(cellStr))
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, rowParts...))
		if r < endRow-1 {
			b.WriteString("\n")
		}
	}

	// Status line
	b.WriteString("\n")
	status := fmt.Sprintf(" %d rows | row %d/%d ", m.result.RowCount, m.cursorRow+1, len(m.result.Rows))
	b.WriteString(styles.StatusBar.Render(status))

	return b.String()
}

func truncPad(s string, w int) string {
	if len(s) > w {
		return s[:w-1] + "…"
	}
	return s + strings.Repeat(" ", w-len(s))
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/ui/results/`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/ui/results/results.go
git commit -m "feat: results grid component with vim-style scrolling"
```

---

### Task 10: Query Bar

**Files:**
- Create: `internal/ui/querybar/querybar.go`

**Step 1: Implement query bar model**

```go
// internal/ui/querybar/querybar.go
package querybar

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ExecuteQueryMsg struct {
	Query string
}

type Model struct {
	input   textinput.Model
	focused bool
	width   int
	history []string
	histIdx int
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "SQL query..."
	ti.Prompt = ": "
	ti.CharLimit = 1000
	return Model{
		input:   ti,
		histIdx: -1,
	}
}

func (m *Model) SetWidth(w int) {
	m.width = w
	m.input.Width = w - 4
}

func (m *Model) SetFocused(f bool) {
	m.focused = f
	if f {
		m.input.Focus()
	} else {
		m.input.Blur()
	}
}

func (m *Model) Focused() bool {
	return m.focused
}

func (m *Model) AddToHistory(q string) {
	if q == "" {
		return
	}
	// Don't add duplicates at the end
	if len(m.history) > 0 && m.history[len(m.history)-1] == q {
		return
	}
	m.history = append(m.history, q)
	m.histIdx = len(m.history)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			query := m.input.Value()
			if query != "" {
				m.AddToHistory(query)
				m.input.SetValue("")
				m.histIdx = len(m.history)
				return m, func() tea.Msg {
					return ExecuteQueryMsg{Query: query}
				}
			}
		case "ctrl+p":
			if len(m.history) > 0 && m.histIdx > 0 {
				m.histIdx--
				m.input.SetValue(m.history[m.histIdx])
				m.input.CursorEnd()
			}
			return m, nil
		case "ctrl+n":
			if m.histIdx < len(m.history)-1 {
				m.histIdx++
				m.input.SetValue(m.history[m.histIdx])
				m.input.CursorEnd()
			} else {
				m.histIdx = len(m.history)
				m.input.SetValue("")
			}
			return m, nil
		case "esc":
			m.input.SetValue("")
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	style := lipgloss.NewStyle().
		Padding(0, 1)
	return style.Render(m.input.View())
}
```

**Step 2: Verify it compiles**

Run: `go build ./internal/ui/querybar/`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/ui/querybar/querybar.go
git commit -m "feat: query bar component with history"
```

---

### Task 11: App Model — Compose Panes & Focus Management

**Files:**
- Modify: `internal/app/app.go`
- Modify: `main.go`

**Step 1: Rewrite app.go to compose all panes**

```go
// internal/app/app.go
package app

import (
	"fmt"

	"github.com/bklimczak/dex/internal/config"
	"github.com/bklimczak/dex/internal/db"
	"github.com/bklimczak/dex/internal/keymap"
	"github.com/bklimczak/dex/internal/ui/querybar"
	"github.com/bklimczak/dex/internal/ui/results"
	"github.com/bklimczak/dex/internal/ui/sidebar"
	"github.com/bklimczak/dex/internal/ui/styles"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pane int

const (
	paneSidebar pane = iota
	paneResults
	paneQueryBar
	paneCount
)

type modal int

const (
	modalNone modal = iota
	modalConnForm
	modalEditor
	modalSchema
)

// Messages
type tablesLoadedMsg struct {
	connName string
	tables   []string
	err      error
}

type queryResultMsg struct {
	result *db.QueryResult
	err    error
}

type connectResultMsg struct {
	connName string
	err      error
}

type Model struct {
	sidebar  sidebar.Model
	results  results.Model
	querybar querybar.Model

	registry *db.Registry
	cfg      *config.Config
	cfgPath  string

	focus    pane
	modal    modal
	width    int
	height   int
	keys     keymap.KeyMap
	status   string
}

func New(cfg *config.Config, cfgPath string) Model {
	sb := sidebar.New()
	sb.SetFocused(true)

	return Model{
		sidebar:  sb,
		results:  results.New(),
		querybar: querybar.New(),
		registry: db.NewRegistry(),
		cfg:      cfg,
		cfgPath:  cfgPath,
		focus:    paneSidebar,
		keys:     keymap.Default,
	}
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, conn := range m.cfg.Connections {
		conn := conn
		cmds = append(cmds, m.connectCmd(conn))
	}
	return tea.Batch(cmds...)
}

func (m Model) connectCmd(conn config.Connection) tea.Cmd {
	return func() tea.Msg {
		engine, err := db.NewEngine(conn.Engine)
		if err != nil {
			return connectResultMsg{connName: conn.Name, err: err}
		}
		cfg := db.ConnectionConfig{
			Name:     conn.Name,
			Engine:   conn.Engine,
			Host:     conn.Host,
			Port:     conn.Port,
			Database: conn.Database,
			User:     conn.User,
			Password: conn.Password,
			SSL:      conn.SSL,
		}
		if err := engine.Connect(cfg); err != nil {
			return connectResultMsg{connName: conn.Name, err: err}
		}
		return connectResultMsg{connName: conn.Name, err: nil}
	}
}

func (m Model) loadTablesCmd(connName string) tea.Cmd {
	return func() tea.Msg {
		engine := m.registry.Get(connName)
		if engine == nil {
			return tablesLoadedMsg{connName: connName, err: fmt.Errorf("not connected")}
		}
		tables, err := engine.Tables()
		return tablesLoadedMsg{connName: connName, tables: tables, err: err}
	}
}

func (m Model) executeQueryCmd(query string) tea.Cmd {
	return func() tea.Msg {
		engine := m.registry.Active()
		if engine == nil {
			return queryResultMsg{err: fmt.Errorf("no active connection")}
		}
		result, err := engine.Execute(query)
		return queryResultMsg{result: result, err: err}
	}
}

func (m *Model) setFocus(p pane) {
	m.focus = p
	m.sidebar.SetFocused(p == paneSidebar)
	m.results.SetFocused(p == paneResults)
	m.querybar.SetFocused(p == paneQueryBar)
}

func (m *Model) cycleFocus(forward bool) {
	if forward {
		m.setFocus((m.focus + 1) % paneCount)
	} else {
		m.setFocus((m.focus - 1 + paneCount) % paneCount)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateSizes()
		return m, nil

	case connectResultMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Failed to connect %s: %v", msg.connName, msg.err)
			return m, nil
		}
		// Find the config and register
		for _, conn := range m.cfg.Connections {
			if conn.Name == msg.connName {
				engine, _ := db.NewEngine(conn.Engine)
				cfg := db.ConnectionConfig{
					Name: conn.Name, Engine: conn.Engine,
					Host: conn.Host, Port: conn.Port,
					Database: conn.Database, User: conn.User,
					Password: conn.Password, SSL: conn.SSL,
				}
				engine.Connect(cfg)
				m.registry.Add(conn.Name, engine, cfg)
				break
			}
		}
		m.status = fmt.Sprintf("Connected to %s", msg.connName)
		m.updateSidebar()
		return m, m.loadTablesCmd(msg.connName)

	case tablesLoadedMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Error loading tables for %s: %v", msg.connName, msg.err)
			return m, nil
		}
		m.sidebar.UpdateTables(msg.connName, msg.tables)
		m.status = fmt.Sprintf("Loaded %d tables from %s", len(msg.tables), msg.connName)
		return m, nil

	case queryResultMsg:
		if msg.err != nil {
			m.results.SetError(msg.err.Error())
			return m, nil
		}
		m.results.SetResult(msg.result)
		m.setFocus(paneResults)
		return m, nil

	case sidebar.TableSelectedMsg:
		m.registry.SetActive(msg.Connection)
		query := fmt.Sprintf("SELECT * FROM %s LIMIT 100", msg.Table)
		m.status = fmt.Sprintf("Loading %s.%s...", msg.Connection, msg.Table)
		return m, m.executeQueryCmd(query)

	case sidebar.ConnectionSelectedMsg:
		m.registry.SetActive(msg.Connection)
		m.status = fmt.Sprintf("Active: %s", msg.Connection)
		return m, nil

	case querybar.ExecuteQueryMsg:
		m.status = "Executing query..."
		return m, m.executeQueryCmd(msg.Query)

	case tea.KeyMsg:
		// Global keys (when no modal is open and query bar is not focused)
		if m.modal == modalNone && m.focus != paneQueryBar {
			switch {
			case key.Matches(msg, m.keys.Quit):
				m.registry.CloseAll()
				return m, tea.Quit
			case key.Matches(msg, m.keys.FocusNext):
				m.cycleFocus(true)
				return m, nil
			case key.Matches(msg, m.keys.FocusPrev):
				m.cycleFocus(false)
				return m, nil
			case key.Matches(msg, m.keys.FocusLeft):
				m.setFocus(paneSidebar)
				return m, nil
			case key.Matches(msg, m.keys.FocusRight):
				m.setFocus(paneResults)
				return m, nil
			case key.Matches(msg, m.keys.QueryBar):
				m.setFocus(paneQueryBar)
				return m, nil
			}

			// Quick-switch connections 1-9
			for i := 1; i <= 9; i++ {
				if key.Matches(msg, keymap.ConnBinding(i)) {
					names := m.registry.Names()
					if i <= len(names) {
						m.registry.SetActive(names[i-1])
						m.status = fmt.Sprintf("Switched to %s", names[i-1])
					}
					return m, nil
				}
			}
		}

		// Escape from query bar
		if m.focus == paneQueryBar && msg.String() == "esc" {
			m.setFocus(paneSidebar)
			return m, nil
		}

		// ctrl+c always quits
		if msg.String() == "ctrl+c" {
			m.registry.CloseAll()
			return m, tea.Quit
		}
	}

	// Delegate to focused pane
	switch m.focus {
	case paneSidebar:
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case paneResults:
		var cmd tea.Cmd
		m.results, cmd = m.results.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case paneQueryBar:
		var cmd tea.Cmd
		m.querybar, cmd = m.querybar.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateSizes() {
	sidebarWidth := m.width / 4
	if sidebarWidth < 20 {
		sidebarWidth = 20
	}
	if sidebarWidth > 40 {
		sidebarWidth = 40
	}
	mainWidth := m.width - sidebarWidth - 3 // borders
	queryBarHeight := 3
	resultsHeight := m.height - queryBarHeight - 2

	m.sidebar.SetSize(sidebarWidth, m.height-2)
	m.results.SetSize(mainWidth, resultsHeight)
	m.querybar.SetWidth(mainWidth)
}

func (m *Model) updateSidebar() {
	names := m.registry.Names()
	tables := make(map[string][]string)
	m.sidebar.SetConnections(names, tables)
}

func (m Model) View() string {
	sidebarWidth := m.width / 4
	if sidebarWidth < 20 {
		sidebarWidth = 20
	}
	if sidebarWidth > 40 {
		sidebarWidth = 40
	}

	// Sidebar
	sidebarStyle := styles.InactiveBorder
	if m.focus == paneSidebar {
		sidebarStyle = styles.ActiveBorder
	}
	sidebarView := sidebarStyle.
		Width(sidebarWidth).
		Height(m.height - 2).
		Render(m.sidebar.View())

	// Results
	resultsStyle := styles.InactiveBorder
	if m.focus == paneResults {
		resultsStyle = styles.ActiveBorder
	}
	queryBarHeight := 3
	resultsHeight := m.height - queryBarHeight - 4
	mainWidth := m.width - sidebarWidth - 5

	resultsView := resultsStyle.
		Width(mainWidth).
		Height(resultsHeight).
		Render(m.results.View())

	// Query bar
	qbStyle := styles.InactiveBorder
	if m.focus == paneQueryBar {
		qbStyle = styles.ActiveBorder
	}
	queryBarView := qbStyle.
		Width(mainWidth).
		Render(m.querybar.View())

	// Compose right side
	rightSide := lipgloss.JoinVertical(lipgloss.Left, resultsView, queryBarView)

	// Main layout
	main := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, rightSide)

	// Status bar
	status := styles.StatusBar.Width(m.width).Render(m.status)

	return lipgloss.JoinVertical(lipgloss.Left, main, status)
}
```

**Step 2: Update main.go to load config**

```go
package main

import (
	"fmt"
	"os"

	"github.com/bklimczak/dex/internal/app"
	"github.com/bklimczak/dex/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfgPath := config.DefaultPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(app.New(cfg, cfgPath), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 3: Verify it compiles and runs**

Run: `go build -o dex . && echo "Build OK"`
Expected: `Build OK`

**Step 4: Commit**

```bash
git add main.go internal/app/app.go
git commit -m "feat: compose pane layout with focus management and db integration"
```

---

### Task 12: Connection Form Modal

**Files:**
- Create: `internal/ui/connform/connform.go`
- Modify: `internal/app/app.go` (wire modal)

**Step 1: Implement connection form modal**

```go
// internal/ui/connform/connform.go
package connform

import (
	"fmt"
	"strconv"

	"github.com/bklimczak/dex/internal/config"
	"github.com/bklimczak/dex/internal/ui/styles"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type field int

const (
	fieldName field = iota
	fieldEngine
	fieldHost
	fieldPort
	fieldDatabase
	fieldUser
	fieldPassword
	fieldCount
)

type SaveConnectionMsg struct {
	Connection config.Connection
}

type CancelMsg struct{}

type TestConnectionMsg struct {
	Connection config.Connection
}

type Model struct {
	inputs     []textinput.Model
	focusIndex field
	width      int
	height     int
	editing    *config.Connection // non-nil if editing existing
	testResult string
}

func New() Model {
	inputs := make([]textinput.Model, fieldCount)

	names := []string{"Name", "Engine (postgres/mysql)", "Host", "Port", "Database", "User", "Password"}
	placeholders := []string{"my-database", "postgres", "localhost", "5432", "mydb", "user", ""}
	for i := range inputs {
		inputs[i] = textinput.New()
		inputs[i].Prompt = names[i] + ": "
		inputs[i].Placeholder = placeholders[i]
		inputs[i].CharLimit = 256
		if field(i) == fieldPassword {
			inputs[i].EchoMode = textinput.EchoPassword
		}
	}
	inputs[0].Focus()

	return Model{inputs: inputs}
}

func NewEditing(conn *config.Connection) Model {
	m := New()
	m.editing = conn
	m.inputs[fieldName].SetValue(conn.Name)
	m.inputs[fieldEngine].SetValue(conn.Engine)
	m.inputs[fieldHost].SetValue(conn.Host)
	m.inputs[fieldPort].SetValue(strconv.Itoa(conn.Port))
	m.inputs[fieldDatabase].SetValue(conn.Database)
	m.inputs[fieldUser].SetValue(conn.User)
	m.inputs[fieldPassword].SetValue(conn.Password)
	return m
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	for i := range m.inputs {
		m.inputs[i].Width = w - 30
	}
}

func (m *Model) SetTestResult(s string) {
	m.testResult = s
}

func (m Model) toConnection() config.Connection {
	port, _ := strconv.Atoi(m.inputs[fieldPort].Value())
	return config.Connection{
		Name:     m.inputs[fieldName].Value(),
		Engine:   m.inputs[fieldEngine].Value(),
		Host:     m.inputs[fieldHost].Value(),
		Port:     port,
		Database: m.inputs[fieldDatabase].Value(),
		User:     m.inputs[fieldUser].Value(),
		Password: m.inputs[fieldPassword].Value(),
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CancelMsg{} }
		case "tab", "down":
			m.inputs[m.focusIndex].Blur()
			m.focusIndex = (m.focusIndex + 1) % fieldCount
			m.inputs[m.focusIndex].Focus()
			return m, nil
		case "shift+tab", "up":
			m.inputs[m.focusIndex].Blur()
			m.focusIndex = (m.focusIndex - 1 + fieldCount) % fieldCount
			m.inputs[m.focusIndex].Focus()
			return m, nil
		case "ctrl+t":
			conn := m.toConnection()
			return m, func() tea.Msg { return TestConnectionMsg{Connection: conn} }
		case "ctrl+s":
			conn := m.toConnection()
			if conn.Name == "" || conn.Engine == "" || conn.Host == "" {
				m.testResult = "Name, engine, and host are required"
				return m, nil
			}
			return m, func() tea.Msg { return SaveConnectionMsg{Connection: conn} }
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	return m, cmd
}

func (m Model) View() string {
	var content string

	title := styles.ModalTitle.Render("New Connection")
	if m.editing != nil {
		title = styles.ModalTitle.Render("Edit Connection")
	}
	content += title + "\n\n"

	for i := range m.inputs {
		content += m.inputs[i].View() + "\n"
	}

	content += "\n"
	if m.testResult != "" {
		content += m.testResult + "\n"
	}
	content += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).
		Render("Ctrl+t: test | Ctrl+s: save | Esc: cancel")

	return styles.ModalOverlay.
		Width(min(60, m.width-10)).
		Render(content)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

**Step 2: Wire modal into app.go**

Add to `internal/app/app.go`:
- Import `connform` package
- Add `connForm connform.Model` field to Model
- Handle `ctrl+n` to open: set `m.modal = modalConnForm`, create `m.connForm = connform.New()`, set size
- When modal is open, delegate Update to `m.connForm`
- Handle `connform.SaveConnectionMsg`: add to config, save file, connect, close modal
- Handle `connform.TestConnectionMsg`: attempt connect, set test result
- Handle `connform.CancelMsg`: close modal
- In View, overlay the modal when active

**Step 3: Verify it compiles**

Run: `go build -o dex . && echo "Build OK"`
Expected: `Build OK`

**Step 4: Commit**

```bash
git add internal/ui/connform/connform.go internal/app/app.go
git commit -m "feat: connection form modal with test and save"
```

---

### Task 13: Full SQL Editor Modal

**Files:**
- Create: `internal/ui/editor/editor.go`
- Modify: `internal/app/app.go` (wire modal)

**Step 1: Implement editor modal**

```go
// internal/ui/editor/editor.go
package editor

import (
	"github.com/bklimczak/dex/internal/ui/styles"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ExecuteMsg struct {
	Query string
}

type CloseMsg struct{}

type Model struct {
	textarea textarea.Model
	width    int
	height   int
	history  []string
	histIdx  int
}

func New() Model {
	ta := textarea.New()
	ta.Placeholder = "Write your SQL query here..."
	ta.ShowLineNumbers = true
	ta.Focus()
	return Model{
		textarea: ta,
		histIdx:  -1,
	}
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.textarea.SetWidth(w - 8)
	m.textarea.SetHeight(h - 8)
}

func (m *Model) SetHistory(h []string) {
	m.history = h
	m.histIdx = len(h)
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return CloseMsg{} }
		case "ctrl+enter":
			query := m.textarea.Value()
			if query != "" {
				return m, func() tea.Msg { return ExecuteMsg{Query: query} }
			}
			return m, nil
		case "ctrl+p":
			if len(m.history) > 0 && m.histIdx > 0 {
				m.histIdx--
				m.textarea.SetValue(m.history[m.histIdx])
			}
			return m, nil
		case "ctrl+n":
			if m.histIdx < len(m.history)-1 {
				m.histIdx++
				m.textarea.SetValue(m.history[m.histIdx])
			} else {
				m.histIdx = len(m.history)
				m.textarea.SetValue("")
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	title := styles.ModalTitle.Render("SQL Editor")
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).
		Render("Ctrl+Enter: execute | Ctrl+p/n: history | Esc: close")

	content := title + "\n\n" + m.textarea.View() + "\n\n" + help

	return styles.ModalOverlay.
		Width(min(m.width-6, 100)).
		Render(content)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

**Step 2: Wire into app.go**

- Import `editor` package
- Add `editor editor.Model` and `queryHistory []string` fields
- Handle `ctrl+e`: open editor modal, pass history
- Delegate Update when modal is open
- Handle `editor.ExecuteMsg`: execute query, add to history, close modal
- Handle `editor.CloseMsg`: close modal

**Step 3: Verify it compiles**

Run: `go build -o dex . && echo "Build OK"`
Expected: `Build OK`

**Step 4: Commit**

```bash
git add internal/ui/editor/editor.go internal/app/app.go
git commit -m "feat: full SQL editor modal with history"
```

---

### Task 14: Schema Modal

**Files:**
- Create: `internal/ui/schema/schema.go`
- Modify: `internal/app/app.go` (wire modal)

**Step 1: Implement schema modal**

```go
// internal/ui/schema/schema.go
package schema

import (
	"fmt"
	"strings"

	"github.com/bklimczak/dex/internal/db"
	"github.com/bklimczak/dex/internal/ui/styles"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type CloseMsg struct{}

type Model struct {
	schema   *db.TableSchema
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

func New(schema *db.TableSchema, w, h int) Model {
	vp := viewport.New(w-8, h-8)
	vp.SetContent(renderSchema(schema))
	return Model{
		schema:   schema,
		viewport: vp,
		width:    w,
		height:   h,
		ready:    true,
	}
}

func renderSchema(s *db.TableSchema) string {
	var b strings.Builder

	b.WriteString(styles.ModalTitle.Render(fmt.Sprintf("Table: %s", s.Name)))
	b.WriteString("\n\n")

	// Columns
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Columns"))
	b.WriteString("\n")
	for _, col := range s.Columns {
		nullable := ""
		if col.Nullable {
			nullable = " (nullable)"
		}
		b.WriteString(fmt.Sprintf("  %-30s %-20s%s\n", col.Name, col.Type, nullable))
	}

	// Indexes
	if len(s.Indexes) > 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Indexes"))
		b.WriteString("\n")
		for _, idx := range s.Indexes {
			unique := ""
			if idx.Unique {
				unique = " UNIQUE"
			}
			b.WriteString(fmt.Sprintf("  %-30s (%s)%s\n", idx.Name, strings.Join(idx.Columns, ", "), unique))
		}
	}

	// Foreign keys
	if len(s.ForeignKeys) > 0 {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("Foreign Keys"))
		b.WriteString("\n")
		for _, fk := range s.ForeignKeys {
			b.WriteString(fmt.Sprintf("  %s -> %s.%s (%s)\n", fk.Column, fk.RefTable, fk.RefColumn, fk.ConstraintName))
		}
	}

	return b.String()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return m, func() tea.Msg { return CloseMsg{} }
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	help := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).
		Render("j/k: scroll | Esc: close")

	content := m.viewport.View() + "\n" + help

	return styles.ModalOverlay.
		Width(min(m.width-6, 80)).
		Render(content)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
```

**Step 2: Wire into app.go**

- Import `schema` package
- Add `schemaModal schema.Model` field
- Handle `S` key when sidebar is focused and a table is selected: load schema async, open modal
- Add `schemaLoadedMsg` message type
- Handle `schema.CloseMsg`: close modal

**Step 3: Verify it compiles**

Run: `go build -o dex . && echo "Build OK"`
Expected: `Build OK`

**Step 4: Commit**

```bash
git add internal/ui/schema/schema.go internal/app/app.go
git commit -m "feat: schema inspection modal"
```

---

### Task 15: Integration & Polish

**Files:**
- Modify: `internal/app/app.go`

**Step 1: Add search/filter to sidebar**

When `/` is pressed with sidebar focused, enter filter mode — show a text input at the top of the sidebar. Characters typed filter the table list. Esc or Enter exits filter mode.

**Step 2: Add `D` describe shortcut**

When `D` is pressed with a table selected in sidebar, run the engine-appropriate describe command:
- Postgres: `SELECT column_name, data_type, is_nullable, column_default FROM information_schema.columns WHERE table_name = '<table>' ORDER BY ordinal_position`
- MySQL: `DESCRIBE <table>`

Display result in the results pane.

**Step 3: Test end-to-end manually**

Create a test config at `~/.dex/connections.yaml` with a local database. Run `./dex` and verify:
- Sidebar shows connections and tables
- j/k navigates, Enter expands/collapses and loads data
- `:` focuses query bar, type and Enter runs query
- `Ctrl+e` opens editor, `Ctrl+Enter` runs
- `Tab` cycles focus
- `1-9` switches connections
- `S` shows schema
- `q` quits

**Step 4: Commit**

```bash
git add internal/app/app.go
git commit -m "feat: search filter, describe shortcut, integration polish"
```

---

### Task 16: README

**Files:**
- Create: `README.md`

**Step 1: Write README**

Include: what dex is, screenshot placeholder, installation (`go install`), configuration (`~/.dex/connections.yaml` example), keybindings table, supported engines.

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README with usage and keybindings"
```
