package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// SimplifiedBrandPromptResponse represents a simplified prompt response for chatbot use
type SimplifiedBrandPromptResponse struct {
	// Response content in markdown format (search_answer only)
	Response string `json:"response"`

	// Prompt information
	Prompt struct {
		ID         string     `json:"id"`
		Template   string     `json:"template"`
		PromptType string     `json:"promptType,omitempty"`
		Category   string     `json:"category,omitempty"`
		Domain     string     `json:"domain,omitempty"`
		Brand      string     `json:"brand,omitempty"`
		Tags       []string   `json:"tags,omitempty"`
		CreatedAt  time.Time  `json:"createdAt"`
	} `json:"prompt"`

	// LLM information
	LLM struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Provider    string  `json:"provider"`
		Model       string  `json:"model"`
		Temperature float64 `json:"temperature,omitempty"`
	} `json:"llm"`

	// Related information (GEO analysis, sources, etc.)
	Analysis struct {
		VisibilityScore    int      `json:"visibilityScore,omitempty"`
		BrandMentioned     bool     `json:"brandMentioned,omitempty"`
		InGroundingSources bool     `json:"inGroundingSources,omitempty"`
		Sentiment          string   `json:"sentiment,omitempty"`
		BrandPosition      int      `json:"brandPosition,omitempty"`
		TotalBrandsListed  int      `json:"totalBrandsListed,omitempty"`
		CompetitorsMention []string `json:"competitorsMention,omitempty"`
	} `json:"analysis"`

	// Grounding sources
	Sources struct {
		GroundingSources []string `json:"groundingSources,omitempty"`
		GroundingDomains []string `json:"groundingDomains,omitempty"`
		WebSearchQueries []string `json:"webSearchQueries,omitempty"`
	} `json:"sources"`

	// Metadata
	Metadata struct {
		TokensUsed int       `json:"tokensUsed,omitempty"`
		LatencyMs  int64     `json:"latencyMs,omitempty"`
		CreatedAt  time.Time `json:"createdAt"`
	} `json:"metadata"`
}

// getBrandPromptsWithLatestResponses handles GET /api/v1/brand/:brand/prompts
// Returns all prompts associated with a brand and their latest response including grounding sources
func (s *Server) getBrandPromptsWithLatestResponses(c *gin.Context) {
	brand := c.Param("brand")
	if brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand parameter is required")
		return
	}

	ctx := c.Request.Context()

	// Get all responses (we'll filter by brand in memory since ResponseFilter doesn't support brand yet)
	// TODO: Add Brand to ResponseFilter for better performance
	filter := shared.ResponseFilter{
		Limit: 10000, // Get a large number to find all prompts
	}
	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to list responses: "+err.Error())
		return
	}

	// Filter responses by brand (case-insensitive) and group by prompt_id
	brandResponses := make(map[string][]*models.Response) // prompt_id -> responses
	for _, resp := range allResponses {
		// Case-insensitive brand comparison
		if resp.Brand != "" && strings.EqualFold(resp.Brand, brand) {
			if resp.PromptID != "" {
				brandResponses[resp.PromptID] = append(brandResponses[resp.PromptID], resp)
			}
		}
	}

	// For each prompt, get the latest response and build simplified structure
	result := make([]SimplifiedBrandPromptResponse, 0)
	for promptID, responses := range brandResponses {
		// Sort responses by created_at descending to get the latest
		sort.Slice(responses, func(i, j int) bool {
			return responses[i].CreatedAt.After(responses[j].CreatedAt)
		})

		// Get the prompt details
		prompt, err := s.db.GetPrompt(ctx, promptID)
		if err != nil {
			// Skip if prompt not found
			continue
		}

		// Build simplified response
		simplified := SimplifiedBrandPromptResponse{}

		// Populate prompt information
		simplified.Prompt.ID = prompt.ID
		simplified.Prompt.Template = prompt.Template
		simplified.Prompt.PromptType = string(prompt.PromptType)
		simplified.Prompt.Category = prompt.Category
		simplified.Prompt.Domain = prompt.Domain
		simplified.Prompt.Brand = prompt.Brand
		simplified.Prompt.Tags = prompt.Tags
		simplified.Prompt.CreatedAt = prompt.CreatedAt

		// Get the latest response if available
		if len(responses) > 0 {
			latest := responses[0]

			// Extract search_answer from response
			// Priority: SearchAnswer field > extract from ResponseText (if it's JSON) > ResponseText as fallback
			searchAnswer := latest.SearchAnswer
			if searchAnswer == "" {
				// Try to extract search_answer from ResponseText if it's JSON
				searchAnswer = extractSearchAnswerFromJSON(latest.ResponseText)
				if searchAnswer == "" {
					// Fallback to ResponseText if no search_answer found
					searchAnswer = latest.ResponseText
				}
			}

			// Clean up: remove \n characters and normalize whitespace
			searchAnswer = cleanMarkdownResponse(searchAnswer)
			simplified.Response = searchAnswer

			// Populate LLM information
			simplified.LLM.ID = latest.LLMID
			simplified.LLM.Name = latest.LLMName
			simplified.LLM.Provider = latest.LLMProvider
			simplified.LLM.Model = latest.LLMModel
			simplified.LLM.Temperature = latest.Temperature

			// Populate analysis information
			simplified.Analysis.VisibilityScore = latest.VisibilityScore
			simplified.Analysis.BrandMentioned = latest.BrandMentioned
			simplified.Analysis.InGroundingSources = latest.InGroundingSources
			simplified.Analysis.Sentiment = latest.Sentiment
			simplified.Analysis.BrandPosition = latest.BrandPosition
			simplified.Analysis.TotalBrandsListed = latest.TotalBrandsListed
			simplified.Analysis.CompetitorsMention = latest.CompetitorsMention

			// Populate sources
			simplified.Sources.GroundingSources = latest.GroundingSources
			simplified.Sources.GroundingDomains = latest.GroundingDomains
			simplified.Sources.WebSearchQueries = latest.WebSearchQueries

			// Populate metadata
			simplified.Metadata.TokensUsed = latest.TokensUsed
			simplified.Metadata.LatencyMs = latest.LatencyMs
			simplified.Metadata.CreatedAt = latest.CreatedAt
		} else {
			// No response available, set empty response
			simplified.Response = ""
		}

		result = append(result, simplified)
	}

	// Sort by response created_at descending (most recent first)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Metadata.CreatedAt.IsZero() && result[j].Metadata.CreatedAt.IsZero() {
			return result[i].Prompt.CreatedAt.After(result[j].Prompt.CreatedAt)
		}
		if result[i].Metadata.CreatedAt.IsZero() {
			return false
		}
		if result[j].Metadata.CreatedAt.IsZero() {
			return true
		}
		return result[i].Metadata.CreatedAt.After(result[j].Metadata.CreatedAt)
	})

	s.successResponse(c, result)
}

// SimplifiedPromptResponse represents a simplified response structure for chatbot use
type SimplifiedPromptResponse struct {
	// Response content in markdown format
	Response string `json:"response"`

	// Prompt information
	Prompt struct {
		ID         string     `json:"id"`
		Template   string     `json:"template"`
		PromptType string     `json:"promptType,omitempty"`
		Category   string     `json:"category,omitempty"`
		Domain     string     `json:"domain,omitempty"`
		Brand      string     `json:"brand,omitempty"`
		Tags       []string   `json:"tags,omitempty"`
		CreatedAt  time.Time  `json:"createdAt"`
	} `json:"prompt"`

	// LLM information
	LLM struct {
		ID       string  `json:"id"`
		Name     string  `json:"name"`
		Provider string  `json:"provider"`
		Model    string  `json:"model"`
		Temperature float64 `json:"temperature,omitempty"`
	} `json:"llm"`

	// Related information (GEO analysis, sources, etc.)
	Analysis struct {
		VisibilityScore    int      `json:"visibilityScore,omitempty"`
		BrandMentioned     bool     `json:"brandMentioned,omitempty"`
		InGroundingSources bool     `json:"inGroundingSources,omitempty"`
		Sentiment          string   `json:"sentiment,omitempty"`
		BrandPosition      int      `json:"brandPosition,omitempty"`
		TotalBrandsListed  int      `json:"totalBrandsListed,omitempty"`
		CompetitorsMention []string `json:"competitorsMention,omitempty"`
	} `json:"analysis"`

	// Grounding sources
	Sources struct {
		GroundingSources []string `json:"groundingSources,omitempty"`
		GroundingDomains []string `json:"groundingDomains,omitempty"`
		WebSearchQueries []string `json:"webSearchQueries,omitempty"`
	} `json:"sources"`

	// Metadata
	Metadata struct {
		TokensUsed int   `json:"tokensUsed,omitempty"`
		LatencyMs  int64 `json:"latencyMs,omitempty"`
		CreatedAt  time.Time `json:"createdAt"`
	} `json:"metadata"`
}

// getPromptResponses handles GET /api/v1/prompts/:id/responses
// Returns the latest response in a simplified markdown format for chatbot use
func (s *Server) getPromptResponses(c *gin.Context) {
	promptID := c.Param("id")
	if promptID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Prompt ID parameter is required")
		return
	}

	ctx := c.Request.Context()

	// Get the prompt
	prompt, err := s.db.GetPrompt(ctx, promptID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Prompt not found: "+err.Error())
		return
	}

	// Get responses for this prompt
	filter := shared.ResponseFilter{
		PromptID: promptID,
		Limit:    1000, // Get enough to find all responses
	}

	responses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to list responses: "+err.Error())
		return
	}

	// Sort by created_at descending to get the latest
	sort.Slice(responses, func(i, j int) bool {
		return responses[i].CreatedAt.After(responses[j].CreatedAt)
	})

	// If no responses found, return empty response
	if len(responses) == 0 {
		emptyResponse := SimplifiedPromptResponse{
			Response: "",
		}
		emptyResponse.Prompt.ID = prompt.ID
		emptyResponse.Prompt.Template = prompt.Template
		emptyResponse.Prompt.PromptType = string(prompt.PromptType)
		emptyResponse.Prompt.Category = prompt.Category
		emptyResponse.Prompt.Domain = prompt.Domain
		emptyResponse.Prompt.Brand = prompt.Brand
		emptyResponse.Prompt.Tags = prompt.Tags
		emptyResponse.Prompt.CreatedAt = prompt.CreatedAt
		s.successResponse(c, emptyResponse)
		return
	}

	// Get the latest response
	latest := responses[0]

	// Extract search_answer from response
	// Priority: SearchAnswer field > extract from ResponseText (if it's JSON) > ResponseText as fallback
	searchAnswer := latest.SearchAnswer
	if searchAnswer == "" {
		// Try to extract search_answer from ResponseText if it's JSON
		searchAnswer = extractSearchAnswerFromJSON(latest.ResponseText)
		if searchAnswer == "" {
			// Fallback to ResponseText if no search_answer found
			searchAnswer = latest.ResponseText
		}
	}

	// Clean up: remove \n characters and normalize whitespace
	searchAnswer = cleanMarkdownResponse(searchAnswer)

	// Build simplified response
	result := SimplifiedPromptResponse{
		Response: searchAnswer, // Markdown format response (search_answer only)
	}

	// Populate prompt information
	result.Prompt.ID = prompt.ID
	result.Prompt.Template = prompt.Template
	result.Prompt.PromptType = string(prompt.PromptType)
	result.Prompt.Category = prompt.Category
	result.Prompt.Domain = prompt.Domain
	result.Prompt.Brand = prompt.Brand
	result.Prompt.Tags = prompt.Tags
	result.Prompt.CreatedAt = prompt.CreatedAt

	// Populate LLM information
	result.LLM.ID = latest.LLMID
	result.LLM.Name = latest.LLMName
	result.LLM.Provider = latest.LLMProvider
	result.LLM.Model = latest.LLMModel
	result.LLM.Temperature = latest.Temperature

	// Populate analysis information
	result.Analysis.VisibilityScore = latest.VisibilityScore
	result.Analysis.BrandMentioned = latest.BrandMentioned
	result.Analysis.InGroundingSources = latest.InGroundingSources
	result.Analysis.Sentiment = latest.Sentiment
	result.Analysis.BrandPosition = latest.BrandPosition
	result.Analysis.TotalBrandsListed = latest.TotalBrandsListed
	result.Analysis.CompetitorsMention = latest.CompetitorsMention

	// Populate sources
	result.Sources.GroundingSources = latest.GroundingSources
	result.Sources.GroundingDomains = latest.GroundingDomains
	result.Sources.WebSearchQueries = latest.WebSearchQueries

	// Populate metadata
	result.Metadata.TokensUsed = latest.TokensUsed
	result.Metadata.LatencyMs = latest.LatencyMs
	result.Metadata.CreatedAt = latest.CreatedAt

	s.successResponse(c, result)
}

// extractSearchAnswerFromJSON extracts search_answer from JSON string
func extractSearchAnswerFromJSON(text string) string {
	if text == "" {
		return ""
	}

	// Try to parse as JSON
	var jsonData map[string]interface{}
	
	// Clean up markdown code blocks if present
	cleanedText := strings.TrimSpace(text)
	jsonBlockRegex := regexp.MustCompile("(?s)```(?:json)?\\s*(.+?)\\s*```")
	if matches := jsonBlockRegex.FindStringSubmatch(cleanedText); len(matches) > 1 {
		cleanedText = strings.TrimSpace(matches[1])
	} else {
		// Try simple prefix/suffix removal
		cleanedText = strings.TrimPrefix(cleanedText, "```json")
		cleanedText = strings.TrimPrefix(cleanedText, "```")
		cleanedText = strings.TrimSuffix(cleanedText, "```")
		cleanedText = strings.TrimSpace(cleanedText)
	}

	// Try to find JSON object in the text if it's mixed with other content
	if !strings.HasPrefix(cleanedText, "{") {
		jsonStartIdx := strings.Index(cleanedText, "{")
		jsonEndIdx := strings.LastIndex(cleanedText, "}")
		if jsonStartIdx != -1 && jsonEndIdx != -1 && jsonEndIdx > jsonStartIdx {
			cleanedText = cleanedText[jsonStartIdx : jsonEndIdx+1]
		}
	}

	// Try to parse JSON
	if err := json.Unmarshal([]byte(cleanedText), &jsonData); err != nil {
		// Not valid JSON, return empty
		return ""
	}

	// Extract search_answer field
	if searchAnswer, ok := jsonData["search_answer"].(string); ok {
		return searchAnswer
	}

	return ""
}

// cleanMarkdownResponse cleans up markdown response by removing \n and normalizing whitespace
func cleanMarkdownResponse(text string) string {
	if text == "" {
		return ""
	}

	// Replace \n with spaces
	text = strings.ReplaceAll(text, "\\n", " ")
	
	// Replace actual newlines with spaces
	text = strings.ReplaceAll(text, "\n", " ")
	
	// Replace \r\n with spaces
	text = strings.ReplaceAll(text, "\r\n", " ")
	
	// Replace \r with spaces
	text = strings.ReplaceAll(text, "\r", " ")
	
	// Replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`\s+`)
	text = spaceRegex.ReplaceAllString(text, " ")
	
	// Trim leading/trailing whitespace
	text = strings.TrimSpace(text)
	
	return text
}

