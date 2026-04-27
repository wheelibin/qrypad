package commands

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
)

// ExportRequested returns a command that emits an ExportRequestedMsg with the
// given format. Used by the export format popup.
func ExportRequested(f ExportFormat) tea.Cmd {
	return func() tea.Msg {
		return ExportRequestedMsg{Format: f}
	}
}

// ExportResults serializes rows (using the supplied column order) into the
// requested format and writes the result to a timestamped file in cwd. On
// success it returns an ExportCompletedMsg; on failure, an ErrMsg.
func ExportResults(rows []map[string]any, columns []string, format ExportFormat, cwd string) tea.Cmd {
	return func() tea.Msg {
		timestamp := time.Now().Format("20060102-150405")
		var (
			ext  string
			data []byte
			err  error
		)
		switch format {
		case ExportFormatJSON:
			ext = "json"
			data, err = marshalJSON(rows)
		case ExportFormatCSV:
			ext = "csv"
			data, err = marshalCSV(rows, columns)
		default:
			return ErrMsg{Err: fmt.Errorf("unknown export format: %d", format)}
		}
		if err != nil {
			return ErrMsg{Err: err}
		}

		path := filepath.Join(cwd, fmt.Sprintf("qrypad-export-%s.%s", timestamp, ext))
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return ErrMsg{Err: err}
		}

		return ExportCompletedMsg{Path: path, RowCount: len(rows)}
	}
}

func marshalJSON(rows []map[string]any) ([]byte, error) {
	if rows == nil {
		return []byte("[]"), nil
	}
	data, err := json.Marshal(rows)
	if err != nil {
		return nil, fmt.Errorf("marshalling rows to JSON: %w", err)
	}
	return data, nil
}

func marshalCSV(rows []map[string]any, columns []string) ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if err := w.Write(columns); err != nil {
		return nil, fmt.Errorf("writing CSV header: %w", err)
	}
	for _, row := range rows {
		record := make([]string, len(columns))
		for i, col := range columns {
			record[i] = csvCell(row[col])
		}
		if err := w.Write(record); err != nil {
			return nil, fmt.Errorf("writing CSV record: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("flushing CSV writer: %w", err)
	}
	return buf.Bytes(), nil
}

func csvCell(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case time.Time:
		return val.Format(time.RFC3339)
	case sql.RawBytes:
		return string(val)
	case []byte:
		return string(val)
	case string:
		return val
	default:
		return fmt.Sprint(val)
	}
}
