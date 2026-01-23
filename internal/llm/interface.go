package llm

import (
	"context"

	"time"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/utils"
)

// Config represents the common configuration for all LLM providers
type Config struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
	TopP        float64 `json:"top_p"`
	TopK        int     `json:"top_k"`
	Stream      bool    `json:"stream"`
	Brand       string  `json:"brand"` // Brand/company name for GEO analysis
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() Config {
	return Config{
		Model:       "",
		Temperature: 0.7,
		MaxTokens:   1000,
		TopP:        1.0,
		TopK:        0,
		Stream:      false,
	}
}

// Provider defines the interface for LLM providers
type Provider interface {
	// Name returns the provider name (e.g., "openai", "anthropic")
	Name() string

	// Generate sends a prompt to the LLM and returns the response
	Generate(ctx context.Context, prompt string, config Config) (*Response, error)

	// Validate validates the provider configuration
	Validate(config map[string]string) error

	// ListModels lists available text-to-text models from this provider
	// Returns models that can be used for text generation
	ListModels(ctx context.Context, apiKey, baseURL string) ([]models.ModelInfo, error)

	GenerateWithWebSearch(ctx context.Context, prompt string, config Config) (*Response, error)
}

// Response represents an LLM response
type Response struct {
	Text             string
	TokensUsed       int
	LatencyMs        int64
	Model            string
	Provider         string
	Error            string
	GroundingSources []string // Citation sources (URLs) from models that support grounding
	WebSearchQueries []string // Search queries used by the model (for ChatGPT/Gemini-like experience)
	SearchAnswer     string   // Original search answer (before GEO analysis, if applicable)
}

// Registry manages LLM providers
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register registers a provider
func (r *Registry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (Provider, bool) {
	provider, ok := r.providers[name]
	return provider, ok
}

// List returns all registered provider names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// GenerateRequest encapsulates a generation request
type GenerateRequest struct {
	Provider string
	Model    string
	Prompt   string
	Config   Config
	Timeout  time.Duration
}

// GenerateGEOPromptTemplate creates a pre-prompt for generating GEO statistics prompts
func GenerateGEOPromptTemplate(userInput string, existingPrompts []string, language string, count int) string {
	return utils.GEOPromptTemplate(userInput, existingPrompts, language, count)
}
