package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fissionx/gego/internal/llm"
	// "github.com/fissionx/gego/internal/llm/anthropic"
	// "github.com/fissionx/gego/internal/llm/google"
	// "github.com/fissionx/gego/internal/llm/ollama"
	"github.com/fissionx/gego/internal/llm/openai"
	// "github.com/fissionx/gego/internal/llm/perplexity"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/services"
)

// geoJSONResponse represents the JSON structure returned by the LLM for GEO analysis
type geoJSONResponse struct {
	SearchAnswer string `json:"search_answer"`
	GEOAnalysis  struct {
		VisibilityScore    int      `json:"visibility_score"`
		BrandMentioned     bool     `json:"brand_mentioned"`
		InGroundingSources bool     `json:"in_grounding_sources"`
		MentionStatus      string   `json:"mention_status"`
		Reason             string   `json:"reason"`
		Insights           []string `json:"insights"`
		Actions            []string `json:"actions"`
		CompetitorInfo     string   `json:"competitor_info"`
		Competitors        []string `json:"competitors"`
		Sentiment          string   `json:"sentiment"`
	} `json:"geo_analysis"`
}

// execute handles POST /api/v1/execute
func (s *Server) execute(c *gin.Context) {
	var req models.ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if len(req.Prompt) < 1 {
		s.errorResponse(c, http.StatusBadRequest, "Prompt cannot be empty")
		return
	}

	if len(req.Prompt) > 10000 {
		s.errorResponse(c, http.StatusBadRequest, "Prompt too long (max 10000 characters)")
		return
	}

	// Get the LLM configuration
	llmConfig, err := s.llmService.GetLLM(c.Request.Context(), req.LLMID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "LLM not found: "+err.Error())
		return
	}

	if !llmConfig.Enabled {
		s.errorResponse(c, http.StatusBadRequest, "LLM is disabled")
		return
	}

	// Validate that API key is present (required for most providers)
	if llmConfig.APIKey == "" && llmConfig.Provider != "ollama" {
		errorMsg := fmt.Sprintf("API key is missing for LLM '%s' (ID: %s, Provider: %s). This may indicate a database connection issue preventing the API key from being retrieved.", 
			llmConfig.Name, llmConfig.ID, llmConfig.Provider)
		if s.logger != nil {
			s.logger.Error("LLM API key is empty", 
				zap.String("llm_id", llmConfig.ID),
				zap.String("llm_name", llmConfig.Name),
				zap.String("provider", llmConfig.Provider))
		}
		s.errorResponse(c, http.StatusBadRequest, errorMsg)
		return
	}

	// Create the LLM provider with the specific API key from config
	var provider llm.Provider
	switch llmConfig.Provider {
	case "openai":
		if llmConfig.APIKey == "" {
			errorMsg := fmt.Sprintf("OpenAI API key is required but not set in LLM configuration (LLM ID: %s). Please check if the database connection is working and the API key is stored correctly.", llmConfig.ID)
			if s.logger != nil {
				s.logger.Error("OpenAI API key is empty", 
					zap.String("llm_id", llmConfig.ID),
					zap.String("llm_name", llmConfig.Name))
			}
			s.errorResponse(c, http.StatusBadRequest, errorMsg)
			return
		}
		provider = openai.New(llmConfig.APIKey, llmConfig.BaseURL)
	// case "anthropic":
	// 	provider = anthropic.New(llmConfig.APIKey, llmConfig.BaseURL)
	// case "ollama":
	// 	provider = ollama.New(llmConfig.BaseURL)
	// case "google":
	// 	provider = google.New(llmConfig.APIKey, llmConfig.BaseURL)
	// case "perplexity":
	// 	provider = perplexity.New(llmConfig.APIKey, llmConfig.BaseURL)
	default:
		s.errorResponse(c, http.StatusInternalServerError, "Unknown LLM provider: "+llmConfig.Provider)
		return
	}

	// Set default temperature if not provided
	temperature := req.Temperature
	if temperature == 0 {
		temperature = 0.7
	}

	// Validate temperature
	if temperature < 0 || temperature > 2 {
		s.errorResponse(c, http.StatusBadRequest, "Temperature must be between 0 and 2")
		return
	}

	// Generate response from LLM
	llmResponse, err := provider.Generate(c.Request.Context(), req.Prompt, llm.Config{
		Model:       llmConfig.Model,
		Temperature: temperature,
		MaxTokens:   4096,
		Brand:       req.Brand,
	})
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to generate response: "+err.Error())
		return
	}

	if llmResponse.Error != "" {
		s.errorResponse(c, http.StatusInternalServerError, "LLM error: "+llmResponse.Error)
		return
	}

	var promptID string

	// Optionally save the prompt
	if req.SavePrompt {
		prompt := &models.Prompt{
			ID:       uuid.New().String(),
			Template: req.Prompt,
			Tags:     req.Tags,
			Enabled:  true,
		}
		if err := s.promptService.CreatePrompt(c.Request.Context(), prompt); err != nil {
			// Log but don't fail the request
			// Just continue without saving the prompt
		} else {
			promptID = prompt.ID
		}
	}

	// Parse GEO analysis if brand was provided
	var geoAnalysis *models.GEOAnalysis
	responseText := llmResponse.Text

	if req.Brand != "" {
		// Try to parse the JSON response
		var geoResponse geoJSONResponse

		// Clean up the response - remove markdown code blocks if present
		cleanedText := strings.TrimSpace(llmResponse.Text)

		// Remove markdown code block wrappers (```json ... ``` or ``` ... ```)
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

		log.Printf("========== ATTEMPTING TO PARSE GEO JSON ==========")
		log.Printf("Cleaned text (first 500 chars): %s", truncateString(cleanedText, 500))

		if err := json.Unmarshal([]byte(cleanedText), &geoResponse); err == nil {
			// Successfully parsed JSON
			log.Printf("✅ Successfully parsed GEO JSON response")
			responseText = geoResponse.SearchAnswer
			geoAnalysis = &models.GEOAnalysis{
				VisibilityScore:    geoResponse.GEOAnalysis.VisibilityScore,
				BrandMentioned:     geoResponse.GEOAnalysis.BrandMentioned,
				InGroundingSources: geoResponse.GEOAnalysis.InGroundingSources,
				MentionStatus:      geoResponse.GEOAnalysis.MentionStatus,
				Reason:             geoResponse.GEOAnalysis.Reason,
				Insights:           geoResponse.GEOAnalysis.Insights,
				Actions:            geoResponse.GEOAnalysis.Actions,
				CompetitorInfo:     geoResponse.GEOAnalysis.CompetitorInfo,
				Competitors:        geoResponse.GEOAnalysis.Competitors,
				Sentiment:          geoResponse.GEOAnalysis.Sentiment,
			}
			log.Printf("GEO Analysis: Score=%d, Mentioned=%v, Grounded=%v, Sentiment=%s",
				geoAnalysis.VisibilityScore, geoAnalysis.BrandMentioned,
				geoAnalysis.InGroundingSources, geoAnalysis.Sentiment)
		} else {
			log.Printf("❌ Failed to parse GEO JSON: %v", err)
			log.Printf("Raw response kept as-is")
		}
	}

	// Get prompt to extract BrandID and OrgID if promptID is available
	var brandID, orgID string
	if promptID != "" {
		if prompt, err := s.promptService.GetPrompt(c.Request.Context(), promptID); err == nil && prompt != nil {
			brandID = prompt.BrandID
			orgID = prompt.OrgID
		}
	}

	// Save the response with GEO metrics
	responseModel := &models.Response{
		ID:           uuid.New().String(),
		BrandID:      brandID,
		OrgID:        orgID,
		PromptID:     promptID,
		LLMID:        llmConfig.ID,
		PromptText:   req.Prompt,
		ResponseText: llmResponse.Text,
		LLMName:      llmConfig.Name,
		LLMProvider:  llmConfig.Provider,
		LLMModel:     llmConfig.Model,
		Brand:        req.Brand,
		Temperature:  temperature,
		TokensUsed:   llmResponse.TokensUsed,
		LatencyMs:    llmResponse.LatencyMs,
		CreatedAt:    time.Now(),
	}

	// Store web search metadata (for ChatGPT/Gemini-like experience)
	responseModel.WebSearchQueries = llmResponse.WebSearchQueries
	responseModel.GroundingSources = llmResponse.GroundingSources
	if llmResponse.SearchAnswer != "" {
		responseModel.SearchAnswer = llmResponse.SearchAnswer
	} else {
		// If no SearchAnswer provided, use ResponseText as fallback
		responseModel.SearchAnswer = llmResponse.Text
	}

	// Always extract domains from grounding sources (even if JSON parsing fails)
	if len(llmResponse.GroundingSources) > 0 {
		responseModel.GroundingDomains = services.ExtractDomainsFromSources(llmResponse.GroundingSources)

		// Check if brand appears in grounding sources (even if JSON parsing fails)
		if req.Brand != "" {
			brandLower := strings.ToLower(req.Brand)
			brandDomain := strings.ReplaceAll(brandLower, " ", "")
			for _, source := range llmResponse.GroundingSources {
				sourceLower := strings.ToLower(source)
				if strings.Contains(sourceLower, brandDomain) ||
					strings.Contains(sourceLower, strings.ReplaceAll(brandLower, ".", "")) {
					responseModel.InGroundingSources = true
					break
				}
			}
		}
	}

	// Add GEO metrics if available
	if geoAnalysis != nil {
		responseModel.VisibilityScore = geoAnalysis.VisibilityScore
		responseModel.BrandMentioned = geoAnalysis.BrandMentioned
		// Override InGroundingSources from JSON if available (more accurate)
		responseModel.InGroundingSources = geoAnalysis.InGroundingSources
		responseModel.Sentiment = geoAnalysis.Sentiment
		responseModel.CompetitorsMention = geoAnalysis.Competitors

		// NEW: Extract position/ranking from response
		if req.Brand != "" && geoAnalysis.BrandMentioned {
			position, totalBrands := services.ExtractBrandPosition(responseText, req.Brand)
			responseModel.BrandPosition = position
			responseModel.TotalBrandsListed = totalBrands
		}
	} else if req.Brand != "" {
		// If JSON parsing failed, still try to extract basic metrics from response text
		responseTextLower := strings.ToLower(responseModel.ResponseText)
		brandLower := strings.ToLower(req.Brand)
		if strings.Contains(responseTextLower, brandLower) {
			responseModel.BrandMentioned = true
		}
	}

	// NEW: Add time-series fields
	now := time.Now()
	responseModel.Week = now.Format("2006-W02")
	responseModel.Month = now.Format("2006-01")
	quarter := (int(now.Month())-1)/3 + 1
	responseModel.Quarter = fmt.Sprintf("%d-Q%d", now.Year(), quarter)

	// NEW: Add region/language if provided
	responseModel.Region = req.Region
	responseModel.Language = req.Language

	if err := s.db.CreateResponse(c.Request.Context(), responseModel); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to save response: "+err.Error())
		return
	}

	response := models.ExecuteResponse{
		ResponseID:  responseModel.ID,
		PromptID:    promptID,
		Prompt:      req.Prompt,
		Brand:       req.Brand,
		Response:    responseText,
		GEOAnalysis: geoAnalysis,
		LLMName:     llmConfig.Name,
		LLMProvider: llmConfig.Provider,
		LLMModel:    llmConfig.Model,
		Temperature: temperature,
		TokensUsed:  llmResponse.TokensUsed,
		LatencyMs:   llmResponse.LatencyMs,
		CreatedAt:   responseModel.CreatedAt,
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
		Message: "Prompt executed successfully",
	})
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
