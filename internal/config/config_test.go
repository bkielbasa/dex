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
