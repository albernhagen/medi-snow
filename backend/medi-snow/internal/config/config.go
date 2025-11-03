package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server ServerConfig
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port int
}

// Load reads configuration from file and environment variables
func Load() (*Config, error) {
	// Set config file name and paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("$HOME/.medi-snow")

	// Set defaults
	viper.SetDefault("server.port", 8080)

	// Read from environment variables
	viper.SetEnvPrefix("MEDI_SNOW")
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, we have defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal into config struct
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// GetServerAddr returns the server address in the format ":port"
func (c *Config) GetServerAddr() string {
	return fmt.Sprintf(":%d", c.Server.Port)
}
