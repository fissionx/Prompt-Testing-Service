package google

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/genai"

	"github.com/fissionx/gego/internal/llm"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/utils"
)

// Provider implements the LLM Provider interface for Google AI
type Provider struct {
	apiKey  string
	baseURL string
	client  *genai.Client
}

// New creates a new Google provider
func New(apiKey, baseURL string) *Provider {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		client = nil
	}

	return &Provider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  client,
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "google"
}

// Validate validates the provider configuration
func (p *Provider) Validate(config map[string]string) error {
	if config["api_key"] == "" {
		return fmt.Errorf("api_key is required")
	}
	return nil
}

// Generate sends a prompt to Google AI and returns the response
func (p *Provider) Generate(ctx context.Context, prompt string, config llm.Config) (*llm.Response, error) {
	startTime := time.Now()

	model := "gemini-2.5-flash"
	if config.Model != "" {
		model = config.Model
	}

	client := p.client
	if client == nil {
		var err error
		client, err = genai.NewClient(ctx, &genai.ClientConfig{
			APIKey:  p.apiKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create Google client: %w", err)
		}
	}

	// Step 1: Get search results with Google Search tool
	content := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: prompt},
			},
		},
	}

	searchConfig := &genai.GenerateContentConfig{
		Temperature: float32Ptr(float32(config.Temperature)),
		TopP:        float32Ptr(float32(config.TopP)),
		TopK:        float32Ptr(float32(config.TopK)),
		// Enable Google Search tool for web search results
		Tools: []*genai.Tool{
			{
				GoogleSearch: &genai.GoogleSearch{},
			},
		},
	}

	result, err := client.Models.GenerateContent(ctx, model, content, searchConfig)
	if err != nil {
		return nil, fmt.Errorf("Google AI API error: %v", err)
	}

	// Print complete response from Google API
	// if resultJSON, jsonErr := json.MarshalIndent(result, "", "  "); jsonErr == nil {
	// 	log.Printf("\n========== GOOGLE LLM SEARCH RESPONSE ==========\n%s\n==================================================\n", string(resultJSON))
	// }

	var searchAnswer string
	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		if text := result.Candidates[0].Content.Parts[0].Text; text != "" {
			searchAnswer = text
		}
	}

	// Extract grounding metadata (sources/URLs) and search queries
	var groundingSources []string
	var webSearchQueries []string
	if len(result.Candidates) > 0 && result.Candidates[0].GroundingMetadata != nil {
		metadata := result.Candidates[0].GroundingMetadata

		if len(metadata.WebSearchQueries) > 0 {
			webSearchQueries = metadata.WebSearchQueries
			log.Printf("Web Search Queries: %v", webSearchQueries)
		}

		if len(metadata.GroundingChunks) > 0 {
			log.Printf("Found %d grounding chunks", len(metadata.GroundingChunks))
			for i, chunk := range metadata.GroundingChunks {
				if chunk.Web != nil && chunk.Web.Title != "" {
					// Use Title (actual source domain) - skip if not available
					// Title contains the real source (e.g., "forbes.com", "reddit.com")
					// URI contains redirect URLs (vertexaisearch.cloud.google.com/...)
					source := chunk.Web.Title
					groundingSources = append(groundingSources, source)
					log.Printf("  Chunk %d: %s (Source: %s)", i+1, chunk.Web.URI, source)
				} else if chunk.Web != nil {
					log.Printf("  Chunk %d: %s (Source: SKIPPED - no title)", i+1, chunk.Web.URI)
				}
			}
		}

		if len(groundingSources) > 0 {
			log.Printf("Total unique sources: %d", len(groundingSources))
		}
	} else {
		log.Printf("No grounding metadata found in response")
	}

	totalTokens := 0
	if result.UsageMetadata != nil {
		totalTokens = int(result.UsageMetadata.TotalTokenCount)
	}

	// If no brand specified, return just the search answer with metadata
	if config.Brand == "" {
		return &llm.Response{
			Text:             searchAnswer,
			TokensUsed:       totalTokens,
			LatencyMs:        time.Since(startTime).Milliseconds(),
			Model:            model,
			Provider:         "google",
			GroundingSources: groundingSources,
			WebSearchQueries: webSearchQueries,
			SearchAnswer:     searchAnswer,
		}, nil
	}

	// Step 2: Analyze the search response for GEO metrics (separate call, JSON mode)
	log.Printf("========== STARTING GEO ANALYSIS FOR BRAND: %s ==========", config.Brand)

	// Check if brand domain appears in grounding sources
	brandInSources := false
	var brandSourceURLs []string
	if len(groundingSources) > 0 {
		// Try to extract brand domain (assume brand might have a website like brand.com or brand.ai)
		brandLower := strings.ToLower(config.Brand)
		brandDomain := strings.ReplaceAll(brandLower, " ", "")

		for _, source := range groundingSources {
			sourceLower := strings.ToLower(source)
			// Check if the source URL contains the brand name
			if strings.Contains(sourceLower, brandDomain) ||
				strings.Contains(sourceLower, strings.ReplaceAll(brandLower, ".", "")) {
				brandInSources = true
				brandSourceURLs = append(brandSourceURLs, source)
				log.Printf("✅ BRAND FOUND IN GROUNDING SOURCE: %s", source)
			}
		}
	}

	sourcesInfo := ""
	if len(groundingSources) > 0 {
		sourcesInfo = fmt.Sprintf("\n\nGROUNDING SOURCES (URLs cited by the AI):\n%s", strings.Join(groundingSources, "\n"))
		if brandInSources {
			sourcesInfo += fmt.Sprintf("\n\n⚠️ IMPORTANT: The brand's website WAS FOUND in the grounding sources: %s", strings.Join(brandSourceURLs, ", "))
		} else {
			sourcesInfo += "\n\n⚠️ IMPORTANT: The brand's website was NOT found in any grounding sources."
		}
	}

	geoPrompt := utils.GEOAnalysisPrompt(config.Brand, prompt, searchAnswer, sourcesInfo)

	geoContent := []*genai.Content{
		{
			Parts: []*genai.Part{
				{Text: geoPrompt},
			},
		},
	}

	geoConfig := &genai.GenerateContentConfig{
		Temperature:      float32Ptr(0.1), // Low temperature for consistent JSON
		ResponseMIMEType: "application/json",
	}

	geoResult, err := client.Models.GenerateContent(ctx, model, geoContent, geoConfig)
	if err != nil {
		log.Printf("GEO analysis failed: %v, returning search answer only", err)
		return &llm.Response{
			Text:             searchAnswer,
			TokensUsed:       totalTokens,
			LatencyMs:        time.Since(startTime).Milliseconds(),
			Model:            model,
			Provider:         "google",
			GroundingSources: groundingSources,
			WebSearchQueries: webSearchQueries,
			SearchAnswer:     searchAnswer,
		}, nil
	}

	// Print GEO analysis response
	if resultJSON, jsonErr := json.MarshalIndent(geoResult, "", "  "); jsonErr == nil {
		log.Printf("\n========== GOOGLE LLM GEO ANALYSIS RESPONSE ==========\n%s\n==================================================\n", string(resultJSON))
	}

	var geoText string
	if len(geoResult.Candidates) > 0 && len(geoResult.Candidates[0].Content.Parts) > 0 {
		if text := geoResult.Candidates[0].Content.Parts[0].Text; text != "" {
			geoText = text
		}
	}

	if geoResult.UsageMetadata != nil {
		totalTokens += int(geoResult.UsageMetadata.TotalTokenCount)
	}

	// Return the GEO JSON response with all metadata
	return &llm.Response{
		Text:             geoText,
		TokensUsed:       totalTokens,
		LatencyMs:        time.Since(startTime).Milliseconds(),
		Model:            model,
		Provider:         "google",
		GroundingSources: groundingSources,
		WebSearchQueries: webSearchQueries,
		SearchAnswer:     searchAnswer, // Store original search answer before GEO analysis
	}, nil
}

// ListModels lists available Google AI models
func (p *Provider) ListModels(ctx context.Context, apiKey, baseURL string) ([]models.ModelInfo, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Google client: %w", err)
	}

	modelPage, err := client.Models.List(ctx, &genai.ListModelsConfig{})
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	var modelList []models.ModelInfo
	for _, model := range modelPage.Items {
		modelName := model.Name

		if strings.Contains(strings.ToLower(modelName), "embed") || strings.Contains(strings.ToLower(modelName), "embedding") {
			continue
		}

		if strings.Contains(strings.ToLower(modelName), "vision") || strings.Contains(strings.ToLower(modelName), "image") {
			continue
		}

		if strings.Contains(strings.ToLower(modelName), "gemini") {
			name := modelName
			if len(name) > 7 && name[:7] == "models/" {
				name = name[7:]
			}

			modelList = append(modelList, models.ModelInfo{
				ID:          model.Name,
				Name:        name,
				Description: model.Description,
			})
		}
	}

	return modelList, nil
}

func float32Ptr(f float32) *float32 {
	return &f
}
