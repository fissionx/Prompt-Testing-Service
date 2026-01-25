package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/fissionx/gego/internal/models"
)

// SQLite implements the Database interface for SQLite
type SQLite struct {
	db     *sql.DB
	config *models.Config
	mu     sync.RWMutex // Protects database connection from concurrent access
}

// New creates a new SQLite database instance
func New(config *models.Config) (*SQLite, error) {
	return &SQLite{
		config: config,
	}, nil
}

// Connect establishes connection to SQLite
func (s *SQLite) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connectInternal(ctx)
}

// Disconnect closes the SQLite connection
func (s *SQLite) Disconnect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.db != nil {
		err := s.db.Close()
		s.db = nil
		return err
	}
	return nil
}

// Ping checks the database connection and reconnects if necessary
func (s *SQLite) Ping(ctx context.Context) error {
	s.mu.RLock()
	db := s.db
	s.mu.RUnlock()
	
	if db == nil {
		// Try to reconnect if not connected
		s.mu.Lock()
		if s.db == nil {
			if err := s.connectInternal(ctx); err != nil {
				s.mu.Unlock()
				return fmt.Errorf("not connected to database and reconnect failed: %w", err)
			}
		}
		db = s.db
		s.mu.Unlock()
	}
	
	// Try to ping the database
	err := db.PingContext(ctx)
	if err != nil {
		// If ping fails with "database is closed", always try to reconnect
		// The connection is definitely closed, so we need a new one
		if strings.Contains(err.Error(), "database is closed") {
			s.mu.Lock()
			// Always reconnect when we get "database is closed" error
			// Close the old connection if it still exists
			if s.db != nil {
				s.db.Close() // Ignore close errors
				s.db = nil
			}
			// Attempt to reconnect
			if reconnectErr := s.connectInternal(ctx); reconnectErr != nil {
				s.mu.Unlock()
				return fmt.Errorf("database connection closed and reconnect failed: %w (original error: %v)", reconnectErr, err)
			}
			db = s.db
			s.mu.Unlock()
			// Retry ping after reconnection
			if db != nil {
				return db.PingContext(ctx)
			}
			return fmt.Errorf("database connection closed and could not reconnect: %v", err)
		}
		return err
	}
	
	return nil
}

// connectInternal performs the actual connection logic (without locking)
func (s *SQLite) connectInternal(ctx context.Context) error {
	// Close existing connection if any
	if s.db != nil {
		s.db.Close()
		s.db = nil
	}
	
	dbPath := s.config.URI
	if strings.HasPrefix(dbPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(home, dbPath[1:])
	} else if !filepath.IsAbs(dbPath) {
		absPath, err := filepath.Abs(dbPath)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path: %w", err)
		}
		dbPath = absPath
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Configure SQLite connection string with proper settings for production
	// Use WAL mode for better concurrency, longer busy timeout for mounted volumes
	// Add _sync=NORMAL for better performance on network filesystems (like Fly.io volumes)
	connStr := dbPath + "?_journal_mode=WAL&_busy_timeout=10000&_foreign_keys=1&_sync=NORMAL&_cache=shared"
	
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database at path '%s': %w", dbPath, err)
	}

	// Configure connection pool settings for production use
	// Keep connections alive longer to avoid "database is closed" errors
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2) // Keep 2 idle connections alive
	db.SetConnMaxLifetime(30 * time.Minute) // Keep connections alive for 30 minutes
	db.SetConnMaxIdleTime(15 * time.Minute) // Close idle connections after 15 minutes

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping SQLite database at path '%s': %w", dbPath, err)
	}

	s.db = db
	return nil
}

// GetDB returns the underlying *sql.DB connection for migrations
func (s *SQLite) GetDB() *sql.DB {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.db
}

func mapToJSON(m map[string]string) string {
	if len(m) == 0 {
		return "{}"
	}
	result := "{"
	first := true
	for k, v := range m {
		if !first {
			result += ","
		}
		result += fmt.Sprintf(`"%s":"%s"`, k, v)
		first = false
	}
	result += "}"
	return result
}

func jsonToMap(jsonStr string) map[string]string {
	if jsonStr == "" || jsonStr == "{}" {
		return make(map[string]string)
	}
	return make(map[string]string)
}

func sliceToJSON(slice []string) string {
	if len(slice) == 0 {
		return "[]"
	}
	result := "["
	for i, s := range slice {
		if i > 0 {
			result += ","
		}
		result += fmt.Sprintf(`"%s"`, s)
	}
	result += "]"
	return result
}

func jsonToSlice(jsonStr string) []string {
	if jsonStr == "" || jsonStr == "[]" {
		return []string{}
	}

	jsonStr = strings.TrimSpace(jsonStr)
	if !strings.HasPrefix(jsonStr, "[") || !strings.HasSuffix(jsonStr, "]") {
		return []string{}
	}

	jsonStr = jsonStr[1 : len(jsonStr)-1]
	if jsonStr == "" {
		return []string{}
	}

	parts := strings.Split(jsonStr, ",")
	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, `"`) && strings.HasSuffix(part, `"`) {
			part = part[1 : len(part)-1]
		}
		result = append(result, part)
	}

	return result
}

// CreateLLM creates a new LLM configuration
func (s *SQLite) CreateLLM(ctx context.Context, llm *models.LLMConfig) error {
	llm.CreatedAt = time.Now()
	llm.UpdatedAt = time.Now()

	query := `
		INSERT INTO llms (id, name, provider, model, api_key, base_url, config, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		llm.ID,
		llm.Name,
		llm.Provider,
		llm.Model,
		llm.APIKey,
		llm.BaseURL,
		mapToJSON(llm.Config),
		llm.Enabled,
		llm.CreatedAt,
		llm.UpdatedAt,
	)

	return err
}

// GetLLM retrieves an LLM configuration by ID
func (s *SQLite) GetLLM(ctx context.Context, id string) (*models.LLMConfig, error) {
	query := `
		SELECT id, name, provider, model, api_key, base_url, config, enabled, created_at, updated_at
		FROM llms WHERE id = ?`

	var llm models.LLMConfig
	var configJSON string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&llm.ID,
		&llm.Name,
		&llm.Provider,
		&llm.Model,
		&llm.APIKey,
		&llm.BaseURL,
		&configJSON,
		&llm.Enabled,
		&llm.CreatedAt,
		&llm.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("LLM not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	llm.Config = jsonToMap(configJSON)
	return &llm, nil
}

// GetLLMsByIDs retrieves multiple LLM configurations by their IDs in a single query
func (s *SQLite) GetLLMsByIDs(ctx context.Context, ids []string) ([]*models.LLMConfig, error) {
	if len(ids) == 0 {
		return []*models.LLMConfig{}, nil
	}

	// Build query with IN clause using placeholders
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, name, provider, model, api_key, base_url, config, enabled, created_at, updated_at
		FROM llms WHERE id IN (%s)
		ORDER BY created_at DESC`,
		strings.Join(placeholders, ","))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var llms []*models.LLMConfig
	for rows.Next() {
		var llm models.LLMConfig
		var configJSON string

		err := rows.Scan(
			&llm.ID,
			&llm.Name,
			&llm.Provider,
			&llm.Model,
			&llm.APIKey,
			&llm.BaseURL,
			&configJSON,
			&llm.Enabled,
			&llm.CreatedAt,
			&llm.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		llm.Config = jsonToMap(configJSON)
		llms = append(llms, &llm)
	}

	return llms, nil
}

// ListLLMs lists all LLM configurations, optionally filtered by enabled status
func (s *SQLite) ListLLMs(ctx context.Context, enabled *bool) ([]*models.LLMConfig, error) {
	query := `
		SELECT id, name, provider, model, api_key, base_url, config, enabled, created_at, updated_at
		FROM llms`
	args := []interface{}{}

	if enabled != nil {
		query += " WHERE enabled = ?"
		args = append(args, *enabled)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var llms []*models.LLMConfig
	for rows.Next() {
		var llm models.LLMConfig
		var configJSON string

		err := rows.Scan(
			&llm.ID,
			&llm.Name,
			&llm.Provider,
			&llm.Model,
			&llm.APIKey,
			&llm.BaseURL,
			&configJSON,
			&llm.Enabled,
			&llm.CreatedAt,
			&llm.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		llm.Config = jsonToMap(configJSON)
		llms = append(llms, &llm)
	}

	return llms, nil
}

// UpdateLLM updates an existing LLM configuration
func (s *SQLite) UpdateLLM(ctx context.Context, llm *models.LLMConfig) error {
	llm.UpdatedAt = time.Now()

	query := `
		UPDATE llms 
		SET name = ?, provider = ?, model = ?, api_key = ?, base_url = ?, config = ?, enabled = ?, updated_at = ?
		WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query,
		llm.Name,
		llm.Provider,
		llm.Model,
		llm.APIKey,
		llm.BaseURL,
		mapToJSON(llm.Config),
		llm.Enabled,
		llm.UpdatedAt,
		llm.ID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("LLM not found: %s", llm.ID)
	}

	return nil
}

// DeleteLLM deletes an LLM configuration
func (s *SQLite) DeleteLLM(ctx context.Context, id string) error {
	query := "DELETE FROM llms WHERE id = ?"
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("LLM not found: %s", id)
	}

	return nil
}

// DeleteAllLLMs deletes all LLM configurations
func (s *SQLite) DeleteAllLLMs(ctx context.Context) (int, error) {
	query := "DELETE FROM llms"
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// CreateSchedule creates a new schedule
func (s *SQLite) CreateSchedule(ctx context.Context, schedule *models.Schedule) error {
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()

	query := `
		INSERT INTO schedules (id, brand_id, name, prompt_ids, llm_ids, cron_expr, temperature, enabled, last_run, next_run, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.ExecContext(ctx, query,
		schedule.ID,
		schedule.BrandID,
		schedule.Name,
		sliceToJSON(schedule.PromptIDs),
		sliceToJSON(schedule.LLMIDs),
		schedule.CronExpr,
		schedule.Temperature,
		schedule.Enabled,
		schedule.LastRun,
		schedule.NextRun,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	)

	return err
}

// GetSchedule retrieves a schedule by ID
func (s *SQLite) GetSchedule(ctx context.Context, id string) (*models.Schedule, error) {
	query := `
		SELECT id, brand_id, name, prompt_ids, llm_ids, cron_expr, temperature, enabled, last_run, next_run, created_at, updated_at
		FROM schedules WHERE id = ?`

	var schedule models.Schedule
	var promptIDsJSON, llmIDsJSON string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&schedule.ID,
		&schedule.BrandID,
		&schedule.Name,
		&promptIDsJSON,
		&llmIDsJSON,
		&schedule.CronExpr,
		&schedule.Temperature,
		&schedule.Enabled,
		&schedule.LastRun,
		&schedule.NextRun,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	schedule.PromptIDs = jsonToSlice(promptIDsJSON)
	schedule.LLMIDs = jsonToSlice(llmIDsJSON)
	return &schedule, nil
}

// ListSchedules lists all schedules, optionally filtered by brandId and enabled status
func (s *SQLite) ListSchedules(ctx context.Context, brandId string, enabled *bool) ([]*models.Schedule, error) {
	query := `
		SELECT id, brand_id, name, prompt_ids, llm_ids, cron_expr, temperature, enabled, last_run, next_run, created_at, updated_at
		FROM schedules`
	args := []interface{}{}
	whereClauses := []string{}

	if brandId != "" {
		whereClauses = append(whereClauses, "brand_id = ?")
		args = append(args, brandId)
	}

	if enabled != nil {
		whereClauses = append(whereClauses, "enabled = ?")
		args = append(args, *enabled)
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []*models.Schedule
	for rows.Next() {
		var schedule models.Schedule
		var promptIDsJSON, llmIDsJSON string

		err := rows.Scan(
			&schedule.ID,
			&schedule.BrandID,
			&schedule.Name,
			&promptIDsJSON,
			&llmIDsJSON,
			&schedule.CronExpr,
			&schedule.Temperature,
			&schedule.Enabled,
			&schedule.LastRun,
			&schedule.NextRun,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		schedule.PromptIDs = jsonToSlice(promptIDsJSON)
		schedule.LLMIDs = jsonToSlice(llmIDsJSON)
		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

// UpdateSchedule updates an existing schedule
func (s *SQLite) UpdateSchedule(ctx context.Context, schedule *models.Schedule) error {
	schedule.UpdatedAt = time.Now()

	query := `
		UPDATE schedules 
		SET brand_id = ?, name = ?, prompt_ids = ?, llm_ids = ?, cron_expr = ?, temperature = ?, enabled = ?, last_run = ?, next_run = ?, updated_at = ?
		WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query,
		schedule.BrandID,
		schedule.Name,
		sliceToJSON(schedule.PromptIDs),
		sliceToJSON(schedule.LLMIDs),
		schedule.CronExpr,
		schedule.Temperature,
		schedule.Enabled,
		schedule.LastRun,
		schedule.NextRun,
		schedule.UpdatedAt,
		schedule.ID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found: %s", schedule.ID)
	}

	return nil
}

// DeleteSchedule deletes a schedule
func (s *SQLite) DeleteSchedule(ctx context.Context, id string) error {
	query := "DELETE FROM schedules WHERE id = ?"
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found: %s", id)
	}

	return nil
}

// DeleteAllSchedules deletes all schedules
func (s *SQLite) DeleteAllSchedules(ctx context.Context) (int, error) {
	query := "DELETE FROM schedules"
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// ==================== Brand Competitors Operations ====================
// Note: SQLite only handles SQL operations (LLMs, Schedules).
// NoSQL operations including competitors are handled by PostgreSQL/Hybrid database.

// SaveBrandCompetitors - Not implemented in SQLite (use PostgreSQL/Hybrid)
func (s *SQLite) SaveBrandCompetitors(ctx context.Context, competitors *models.BrandCompetitors) error {
	return fmt.Errorf("brand competitors operations not supported in SQLite-only mode")
}

// GetBrandCompetitors - Not implemented in SQLite (use PostgreSQL/Hybrid)
func (s *SQLite) GetBrandCompetitors(ctx context.Context, brand string) (*models.BrandCompetitors, error) {
	return nil, fmt.Errorf("brand competitors operations not supported in SQLite-only mode")
}

// SaveBrandPrompts - Not implemented in SQLite (use PostgreSQL/Hybrid)
func (s *SQLite) SaveBrandPrompts(ctx context.Context, prompts *models.BrandPrompts) error {
	return fmt.Errorf("brand prompts operations not supported in SQLite-only mode")
}

// GetBrandPrompts - Not implemented in SQLite (use PostgreSQL/Hybrid)
func (s *SQLite) GetBrandPrompts(ctx context.Context, brandID string) (*models.BrandPrompts, error) {
	return nil, fmt.Errorf("brand prompts operations not supported in SQLite-only mode")
}

// DeleteBrandCompetitors - Not implemented in SQLite (use PostgreSQL/Hybrid)
func (s *SQLite) DeleteBrandCompetitors(ctx context.Context, brand string) error {
	return fmt.Errorf("brand competitors operations not supported in SQLite-only mode")
}

// ListBrandCompetitors - Not implemented in SQLite (use PostgreSQL/Hybrid)
func (s *SQLite) ListBrandCompetitors(ctx context.Context) ([]*models.BrandCompetitors, error) {
	return nil, fmt.Errorf("brand competitors operations not supported in SQLite-only mode")
}
