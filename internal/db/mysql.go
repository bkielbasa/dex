package db

import (
	"database/sql"
	"fmt"
	"strings"

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
