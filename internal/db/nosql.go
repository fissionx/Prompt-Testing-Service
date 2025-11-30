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
}
