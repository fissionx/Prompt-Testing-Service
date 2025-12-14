package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/services"
)

// deletePromptsByBrand handles DELETE /api/v1/geo/prompts?brand=X
// Deletes all prompts for a specific brand
func (s *Server) deletePromptsByBrand(c *gin.Context) {
	brand := c.Query("brand")
	if brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand parameter is required")
		return
	}

	ctx := c.Request.Context()

	// Delete all prompts for this brand
	deletedCount, err := s.db.DeletePromptsByBrand(ctx, brand)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to delete prompts: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"brand":   brand,
			"deleted": deletedCount,
		},
		Message: "Prompts deleted successfully",
	})
}

// getBrandPrompts handles GET /api/v1/geo/prompts?brand=X
// Returns all prompts for a brand with available LLMs
func (s *Server) getBrandPrompts(c *gin.Context) {
	brand := c.Query("brand")
	if brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand parameter is required")
		return
	}

	ctx := c.Request.Context()

	// Get ALL prompts for this brand from the prompts collection
	allPrompts, err := s.db.ListPrompts(ctx, nil)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to list prompts: "+err.Error())
		return
	}

	// Filter prompts for this brand (case-insensitive)
	var brandPrompts []models.PromptDetail
	for _, prompt := range allPrompts {
		if prompt.Brand != "" && strings.EqualFold(prompt.Brand, brand) {
			brandPrompts = append(brandPrompts, models.PromptDetail{
				ID:         prompt.ID,
				Template:   prompt.Template,
				PromptType: prompt.PromptType,
				Category:   prompt.Category,
			})
		}
	}

	// Get all enabled LLMs
	allLLMs, _ := s.db.ListLLMs(ctx, nil)
	var llms []models.LLMDetail
	for _, llm := range allLLMs {
		if llm.Enabled {
			llms = append(llms, models.LLMDetail{
				ID:       llm.ID,
				Name:     llm.Name,
				Provider: llm.Provider,
				Model:    llm.Model,
			})
		}
	}

	// Build response
	response := map[string]interface{}{
		"brand":   brand,
		"prompts": brandPrompts,
		"llms":    llms,
		"total":   len(brandPrompts),
	}

	message := "Prompts retrieved successfully"
	if len(brandPrompts) == 0 {
		message = "No prompts found for this brand. Generate prompts first using /geo/prompts/generate"
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
		Message: message,
	})
}

// SaveAndExecutePromptsRequest represents the request to save and execute prompts
type SaveAndExecutePromptsRequest struct {
	Brand         string                `json:"brand" binding:"required"`
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

	if req.Brand == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand is required")
		return
	}

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

	if _, err := s.db.DeletePromptsByBrand(ctx, req.Brand); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to clear existing prompts: "+err.Error())
		return
	}

	allPromptIDs := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		newPrompt := &models.Prompt{
			ID:         uuid.New().String(),
			Template:   seed.Template,
			PromptType: seed.PromptType,
			Category:   seed.Category,
			Tags:       seed.Tags,
			Brand:      req.Brand,
			Generated:  seed.Generated,
			Enabled:    true,
			SourceID:   seed.SourcePromptID,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if err := s.db.CreatePrompt(ctx, newPrompt); err != nil {
			s.errorResponse(c, http.StatusInternalServerError, "Failed to save prompt: "+err.Error())
			return
		}

		allPromptIDs = append(allPromptIDs, newPrompt.ID)
	}

	// Set defaults
	if req.CampaignName == "" {
		req.CampaignName = req.Brand + " Execution"
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
			req.Brand,
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
		req.Brand,
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
