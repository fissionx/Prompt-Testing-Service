package openai

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"

	"github.com/fissionx/gego/internal/llm"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/utils"
)

// Provider implements the LLM Provider interface for OpenAI
type Provider struct {
	apiKey  string
	baseURL string
	client  openai.Client
}

// New creates a new OpenAI provider
func New(apiKey, baseURL string) *Provider {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
	)

	if baseURL != "" && baseURL != "https://api.openai.com/v1" {
		client = openai.NewClient(
			option.WithAPIKey(apiKey),
			option.WithBaseURL(baseURL),
		)
	}

	return &Provider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  client,
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "openai"
}

// Validate validates the provider configuration
func (p *Provider) Validate(config map[string]string) error {
	if config["api_key"] == "" {
		return fmt.Errorf("api_key is required")
	}
	return nil
}

// Generate sends a prompt to OpenAI and returns the response
// If Brand is provided in config, it performs web search + GEO analysis (matching Google's behavior)
func (p *Provider) Generate(ctx context.Context, prompt string, config llm.Config) (*llm.Response, error) {
	startTime := time.Now()

	// If brand is provided, use web search + GEO analysis workflow (matching Google's behavior)
	if config.Brand != "" {
		return p.generateWithGEOAnalysis(ctx, prompt, config)
	}

	// Standard generation without web search
	model := shared.ChatModelGPT3_5Turbo
	if config.Model != "" {
		model = shared.ChatModel(config.Model)
	}

	temperature := config.Temperature
	maxTokens := config.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1000
	}

	chatCompletion, err := p.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model: model,
			Messages: []openai.ChatCompletionMessageParamUnion{
				{
					OfUser: &openai.ChatCompletionUserMessageParam{
						Content: openai.ChatCompletionUserMessageParamContentUnion{
							OfString: openai.String(prompt),
						},
					},
				},
			},
			Temperature: openai.Float(temperature),
			MaxTokens:   openai.Int(int64(maxTokens)),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API error: %w", err)
	}

	var generatedText string
	if len(chatCompletion.Choices) > 0 && chatCompletion.Choices[0].Message.Content != "" {
		generatedText = chatCompletion.Choices[0].Message.Content
	}

	tokensUsed := 0
	if chatCompletion.Usage.TotalTokens != 0 {
		tokensUsed = int(chatCompletion.Usage.TotalTokens)
	}

	return &llm.Response{
		Text:       generatedText,
		TokensUsed: tokensUsed,
		LatencyMs:  time.Since(startTime).Milliseconds(),
		Model:      string(model),
		Provider:   "openai",
	}, nil
}

// generateWithGEOAnalysis performs web search + GEO analysis (matching Google's workflow)
func (p *Provider) generateWithGEOAnalysis(ctx context.Context, prompt string, config llm.Config) (*llm.Response, error) {
	startTime := time.Now()

	// Step 1: Perform web search to get search answer and grounding sources
	webSearchResponse, err := p.GenerateWithWebSearchSimple(ctx, prompt, llm.Config{
		Model:       config.Model,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
		// Don't pass Brand here - we'll do GEO analysis separately
	})
	if err != nil {
		log.Printf("Web search failed: %v, falling back to regular generation", err)
		// Fallback to regular generation if web search fails
		return p.Generate(ctx, prompt, llm.Config{
			Model:       config.Model,
			Temperature: config.Temperature,
			MaxTokens:   config.MaxTokens,
		})
	}

	searchAnswer := webSearchResponse.Text
	groundingSources := webSearchResponse.GroundingSources
	webSearchQueries := webSearchResponse.WebSearchQueries
	totalTokens := webSearchResponse.TokensUsed

	// If no brand specified, return just the search answer with all metadata
	if config.Brand == "" {
		return &llm.Response{
			Text:             searchAnswer,
			TokensUsed:       totalTokens,
			LatencyMs:        time.Since(startTime).Milliseconds(),
			Model:            webSearchResponse.Model,
			Provider:         "openai",
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

	// Use Chat Completions API for GEO analysis (JSON mode)
	model := shared.ChatModelGPT4o
	if config.Model != "" {
		// Try to use the same model, but fallback to gpt-4o if it doesn't support JSON mode
		model = shared.ChatModel(config.Model)
	}

	geoPrompt := utils.GEOAnalysisPrompt(config.Brand, prompt, searchAnswer, sourcesInfo)

	geoCompletion, err := p.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Model: model,
			Messages: []openai.ChatCompletionMessageParamUnion{
				{
					OfUser: &openai.ChatCompletionUserMessageParam{
						Content: openai.ChatCompletionUserMessageParamContentUnion{
							OfString: openai.String(geoPrompt),
						},
					},
				},
			},
			Temperature: openai.Float(0.1), // Low temperature for consistent JSON
			MaxTokens:   openai.Int(2000),
			ResponseFormat: func() openai.ChatCompletionNewParamsResponseFormatUnion {
				jsonObj := shared.NewResponseFormatJSONObjectParam()
				return openai.ChatCompletionNewParamsResponseFormatUnion{OfJSONObject: &jsonObj}
			}(),
		},
	)
	if err != nil {
		log.Printf("GEO analysis failed: %v, returning search answer only", err)
		return &llm.Response{
			Text:             searchAnswer,
			TokensUsed:       totalTokens,
			LatencyMs:        time.Since(startTime).Milliseconds(),
			Model:            webSearchResponse.Model,
			Provider:         "openai",
			GroundingSources: groundingSources,
		}, nil
	}

	var geoText string
	if len(geoCompletion.Choices) > 0 && geoCompletion.Choices[0].Message.Content != "" {
		geoText = geoCompletion.Choices[0].Message.Content
	}

	if geoCompletion.Usage.TotalTokens != 0 {
		totalTokens += int(geoCompletion.Usage.TotalTokens)
	}

	// Return the GEO JSON response (matching Google's format) with all metadata
	return &llm.Response{
		Text:             geoText,
		TokensUsed:       totalTokens,
		LatencyMs:        time.Since(startTime).Milliseconds(),
		Model:            string(model),
		Provider:         "openai",
		GroundingSources: groundingSources,
		WebSearchQueries: webSearchQueries,
		SearchAnswer:     searchAnswer, // Store original search answer before GEO analysis
	}, nil
}

// ListModels lists available text-to-text models from OpenAI
func (p *Provider) ListModels(ctx context.Context, apiKey, baseURL string) ([]models.ModelInfo, error) {
	client := p.client
	if apiKey != "" && apiKey != p.apiKey {
		client = openai.NewClient(
			option.WithAPIKey(apiKey),
		)
		if baseURL != "" && baseURL != "https://api.openai.com/v1" {
			client = openai.NewClient(
				option.WithAPIKey(apiKey),
				option.WithBaseURL(baseURL),
			)
		}
	}

	modelList, err := client.Models.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	var textModels []models.ModelInfo
	for _, model := range modelList.Data {
		modelID := string(model.ID)

		if strings.HasPrefix(strings.ToLower(modelID), "gpt") {
			if strings.Contains(modelID, ":") {
				continue
			}

			if strings.Contains(strings.ToLower(modelID), "embed") || strings.Contains(strings.ToLower(modelID), "embedding") {
				continue
			}

			if strings.Contains(strings.ToLower(modelID), "vision") || strings.Contains(strings.ToLower(modelID), "image") {
				continue
			}

			if strings.Contains(strings.ToLower(modelID), "whisper") || strings.Contains(strings.ToLower(modelID), "audio") {
				continue
			}

			textModels = append(textModels, models.ModelInfo{
				ID:          modelID,
				Name:        modelID,
				Description: fmt.Sprintf("OpenAI %s", modelID),
			})
		}
	}

	return textModels, nil
}

// WebSearchConfig represents configuration for web search operations
type WebSearchConfig struct {
	// AllowedDomains restricts search to specific domains (optional)
	AllowedDomains []string
	// UserLocation provides geographic context for search results (optional)
	UserLocation *UserLocation
	// MaxResults limits the number of search results to use
	MaxResults int
}

// UserLocation represents geographic context for web search
type UserLocation struct {
	Country string
	City    string
	Region  string
}

// GenerateWithWebSearch performs a web search using OpenAI's Responses API with web_search tool
// This method uses the Responses API which properly supports web search functionality
func (p *Provider) GenerateWithWebSearch(ctx context.Context, prompt string, config llm.Config) (*llm.Response, error) {
	startTime := time.Now()

	// Use a model that supports web search (gpt-4.1, gpt-5, o1, etc.)
	modelName := config.Model
	if modelName == "" {
		modelName = "gpt-4.1" // Default model for Responses API
	}

	temperature := config.Temperature
	maxOutputTokens := config.MaxTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = 2000 // Higher default for web search responses
	}

	// Build tools array with web_search tool
	tools := []responses.ToolUnionParam{
		responses.ToolParamOfWebSearch(responses.WebSearchToolTypeWebSearch),
	}

	// Create response request with web search tool
	// Note: Some models (like o1, gpt-4.1) don't support temperature parameter
	params := responses.ResponseNewParams{
		Model: shared.ResponsesModel(modelName),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(prompt),
		},
		Tools:           tools,
		MaxOutputTokens: openai.Int(int64(maxOutputTokens)),
		Include:         []responses.ResponseIncludable{responses.ResponseIncludableWebSearchCallActionSources}, // Include sources
	}

	// Only set temperature if the model supports it (skip for o1, gpt-4.1, gpt-5, etc.)
	// These models don't support temperature parameter
	modelLower := strings.ToLower(modelName)
	if !strings.Contains(modelLower, "o1") &&
		!strings.Contains(modelLower, "gpt-4.1") &&
		!strings.Contains(modelLower, "gpt-5") &&
		!strings.Contains(modelLower, "gpt-4o") {
		params.Temperature = openai.Float(temperature)
	} else {
		log.Printf("Skipping temperature parameter for model %s (not supported)", modelName)
	}

	responseAPI, err := p.client.Responses.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI Responses API error: %w", err)
	}

	// Extract response text and grounding sources
	var generatedText string
	var groundingSources []string

	// Get output text from the response (OutputText is a method that extracts text from output items)
	generatedText = responseAPI.OutputText()

	// If OutputText() is empty, try to extract manually from output items
	if generatedText == "" {
		for _, item := range responseAPI.Output {
			if item.Type == "message" {
				for _, content := range item.Content {
					if content.Type == "output_text" {
						textContent := content.AsOutputText()
						generatedText += textContent.Text
					}
				}
			}
		}
	}

	// Extract grounding sources and search queries from web search calls and annotations
	groundingSources, searchQueries := p.extractGroundingSourcesAndQueriesFromResponse(responseAPI, prompt)

	tokensUsed := 0
	if responseAPI.Usage.TotalTokens != 0 {
		tokensUsed = int(responseAPI.Usage.TotalTokens)
	}

	response := &llm.Response{
		Text:             generatedText,
		TokensUsed:       tokensUsed,
		LatencyMs:        time.Since(startTime).Milliseconds(),
		Model:            modelName,
		Provider:         "openai",
		GroundingSources: groundingSources,
		WebSearchQueries: searchQueries,
		SearchAnswer:     generatedText, // Store the original search answer
	}

	log.Printf("Web search completed: %d tokens, %d sources, %dms latency",
		tokensUsed, len(groundingSources), response.LatencyMs)

	return response, nil
}

// extractGroundingSourcesAndQueriesFromResponse extracts source URLs and search queries from Responses API
func (p *Provider) extractGroundingSourcesAndQueriesFromResponse(response *responses.Response, originalPrompt string) ([]string, []string) {
	var sources []string
	var searchQueries []string
	sourceMap := make(map[string]bool) // Deduplicate sources

	// Extract from web_search_call items in the output
	for _, item := range response.Output {
		if item.Type == "web_search_call" {
			// Extract search query and status from web search call
			if item.Status != "" {
				log.Printf("Web search call status: %s", item.Status)
			}
			// Note: The actual search queries used by the model are internal to the API
			// and may not be directly accessible from the response structure.
			// We'll use the original prompt as a reference for what was searched.
			// The model generates queries based on the input prompt.
			if originalPrompt != "" {
				searchQueries = append(searchQueries, originalPrompt)
			}
		}

		// Check if this is a message output item
		if item.Type == "message" {
			for _, content := range item.Content {
				// Check if this content has URL citations
				if content.Type == "output_text" {
					// Get annotations which contain URL citations
					textContent := content.AsOutputText()
					for _, annotation := range textContent.Annotations {
						// Check if this is a URL citation (check type first)
						if annotation.Type == "url_citation" {
							urlCitation := annotation.AsURLCitation()
							if urlCitation.URL != "" && !sourceMap[urlCitation.URL] {
								sources = append(sources, urlCitation.URL)
								sourceMap[urlCitation.URL] = true
								log.Printf("Found source from URL citation: %s (Title: %s)", urlCitation.URL, urlCitation.Title)
							}
						}
					}
				}
			}
		}
	}

	// Fallback: Extract URLs from output text if no citations found in annotations
	if len(sources) == 0 {
		outputText := response.OutputText()
		if outputText != "" {
			urls := p.extractURLsFromText(outputText)
			for _, url := range urls {
				if !sourceMap[url] {
					sources = append(sources, url)
					sourceMap[url] = true
					log.Printf("Found source from output text: %s", url)
				}
			}
		}
	}

	// Use the original prompt as a search query reference
	// The model generates internal search queries based on the prompt, but we store
	// the original prompt as a reference for what was searched
	if originalPrompt != "" && len(searchQueries) == 0 {
		// Store the prompt as the search query reference
		searchQueries = append(searchQueries, originalPrompt)
		log.Printf("Using original prompt as search query reference: %s", truncateString(originalPrompt, 100))
	}

	return sources, searchQueries
}

// truncateString truncates a string for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractGroundingSources extracts source URLs from annotations in the response
func (p *Provider) extractGroundingSources(message openai.ChatCompletionMessage) []string {
	var sources []string
	sourceMap := make(map[string]bool) // Deduplicate sources

	// Extract sources from annotations (URL citations from web search)
	if len(message.Annotations) > 0 {
		for _, annotation := range message.Annotations {
			// Check if this is a URL citation annotation
			if annotation.URLCitation.URL != "" {
				url := annotation.URLCitation.URL
				title := annotation.URLCitation.Title

				// Use URL as the primary source identifier
				if !sourceMap[url] {
					sources = append(sources, url)
					sourceMap[url] = true
					log.Printf("Found source: %s (Title: %s)", url, title)
				}
			}
		}
	}

	// Also check for URLs in the response text as a fallback
	if message.Content != "" {
		urls := p.extractURLsFromText(message.Content)
		for _, url := range urls {
			if !sourceMap[url] {
				sources = append(sources, url)
				sourceMap[url] = true
			}
		}
	}

	return sources
}

// extractURLsFromText extracts URLs from text using pattern matching
// Handles both direct URLs and markdown format [text](url)
func (p *Provider) extractURLsFromText(text string) []string {
	var urls []string
	sourceMap := make(map[string]bool) // Deduplicate

	// Pattern 2: Direct URLs (http:// or https://)
	words := strings.Fields(text)
	for _, word := range words {
		// Check for direct URLs
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			// Clean up URL (remove trailing punctuation)
			url := strings.TrimRight(word, ".,;:!?)")
			if !sourceMap[url] {
				urls = append(urls, url)
				sourceMap[url] = true
			}
		}
	}

	// Pattern 3: Extract URLs from markdown links using simple string parsing
	// Look for patterns like [text](url) or (url)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		// Find URLs in parentheses (markdown links)
		start := 0
		for {
			openParen := strings.Index(line[start:], "(")
			if openParen == -1 {
				break
			}
			openParen += start
			closeParen := strings.Index(line[openParen:], ")")
			if closeParen == -1 {
				break
			}
			closeParen += openParen

			potentialURL := line[openParen+1 : closeParen]
			if strings.HasPrefix(potentialURL, "http://") || strings.HasPrefix(potentialURL, "https://") {
				// Clean up URL
				url := strings.TrimSpace(potentialURL)
				url = strings.TrimRight(url, ".,;:!?)")
				if !sourceMap[url] {
					urls = append(urls, url)
					sourceMap[url] = true
				}
			}
			start = closeParen + 1
		}
	}

	return urls
}

// supportsWebSearch checks if a model supports web search
func (p *Provider) supportsWebSearch(model string) bool {
	// Models that support web search
	supportedModels := []string{
		"gpt-4.1",
		"gpt-4-turbo",
		"gpt-4o-2024-08-06",
		"gpt-4-turbo-2024-04-09",
		"o1",
		"o1-preview",
		"o1-mini",
	}

	modelLower := strings.ToLower(model)
	for _, supported := range supportedModels {
		if strings.Contains(modelLower, strings.ToLower(supported)) {
			return true
		}
	}

	return false
}

// GenerateWithWebSearchSimple is a convenience method that performs web search with default settings
func (p *Provider) GenerateWithWebSearchSimple(ctx context.Context, prompt string, config llm.Config) (*llm.Response, error) {
	return p.GenerateWithWebSearch(ctx, prompt, config)
}
