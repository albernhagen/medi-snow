package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server ServerConfig
	Log    LogConfig
	App    AppConfig
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port    int
	GinMode string // debug, release, test
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string // debug, info, warn, error
	Format string // json, text
}

// AppConfig holds application-specific configuration
type AppConfig struct {
	ForecastDays int // Number of days to forecast
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
	viper.SetDefault("server.ginmode", "release")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "text")
	viper.SetDefault("app.forecastDays", 16)

	// Read from environment variables
	viper.SetEnvPrefix("MEDI_SNOW")
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist, we have defaults
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
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

// NewLogger creates a new slog.Logger based on the configuration
func (c *Config) NewLogger() *slog.Logger {
	// Parse log level
	var level slog.Level
	switch strings.ToLower(c.Log.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler options
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Choose handler based on format
	var handler slog.Handler
	switch strings.ToLower(c.Log.Format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default: // "text" or anything else
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
