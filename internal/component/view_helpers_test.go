package component_test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/wheelibin/qrypad/internal/style"
	"github.com/wheelibin/qrypad/internal/theme"
)

//nolint:gochecknoglobals // test flag must be a package-level var for flag.Bool
var update = flag.Bool("update", false, "update golden files")

// setupViewTest resets the theme singleton and clears caches so View() output
// is reproducible. In lipgloss v2, Style.Render() always emits full ANSI codes
// and does not depend on a renderer, so no renderer pinning is needed.
func setupViewTest(t *testing.T) {
	t.Helper()
	theme.ResetThemeOnce()
	style.ClearHighlightCache()
}

// assertGolden strips ANSI escape codes from got and compares against the
// contents of testdata/<name>.golden.  Golden files contain only the visible
// characters (box-drawing chars, text, spaces) so they are human-readable and
// stable across environments.
// If the file does not exist or -update is set, it writes the stripped output.
func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	got = ansi.Strip(got)
	path := filepath.Join("testdata", name+".golden")

	if *update {
		if err := os.MkdirAll("testdata", 0o750); err != nil {
			t.Fatalf("assertGolden: mkdir testdata: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o600); err != nil {
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
		maxLines := len(gotLines)
		if len(wantLines) > maxLines {
			maxLines = len(wantLines)
		}
		for i := range maxLines {
			g, w := "", ""
			if i < len(gotLines) {
				g = gotLines[i]
			}
			if i < len(wantLines) {
				w = wantLines[i]
			}
			if g != w {
				fmt.Fprintf(&sb, "line %d:\n  want: %q\n   got: %q\n", i+1, w, g)
			}
		}
		t.Errorf("view output mismatch for %s:\n%s", name, sb.String())
	}
}
