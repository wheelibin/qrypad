package db

import (
	"fmt"

	"github.com/spf13/viper"
)

func GetConnections() (map[string]ConnectionConfig, error) {
	type config struct {
		Connections map[string]ConnectionConfig `mapstructure:"connections"`
	}
	var cfg config
	err := viper.Unmarshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}
	return cfg.Connections, nil
}
