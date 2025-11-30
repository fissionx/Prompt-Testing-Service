package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	SQLDatabase           DatabaseConfig `yaml:"sql_database"`                      // SQLite for LLMs and Schedules
	NoSQLDatabase         DatabaseConfig `yaml:"nosql_database"`                    // MongoDB for Prompts and Responses
	CORSOrigin            string         `yaml:"cors_origin,omitempty"`             // CORS origin for API server
	KeywordsExclusionPath string         `yaml:"keywords_exclusion_path,omitempty"` // Path to keywords exclusion file
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Provider string            `yaml:"provider"` // sqlite, mongodb, cassandra
	URI      string            `yaml:"uri"`
	Database string            `yaml:"database"`
	Options  map[string]string `yaml:"options,omitempty"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	configPath := GetConfigPath()
	configDir := filepath.Dir(configPath)
	return &Config{
		SQLDatabase: DatabaseConfig{
			Provider: "sqlite",
			URI:      "gego.db",
			Database: "gego",
		},
		NoSQLDatabase: DatabaseConfig{
			Provider: "mongodb",
			URI:      "mongodb://localhost:27017",
			Database: "gego",
		},
		CORSOrigin:            "*",
		KeywordsExclusionPath: filepath.Join(configDir, "keywords_exclusion"),
	}
}

// Load loads configuration from file and applies environment variable overrides
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	applyEnvironmentOverrides(&config)

	return &config, nil
}

// applyEnvironmentOverrides applies environment variables to override configuration
func applyEnvironmentOverrides(cfg *Config) {
	// Check for GEGO_ENV to determine environment (local, dev, prod)
	env := strings.ToLower(os.Getenv("GEGO_ENV"))

	// MongoDB URI override based on environment or direct variable
	if mongoURI := os.Getenv("MONGODB_URI"); mongoURI != "" {
		// Direct override takes precedence
		cfg.NoSQLDatabase.URI = mongoURI
	} else if env != "" {
		// Environment-based configuration
		switch env {
		case "local":
			cfg.NoSQLDatabase.URI = "mongodb://localhost:27017"
		case "dev", "development":
			// Use Atlas cloud URI from environment or keep existing config
			if cloudURI := os.Getenv("MONGODB_CLOUD_URI"); cloudURI != "" {
				cfg.NoSQLDatabase.URI = cloudURI
			}
		case "prod", "production":
			// Use production Atlas URI from environment
			if prodURI := os.Getenv("MONGODB_PROD_URI"); prodURI != "" {
				cfg.NoSQLDatabase.URI = prodURI
			}
		}
	}

	// MongoDB Database name override
	if dbName := os.Getenv("MONGODB_DATABASE"); dbName != "" {
		cfg.NoSQLDatabase.Database = dbName
	}

	// SQL Database URI override (for SQLite)
	if sqlURI := os.Getenv("SQL_DATABASE_URI"); sqlURI != "" {
		cfg.SQLDatabase.URI = sqlURI
	}

	// CORS origin override
	if corsOrigin := os.Getenv("CORS_ORIGIN"); corsOrigin != "" {
		cfg.CORSOrigin = corsOrigin
	}
}

// Save saves configuration to file
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigPath returns the default config file path
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".gego/config.yaml"
	}
	return filepath.Join(home, ".gego", "config.yaml")
}

// Exists checks if config file exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
