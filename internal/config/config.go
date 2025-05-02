package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	ServerAddress string `mapstructure:"server_address"`
	DatabaseURL   string `mapstructure:"database_url"`
}

func LoadConfig() (*Config, error) {
	viper.SetDefault("server_address", ":8080")
	viper.SetDefault("database_url", "postgres://user:pass@localhost:5432/dbname")
	viper.AutomaticEnv()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
