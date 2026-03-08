package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
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
			switch val := v.(type) {
			case nil:
				row[i] = "NULL"
			case []byte:
				row[i] = string(val)
			case time.Time:
				if val.Hour() == 0 && val.Minute() == 0 && val.Second() == 0 && val.Nanosecond() == 0 {
					row[i] = val.Format("2006-01-02")
				} else {
					row[i] = val.Format("2006-01-02 15:04:05")
				}
			case bool:
				row[i] = fmt.Sprintf("%t", val)
			case int64:
				row[i] = fmt.Sprintf("%d", val)
			case float64:
				row[i] = fmt.Sprintf("%g", val)
			default:
				row[i] = fmt.Sprintf("%v", val)
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
