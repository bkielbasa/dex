package app

import (
	"encoding/csv"
	"encoding/json"
	"os"

	"github.com/bklimczak/dex/internal/db"
)

func exportCSV(path string, result *db.QueryResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write(result.Columns)
	for _, row := range result.Rows {
		w.Write(row)
	}
	w.Flush()
	return w.Error()
}

func exportJSON(path string, result *db.QueryResult) error {
	var records []map[string]string
	for _, row := range result.Rows {
		record := make(map[string]string)
		for i, col := range result.Columns {
			if i < len(row) {
				record[col] = row[i]
			}
		}
		records = append(records, record)
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
