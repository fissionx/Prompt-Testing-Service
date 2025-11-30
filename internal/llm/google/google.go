package google

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/genai"

	"github.com/AI2HU/gego/internal/llm"
	"github.com/AI2HU/gego/internal/models"
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

	// Extract grounding metadata (sources/URLs)
	var groundingSources []string
	if len(result.Candidates) > 0 && result.Candidates[0].GroundingMetadata != nil {
		metadata := result.Candidates[0].GroundingMetadata

		if len(metadata.WebSearchQueries) > 0 {
			log.Printf("Web Search Queries: %v", metadata.WebSearchQueries)
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

	// If no brand specified, return just the search answer
	if config.Brand == "" {
		return &llm.Response{
			Text:       searchAnswer,
			TokensUsed: totalTokens,
			LatencyMs:  time.Since(startTime).Milliseconds(),
			Model:      model,
			Provider:   "google",
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

	geoPrompt := fmt.Sprintf(`Analyze the following search response for brand visibility, sentiment, and competitors.

BRAND TO ANALYZE: %s

SEARCH QUERY: %s

SEARCH RESPONSE:
%s%s

---

CRITICAL ANALYSIS INSTRUCTIONS:
1. Check if "%s" is mentioned in the search response text
2. Check if the brand's domain appears in the GROUNDING SOURCES (cited URLs)  
3. Identify ALL competitor brands/products mentioned in the response
4. If brand is mentioned, analyze the sentiment (positive/neutral/negative)
5. Scoring: 
   - Score 0: Not in text, not in sources
   - Score 1-3: In sources but not in text (low visibility)
   - Score 4-6: Mentioned in text with context
   - Score 7-10: Prominently featured in text AND sources

You MUST respond with ONLY a valid JSON object (no markdown, no code blocks):

{"search_answer":"%s","geo_analysis":{"visibility_score":0,"brand_mentioned":false,"in_grounding_sources":false,"mention_status":"Where/how brand appeared or why absent","reason":"Why brand is/isn't cited, considering text and sources","sentiment":"positive|neutral|negative (only if brand mentioned)","competitors":["Competitor1","Competitor2"],"insights":["Insight 1","Insight 2","Insight 3"],"actions":["Action 1: Blog topic","Action 2: LinkedIn content","Action 3: Get featured on X","Action 4: Target keyword Y","Action 5: Technical SEO tip"],"competitor_info":"What competitors are doing to get cited"}}

Rules:
- visibility_score: integer 0-10
- brand_mentioned: true if in text OR sources
- in_grounding_sources: true if brand domain in cited URLs
- sentiment: "positive" (recommended/praised), "neutral" (just mentioned), "negative" (criticized), or empty string if not mentioned
- competitors: array of competitor names mentioned (empty array if none)
- insights: 3-5 insights about visibility
- actions: 5 specific actionable recommendations
- competitor_info: what competitors do well

RESPOND WITH ONLY THE JSON OBJECT, NO OTHER TEXT.`, config.Brand, prompt, searchAnswer, sourcesInfo, config.Brand, escapeJSONString(searchAnswer))

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

	// Return the GEO JSON response
	return &llm.Response{
		Text:             geoText,
		TokensUsed:       totalTokens,
		LatencyMs:        time.Since(startTime).Milliseconds(),
		Model:            model,
		Provider:         "google",
		GroundingSources: groundingSources,
	}, nil
}

// escapeJSONString escapes special characters for JSON string embedding
func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
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
