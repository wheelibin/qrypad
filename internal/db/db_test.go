package db_test

import (
	"testing"

	"github.com/wheelibin/qrypad/internal/db"
)

func TestConnectionConfig_UseSingleQueryFile(t *testing.T) {
	tests := []struct {
		name   string
		config db.ConnectionConfig
		want   bool
	}{
		{
			name:   "sqlite always returns true",
			config: db.ConnectionConfig{Driver: db.DriverName.SQLite},
			want:   true,
		},
		{
			name:   "sqlite returns true even when SingleQueryFile is false",
			config: db.ConnectionConfig{Driver: db.DriverName.SQLite, SingleQueryFile: false},
			want:   true,
		},
		{
			name:   "postgres defaults to false (per-database mode)",
			config: db.ConnectionConfig{Driver: db.DriverName.Postgres},
			want:   false,
		},
		{
			name:   "postgres with SingleQueryFile true returns true",
			config: db.ConnectionConfig{Driver: db.DriverName.Postgres, SingleQueryFile: true},
			want:   true,
		},
		{
			name:   "mysql defaults to false (per-database mode)",
			config: db.ConnectionConfig{Driver: db.DriverName.MySQL},
			want:   false,
		},
		{
			name:   "mysql with SingleQueryFile true returns true",
			config: db.ConnectionConfig{Driver: db.DriverName.MySQL, SingleQueryFile: true},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.UseSingleQueryFile()
			if got != tt.want {
				t.Errorf("UseSingleQueryFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
