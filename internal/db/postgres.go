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
