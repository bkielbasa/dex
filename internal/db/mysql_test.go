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
