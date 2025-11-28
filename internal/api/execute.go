package api

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/AI2HU/gego/internal/llm"
	"github.com/AI2HU/gego/internal/llm/anthropic"
	"github.com/AI2HU/gego/internal/llm/google"
	"github.com/AI2HU/gego/internal/llm/ollama"
	"github.com/AI2HU/gego/internal/llm/openai"
	"github.com/AI2HU/gego/internal/llm/perplexity"
	"github.com/AI2HU/gego/internal/models"
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

	// Create the LLM provider with the specific API key from config
	var provider llm.Provider
	switch llmConfig.Provider {
	case "openai":
		provider = openai.New(llmConfig.APIKey, llmConfig.BaseURL)
	case "anthropic":
		provider = anthropic.New(llmConfig.APIKey, llmConfig.BaseURL)
	case "ollama":
		provider = ollama.New(llmConfig.BaseURL)
	case "google":
		provider = google.New(llmConfig.APIKey, llmConfig.BaseURL)
	case "perplexity":
		provider = perplexity.New(llmConfig.APIKey, llmConfig.BaseURL)
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

	// Save the response with GEO metrics
	responseModel := &models.Response{
		ID:           uuid.New().String(),
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

	// Add GEO metrics if available
	if geoAnalysis != nil {
		responseModel.VisibilityScore = geoAnalysis.VisibilityScore
		responseModel.BrandMentioned = geoAnalysis.BrandMentioned
		responseModel.InGroundingSources = geoAnalysis.InGroundingSources
		responseModel.Sentiment = geoAnalysis.Sentiment
		responseModel.CompetitorsMention = geoAnalysis.Competitors
	}

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

