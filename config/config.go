package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	APIKey string `mapstructure:"api_key"`
	Host   string `mapstructure:"host"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName(".redmine-tui")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")

	// Bind environment variables
	viper.BindEnv("api_key", "REDMINE_API_KEY")
	viper.BindEnv("host", "REDMINE_HOST")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	return &config, nil
}
