package db

import (
	"context"
	"fmt"
	"time"

	"github.com/fissionx/gego/internal/db/postgresql"
	"github.com/fissionx/gego/internal/db/sqlite"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// HybridDB implements the Database interface using both SQLite and PostgreSQL
type HybridDB struct {
	sqlDB   SQLDatabase   // SQLite for LLMs and Schedules
	nosqlDB NoSQLDatabase // PostgreSQL for Prompts and Responses
}

// New creates a new hybrid database instance
func New(sqlConfig, nosqlConfig *models.Config) (*HybridDB, error) {
	var sqlDB SQLDatabase
	var nosqlDB NoSQLDatabase
	var err error

	switch sqlConfig.Provider {
	case "sqlite":
		sqlDB, err = sqlite.New(sqlConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite database: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported SQL database provider: %s", sqlConfig.Provider)
	}

	// Only PostgreSQL is supported for NoSQL operations
	switch nosqlConfig.Provider {
	case "postgresql", "postgres":
		nosqlDB, err = postgresql.New(nosqlConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create PostgreSQL database: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported NoSQL database provider: %s (only postgresql/postgres is supported)", nosqlConfig.Provider)
	}

	return &HybridDB{
		sqlDB:   sqlDB,
		nosqlDB: nosqlDB,
	}, nil
}

// Connection management
func (h *HybridDB) Connect(ctx context.Context) error {
	if err := h.sqlDB.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to SQL database: %w", err)
	}

	if err := h.nosqlDB.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to NoSQL database: %w", err)
	}

	return nil
}

func (h *HybridDB) Disconnect(ctx context.Context) error {
	var errs []error

	if err := h.sqlDB.Disconnect(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to disconnect from SQL database: %w", err))
	}

	if err := h.nosqlDB.Disconnect(ctx); err != nil {
		errs = append(errs, fmt.Errorf("failed to disconnect from NoSQL database: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("disconnect errors: %v", errs)
	}

	return nil
}

func (h *HybridDB) Ping(ctx context.Context) error {
	if err := h.sqlDB.Ping(ctx); err != nil {
		return fmt.Errorf("SQL database ping failed: %w", err)
	}

	if err := h.nosqlDB.Ping(ctx); err != nil {
		return fmt.Errorf("NoSQL database ping failed: %w", err)
	}

	return nil
}

func (h *HybridDB) CreateLLM(ctx context.Context, llm *models.LLMConfig) error {
	return h.sqlDB.CreateLLM(ctx, llm)
}

func (h *HybridDB) GetLLM(ctx context.Context, id string) (*models.LLMConfig, error) {
	return h.sqlDB.GetLLM(ctx, id)
}

func (h *HybridDB) GetLLMsByIDs(ctx context.Context, ids []string) ([]*models.LLMConfig, error) {
	return h.sqlDB.GetLLMsByIDs(ctx, ids)
}

func (h *HybridDB) ListLLMs(ctx context.Context, enabled *bool) ([]*models.LLMConfig, error) {
	return h.sqlDB.ListLLMs(ctx, enabled)
}

func (h *HybridDB) UpdateLLM(ctx context.Context, llm *models.LLMConfig) error {
	return h.sqlDB.UpdateLLM(ctx, llm)
}

func (h *HybridDB) DeleteLLM(ctx context.Context, id string) error {
	return h.sqlDB.DeleteLLM(ctx, id)
}

func (h *HybridDB) DeleteAllLLMs(ctx context.Context) (int, error) {
	return h.sqlDB.DeleteAllLLMs(ctx)
}

func (h *HybridDB) CreateSchedule(ctx context.Context, schedule *models.Schedule) error {
	return h.sqlDB.CreateSchedule(ctx, schedule)
}

func (h *HybridDB) GetSchedule(ctx context.Context, id string) (*models.Schedule, error) {
	return h.sqlDB.GetSchedule(ctx, id)
}

func (h *HybridDB) ListSchedules(ctx context.Context, brandId string, enabled *bool) ([]*models.Schedule, error) {
	return h.sqlDB.ListSchedules(ctx, brandId, enabled)
}

func (h *HybridDB) UpdateSchedule(ctx context.Context, schedule *models.Schedule) error {
	return h.sqlDB.UpdateSchedule(ctx, schedule)
}

func (h *HybridDB) DeleteSchedule(ctx context.Context, id string) error {
	return h.sqlDB.DeleteSchedule(ctx, id)
}

func (h *HybridDB) DeleteAllSchedules(ctx context.Context) (int, error) {
	return h.sqlDB.DeleteAllSchedules(ctx)
}

// Prompt operations - Use NoSQL
func (h *HybridDB) CreatePrompt(ctx context.Context, prompt *models.Prompt) error {
	return h.nosqlDB.CreatePrompt(ctx, prompt)
}

func (h *HybridDB) GetPrompt(ctx context.Context, id string) (*models.Prompt, error) {
	return h.nosqlDB.GetPrompt(ctx, id)
}

func (h *HybridDB) GetPromptsByIDs(ctx context.Context, ids []string) ([]*models.Prompt, error) {
	return h.nosqlDB.GetPromptsByIDs(ctx, ids)
}

func (h *HybridDB) ListPrompts(ctx context.Context, enabled *bool) ([]*models.Prompt, error) {
	return h.nosqlDB.ListPrompts(ctx, enabled)
}

func (h *HybridDB) ListPromptsByBrand(ctx context.Context, brand string, enabled *bool) ([]*models.Prompt, error) {
	return h.nosqlDB.ListPromptsByBrand(ctx, brand, enabled)
}

func (h *HybridDB) UpdatePrompt(ctx context.Context, prompt *models.Prompt) error {
	return h.nosqlDB.UpdatePrompt(ctx, prompt)
}

func (h *HybridDB) DeletePrompt(ctx context.Context, id string) error {
	return h.nosqlDB.DeletePrompt(ctx, id)
}

func (h *HybridDB) DeleteAllPrompts(ctx context.Context) (int, error) {
	return h.nosqlDB.DeleteAllPrompts(ctx)
}

func (h *HybridDB) DeletePromptsByBrand(ctx context.Context, brand string) (int, error) {
	return h.nosqlDB.DeletePromptsByBrand(ctx, brand)
}

func (h *HybridDB) CreateResponse(ctx context.Context, response *models.Response) error {
	return h.nosqlDB.CreateResponse(ctx, response)
}

func (h *HybridDB) GetResponse(ctx context.Context, id string) (*models.Response, error) {
	return h.nosqlDB.GetResponse(ctx, id)
}

func (h *HybridDB) ListResponses(ctx context.Context, filter shared.ResponseFilter) ([]*models.Response, error) {
	return h.nosqlDB.ListResponses(ctx, filter)
}

func (h *HybridDB) CountResponses(ctx context.Context, filter shared.ResponseFilter) (int64, error) {
	return h.nosqlDB.CountResponses(ctx, filter)
}

func (h *HybridDB) DeleteAllResponses(ctx context.Context) (int, error) {
	return h.nosqlDB.DeleteAllResponses(ctx)
}

func (h *HybridDB) SearchKeyword(ctx context.Context, keyword string, startTime, endTime *time.Time) (*models.KeywordStats, error) {
	return h.nosqlDB.SearchKeyword(ctx, keyword, startTime, endTime)
}

func (h *HybridDB) GetTopKeywords(ctx context.Context, limit int, startTime, endTime *time.Time) ([]models.KeywordCount, error) {
	return h.nosqlDB.GetTopKeywords(ctx, limit, startTime, endTime)
}

func (h *HybridDB) GetPromptStats(ctx context.Context, promptID string) (*models.PromptStats, error) {
	return h.nosqlDB.GetPromptStats(ctx, promptID)
}

func (h *HybridDB) GetLLMStats(ctx context.Context, llmID string) (*models.LLMStats, error) {
	return h.nosqlDB.GetLLMStats(ctx, llmID)
}

func (h *HybridDB) GetPostgreSQLDatabase() *postgresql.PostgreSQL {
	if pgDB, ok := h.nosqlDB.(*postgresql.PostgreSQL); ok {
		return pgDB
	}
	return nil
}

func (h *HybridDB) GetSQLiteDatabase() *sqlite.SQLite {
	if sqliteDB, ok := h.sqlDB.(*sqlite.SQLite); ok {
		return sqliteDB
	}
	return nil
}

// Prompt Library operations - Use NoSQL
func (h *HybridDB) CreatePromptLibrary(ctx context.Context, library *models.PromptLibrary) error {
	return h.nosqlDB.CreatePromptLibrary(ctx, library)
}

func (h *HybridDB) GetPromptLibrary(ctx context.Context, brand, domain, category string) (*models.PromptLibrary, error) {
	return h.nosqlDB.GetPromptLibrary(ctx, brand, domain, category)
}

func (h *HybridDB) UpdatePromptLibrary(ctx context.Context, library *models.PromptLibrary) error {
	return h.nosqlDB.UpdatePromptLibrary(ctx, library)
}

func (h *HybridDB) ListPromptLibraries(ctx context.Context) ([]*models.PromptLibrary, error) {
	return h.nosqlDB.ListPromptLibraries(ctx)
}

// Brand Profile operations - Use NoSQL
func (h *HybridDB) CreateBrandProfile(ctx context.Context, profile *models.BrandProfile) error {
	return h.nosqlDB.CreateBrandProfile(ctx, profile)
}

func (h *HybridDB) GetBrandProfile(ctx context.Context, brandName string) (*models.BrandProfile, error) {
	return h.nosqlDB.GetBrandProfile(ctx, brandName)
}

func (h *HybridDB) UpdateBrandProfile(ctx context.Context, profile *models.BrandProfile) error {
	return h.nosqlDB.UpdateBrandProfile(ctx, profile)
}

func (h *HybridDB) ListBrandProfiles(ctx context.Context) ([]*models.BrandProfile, error) {
	return h.nosqlDB.ListBrandProfiles(ctx)
}

func (h *HybridDB) SaveBrandLogo(ctx context.Context, logo *models.BrandLogoCache) error {
	return h.nosqlDB.SaveBrandLogo(ctx, logo)
}

func (h *HybridDB) GetBrandLogo(ctx context.Context, brandName string) (*models.BrandLogoCache, error) {
	return h.nosqlDB.GetBrandLogo(ctx, brandName)
}

// Cached GEO Insights operations - Use NoSQL
func (h *HybridDB) SaveCachedGEOInsights(ctx context.Context, insights *models.CachedGEOInsights) error {
	return h.nosqlDB.SaveCachedGEOInsights(ctx, insights)
}

func (h *HybridDB) GetCachedGEOInsights(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedGEOInsights, error) {
	return h.nosqlDB.GetCachedGEOInsights(ctx, query)
}

func (h *HybridDB) DeleteCachedGEOInsights(ctx context.Context, id string) error {
	return h.nosqlDB.DeleteCachedGEOInsights(ctx, id)
}

func (h *HybridDB) DeleteCachedGEOInsightsByBrand(ctx context.Context, brand string) error {
	return h.nosqlDB.DeleteCachedGEOInsightsByBrand(ctx, brand)
}

// Cached Source Analytics operations - Use NoSQL
func (h *HybridDB) SaveCachedSourceAnalytics(ctx context.Context, analytics *models.CachedSourceAnalytics) error {
	return h.nosqlDB.SaveCachedSourceAnalytics(ctx, analytics)
}

func (h *HybridDB) GetCachedSourceAnalytics(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedSourceAnalytics, error) {
	return h.nosqlDB.GetCachedSourceAnalytics(ctx, query)
}

func (h *HybridDB) DeleteCachedSourceAnalytics(ctx context.Context, id string) error {
	return h.nosqlDB.DeleteCachedSourceAnalytics(ctx, id)
}

func (h *HybridDB) DeleteCachedSourceAnalyticsByBrand(ctx context.Context, brand string) error {
	return h.nosqlDB.DeleteCachedSourceAnalyticsByBrand(ctx, brand)
}

// Cached Competitive Benchmark operations - Use NoSQL
func (h *HybridDB) SaveCachedCompetitiveBenchmark(ctx context.Context, benchmark *models.CachedCompetitiveBenchmark) error {
	return h.nosqlDB.SaveCachedCompetitiveBenchmark(ctx, benchmark)
}

func (h *HybridDB) GetCachedCompetitiveBenchmark(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedCompetitiveBenchmark, error) {
	return h.nosqlDB.GetCachedCompetitiveBenchmark(ctx, query)
}

func (h *HybridDB) DeleteCachedCompetitiveBenchmark(ctx context.Context, id string) error {
	return h.nosqlDB.DeleteCachedCompetitiveBenchmark(ctx, id)
}

func (h *HybridDB) DeleteCachedCompetitiveBenchmarkByBrand(ctx context.Context, brand string) error {
	return h.nosqlDB.DeleteCachedCompetitiveBenchmarkByBrand(ctx, brand)
}

// Cached Prompt Performance operations - Use NoSQL
func (h *HybridDB) SaveCachedPromptPerformance(ctx context.Context, performance *models.CachedPromptPerformance) error {
	return h.nosqlDB.SaveCachedPromptPerformance(ctx, performance)
}

func (h *HybridDB) GetCachedPromptPerformance(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedPromptPerformance, error) {
	return h.nosqlDB.GetCachedPromptPerformance(ctx, query)
}

func (h *HybridDB) DeleteCachedPromptPerformance(ctx context.Context, id string) error {
	return h.nosqlDB.DeleteCachedPromptPerformance(ctx, id)
}

func (h *HybridDB) DeleteCachedPromptPerformanceByBrand(ctx context.Context, brand string) error {
	return h.nosqlDB.DeleteCachedPromptPerformanceByBrand(ctx, brand)
}

// Scheduled Campaign operations - Use NoSQL
func (h *HybridDB) SaveScheduledCampaign(ctx context.Context, campaign *models.ScheduledCampaign) error {
	return h.nosqlDB.SaveScheduledCampaign(ctx, campaign)
}

func (h *HybridDB) GetScheduledCampaign(ctx context.Context, id string) (*models.ScheduledCampaign, error) {
	return h.nosqlDB.GetScheduledCampaign(ctx, id)
}

func (h *HybridDB) GetScheduledCampaignByBrand(ctx context.Context, brand string) (*models.ScheduledCampaign, error) {
	return h.nosqlDB.GetScheduledCampaignByBrand(ctx, brand)
}

func (h *HybridDB) ListScheduledCampaigns(ctx context.Context, status string) ([]*models.ScheduledCampaign, error) {
	return h.nosqlDB.ListScheduledCampaigns(ctx, status)
}

func (h *HybridDB) UpdateScheduledCampaign(ctx context.Context, campaign *models.ScheduledCampaign) error {
	return h.nosqlDB.UpdateScheduledCampaign(ctx, campaign)
}

func (h *HybridDB) DeleteScheduledCampaign(ctx context.Context, id string) error {
	return h.nosqlDB.DeleteScheduledCampaign(ctx, id)
}

// Cached Prompt Time Series operations - Use NoSQL
func (h *HybridDB) SaveCachedPromptTimeSeries(ctx context.Context, timeSeries *models.CachedPromptTimeSeries) error {
	return h.nosqlDB.SaveCachedPromptTimeSeries(ctx, timeSeries)
}

func (h *HybridDB) GetCachedPromptTimeSeries(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedPromptTimeSeries, error) {
	return h.nosqlDB.GetCachedPromptTimeSeries(ctx, query)
}

func (h *HybridDB) DeleteCachedPromptTimeSeries(ctx context.Context, id string) error {
	return h.nosqlDB.DeleteCachedPromptTimeSeries(ctx, id)
}

func (h *HybridDB) DeleteCachedPromptTimeSeriesByBrand(ctx context.Context, brand string) error {
	return h.nosqlDB.DeleteCachedPromptTimeSeriesByBrand(ctx, brand)
}

// Brand Competitors operations - Use NoSQL
func (h *HybridDB) SaveBrandCompetitors(ctx context.Context, competitors *models.BrandCompetitors) error {
	return h.nosqlDB.SaveBrandCompetitors(ctx, competitors)
}

func (h *HybridDB) GetBrandCompetitors(ctx context.Context, brandID string) (*models.BrandCompetitors, error) {
	return h.nosqlDB.GetBrandCompetitors(ctx, brandID)
}

func (h *HybridDB) SaveBrandPrompts(ctx context.Context, prompts *models.BrandPrompts) error {
	return h.nosqlDB.SaveBrandPrompts(ctx, prompts)
}

func (h *HybridDB) GetBrandPrompts(ctx context.Context, brandID string) (*models.BrandPrompts, error) {
	return h.nosqlDB.GetBrandPrompts(ctx, brandID)
}

func (h *HybridDB) DeleteBrandCompetitors(ctx context.Context, brandID string) error {
	return h.nosqlDB.DeleteBrandCompetitors(ctx, brandID)
}

func (h *HybridDB) ListBrandCompetitors(ctx context.Context) ([]*models.BrandCompetitors, error) {
	return h.nosqlDB.ListBrandCompetitors(ctx)
}

// GEO Campaign operations - Use NoSQL
func (h *HybridDB) SaveGEOCampaign(ctx context.Context, campaign *models.GEOCampaign) error {
	return h.nosqlDB.SaveGEOCampaign(ctx, campaign)
}

func (h *HybridDB) GetGEOCampaign(ctx context.Context, id string) (*models.GEOCampaign, error) {
	return h.nosqlDB.GetGEOCampaign(ctx, id)
}

func (h *HybridDB) GetGEOCampaignsByBrandID(ctx context.Context, brandID string) ([]*models.GEOCampaign, error) {
	return h.nosqlDB.GetGEOCampaignsByBrandID(ctx, brandID)
}

func (h *HybridDB) GetRunningGEOCampaignByBrand(ctx context.Context, brand string) (*models.GEOCampaign, error) {
	return h.nosqlDB.GetRunningGEOCampaignByBrand(ctx, brand)
}

func (h *HybridDB) UpdateGEOCampaign(ctx context.Context, campaign *models.GEOCampaign) error {
	return h.nosqlDB.UpdateGEOCampaign(ctx, campaign)
}
