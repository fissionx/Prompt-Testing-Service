package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	SQLDatabase           DatabaseConfig `yaml:"sql_database"`                      // SQLite for LLMs and Schedules
	NoSQLDatabase         DatabaseConfig `yaml:"nosql_database"`                    // PostgreSQL for Prompts and Responses
	CORSOrigin            string         `yaml:"cors_origin,omitempty"`             // CORS origin for API server
	KeywordsExclusionPath string         `yaml:"keywords_exclusion_path,omitempty"` // Path to keywords exclusion file
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Provider string            `yaml:"provider"` // sqlite, postgresql, postgres
	URI      string            `yaml:"uri"`
	Database string            `yaml:"database"`
	Options  map[string]string `yaml:"options,omitempty"`
}

// ClerkConfig represents Clerk authentication configuration
type ClerkConfig struct {
	SecretKey      string `mapstructure:"secret_key"`      // CLERK_SECRET_KEY for JWT verification
	PublishableKey string `mapstructure:"publishable_key"` // CLERK_PUBLISHABLE_KEY (optional)
	JWKSURL        string `mapstructure:"jwks_url"`        // Clerk JWKS endpoint (auto-derived if empty)
}

// Logger interface for structured logging (wraps zap.Logger)
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
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
			Provider: "postgresql",
			URI:      "postgres://localhost:5432/gego?sslmode=disable",
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

	// PostgreSQL/NoSQL Database Provider override (takes precedence)
	if nosqlProvider := os.Getenv("NOSQL_DATABASE_PROVIDER"); nosqlProvider != "" {
		cfg.NoSQLDatabase.Provider = nosqlProvider
	} else if cfg.NoSQLDatabase.Provider == "" {
		// Default to postgresql if not set
		cfg.NoSQLDatabase.Provider = "postgresql"
	}

	// PostgreSQL URI override
	if pgURI := os.Getenv("POSTGRESQL_URI"); pgURI != "" {
		cfg.NoSQLDatabase.Provider = "postgresql"
		cfg.NoSQLDatabase.URI = pgURI
	} else if env != "" && cfg.NoSQLDatabase.URI == "" {
		// Environment-based configuration
		switch env {
		case "local":
			cfg.NoSQLDatabase.URI = "postgres://localhost:5432/gego?sslmode=disable"
		case "dev", "development":
			// Use cloud URI from environment or keep existing config
			if cloudURI := os.Getenv("POSTGRESQL_CLOUD_URI"); cloudURI != "" {
				cfg.NoSQLDatabase.URI = cloudURI
			}
		case "prod", "production":
			// Use production URI from environment
			if prodURI := os.Getenv("POSTGRESQL_PROD_URI"); prodURI != "" {
				cfg.NoSQLDatabase.URI = prodURI
			}
		}
	}

	// PostgreSQL Database name override
	if pgDBName := os.Getenv("POSTGRESQL_DATABASE"); pgDBName != "" {
		cfg.NoSQLDatabase.Database = pgDBName
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

// LoadClerkConfig loads Clerk configuration from environment variables
func LoadClerkConfig() ClerkConfig {
	return ClerkConfig{
		SecretKey:      os.Getenv("CLERK_SECRET_KEY"),
		PublishableKey: os.Getenv("CLERK_PUBLISHABLE_KEY"),
		JWKSURL:        os.Getenv("CLERK_JWKS_URL"),
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
