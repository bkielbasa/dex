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
