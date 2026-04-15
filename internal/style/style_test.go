package style_test

import (
	"testing"

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
