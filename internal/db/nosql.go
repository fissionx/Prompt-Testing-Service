package db

import (
	"context"
	"time"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// NoSQLDatabase defines the interface for NoSQL database operations (Prompts and Responses)
type NoSQLDatabase interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Ping(ctx context.Context) error

	// Prompt operations
	CreatePrompt(ctx context.Context, prompt *models.Prompt) error
	GetPrompt(ctx context.Context, id string) (*models.Prompt, error)
	ListPrompts(ctx context.Context, enabled *bool) ([]*models.Prompt, error)
	UpdatePrompt(ctx context.Context, prompt *models.Prompt) error
	DeletePrompt(ctx context.Context, id string) error
	DeleteAllPrompts(ctx context.Context) (int, error)
	DeletePromptsByBrand(ctx context.Context, brand string) (int, error)

	// Response operations
	CreateResponse(ctx context.Context, response *models.Response) error
	GetResponse(ctx context.Context, id string) (*models.Response, error)
	ListResponses(ctx context.Context, filter shared.ResponseFilter) ([]*models.Response, error)
	CountResponses(ctx context.Context, filter shared.ResponseFilter) (int64, error)
	DeleteAllResponses(ctx context.Context) (int, error)

	// Keyword search (on-demand, searches through response_text)
	SearchKeyword(ctx context.Context, keyword string, startTime, endTime *time.Time) (*models.KeywordStats, error)
	GetTopKeywords(ctx context.Context, limit int, startTime, endTime *time.Time) ([]models.KeywordCount, error)

	// Statistics operations
	GetPromptStats(ctx context.Context, promptID string) (*models.PromptStats, error)
	GetLLMStats(ctx context.Context, llmID string) (*models.LLMStats, error)

	// Prompt Library operations (for organized prompt reuse)
	CreatePromptLibrary(ctx context.Context, library *models.PromptLibrary) error
	GetPromptLibrary(ctx context.Context, brand, domain, category string) (*models.PromptLibrary, error)
	UpdatePromptLibrary(ctx context.Context, library *models.PromptLibrary) error
	ListPromptLibraries(ctx context.Context) ([]*models.PromptLibrary, error)

	// Brand Profile operations (for metadata and categorization)
	CreateBrandProfile(ctx context.Context, profile *models.BrandProfile) error
	GetBrandProfile(ctx context.Context, brandName string) (*models.BrandProfile, error)
	UpdateBrandProfile(ctx context.Context, profile *models.BrandProfile) error
	ListBrandProfiles(ctx context.Context) ([]*models.BrandProfile, error)

	// Brand Logo cache operations
	SaveBrandLogo(ctx context.Context, logo *models.BrandLogoCache) error
	GetBrandLogo(ctx context.Context, brandName string) (*models.BrandLogoCache, error)

	// Cached GEO Insights operations
	SaveCachedGEOInsights(ctx context.Context, insights *models.CachedGEOInsights) error
	GetCachedGEOInsights(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedGEOInsights, error)
	DeleteCachedGEOInsights(ctx context.Context, id string) error
	DeleteCachedGEOInsightsByBrand(ctx context.Context, brand string) error

	// Cached Source Analytics operations
	SaveCachedSourceAnalytics(ctx context.Context, analytics *models.CachedSourceAnalytics) error
	GetCachedSourceAnalytics(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedSourceAnalytics, error)
	DeleteCachedSourceAnalytics(ctx context.Context, id string) error
	DeleteCachedSourceAnalyticsByBrand(ctx context.Context, brand string) error

	// Cached Competitive Benchmark operations
	SaveCachedCompetitiveBenchmark(ctx context.Context, benchmark *models.CachedCompetitiveBenchmark) error
	GetCachedCompetitiveBenchmark(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedCompetitiveBenchmark, error)
	DeleteCachedCompetitiveBenchmark(ctx context.Context, id string) error
	DeleteCachedCompetitiveBenchmarkByBrand(ctx context.Context, brand string) error

	// Cached Prompt Performance operations
	SaveCachedPromptPerformance(ctx context.Context, performance *models.CachedPromptPerformance) error
	GetCachedPromptPerformance(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedPromptPerformance, error)
	DeleteCachedPromptPerformance(ctx context.Context, id string) error
	DeleteCachedPromptPerformanceByBrand(ctx context.Context, brand string) error

	// Scheduled Campaign operations
	SaveScheduledCampaign(ctx context.Context, campaign *models.ScheduledCampaign) error
	GetScheduledCampaign(ctx context.Context, id string) (*models.ScheduledCampaign, error)
	GetScheduledCampaignByBrand(ctx context.Context, brand string) (*models.ScheduledCampaign, error)
	ListScheduledCampaigns(ctx context.Context, status string) ([]*models.ScheduledCampaign, error)
	UpdateScheduledCampaign(ctx context.Context, campaign *models.ScheduledCampaign) error
	DeleteScheduledCampaign(ctx context.Context, id string) error

	// Cached Prompt Time Series operations
	SaveCachedPromptTimeSeries(ctx context.Context, timeSeries *models.CachedPromptTimeSeries) error
	GetCachedPromptTimeSeries(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedPromptTimeSeries, error)
	DeleteCachedPromptTimeSeries(ctx context.Context, id string) error
	DeleteCachedPromptTimeSeriesByBrand(ctx context.Context, brand string) error

	// Brand Competitors operations
	SaveBrandCompetitors(ctx context.Context, competitors *models.BrandCompetitors) error
	GetBrandCompetitors(ctx context.Context, brand string) (*models.BrandCompetitors, error)
	SaveBrandPrompts(ctx context.Context, prompts *models.BrandPrompts) error
	GetBrandPrompts(ctx context.Context, brand string) (*models.BrandPrompts, error)
	DeleteBrandCompetitors(ctx context.Context, brand string) error
	ListBrandCompetitors(ctx context.Context) ([]*models.BrandCompetitors, error)

	// GEO Campaign operations
	SaveGEOCampaign(ctx context.Context, campaign *models.GEOCampaign) error
	GetGEOCampaign(ctx context.Context, id string) (*models.GEOCampaign, error)
	GetRunningGEOCampaignByBrand(ctx context.Context, brand string) (*models.GEOCampaign, error)
	UpdateGEOCampaign(ctx context.Context, campaign *models.GEOCampaign) error
}
