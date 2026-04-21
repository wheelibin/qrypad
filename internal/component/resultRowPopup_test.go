package component_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/component"
)

func TestClampToLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		n     int
		want  string
	}{
		{
			name:  "fewer lines than limit — unchanged",
			input: "line1\nline2",
			n:     5,
			want:  "line1\nline2",
		},
		{
			name:  "exactly at limit — unchanged",
			input: "a\nb\nc\nd\ne",
			n:     5,
			want:  "a\nb\nc\nd\ne",
		},
		{
			name:  "exceeds limit — truncated with ellipsis",
			input: "a\nb\nc\nd\ne\nf",
			n:     5,
			want:  "a\nb\nc\nd\ne…",
		},
		{
			name:  "single line — unchanged",
			input: "hello world",
			n:     5,
			want:  "hello world",
		},
		{
			name:  "empty string — unchanged",
			input: "",
			n:     5,
			want:  "",
		},
		{
			name:  "limit of 1 — returns first line with ellipsis if truncated",
			input: "first\nsecond",
			n:     1,
			want:  "first…",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := component.ClampToLines(tt.input, tt.n)
			if got != tt.want {
				t.Errorf("ClampToLines(%q, %d) = %q; want %q", tt.input, tt.n, got, tt.want)
			}
		})
	}
}
