package commands_test

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/wheelibin/qrypad/internal/commands"
)

func runExport(t *testing.T, rows []map[string]any, cols []string, format commands.ExportFormat, cwd string) any {
	t.Helper()
	cmd := commands.ExportResults(rows, cols, format, cwd)
	if cmd == nil {
		t.Fatalf("ExportResults returned nil cmd")
	}
	return cmd()
}

func TestExportResults_JSON(t *testing.T) {
	dir := t.TempDir()
	rows := []map[string]any{
		{"id": 1, "name": "alice"},
		{"id": 2, "name": "bob"},
	}
	cols := []string{"id", "name"}

	msg := runExport(t, rows, cols, commands.ExportFormatJSON, dir)
	completed, ok := msg.(commands.ExportCompletedMsg)
	if !ok {
		t.Fatalf("expected ExportCompletedMsg, got %T (%v)", msg, msg)
	}
	if completed.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", completed.RowCount)
	}

	base := filepath.Base(completed.Path)
	matched, _ := regexp.MatchString(`^qrypad-export-\d{8}-\d{6}\.json$`, base)
	if !matched {
		t.Errorf("filename %q does not match expected pattern", base)
	}

	data, err := os.ReadFile(completed.Path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2", len(got))
	}
}

func TestExportResults_CSV(t *testing.T) {
	dir := t.TempDir()
	rows := []map[string]any{
		{"id": 1, "name": "alice", "note": "hi, there"},
		{"id": 2, "name": "bob", "note": "line1\nline2"},
		{"id": 3, "name": `he said "hi"`, "note": nil},
	}
	cols := []string{"id", "name", "note"}

	msg := runExport(t, rows, cols, commands.ExportFormatCSV, dir)
	completed, ok := msg.(commands.ExportCompletedMsg)
	if !ok {
		t.Fatalf("expected ExportCompletedMsg, got %T (%v)", msg, msg)
	}
	base := filepath.Base(completed.Path)
	matched, _ := regexp.MatchString(`^qrypad-export-\d{8}-\d{6}\.csv$`, base)
	if !matched {
		t.Errorf("filename %q does not match expected pattern", base)
	}

	f, err := os.Open(completed.Path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("csv read: %v", err)
	}
	if len(records) != 4 {
		t.Fatalf("got %d records (inc. header), want 4", len(records))
	}
	if !slices.Equal(records[0], []string{"id", "name", "note"}) {
		t.Errorf("header = %v, want [id name note]", records[0])
	}
	if !slices.Equal(records[1], []string{"1", "alice", "hi, there"}) {
		t.Errorf("row1 = %v", records[1])
	}
	if !slices.Equal(records[2], []string{"2", "bob", "line1\nline2"}) {
		t.Errorf("row2 = %v", records[2])
	}
	if !slices.Equal(records[3], []string{"3", `he said "hi"`, ""}) {
		t.Errorf("row3 = %v", records[3])
	}
}

func TestExportResults_EmptyRows(t *testing.T) {
	dir := t.TempDir()
	cols := []string{"id", "name"}

	// JSON: empty array
	jsonMsg := runExport(t, nil, cols, commands.ExportFormatJSON, dir)
	jsonDone, ok := jsonMsg.(commands.ExportCompletedMsg)
	if !ok {
		t.Fatalf("json: expected ExportCompletedMsg, got %T", jsonMsg)
	}
	if jsonDone.RowCount != 0 {
		t.Errorf("json RowCount = %d, want 0", jsonDone.RowCount)
	}
	jsonBytes, _ := os.ReadFile(jsonDone.Path)
	if strings.TrimSpace(string(jsonBytes)) != "[]" {
		t.Errorf("json content = %q, want \"[]\"", string(jsonBytes))
	}

	// CSV: header only
	csvMsg := runExport(t, nil, cols, commands.ExportFormatCSV, dir)
	csvDone, ok := csvMsg.(commands.ExportCompletedMsg)
	if !ok {
		t.Fatalf("csv: expected ExportCompletedMsg, got %T", csvMsg)
	}
	csvBytes, _ := os.ReadFile(csvDone.Path)
	if strings.TrimSpace(string(csvBytes)) != "id,name" {
		t.Errorf("csv content = %q, want \"id,name\"", string(csvBytes))
	}
}

func TestExportResults_BytesValueInCSV(t *testing.T) {
	dir := t.TempDir()
	rows := []map[string]any{
		{"data": []byte("hello")},
	}
	cols := []string{"data"}
	msg := runExport(t, rows, cols, commands.ExportFormatCSV, dir)
	completed, ok := msg.(commands.ExportCompletedMsg)
	if !ok {
		t.Fatalf("expected ExportCompletedMsg, got %T", msg)
	}
	data, _ := os.ReadFile(completed.Path)
	want := "data\nhello\n"
	if string(data) != want {
		t.Errorf("csv content = %q, want %q", string(data), want)
	}
}

func TestExportResults_WriteError(t *testing.T) {
	// /dev/null/does-not-exist can't be a directory, so writing there fails
	msg := runExport(t, nil, []string{"id"}, commands.ExportFormatJSON, "/dev/null/does-not-exist")
	if _, ok := msg.(commands.ErrMsg); !ok {
		t.Fatalf("expected ErrMsg, got %T (%v)", msg, msg)
	}
}

func TestExportResults_UnknownFormat(t *testing.T) {
	dir := t.TempDir()
	msg := runExport(t, nil, []string{"id"}, commands.ExportFormat(99), dir)
	errMsg, ok := msg.(commands.ErrMsg)
	if !ok {
		t.Fatalf("expected ErrMsg, got %T", msg)
	}
	if errMsg.Err == nil || !strings.Contains(errMsg.Err.Error(), "unknown export format") {
		t.Errorf("error = %v, want to contain 'unknown export format'", errMsg.Err)
	}
}

func TestExportResults_TimeValueInCSV(t *testing.T) {
	dir := t.TempDir()
	ts := time.Date(2026, 4, 22, 10, 30, 0, 0, time.UTC)
	rows := []map[string]any{
		{"when": ts},
	}
	cols := []string{"when"}
	msg := runExport(t, rows, cols, commands.ExportFormatCSV, dir)
	completed, ok := msg.(commands.ExportCompletedMsg)
	if !ok {
		t.Fatalf("expected ExportCompletedMsg, got %T", msg)
	}
	data, _ := os.ReadFile(completed.Path)
	want := "when\n2026-04-22T10:30:00Z\n"
	if string(data) != want {
		t.Errorf("csv content = %q, want %q", string(data), want)
	}
}
