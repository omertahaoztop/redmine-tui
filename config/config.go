package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	APIKey  string  `mapstructure:"api_key"`
	Host    string  `mapstructure:"host"`
	Vikunja Vikunja `mapstructure:"vikunja"`
}

type Vikunja struct {
	BaseURL      string `mapstructure:"base_url"`
	Token        string `mapstructure:"token"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	ProjectID    int64  `mapstructure:"project_id"`
	ViewID       int64  `mapstructure:"view_id"`
	BucketID     int64  `mapstructure:"bucket_id"`
	DoneBucketID int64  `mapstructure:"done_bucket_id"`
}

func LoadConfig() (*Config, error) {
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/default")

	viper.SetConfigName(".redmine-tui")
	viper.SetConfigType("yaml")

	viper.BindEnv("api_key", "REDMINE_API_KEY")
	viper.BindEnv("host", "REDMINE_HOST")
	viper.BindEnv("vikunja.base_url", "VIKUNJA_API_URL")
	viper.BindEnv("vikunja.token", "VIKUNJA_TOKEN")
	viper.BindEnv("vikunja.username", "VIKUNJA_USERNAME")
	viper.BindEnv("vikunja.password", "VIKUNJA_PASSWORD")
	viper.BindEnv("vikunja.project_id", "VIKUNJA_PROJECT_ID")
	viper.BindEnv("vikunja.view_id", "VIKUNJA_VIEW_ID")
	viper.BindEnv("vikunja.bucket_id", "VIKUNJA_BUCKET_ID")
	viper.BindEnv("vikunja.done_bucket_id", "VIKUNJA_DONE_BUCKET_ID")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			viper.SetConfigName("redmine-tui")
			if err := viper.ReadInConfig(); err != nil {
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					return nil, fmt.Errorf("error reading config file: %w", err)
				}
			}
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	if config.Vikunja.BaseURL == "" {
		config.Vikunja.BaseURL = viper.GetString("vikunja.base_url")
	}
	if config.Vikunja.Token == "" {
		config.Vikunja.Token = viper.GetString("vikunja.token")
	}
	if config.Vikunja.Username == "" {
		config.Vikunja.Username = viper.GetString("vikunja.username")
	}
	if config.Vikunja.Password == "" {
		config.Vikunja.Password = viper.GetString("vikunja.password")
	}
	if config.Vikunja.ProjectID == 0 {
		config.Vikunja.ProjectID = viper.GetInt64("vikunja.project_id")
	}
	if config.Vikunja.ViewID == 0 {
		config.Vikunja.ViewID = viper.GetInt64("vikunja.view_id")
	}
	if config.Vikunja.BucketID == 0 {
		config.Vikunja.BucketID = viper.GetInt64("vikunja.bucket_id")
	}
	if config.Vikunja.DoneBucketID == 0 {
		config.Vikunja.DoneBucketID = viper.GetInt64("vikunja.done_bucket_id")
	}

	return &config, nil
}
