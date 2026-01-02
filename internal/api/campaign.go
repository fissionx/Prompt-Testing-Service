package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/services"
	"github.com/fissionx/gego/internal/shared"
)

// deletePromptsByBrand handles DELETE /api/v1/geo/prompts/brand?brandId=X
// Deletes all prompts for a specific brand
func (s *Server) deletePromptsByBrand(c *gin.Context) {
	brandID := c.Query("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId parameter is required")
		return
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), brandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	brandName := brandInfo.Name

	ctx := c.Request.Context()

	// Delete all prompts for this brand
	deletedCount, err := s.db.DeletePromptsByBrand(ctx, brandName)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to delete prompts: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"brand":   brandName,
			"deleted": deletedCount,
		},
		Message: "Prompts deleted successfully",
	})
}

// getBrandPrompts handles GET /api/v1/geo/prompts?brandId=X&count=Y&forceRefresh=true
// Returns both active prompts and suggested prompts for a brand
// Automatically uses domain and category from brand info API
// Query parameters:
//   - brandId (required): Brand UUID from external API
//   - count (optional): Number of prompts to suggest (default: 20)
//   - forceRefresh (optional): Force refresh suggestions (default: false)
func (s *Server) getBrandPrompts(c *gin.Context) {
	brandID := c.Query("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId parameter is required")
		return
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), brandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	brandName := brandInfo.Name
	normalizedDomain := shared.NormalizeDomainToURL(brandInfo.Domain)

	// Use domain from brand info as website (convert to https URL)
	website := ""
	if normalizedDomain != "" {
		website = "https://" + normalizedDomain
	}

	// Use category from brand info
	category := brandInfo.Category

	// Use normalized domain
	domain := normalizedDomain

	// No description needed - will be derived from brand info if needed
	description := ""

	// Parse count parameter (default: 20)
	count := 20
	if countStr := c.Query("count"); countStr != "" {
		if parsed, err := strconv.Atoi(countStr); err == nil && parsed > 0 {
			count = parsed
		}
	}

	// Parse forceRefresh parameter (default: false)
	forceRefresh := c.Query("forceRefresh") == "true"

	s.getBrandPromptsResponse(c, brandName, website, category, domain, description, count, forceRefresh)
}

// getBrandPromptsResponse is a helper function that builds and returns the prompts response
// This is used by both GET and POST/DELETE endpoints to ensure consistent response format
func (s *Server) getBrandPromptsResponse(c *gin.Context, brand, website, category, domain, description string, count int, forceRefresh bool) {
	ctx := c.Request.Context()

	// Get active and suggested prompts
	response, err := s.brandPromptService.GetPrompts(ctx, brand)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get prompts: "+err.Error())
		return
	}

	// If website is provided, ensure we get suggestions (from cache or LLM)
	// When prompts are empty, automatically generate prompts using LLM and cache them
	if website != "" {
		// Check if we need fresh suggestions (empty prompts list)
		needsFreshSuggestions := len(response.ActivePrompts) == 0 && len(response.SuggestedPrompts) == 0

		// Only force refresh if:
		// 1. Explicitly requested via forceRefresh parameter, OR
		// 2. We need fresh suggestions (empty prompts list)
		// Otherwise, use cached suggestions if available
		shouldForceRefresh := forceRefresh || needsFreshSuggestions

		suggestResponse, err := s.brandPromptService.SuggestPrompts(
			ctx,
			brand,
			website,
			category,
			domain,
			description,
			count,
			shouldForceRefresh, // Use cache if available, only force refresh when needed
		)
		if err != nil {
			// If suggestion fails and we have no suggestions, return error
			if len(response.SuggestedPrompts) == 0 {
				s.errorResponse(c, http.StatusInternalServerError,
					"Failed to get prompt suggestions: "+err.Error())
				return
			}
			// Otherwise, fall back to existing SuggestedPrompts from database
		} else {
			// Filter the suggestions to exclude already active prompts
			activePromptIDMap := make(map[string]bool)
			for _, active := range response.ActivePrompts {
				activePromptIDMap[active.ID] = true
			}

			var filteredSuggested []models.PromptDetail
			for _, suggested := range suggestResponse.Prompts {
				if !activePromptIDMap[suggested.ID] {
					filteredSuggested = append(filteredSuggested, suggested)
				}
			}

			response.SuggestedPrompts = filteredSuggested
			response.LLMDetails = suggestResponse.LLMDetails

			// When active prompts list is empty, ensure we have suggestions
			if len(response.ActivePrompts) == 0 && len(response.SuggestedPrompts) == 0 && len(suggestResponse.Prompts) > 0 {
				response.SuggestedPrompts = suggestResponse.Prompts
			}
		}
	}

	s.successResponse(c, response)
}

// SaveAndExecutePromptsRequest represents the request to save and execute prompts
type SaveAndExecutePromptsRequest struct {
	BrandID       string                `json:"brandId" binding:"required"` // Brand ID (UUID) from external API
	CampaignName  string                `json:"campaignName"`
	PromptIDs     []string              `json:"promptIds"`     // Existing prompt IDs (suggested prompts)
	CustomPrompts []models.CustomPrompt `json:"customPrompts"` // User's custom prompts
	LLMIDs        []string              `json:"llmIds" binding:"required"`
	Temperature   float64               `json:"temperature"`
	ScheduleCron  string                `json:"scheduleCron"` // Optional: for scheduled execution
	TotalRuns     int                   `json:"totalRuns"`    // Number of runs per prompt (default: 1)
}

// saveAndExecutePrompts handles POST /api/v1/geo/prompts/execute/bulk
// Saves prompts for a brand (upsert - replaces existing) and executes them
func (s *Server) saveAndExecutePrompts(c *gin.Context) {
	var req SaveAndExecutePromptsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if req.BrandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId is required")
		return
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), req.BrandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	brandName := brandInfo.Name

	if len(req.PromptIDs) == 0 && len(req.CustomPrompts) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "At least one prompt is required (promptIds or customPrompts)")
		return
	}

	if len(req.LLMIDs) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "At least one LLM is required")
		return
	}

	ctx := c.Request.Context()

	type promptSeed struct {
		Template       string
		PromptType     models.PromptType
		Category       string
		Tags           []string
		Generated      bool
		SourcePromptID string
	}

	seeds := make([]promptSeed, 0, len(req.PromptIDs)+len(req.CustomPrompts))
	seen := make(map[string]struct{})

	addSeed := func(seed promptSeed) {
		template := strings.TrimSpace(seed.Template)
		if template == "" {
			return
		}
		seed.Template = template

		key := seed.SourcePromptID
		if key == "" {
			key = strings.ToLower(template)
		}

		if _, exists := seen[key]; exists {
			return
		}

		seen[key] = struct{}{}
		seeds = append(seeds, seed)
	}

	for _, promptID := range req.PromptIDs {
		existingPrompt, err := s.db.GetPrompt(ctx, promptID)
		if err != nil || existingPrompt == nil {
			continue
		}

		addSeed(promptSeed{
			Template:       existingPrompt.Template,
			PromptType:     existingPrompt.PromptType,
			Category:       existingPrompt.Category,
			Tags:           append([]string(nil), existingPrompt.Tags...),
			Generated:      existingPrompt.Generated,
			SourcePromptID: promptID,
		})
	}

	for _, customPrompt := range req.CustomPrompts {
		promptType := models.PromptType(customPrompt.PromptType)
		if promptType == "" {
			promptType = models.PromptTypeCustom
		}

		addSeed(promptSeed{
			Template:       customPrompt.Template,
			PromptType:     promptType,
			Category:       customPrompt.Category,
			Tags:           append([]string(nil), customPrompt.Tags...),
			Generated:      false,
			SourcePromptID: "",
		})
	}

	if len(seeds) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "No valid prompts provided")
		return
	}

	// Use BrandPromptService to properly manage prompts and preserve suggested prompts
	// First, delete all active prompts (but preserve suggested prompts in BrandPrompts record)
	if err := s.brandPromptService.DeleteAllPrompts(ctx, brandName); err != nil {
		// If no BrandPrompts record exists, that's okay - we'll create one
		fmt.Printf("Warning: failed to delete existing prompts: %v\n", err)
	}

	// Prepare prompt IDs and custom prompts for SavePrompts
	var promptIDsFromSuggested []string
	var customPromptsToSave []models.CustomPrompt

	// Separate existing prompts (by ID) from new custom prompts
	for _, seed := range seeds {
		if seed.SourcePromptID != "" {
			// This is an existing prompt ID - check if it exists
			prompt, err := s.db.GetPrompt(ctx, seed.SourcePromptID)
			if err == nil && prompt != nil {
				// Use existing prompt - will be moved to active by SavePrompts
				promptIDsFromSuggested = append(promptIDsFromSuggested, seed.SourcePromptID)
			} else {
				// Prompt doesn't exist, create it as custom
				customPromptsToSave = append(customPromptsToSave, models.CustomPrompt{
					Template:   seed.Template,
					PromptType: string(seed.PromptType),
					Category:   seed.Category,
					Tags:       seed.Tags,
				})
			}
		} else {
			// This is a new custom prompt
			customPromptsToSave = append(customPromptsToSave, models.CustomPrompt{
				Template:   seed.Template,
				PromptType: string(seed.PromptType),
				Category:   seed.Category,
				Tags:       seed.Tags,
			})
		}
	}

	// Use BrandPromptService to save prompts properly (preserves suggested prompts)
	// This ensures the BrandPrompts record is properly maintained
	source := "custom"
	if len(promptIDsFromSuggested) > 0 && len(customPromptsToSave) > 0 {
		source = "mixed"
	} else if len(promptIDsFromSuggested) > 0 {
		source = "suggested"
	}

	saveResponse, err := s.brandPromptService.SavePrompts(
		ctx,
		brandName,
		promptIDsFromSuggested,
		customPromptsToSave,
		source,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to save prompts: "+err.Error())
		return
	}

	allPromptIDs := saveResponse.SavedPromptIDs

	// Set defaults
	if req.CampaignName == "" {
		req.CampaignName = brandName + " Execution"
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	totalRuns := req.TotalRuns
	if totalRuns == 0 {
		totalRuns = 1
	}

	// If scheduleCron is provided, create a scheduled campaign
	if req.ScheduleCron != "" {
		scheduledCampaign, err := s.scheduledCampaignManager.CreateScheduledCampaign(
			ctx,
			req.CampaignName,
			brandName,
			allPromptIDs,
			req.LLMIDs,
			req.Temperature,
			req.ScheduleCron,
			totalRuns,
		)
		if err != nil {
			s.errorResponse(c, http.StatusInternalServerError, "Failed to create scheduled campaign: "+err.Error())
			return
		}

		response := models.BulkExecuteResponse{
			CampaignID:   scheduledCampaign.ID,
			CampaignName: scheduledCampaign.CampaignName,
			Brand:        scheduledCampaign.Brand,
			TotalRuns:    scheduledCampaign.TotalRuns,
			Status:       scheduledCampaign.Status,
			StartedAt:    scheduledCampaign.CreatedAt,
			NextRunAt:    scheduledCampaign.NextRunAt,
			ScheduleCron: scheduledCampaign.ScheduleCron,
			Message:      "Prompts saved and scheduled campaign created. First execution started in background. Next run at " + scheduledCampaign.NextRunAt.Format("2006-01-02 15:04:05 UTC"),
		}

		c.JSON(http.StatusAccepted, models.APIResponse{
			Success: true,
			Data:    response,
			Message: "Prompts saved and scheduled campaign created",
		})
		return
	}

	// Create bulk execution service for one-time execution
	bulkService := services.NewBulkExecutionService(s.db, s.llmRegistry)

	// Start campaign execution
	campaign, err := bulkService.ExecuteCampaign(
		ctx,
		req.CampaignName,
		brandName,
		allPromptIDs,
		req.LLMIDs,
		req.Temperature,
		totalRuns,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to start execution: "+err.Error())
		return
	}

	response := models.BulkExecuteResponse{
		CampaignID:   campaign.ID,
		CampaignName: campaign.Name,
		Brand:        campaign.Brand,
		TotalRuns:    campaign.TotalRuns,
		Status:       campaign.Status,
		StartedAt:    campaign.CreatedAt,
		Message:      "Prompts saved and execution started successfully. Running in background.",
	}

	c.JSON(http.StatusAccepted, models.APIResponse{
		Success: true,
		Data:    response,
		Message: "Prompts saved and execution started",
	})
}
