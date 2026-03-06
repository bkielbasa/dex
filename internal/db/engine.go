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
	Columns  []string
	Rows     [][]string
	RowCount int
	Error    string
}

type Engine interface {
	Connect(cfg ConnectionConfig) error
	Close() error
	DB() *sql.DB
	Tables() ([]string, error)
	Schema(table string) (*TableSchema, error)
	Execute(query string) (*QueryResult, error)
}
