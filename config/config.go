package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	APIKey string `mapstructure:"api_key"`
	Host   string `mapstructure:"host"`
	Planka Planka `mapstructure:"planka"`
}

type Planka struct {
	BaseURL      string `mapstructure:"base_url"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	BoardID      string `mapstructure:"board_id"`
	ListID       string `mapstructure:"list_id"`
	ClosedListID string `mapstructure:"closed_list_id"`
}

func LoadConfig() (*Config, error) {
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/default")

	// Try .redmine-tui first (standard for home dir)
	viper.SetConfigName(".redmine-tui")
	viper.SetConfigType("yaml")

	// Bind environment variables
	viper.BindEnv("api_key", "REDMINE_API_KEY")
	viper.BindEnv("host", "REDMINE_HOST")
	viper.BindEnv("planka.base_url", "PLANKA_API_URL")
	viper.BindEnv("planka.username", "PLANKA_USERNAME")
	viper.BindEnv("planka.password", "PLANKA_PASSWORD")
	viper.BindEnv("planka.board_id", "PLANKA_BOARD_ID")
	viper.BindEnv("planka.list_id", "PLANKA_LIST_ID")
	viper.BindEnv("planka.closed_list_id", "PLANKA_CLOSED_LIST_ID")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Try redmine-tui (standard for system config)
			viper.SetConfigName("redmine-tui")
			if err := viper.ReadInConfig(); err != nil {
				// If still not found, check if it's really missing or another error
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					return nil, fmt.Errorf("error reading config file: %w", err)
				}
				// If not found in either, that's fine, we might rely entirely on Env vars
			}
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	// Manually set Planka config if not populated (Viper Unmarshal sometimes misses Env vars for nested structs if keys missing in file)
	if config.Planka.BaseURL == "" {
		config.Planka.BaseURL = viper.GetString("planka.base_url")
	}
	if config.Planka.Username == "" {
		config.Planka.Username = viper.GetString("planka.username")
	}
	if config.Planka.Password == "" {
		config.Planka.Password = viper.GetString("planka.password")
	}
	if config.Planka.BoardID == "" {
		config.Planka.BoardID = viper.GetString("planka.board_id")
	}
	if config.Planka.ListID == "" {
		config.Planka.ListID = viper.GetString("planka.list_id")
	}
	if config.Planka.ClosedListID == "" {
		config.Planka.ClosedListID = viper.GetString("planka.closed_list_id")
	}
	// Also ensure defaults for Board/List IDs if configured via env vars but not bound explicitly above (if user added them to env)
	// But we only bound base_url, username, password. The IDs are usually in config file or not bound.
	// Let's rely on config file for IDs or bind them if needed.
	// The user walkthrough said "Credentials are loaded from environment variables ... or config file". IDs usually in config.

	// Just in case, let's verify BoardID/ListID from config file map if struct missed it (unlikely if in file).

	return &config, nil
}
