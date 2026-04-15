package component_test

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

var update = flag.Bool("update", false, "update golden files")

// setupViewTest resets the theme singleton and pins the lipgloss renderer to
// TrueColor so View() output is identical across all machines and CI.
func setupViewTest(t *testing.T) {
	t.Helper()
	theme.ResetThemeOnce()
	style.ClearHighlightCache()
	r := lipgloss.NewRenderer(io.Discard, termenv.WithProfile(termenv.TrueColor))
	lipgloss.SetDefaultRenderer(r)
	t.Cleanup(func() {
		lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(os.Stdout))
	})
}

// assertGolden compares got against the contents of testdata/<name>.golden.
// If the file does not exist or -update is set, it writes got to the file.
func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")

	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("assertGolden: mkdir testdata: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("assertGolden: write %s: %v", path, err)
		}
		return
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Fatalf("assertGolden: golden file %s does not exist; run with -update to create it", path)
	}
	if err != nil {
		t.Fatalf("assertGolden: read %s: %v", path, err)
	}

	want := string(data)
	if got != want {
		gotLines := strings.Split(got, "\n")
		wantLines := strings.Split(want, "\n")
		var sb strings.Builder
		max := len(gotLines)
		if len(wantLines) > max {
			max = len(wantLines)
		}
		for i := 0; i < max; i++ {
			g, w := "", ""
			if i < len(gotLines) {
				g = gotLines[i]
			}
			if i < len(wantLines) {
				w = wantLines[i]
			}
			if g != w {
				sb.WriteString(fmt.Sprintf("line %d:\n  want: %q\n   got: %q\n", i+1, w, g))
			}
		}
		t.Errorf("view output mismatch for %s:\n%s", name, sb.String())
	}
}
