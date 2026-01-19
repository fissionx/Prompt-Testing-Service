package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// trackLatency accumulates PostgreSQL operation latency in context for summary logging
func trackLatency(ctx context.Context, operation, table string, start time.Time, err error) {
	duration := time.Since(start)

	// Accumulate time in context if available (from API requests)
	// Uses context key for database operation latency tracking
	if dbTotalTime, ok := ctx.Value("mongo_total_time").(*time.Duration); ok {
		*dbTotalTime += duration
	}
}

// PostgreSQL implements the Database interface for PostgreSQL
type PostgreSQL struct {
	db     *sql.DB
	config *models.Config
}

// New creates a new PostgreSQL database instance
func New(config *models.Config) (*PostgreSQL, error) {
	return &PostgreSQL{
		config: config,
	}, nil
}

// Connect establishes connection to PostgreSQL
func (p *PostgreSQL) Connect(ctx context.Context) error {
	// Parse connection string from URI
	connStr := p.config.URI
	if connStr == "" {
		return fmt.Errorf("PostgreSQL URI is required")
	}

	// Add connection parameters to reduce prepared statement caching issues
	// binary_parameters=yes helps with prepared statement handling
	// Also ensure we don't use server-side prepared statements by default
	// The lib/pq driver will handle this, but we add this for safety
	if !strings.Contains(connStr, "binary_parameters") {
		separator := "?"
		if strings.Contains(connStr, "?") {
			separator = "&"
		}
		connStr += separator + "binary_parameters=yes"
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}

	// Set connection pool settings
	// Reduced connection lifetime to help clear stale prepared statements
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute) // Reduced from 5 to 3 minutes to clear stale connections faster
	db.SetConnMaxIdleTime(1 * time.Minute) // Close idle connections after 1 minute

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	p.db = db

	// Create schema and indexes
	if err := p.createSchema(ctx); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	if err := p.createIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	if err := p.createAnalyticsCacheIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create analytics cache indexes: %w", err)
	}

	return nil
}

// Disconnect closes the PostgreSQL connection
func (p *PostgreSQL) Disconnect(ctx context.Context) error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// Ping checks the database connection
func (p *PostgreSQL) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("not connected to database")
	}
	return p.db.PingContext(ctx)
}

// GetDB returns the underlying *sql.DB connection
func (p *PostgreSQL) GetDB() *sql.DB {
	return p.db
}

// createSchema creates all necessary tables
func (p *PostgreSQL) createSchema(ctx context.Context) error {
	schema := `
	-- Prompts table
	CREATE TABLE IF NOT EXISTS prompts (
		id TEXT PRIMARY KEY,
		template TEXT NOT NULL,
		prompt_type TEXT,
		tags JSONB DEFAULT '[]'::jsonb,
		category TEXT,
		domain TEXT,
		brand TEXT,
		source_id TEXT,
		generated BOOLEAN DEFAULT false,
		enabled BOOLEAN DEFAULT true,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Responses table
	CREATE TABLE IF NOT EXISTS responses (
		id TEXT PRIMARY KEY,
		prompt_id TEXT NOT NULL,
		prompt_text TEXT,
		llm_id TEXT,
		llm_name TEXT,
		llm_provider TEXT,
		llm_model TEXT,
		response_text TEXT,
		brand TEXT,
		temperature DOUBLE PRECISION,
		metadata JSONB,
		schedule_id TEXT,
		tokens_used INTEGER,
		latency_ms BIGINT,
		error TEXT,
		visibility_score INTEGER,
		brand_mentioned BOOLEAN,
		in_grounding_sources BOOLEAN,
		grounding_sources JSONB DEFAULT '[]'::jsonb,
		sentiment TEXT,
		competitors_mention JSONB DEFAULT '[]'::jsonb,
		brand_position INTEGER,
		total_brands_listed INTEGER,
		grounding_domains JSONB DEFAULT '[]'::jsonb,
		web_search_queries JSONB DEFAULT '[]'::jsonb,
		web_search_calls JSONB DEFAULT '[]'::jsonb,
		search_answer TEXT,
		week TEXT,
		month TEXT,
		quarter TEXT,
		region TEXT,
		language TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Prompt Library table
	CREATE TABLE IF NOT EXISTS prompt_library (
		id TEXT PRIMARY KEY,
		brand TEXT,
		domain TEXT NOT NULL,
		category TEXT NOT NULL,
		prompt_ids JSONB DEFAULT '[]'::jsonb,
		usage_count INTEGER DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Brand Profiles table
	CREATE TABLE IF NOT EXISTS brand_profiles (
		id TEXT PRIMARY KEY,
		brand_name TEXT NOT NULL,
		domain TEXT,
		category TEXT,
		website TEXT,
		description TEXT,
		competitors JSONB DEFAULT '[]'::jsonb,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Brand Logos table
	CREATE TABLE IF NOT EXISTS brand_logos (
		id TEXT PRIMARY KEY,
		brand_name TEXT NOT NULL UNIQUE,
		domain TEXT,
		logo_url TEXT,
		fallback_logo_url TEXT,
		source TEXT,
		last_checked TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Brand Competitors table
	CREATE TABLE IF NOT EXISTS brand_competitors (
		id TEXT PRIMARY KEY,
		brand TEXT NOT NULL UNIQUE,
		competitors JSONB DEFAULT '[]'::jsonb,
		suggested_list JSONB DEFAULT '[]'::jsonb,
		source TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Brand Prompts table
	CREATE TABLE IF NOT EXISTS brand_prompts (
		id TEXT PRIMARY KEY,
		brand TEXT NOT NULL UNIQUE,
		active_prompt_ids JSONB DEFAULT '[]'::jsonb,
		suggested_prompt_ids JSONB DEFAULT '[]'::jsonb,
		source TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- GEO Campaigns table
	CREATE TABLE IF NOT EXISTS geo_campaigns (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		brand_id TEXT,
		brand TEXT,
		prompt_ids JSONB DEFAULT '[]'::jsonb,
		llm_ids JSONB DEFAULT '[]'::jsonb,
		status TEXT,
		total_runs INTEGER DEFAULT 0,
		completed_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Cached GEO Insights table
	CREATE TABLE IF NOT EXISTS cached_geo_insights (
		id TEXT PRIMARY KEY,
		campaign_id TEXT,
		brand TEXT,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP NOT NULL,
		data JSONB NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Cached Source Analytics table
	CREATE TABLE IF NOT EXISTS cached_source_analytics (
		id TEXT PRIMARY KEY,
		campaign_id TEXT,
		brand TEXT,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP NOT NULL,
		data JSONB NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Cached Competitive Benchmark table
	CREATE TABLE IF NOT EXISTS cached_competitive_benchmark (
		id TEXT PRIMARY KEY,
		campaign_id TEXT,
		main_brand TEXT,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP NOT NULL,
		data JSONB NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Cached Prompt Performance table
	CREATE TABLE IF NOT EXISTS cached_prompt_performance (
		id TEXT PRIMARY KEY,
		campaign_id TEXT,
		brand TEXT,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP NOT NULL,
		data JSONB NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Scheduled Campaigns table
	CREATE TABLE IF NOT EXISTS scheduled_campaigns (
		id TEXT PRIMARY KEY,
		campaign_name TEXT NOT NULL,
		brand TEXT,
		prompt_ids JSONB DEFAULT '[]'::jsonb,
		llm_ids JSONB DEFAULT '[]'::jsonb,
		temperature DOUBLE PRECISION,
		schedule_cron TEXT,
		status TEXT,
		total_runs INTEGER DEFAULT 0,
		run_count INTEGER DEFAULT 0,
		last_run_at TIMESTAMP,
		next_run_at TIMESTAMP,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	-- Cached Prompt Time Series table
	CREATE TABLE IF NOT EXISTS cached_prompt_time_series (
		id TEXT PRIMARY KEY,
		campaign_id TEXT,
		prompt_id TEXT,
		brand TEXT,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP NOT NULL,
		data JSONB NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);
	`

	_, err := p.db.ExecContext(ctx, schema)
	return err
}

// createIndexes creates necessary indexes for optimal query performance
func (p *PostgreSQL) createIndexes(ctx context.Context) error {
	indexes := []string{
		// Prompts indexes
		"CREATE INDEX IF NOT EXISTS idx_prompts_brand ON prompts(brand)",
		"CREATE INDEX IF NOT EXISTS idx_prompts_brand_created_at ON prompts(brand, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_prompts_enabled ON prompts(enabled)",

		// Responses indexes
		"CREATE INDEX IF NOT EXISTS idx_responses_prompt_id_created_at ON responses(prompt_id, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_responses_created_at ON responses(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_responses_brand ON responses(brand)",
		"CREATE INDEX IF NOT EXISTS idx_responses_brand_created_at ON responses(brand, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_responses_schedule_id ON responses(schedule_id) WHERE schedule_id IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_responses_llm_id ON responses(llm_id) WHERE llm_id IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_responses_response_text_gin ON responses USING gin(to_tsvector('english', response_text))",

		// Prompt Library indexes
		"CREATE INDEX IF NOT EXISTS idx_prompt_library_domain_category ON prompt_library(domain, category)",

		// Brand Profiles indexes
		"CREATE INDEX IF NOT EXISTS idx_brand_profiles_brand_name ON brand_profiles(brand_name)",

		// Brand Logos indexes (unique constraint already exists)
		"CREATE INDEX IF NOT EXISTS idx_brand_logos_brand_name ON brand_logos(brand_name)",

		// Brand Competitors indexes
		"CREATE INDEX IF NOT EXISTS idx_brand_competitors_brand ON brand_competitors(brand)",

		// Brand Prompts indexes
		"CREATE INDEX IF NOT EXISTS idx_brand_prompts_brand ON brand_prompts(brand)",

		// GEO Campaigns indexes
		"CREATE INDEX IF NOT EXISTS idx_geo_campaigns_brand ON geo_campaigns(brand)",
		"CREATE INDEX IF NOT EXISTS idx_geo_campaigns_status ON geo_campaigns(status)",
		"CREATE INDEX IF NOT EXISTS idx_geo_campaigns_brand_status_completed ON geo_campaigns(brand, status, completed_at) WHERE completed_at IS NULL",
	}

	for _, indexSQL := range indexes {
		if _, err := p.db.ExecContext(ctx, indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// createAnalyticsCacheIndexes creates indexes for analytics cache collections
func (p *PostgreSQL) createAnalyticsCacheIndexes(ctx context.Context) error {
	indexes := []string{
		// Cached GEO Insights indexes
		"CREATE INDEX IF NOT EXISTS idx_cached_geo_insights_brand_campaign ON cached_geo_insights(brand, campaign_id)",
		"CREATE INDEX IF NOT EXISTS idx_cached_geo_insights_brand_time ON cached_geo_insights(brand, start_time DESC, end_time DESC)",

		// Cached Source Analytics indexes
		"CREATE INDEX IF NOT EXISTS idx_cached_source_analytics_brand_campaign ON cached_source_analytics(brand, campaign_id)",
		"CREATE INDEX IF NOT EXISTS idx_cached_source_analytics_brand_time ON cached_source_analytics(brand, start_time DESC, end_time DESC)",

		// Cached Competitive Benchmark indexes
		"CREATE INDEX IF NOT EXISTS idx_cached_competitive_benchmark_main_brand_campaign ON cached_competitive_benchmark(main_brand, campaign_id)",
		"CREATE INDEX IF NOT EXISTS idx_cached_competitive_benchmark_main_brand_time ON cached_competitive_benchmark(main_brand, start_time DESC, end_time DESC)",

		// Cached Prompt Performance indexes
		"CREATE INDEX IF NOT EXISTS idx_cached_prompt_performance_brand_campaign ON cached_prompt_performance(brand, campaign_id)",
		"CREATE INDEX IF NOT EXISTS idx_cached_prompt_performance_brand_time ON cached_prompt_performance(brand, start_time DESC, end_time DESC)",

		// Scheduled Campaigns indexes
		"CREATE INDEX IF NOT EXISTS idx_scheduled_campaigns_brand ON scheduled_campaigns(brand)",
		"CREATE INDEX IF NOT EXISTS idx_scheduled_campaigns_status ON scheduled_campaigns(status)",
		"CREATE INDEX IF NOT EXISTS idx_scheduled_campaigns_next_run_at ON scheduled_campaigns(next_run_at) WHERE next_run_at IS NOT NULL",

		// Cached Prompt Time Series indexes
		"CREATE INDEX IF NOT EXISTS idx_cached_prompt_time_series_prompt_brand ON cached_prompt_time_series(prompt_id, brand)",
		"CREATE INDEX IF NOT EXISTS idx_cached_prompt_time_series_prompt_campaign ON cached_prompt_time_series(prompt_id, campaign_id)",
		"CREATE INDEX IF NOT EXISTS idx_cached_prompt_time_series_prompt_time ON cached_prompt_time_series(prompt_id, start_time DESC, end_time DESC)",
	}

	for _, indexSQL := range indexes {
		if _, err := p.db.ExecContext(ctx, indexSQL); err != nil {
			return fmt.Errorf("failed to create analytics cache index: %w", err)
		}
	}

	return nil
}

// Helper functions for JSON operations
func jsonToSlice(jsonStr string) []string {
	if jsonStr == "" || jsonStr == "[]" || jsonStr == "null" {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return []string{}
	}
	return result
}

func sliceToJSON(slice []string) string {
	if len(slice) == 0 {
		return "[]"
	}
	data, err := json.Marshal(slice)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func jsonToMap(jsonStr string) map[string]interface{} {
	if jsonStr == "" || jsonStr == "{}" || jsonStr == "null" {
		return make(map[string]interface{})
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return make(map[string]interface{})
	}
	return result
}

func mapToJSON(m map[string]interface{}) string {
	if len(m) == 0 {
		return "{}"
	}
	data, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// CreatePrompt creates a new prompt
func (p *PostgreSQL) CreatePrompt(ctx context.Context, prompt *models.Prompt) error {
	start := time.Now()
	prompt.CreatedAt = time.Now()
	prompt.UpdatedAt = time.Now()

	// Compress template if large
	template := prompt.Template
	if shared.ShouldCompress(template) {
		if compressed, err := shared.CompressString(template); err == nil {
			template = compressed
		}
	}

	query := `
		INSERT INTO prompts (id, template, prompt_type, tags, category, domain, brand, source_id, generated, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := p.db.ExecContext(ctx, query,
		prompt.ID,
		template,
		string(prompt.PromptType),
		sliceToJSON(prompt.Tags),
		prompt.Category,
		prompt.Domain,
		prompt.Brand,
		prompt.SourceID,
		prompt.Generated,
		prompt.Enabled,
		prompt.CreatedAt,
		prompt.UpdatedAt,
	)

	trackLatency(ctx, "CreatePrompt", "prompts", start, err)
	return err
}

// GetPrompt retrieves a prompt by ID
func (p *PostgreSQL) GetPrompt(ctx context.Context, id string) (*models.Prompt, error) {
	start := time.Now()
	query := `
		SELECT id, template, prompt_type, tags, category, domain, brand, source_id, generated, enabled, created_at, updated_at
		FROM prompts WHERE id = $1
	`

	var prompt models.Prompt
	var tagsJSON, promptTypeStr string

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&prompt.ID,
		&prompt.Template,
		&promptTypeStr,
		&tagsJSON,
		&prompt.Category,
		&prompt.Domain,
		&prompt.Brand,
		&prompt.SourceID,
		&prompt.Generated,
		&prompt.Enabled,
		&prompt.CreatedAt,
		&prompt.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		trackLatency(ctx, "GetPrompt", "prompts", start, fmt.Errorf("prompt not found: %s", id))
		return nil, fmt.Errorf("prompt not found: %s", id)
	}
	if err != nil {
		trackLatency(ctx, "GetPrompt", "prompts", start, err)
		return nil, err
	}

	prompt.PromptType = models.PromptType(promptTypeStr)
	prompt.Tags = jsonToSlice(tagsJSON)

	// Decompress template if it was compressed
	if decompressed, err := shared.DecompressString(prompt.Template); err == nil {
		prompt.Template = decompressed
	}

	trackLatency(ctx, "GetPrompt", "prompts", start, nil)
	return &prompt, nil
}

// ListPrompts lists all prompts, optionally filtered by enabled status
func (p *PostgreSQL) ListPrompts(ctx context.Context, enabled *bool) ([]*models.Prompt, error) {
	start := time.Now()
	query := `SELECT id, template, prompt_type, tags, category, domain, brand, source_id, generated, enabled, created_at, updated_at 
		FROM prompts WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	if enabled != nil {
		query += fmt.Sprintf(" AND enabled = $%d", argPos)
		args = append(args, *enabled)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	// Use QueryContext directly to avoid prepared statement caching issues
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		trackLatency(ctx, "ListPrompts", "prompts", start, err)
		return nil, err
	}
	defer rows.Close()

	var prompts []*models.Prompt
	for rows.Next() {
		var prompt models.Prompt
		var tagsJSON, promptTypeStr string

		err := rows.Scan(
			&prompt.ID,
			&prompt.Template,
			&promptTypeStr,
			&tagsJSON,
			&prompt.Category,
			&prompt.Domain,
			&prompt.Brand,
			&prompt.SourceID,
			&prompt.Generated,
			&prompt.Enabled,
			&prompt.CreatedAt,
			&prompt.UpdatedAt,
		)
		if err != nil {
			trackLatency(ctx, "ListPrompts", "prompts", start, err)
			return nil, err
		}

		prompt.PromptType = models.PromptType(promptTypeStr)
		prompt.Tags = jsonToSlice(tagsJSON)

		// Decompress template if it was compressed
		if decompressed, err := shared.DecompressString(prompt.Template); err == nil {
			prompt.Template = decompressed
		}

		prompts = append(prompts, &prompt)
	}

	trackLatency(ctx, "ListPrompts", "prompts", start, nil)
	return prompts, nil
}

// GetPromptsByIDs retrieves multiple prompts by their IDs in a single query
func (p *PostgreSQL) GetPromptsByIDs(ctx context.Context, ids []string) ([]*models.Prompt, error) {
	start := time.Now()
	if len(ids) == 0 {
		return []*models.Prompt{}, nil
	}

	// Build query with IN clause
	query := `SELECT id, template, prompt_type, tags, category, domain, brand, source_id, generated, enabled, created_at, updated_at 
		FROM prompts WHERE id = ANY($1)`

	rows, err := p.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		trackLatency(ctx, "GetPromptsByIDs", "prompts", start, err)
		return nil, err
	}
	defer rows.Close()

	var prompts []*models.Prompt
	for rows.Next() {
		var prompt models.Prompt
		var tagsJSON, promptTypeStr string

		err := rows.Scan(
			&prompt.ID,
			&prompt.Template,
			&promptTypeStr,
			&tagsJSON,
			&prompt.Category,
			&prompt.Domain,
			&prompt.Brand,
			&prompt.SourceID,
			&prompt.Generated,
			&prompt.Enabled,
			&prompt.CreatedAt,
			&prompt.UpdatedAt,
		)
		if err != nil {
			trackLatency(ctx, "GetPromptsByIDs", "prompts", start, err)
			return nil, err
		}

		prompt.PromptType = models.PromptType(promptTypeStr)
		prompt.Tags = jsonToSlice(tagsJSON)

		// Decompress template if it was compressed
		if decompressed, err := shared.DecompressString(prompt.Template); err == nil {
			prompt.Template = decompressed
		}

		prompts = append(prompts, &prompt)
	}

	trackLatency(ctx, "GetPromptsByIDs", "prompts", start, nil)
	return prompts, nil
}

// ListPromptsByBrand lists prompts filtered by brand and optionally by enabled status
func (p *PostgreSQL) ListPromptsByBrand(ctx context.Context, brand string, enabled *bool) ([]*models.Prompt, error) {
	start := time.Now()
	query := `SELECT id, template, prompt_type, tags, category, domain, brand, source_id, generated, enabled, created_at, updated_at 
		FROM prompts WHERE brand = $1`
	args := []interface{}{brand}
	argPos := 2

	if enabled != nil {
		query += fmt.Sprintf(" AND enabled = $%d", argPos)
		args = append(args, *enabled)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		trackLatency(ctx, "ListPromptsByBrand", "prompts", start, err)
		return nil, err
	}
	defer rows.Close()

	var prompts []*models.Prompt
	for rows.Next() {
		var prompt models.Prompt
		var tagsJSON, promptTypeStr string

		err := rows.Scan(
			&prompt.ID,
			&prompt.Template,
			&promptTypeStr,
			&tagsJSON,
			&prompt.Category,
			&prompt.Domain,
			&prompt.Brand,
			&prompt.SourceID,
			&prompt.Generated,
			&prompt.Enabled,
			&prompt.CreatedAt,
			&prompt.UpdatedAt,
		)
		if err != nil {
			trackLatency(ctx, "ListPromptsByBrand", "prompts", start, err)
			return nil, err
		}

		prompt.PromptType = models.PromptType(promptTypeStr)
		prompt.Tags = jsonToSlice(tagsJSON)

		// Decompress template if it was compressed
		if decompressed, err := shared.DecompressString(prompt.Template); err == nil {
			prompt.Template = decompressed
		}

		prompts = append(prompts, &prompt)
	}

	trackLatency(ctx, "ListPromptsByBrand", "prompts", start, nil)
	return prompts, nil
}

// UpdatePrompt updates an existing prompt
func (p *PostgreSQL) UpdatePrompt(ctx context.Context, prompt *models.Prompt) error {
	start := time.Now()
	prompt.UpdatedAt = time.Now()

	// Compress template if large
	template := prompt.Template
	if shared.ShouldCompress(template) {
		if compressed, err := shared.CompressString(template); err == nil {
			template = compressed
		}
	}

	query := `
		UPDATE prompts 
		SET template = $1, prompt_type = $2, tags = $3, category = $4, domain = $5, brand = $6, source_id = $7, generated = $8, enabled = $9, updated_at = $10
		WHERE id = $11
	`

	result, err := p.db.ExecContext(ctx, query,
		template,
		string(prompt.PromptType),
		sliceToJSON(prompt.Tags),
		prompt.Category,
		prompt.Domain,
		prompt.Brand,
		prompt.SourceID,
		prompt.Generated,
		prompt.Enabled,
		prompt.UpdatedAt,
		prompt.ID,
	)

	if err != nil {
		trackLatency(ctx, "UpdatePrompt", "prompts", start, err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		trackLatency(ctx, "UpdatePrompt", "prompts", start, err)
		return err
	}

	if rowsAffected == 0 {
		err := fmt.Errorf("prompt not found: %s", prompt.ID)
		trackLatency(ctx, "UpdatePrompt", "prompts", start, err)
		return err
	}

	trackLatency(ctx, "UpdatePrompt", "prompts", start, nil)
	return nil
}

// DeletePrompt deletes a prompt by ID
func (p *PostgreSQL) DeletePrompt(ctx context.Context, id string) error {
	start := time.Now()
	query := "DELETE FROM prompts WHERE id = $1"
	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		trackLatency(ctx, "DeletePrompt", "prompts", start, err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		trackLatency(ctx, "DeletePrompt", "prompts", start, err)
		return err
	}

	if rowsAffected == 0 {
		err := fmt.Errorf("prompt not found: %s", id)
		trackLatency(ctx, "DeletePrompt", "prompts", start, err)
		return err
	}

	trackLatency(ctx, "DeletePrompt", "prompts", start, nil)
	return nil
}

// DeleteAllPrompts deletes all prompts
func (p *PostgreSQL) DeleteAllPrompts(ctx context.Context) (int, error) {
	start := time.Now()
	query := "DELETE FROM prompts"
	result, err := p.db.ExecContext(ctx, query)
	if err != nil {
		trackLatency(ctx, "DeleteAllPrompts", "prompts", start, err)
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		trackLatency(ctx, "DeleteAllPrompts", "prompts", start, err)
		return 0, err
	}

	trackLatency(ctx, "DeleteAllPrompts", "prompts", start, nil)
	return int(rowsAffected), nil
}

// DeletePromptsByBrand deletes all prompts for a specific brand
func (p *PostgreSQL) DeletePromptsByBrand(ctx context.Context, brand string) (int, error) {
	start := time.Now()
	query := "DELETE FROM prompts WHERE LOWER(brand) = LOWER($1)"
	result, err := p.db.ExecContext(ctx, query, brand)
	if err != nil {
		trackLatency(ctx, "DeletePromptsByBrand", "prompts", start, err)
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		trackLatency(ctx, "DeletePromptsByBrand", "prompts", start, err)
		return 0, err
	}

	trackLatency(ctx, "DeletePromptsByBrand", "prompts", start, nil)
	return int(rowsAffected), nil
}
