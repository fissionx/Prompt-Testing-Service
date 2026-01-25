package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"go.uber.org/zap"

	"github.com/fissionx/gego/internal/api"
	"github.com/fissionx/gego/internal/config"
	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/llm"
	"github.com/fissionx/gego/internal/llm/openai"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

var (
	apiPort    = flag.String("port", "8989", "Port to run the API server on")
	apiHost    = flag.String("host", "0.0.0.0", "Host to bind the API server to")
	corsOrigin = flag.String("cors-origin", "", "CORS origin to allow (overrides config file, use '*' for all origins)")
	configPath = flag.String("config", "", "Config file path (default is $HOME/.gego/config.yaml)")
)

func main() {
	flag.Parse()

	var cfgPath string
	if *configPath != "" {
		cfgPath = *configPath
	} else if envPath := os.Getenv("GEGO_CONFIG_PATH"); envPath != "" {
		cfgPath = envPath
	} else {
		cfgPath = config.GetConfigPath()
	}

	if !config.Exists(cfgPath) {
		fmt.Fprintf(os.Stderr, "Error: configuration file not found at %s\n", cfgPath)
		fmt.Fprintf(os.Stderr, "Please create a configuration file or set GEGO_CONFIG_PATH environment variable\n")
		os.Exit(1)
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	if cfg.KeywordsExclusionPath != "" {
		exclusionPath := cfg.KeywordsExclusionPath
		if !filepath.IsAbs(exclusionPath) {
			configDir := filepath.Dir(cfgPath)
			exclusionPath = filepath.Join(configDir, exclusionPath)
		}
		shared.SetExclusionFilePath(exclusionPath)
	}

	selectedCORSOrigin := *corsOrigin
	if selectedCORSOrigin == "" {
		if cfg.CORSOrigin != "" {
			selectedCORSOrigin = cfg.CORSOrigin
		} else {
			selectedCORSOrigin = "*"
		}
	}

	fmt.Printf("🚀 Starting Gego API Server\n")
	fmt.Printf("===========================\n")
	fmt.Printf("Host: %s\n", *apiHost)
	fmt.Printf("Port: %s\n", *apiPort)
	fmt.Printf("CORS Origin: %s\n", selectedCORSOrigin)
	fmt.Printf("URL: http://%s:%s/api/v1\n", *apiHost, *apiPort)
	fmt.Println()

	sqlConfig := &models.Config{
		Provider: cfg.SQLDatabase.Provider,
		URI:      cfg.SQLDatabase.URI,
		Database: cfg.SQLDatabase.Database,
		Options:  cfg.SQLDatabase.Options,
	}

	nosqlConfig := &models.Config{
		Provider: cfg.NoSQLDatabase.Provider,
		URI:      cfg.NoSQLDatabase.URI,
		Database: cfg.NoSQLDatabase.Database,
		Options:  cfg.NoSQLDatabase.Options,
	}

	fmt.Printf("📊 Database Configuration:\n")
	fmt.Printf("  SQL Database: %s (%s)\n", sqlConfig.Provider, sqlConfig.URI)
	fmt.Printf("  NoSQL Database: %s\n", nosqlConfig.Provider)
	if nosqlConfig.Provider == "postgresql" || nosqlConfig.Provider == "postgres" {
		if nosqlConfig.URI == "" {
			fmt.Fprintf(os.Stderr, "\n❌ ERROR: PostgreSQL URI is required but not set!\n\n")
			fmt.Fprintf(os.Stderr, "For Fly.io deployment, set the secret with:\n")
			fmt.Fprintf(os.Stderr, "  flyctl secrets set POSTGRESQL_URI=\"postgresql://user:password@host:port/database?sslmode=require\" --app gego\n\n")
			fmt.Fprintf(os.Stderr, "Or set the POSTGRESQL_URI environment variable.\n")
			fmt.Fprintf(os.Stderr, "Example: postgresql://user:pass@hostname:5432/gego?sslmode=require\n\n")
			os.Exit(1)
		}
		maskedURI := maskPasswordInURI(nosqlConfig.URI)
		fmt.Printf("  PostgreSQL URI: %s\n", maskedURI)
		fmt.Printf("  Database Name: %s\n", nosqlConfig.Database)
		fmt.Printf("  ✅ Using PostgreSQL for NoSQL operations\n")
	} else {
		fmt.Printf("  ⚠️  Unsupported NoSQL database provider: %s (only postgresql/postgres is supported)\n", nosqlConfig.Provider)
	}
	fmt.Println()

	database, err := db.New(sqlConfig, nosqlConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: failed to create hybrid database: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	fmt.Println("🔌 Connecting to databases...")
	if err := database.Connect(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Error: failed to connect to database: %v\n\n", err)
		if nosqlConfig.Provider == "postgresql" || nosqlConfig.Provider == "postgres" {
			fmt.Fprintf(os.Stderr, "This is likely a PostgreSQL connection issue.\n")
			fmt.Fprintf(os.Stderr, "Please verify:\n")
			fmt.Fprintf(os.Stderr, "  1. POSTGRESQL_URI secret is set correctly in Fly.io\n")
			fmt.Fprintf(os.Stderr, "  2. The database is accessible from Fly.io\n")
			fmt.Fprintf(os.Stderr, "  3. The connection string format is correct\n")
			fmt.Fprintf(os.Stderr, "\nCheck secrets with: flyctl secrets list --app gego\n")
		}
		os.Exit(1)
	}
	defer database.Disconnect(ctx)

	fmt.Println("🔍 Testing database connection...")
	if err := database.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Error: database ping failed: %v\n\n", err)
		if nosqlConfig.Provider == "postgresql" || nosqlConfig.Provider == "postgres" {
			fmt.Fprintf(os.Stderr, "The database connection string may be incorrect or the database is unreachable.\n")
			fmt.Fprintf(os.Stderr, "Verify your POSTGRESQL_URI secret in Fly.io.\n")
		}
		os.Exit(1)
	}

	fmt.Println("✅ Database connection successful!")

	fmt.Println("\n🔄 Running database migrations...")
	if err := runMigrations(ctx, database); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to run migrations: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ Database migrations completed successfully!")

	// Initialize LLM registry with all providers
	apiLLMRegistry := llm.NewRegistry()
	apiLLMRegistry.Register(openai.New("", ""))

	// Initialize providers with API keys from database
	llms, err := database.ListLLMs(ctx, nil)
	if err == nil {
		for _, llmConfig := range llms {
			if llmConfig.APIKey != "" {
				switch llmConfig.Provider {
				case "openai":
					apiLLMRegistry.Register(openai.New(llmConfig.APIKey, llmConfig.BaseURL))
				}
			}
		}
	}
	fmt.Println("✅ LLM providers initialized!")

	// Load Clerk configuration from environment variables
	clerkConfig := config.LoadClerkConfig()

	// Initialize zap logger
	zapLogger, err := zap.NewProduction()
	if err != nil {
		zapLogger, _ = zap.NewDevelopment()
	}
	defer zapLogger.Sync()

	var apiLogger config.Logger = zapLogger

	// Display auth configuration
	env := strings.ToLower(os.Getenv("GEGO_ENV"))
	isLocalEnv := env == "local"

	if isLocalEnv {
		fmt.Printf("🔓 Authentication: Disabled (GEGO_ENV=local - local development mode)\n")
	} else if clerkConfig.SecretKey != "" {
		fmt.Printf("🔐 Authentication: Enabled (Clerk)\n")
		if clerkConfig.JWKSURL != "" {
			fmt.Printf("  JWKS URL: %s\n", clerkConfig.JWKSURL)
		}
	} else {
		fmt.Printf("⚠️  Authentication: Disabled (CLERK_SECRET_KEY not set)\n")
	}
	fmt.Println()

	server := api.NewServer(database, apiLLMRegistry, selectedCORSOrigin, clerkConfig, apiLogger)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\n🛑 Shutting down API server...")
		database.Disconnect(ctx)
		os.Exit(0)
	}()

	fmt.Println("🌐 API Server is running!")
	fmt.Println()
	fmt.Println("📚 Available Endpoints:")
	fmt.Println("  LLMs:")
	fmt.Println("    GET    /api/v1/llms              - List all LLMs")
	fmt.Println("    GET    /api/v1/llms/:id          - Get specific LLM")
	fmt.Println("    POST   /api/v1/llms              - Create new LLM")
	fmt.Println("    PUT    /api/v1/llms/:id          - Update LLM")
	fmt.Println("    DELETE /api/v1/llms/:id          - Delete LLM")
	fmt.Println()
	fmt.Println("  Prompts:")
	fmt.Println("    GET    /api/v1/prompts           - List all prompts")
	fmt.Println("    GET    /api/v1/prompts/:id       - Get specific prompt")
	fmt.Println("    POST   /api/v1/prompts           - Create new prompt")
	fmt.Println("    PUT    /api/v1/prompts/:id       - Update prompt")
	fmt.Println("    DELETE /api/v1/prompts/:id       - Delete prompt")
	fmt.Println()
	fmt.Println("  Schedules:")
	fmt.Println("    GET    /api/v1/schedules         - List all schedules")
	fmt.Println("    GET    /api/v1/schedules/:id     - Get specific schedule")
	fmt.Println("    POST   /api/v1/schedules         - Create new schedule")
	fmt.Println("    PUT    /api/v1/schedules/:id     - Update schedule")
	fmt.Println("    DELETE /api/v1/schedules/:id     - Delete schedule")
	fmt.Println()
	fmt.Println("  Stats & Search:")
	fmt.Println("    GET    /api/v1/stats             - Get statistics")
	fmt.Println("    POST   /api/v1/search            - Search keywords")
	fmt.Println("    GET    /api/v1/health            - Health check")
	fmt.Println()
	fmt.Println("  Execute:")
	fmt.Println("    POST   /api/v1/execute           - Execute prompt with LLM")
	fmt.Println()
	fmt.Println("  Actions:")
	fmt.Println("    GET    /api/v1/opportunities/:opportunityId/actions - Get actions for an opportunity")
	fmt.Println("    GET    /api/v1/actions/:actionId                    - Get a single action")
	fmt.Println("    POST   /api/v1/actions/:actionId/status             - Update action status")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop the server")

	address := fmt.Sprintf("%s:%s", *apiHost, *apiPort)
	if err := server.Run(address); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to start server: %v\n", err)
		os.Exit(1)
	}
}

// maskPasswordInURI masks the password in a database URI for safe display
func maskPasswordInURI(uri string) string {
	if strings.Contains(uri, "@") {
		parts := strings.Split(uri, "@")
		if len(parts) == 2 {
			beforeAt := parts[0]
			if strings.Contains(beforeAt, ":") {
				userPass := strings.Split(beforeAt, ":")
				if len(userPass) >= 3 {
					userPass[2] = "***"
					masked := strings.Join(userPass, ":") + "@" + parts[1]
					return masked
				} else if len(userPass) == 2 {
					userPass[1] = "***"
					masked := strings.Join(userPass, ":") + "@" + parts[1]
					return masked
				}
			}
		}
	}
	return uri
}

func runMigrations(ctx context.Context, database db.Database) error {
	hybridDB, ok := database.(*db.HybridDB)
	if !ok {
		return fmt.Errorf("database is not a HybridDB instance")
	}

	sqliteDB := hybridDB.GetSQLiteDatabase()
	if sqliteDB == nil {
		return fmt.Errorf("SQLite database not available")
	}

	migrationsDir := os.Getenv("GEGO_MIGRATIONS_DIR")
	if migrationsDir == "" {
		migrationsDir = "/migrations"
		if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
			workDir, _ := os.Getwd()
			migrationsDir = filepath.Join(workDir, "internal", "db", "migrations")
		}
	}

	sqlDB := sqliteDB.GetDB()
	if sqlDB == nil {
		return fmt.Errorf("database connection not available")
	}

	return db.RunMigrations(ctx, sqlDB, migrationsDir)
}
