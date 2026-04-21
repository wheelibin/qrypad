package style_test

import (
	"image/color"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/wheelibin/qrypad/internal/style"
)

func TestGetSpan(t *testing.T) {
	tests := []struct {
		name  string
		span  int
		total int
		want  int
	}{
		{
			name:  "span 12 returns total exactly",
			span:  12,
			total: 100,
			want:  100,
		},
		{
			name:  "span 6 returns half rounded up",
			span:  6,
			total: 100,
			want:  50,
		},
		{
			name:  "span 6 with odd total rounds up",
			span:  6,
			total: 101,
			want:  51, // ceil(101/2) = 51
		},
		{
			name:  "span 4 is one third",
			span:  4,
			total: 120,
			want:  40,
		},
		{
			name:  "span 1 with small total",
			span:  1,
			total: 12,
			want:  1,
		},
		{
			name:  "span 1 with total not divisible by 12 rounds up",
			span:  1,
			total: 13,
			want:  2, // ceil(13/12) = 2
		},
		{
			name:  "span 3 is one quarter",
			span:  3,
			total: 120,
			want:  30,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := style.GetSpan(tt.span, tt.total)
			if got != tt.want {
				t.Errorf("GetSpan(%d, %d) = %d, want %d", tt.span, tt.total, got, tt.want)
			}
		})
	}
}

func TestExtractStyledChar(t *testing.T) {
	red := "\033[31m"
	blue := "\033[34m"
	reset := "\033[0m"

	tests := []struct {
		name         string
		styled       string
		offset       int
		wantContains string // the raw character that must appear in the result
		wantHasColor bool   // result should contain an ANSI escape
		wantEmpty    bool   // result should be empty (out of range)
	}{
		{
			name:         "plain string offset 0",
			styled:       "abc",
			offset:       0,
			wantContains: "a",
			wantHasColor: false,
		},
		{
			name:         "plain string offset 1",
			styled:       "abc",
			offset:       1,
			wantContains: "b",
			wantHasColor: false,
		},
		{
			name:         "colored char at offset 0",
			styled:       red + "a" + reset + "bc",
			offset:       0,
			wantContains: "a",
			wantHasColor: true,
		},
		{
			name:         "colored char at offset 1 picks up its own color",
			styled:       "a" + blue + "b" + reset + "c",
			offset:       1,
			wantContains: "b",
			wantHasColor: true,
		},
		{
			name:         "char inherits color set before it",
			styled:       red + "ab" + reset,
			offset:       1,
			wantContains: "b",
			wantHasColor: true, // red was set before 'b'
		},
		{
			name:      "offset out of range returns empty",
			styled:    "abc",
			offset:    10,
			wantEmpty: true,
		},
		{
			name:      "empty string returns empty",
			styled:    "",
			offset:    0,
			wantEmpty: true,
		},
		{
			name:         "result always ends with reset",
			styled:       red + "x" + reset,
			offset:       0,
			wantContains: "x",
			wantHasColor: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := style.ExtractStyledChar(tt.styled, tt.offset)
			if tt.wantEmpty {
				if got != "" {
					t.Errorf("ExtractStyledChar: expected empty, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("ExtractStyledChar: expected result to contain %q, got %q", tt.wantContains, got)
			}
			if tt.wantHasColor && !strings.Contains(got, "\033[") {
				t.Errorf("ExtractStyledChar: expected ANSI escape in result, got %q", got)
			}
			if !strings.HasSuffix(got, "\033[0m") {
				t.Errorf("ExtractStyledChar: expected result to end with reset, got %q", got)
			}
		})
	}
}

func TestParseFgColor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string // expected lipgloss color string ("196", "#FF6400", or "" for none)
	}{
		{
			name:  "256-color fg escape",
			input: "\033[38;5;196mx\033[0m",
			want:  "196",
		},
		{
			name:  "truecolor fg escape",
			input: "\033[38;2;255;100;0mx\033[0m",
			want:  "#FF6400",
		},
		{
			name:  "no color escape returns empty",
			input: "x\033[0m",
			want:  "",
		},
		{
			name:  "empty string returns empty",
			input: "",
			want:  "",
		},
		{
			name:  "bg escape (not fg) returns empty",
			input: "\033[48;5;196mx\033[0m",
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := style.ParseFgColor(tt.input)
			var wantColor color.Color
			if tt.want != "" {
				wantColor = lipgloss.Color(tt.want)
			}
			if got != wantColor {
				t.Errorf("ParseFgColor(%q) = %v, want %v", tt.input, got, wantColor)
			}
		})
	}
}
