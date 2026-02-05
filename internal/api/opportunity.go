package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fissionx/gego/internal/llm"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/services"
)

// listOpportunities handles GET /api/v1/geo/brand/:brandId/opportunities
func (s *Server) listOpportunities(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand ID is required")
		return
	}

	// Parse query parameters
	filter := models.OpportunityFilter{
		BrandID:   brandID,
		Status:    c.Query("status"),
		Type:      c.Query("type"),
		MinImpact: parseIntOrDefault(c.Query("minImpact"), 0),
		Limit:     parseIntOrDefault(c.Query("limit"), 50),
		Offset:    parseIntOrDefault(c.Query("offset"), 0),
	}

	opportunityService := services.NewOpportunityService(s.db, s.llmRegistry)
	opportunities, total, err := opportunityService.ListOpportunities(c.Request.Context(), filter)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to list opportunities: "+err.Error())
		return
	}

	// Get brand name
	brand, _ := s.brandService.GetBrandInfo(c.Request.Context(), brandID)
	brandName := brandID
	if brand != nil {
		brandName = brand.Name
	}

	response := models.ListOpportunitiesResponse{
		Brand:         brandName,
		BrandID:       brandID,
		Opportunities: make([]models.Opportunity, 0, len(opportunities)),
		Total:         total,
		Pagination: models.Pagination{
			Page:       (filter.Offset / filter.Limit) + 1,
			Limit:      filter.Limit,
			Total:      total,
			TotalPages: int((total + int64(filter.Limit) - 1) / int64(filter.Limit)),
		},
	}

	for _, opp := range opportunities {
		response.Opportunities = append(response.Opportunities, *opp)
	}

	s.successResponse(c, response)
}

// listOpportunitiesByPrompt handles GET /api/v1/geo/brand/:brandId/opportunities/prompt/:promptId
func (s *Server) listOpportunitiesByPrompt(c *gin.Context) {
	brandID := c.Param("brandId")
	promptID := c.Param("promptId")

	if brandID == "" || promptID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand ID and Prompt ID are required")
		return
	}

	// Parse query parameters
	filter := models.OpportunityFilter{
		BrandID:   brandID,
		PromptID:  promptID,
		Status:    c.Query("status"),
		Type:      c.Query("type"),
		MinImpact: parseIntOrDefault(c.Query("minImpact"), 0),
		Limit:     parseIntOrDefault(c.Query("limit"), 50),
		Offset:    parseIntOrDefault(c.Query("offset"), 0),
	}

	opportunityService := services.NewOpportunityService(s.db, s.llmRegistry)
	opportunities, total, err := opportunityService.ListOpportunities(c.Request.Context(), filter)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to list opportunities: "+err.Error())
		return
	}

	// Get brand name
	brand, _ := s.brandService.GetBrandInfo(c.Request.Context(), brandID)
	brandName := brandID
	if brand != nil {
		brandName = brand.Name
	}

	response := models.ListOpportunitiesResponse{
		Brand:         brandName,
		BrandID:       brandID,
		Opportunities: make([]models.Opportunity, 0, len(opportunities)),
		Total:         total,
		Pagination: models.Pagination{
			Page:       (filter.Offset / filter.Limit) + 1,
			Limit:      filter.Limit,
			Total:      total,
			TotalPages: int((total + int64(filter.Limit) - 1) / int64(filter.Limit)),
		},
	}

	for _, opp := range opportunities {
		response.Opportunities = append(response.Opportunities, *opp)
	}

	s.successResponse(c, response)
}

// getOpportunity handles GET /api/v1/geo/brand/:brandId/opportunities/:opportunityId
func (s *Server) getOpportunity(c *gin.Context) {
	brandID := c.Param("brandId")
	opportunityID := c.Param("opportunityId")

	if brandID == "" || opportunityID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand ID and Opportunity ID are required")
		return
	}

	opportunityService := services.NewOpportunityService(s.db, s.llmRegistry)
	opportunity, action, prompt, err := opportunityService.GetOpportunityWithDetails(c.Request.Context(), opportunityID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Opportunity not found: "+err.Error())
		return
	}

	// Verify the opportunity belongs to the brand
	if opportunity.BrandID != brandID {
		s.errorResponse(c, http.StatusForbidden, "Opportunity does not belong to this brand")
		return
	}

	response := models.OpportunityDetailResponse{
		Opportunity: *opportunity,
		Action:      action,
		Prompt:      prompt,
	}

	s.successResponse(c, response)
}

// suppressOpportunity handles POST /api/v1/geo/brand/:brandId/opportunities/:opportunityId/suppress
func (s *Server) suppressOpportunity(c *gin.Context) {
	brandID := c.Param("brandId")
	opportunityID := c.Param("opportunityId")

	if brandID == "" || opportunityID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand ID and Opportunity ID are required")
		return
	}

	opportunityService := services.NewOpportunityService(s.db, s.llmRegistry)

	// Verify the opportunity belongs to the brand first
	opportunity, err := s.db.GetOpportunity(c.Request.Context(), opportunityID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Opportunity not found")
		return
	}
	if opportunity.BrandID != brandID {
		s.errorResponse(c, http.StatusForbidden, "Opportunity does not belong to this brand")
		return
	}

	if err := opportunityService.SuppressOpportunity(c.Request.Context(), opportunityID); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to suppress opportunity: "+err.Error())
		return
	}

	response := models.SuppressOpportunityResponse{
		OpportunityID: opportunityID,
		IsArchived:    true,
		SuppressedAt:  time.Now(),
		Message:       "Opportunity archived successfully",
	}

	s.successResponse(c, response)
}

// convertOpportunityToAction handles POST /api/v1/geo/brand/:brandId/opportunities/:opportunityId/convert
func (s *Server) convertOpportunityToAction(c *gin.Context) {
	brandID := c.Param("brandId")
	opportunityID := c.Param("opportunityId")

	if brandID == "" || opportunityID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand ID and Opportunity ID are required")
		return
	}

	// Parse request body (optional)
	var req models.ConvertToActionRequest
	c.ShouldBindJSON(&req) // Ignore error, request body is optional

	opportunityService := services.NewOpportunityService(s.db, s.llmRegistry)

	// Verify the opportunity belongs to the brand first
	opportunity, err := s.db.GetOpportunity(c.Request.Context(), opportunityID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Opportunity not found")
		return
	}
	if opportunity.BrandID != brandID {
		s.errorResponse(c, http.StatusForbidden, "Opportunity does not belong to this brand")
		return
	}

	// Get brand language for action plan output
	brandLanguage := ""
	if brand, err := s.brandService.GetBrandInfo(c.Request.Context(), brandID); err == nil && brand != nil {
		brandLanguage = brand.Language
	}
	// ConvertToAction returns immediately; preparation runs in the background (status "preparing" → "ready")
	action, err := opportunityService.ConvertToAction(c.Request.Context(), opportunityID, req.AdditionalContext, brandLanguage)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to convert opportunity to action: "+err.Error())
		return
	}

	response := models.ConvertToActionResponse{
		OpportunityID: opportunityID,
		Action:        action,
		Message:       "Actions have been captured. Check status via GET /actions (preparing → ready).",
	}

	s.successResponse(c, response)
}

// listActions handles GET /api/v1/geo/brand/:brandId/actions
func (s *Server) listActions(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand ID is required")
		return
	}

	// Parse query parameters
	filter := models.ActionFilter{
		BrandID:       brandID,
		OpportunityID: c.Query("opportunityId"),
		Status:        c.Query("status"),
		Limit:         parseIntOrDefault(c.Query("limit"), 50),
		Offset:        parseIntOrDefault(c.Query("offset"), 0),
	}

	opportunityService := services.NewOpportunityService(s.db, s.llmRegistry)
	actions, total, err := opportunityService.ListActions(c.Request.Context(), filter)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to list actions: "+err.Error())
		return
	}

	// Get brand name
	brand, _ := s.brandService.GetBrandInfo(c.Request.Context(), brandID)
	brandName := brandID
	if brand != nil {
		brandName = brand.Name
	}

	response := models.ListActionsResponse{
		Brand:   brandName,
		BrandID: brandID,
		Actions: make([]models.Action, 0, len(actions)),
		Total:   total,
		Pagination: models.Pagination{
			Page:       (filter.Offset / filter.Limit) + 1,
			Limit:      filter.Limit,
			Total:      total,
			TotalPages: int((total + int64(filter.Limit) - 1) / int64(filter.Limit)),
		},
	}

	for _, action := range actions {
		response.Actions = append(response.Actions, *action)
	}

	s.successResponse(c, response)
}

// updateAction handles PATCH /api/v1/geo/brand/:brandId/actions/:actionId
func (s *Server) updateAction(c *gin.Context) {
	brandID := c.Param("brandId")
	actionID := c.Param("actionId")

	if brandID == "" || actionID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand ID and Action ID are required")
		return
	}

	var req models.UpdateActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	opportunityService := services.NewOpportunityService(s.db, s.llmRegistry)

	// Verify the action belongs to the brand first
	action, err := s.db.GetAction(c.Request.Context(), actionID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Action not found")
		return
	}
	if action.BrandID != brandID {
		s.errorResponse(c, http.StatusForbidden, "Action does not belong to this brand")
		return
	}

	updatedAction, err := opportunityService.UpdateActionStatus(c.Request.Context(), actionID, req.Status, req.CompletedStep)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to update action: "+err.Error())
		return
	}

	response := models.UpdateActionResponse{
		Action:  updatedAction,
		Message: "Action updated successfully",
	}

	s.successResponse(c, response)
}

// getOpportunitySummary handles GET /api/v1/geo/brand/:brandId/opportunities/summary
func (s *Server) getOpportunitySummary(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand ID is required")
		return
	}

	opportunityService := services.NewOpportunityService(s.db, s.llmRegistry)
	summary, err := opportunityService.GetOpportunitySummary(c.Request.Context(), brandID)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get opportunity summary: "+err.Error())
		return
	}

	s.successResponse(c, summary)
}

// batchConvertOpportunitiesToActions handles POST /api/v1/geo/brand/:brandId/opportunities/convert
// Converts multiple opportunities to actions in a single batch operation
// Each opportunity uses its stored LLM ID for action generation
func (s *Server) batchConvertOpportunitiesToActions(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand ID is required")
		return
	}

	var req models.BatchConvertToActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if len(req.OpportunityIDs) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "At least one opportunity ID is required")
		return
	}

	// Limit batch size to prevent abuse
	const maxBatchSize = 50
	if len(req.OpportunityIDs) > maxBatchSize {
		s.errorResponse(c, http.StatusBadRequest, fmt.Sprintf("Batch size exceeds maximum allowed (%d)", maxBatchSize))
		return
	}

	// Get brand language for action plan output
	brandLanguage := ""
	if brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), brandID); err == nil && brandInfo != nil {
		brandLanguage = brandInfo.Language
	}
	opportunityService := services.NewOpportunityService(s.db, s.llmRegistry)
	response, err := opportunityService.BatchConvertToActions(
		c.Request.Context(),
		brandID,
		req.OpportunityIDs,
		req.AdditionalContext,
		brandLanguage,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Batch conversion failed: "+err.Error())
		return
	}

	s.successResponse(c, response)
}

// batchSuppressOpportunities handles POST /api/v1/geo/brand/:brandId/opportunities/suppress
// Suppresses/archives multiple opportunities in a single batch operation
func (s *Server) batchSuppressOpportunities(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand ID is required")
		return
	}

	var req models.BatchSuppressOpportunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if len(req.OpportunityIDs) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "At least one opportunity ID is required")
		return
	}

	// Limit batch size to prevent abuse
	const maxBatchSize = 100
	if len(req.OpportunityIDs) > maxBatchSize {
		s.errorResponse(c, http.StatusBadRequest, fmt.Sprintf("Batch size exceeds maximum allowed (%d)", maxBatchSize))
		return
	}

	opportunityService := services.NewOpportunityService(s.db, s.llmRegistry)
	response, err := opportunityService.BatchSuppressOpportunities(
		c.Request.Context(),
		brandID,
		req.OpportunityIDs,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Batch suppression failed: "+err.Error())
		return
	}

	s.successResponse(c, response)
}

// getOpportunityActions handles GET /api/v1/geo/brand/:brandId/opportunities/:opportunityId/actions
func (s *Server) getOpportunityActions(c *gin.Context) {
	brandID := c.Param("brandId")
	opportunityID := c.Param("opportunityId")
	if brandID == "" || opportunityID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand ID and Opportunity ID are required")
		return
	}

	// Verify opportunity exists and belongs to brand
	opportunity, err := s.db.GetOpportunity(c.Request.Context(), opportunityID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Opportunity not found: "+err.Error())
		return
	}
	if opportunity.BrandID != brandID {
		s.errorResponse(c, http.StatusForbidden, "Opportunity does not belong to this brand")
		return
	}

	// Get actions for this opportunity
	filter := models.ActionFilter{
		OpportunityID: opportunityID,
		Limit:         100, // Get all actions for the opportunity
		Offset:        0,
	}

	opportunityService := services.NewOpportunityService(s.db, s.llmRegistry)
	actions, total, err := opportunityService.ListActions(c.Request.Context(), filter)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to list actions: "+err.Error())
		return
	}

	response := models.GetOpportunityActionsResponse{
		OpportunityID: opportunityID,
		Actions:       make([]models.Action, 0, len(actions)),
		Total:         int(total),
	}

	for _, action := range actions {
		response.Actions = append(response.Actions, *action)
	}

	s.successResponse(c, response)
}

// getAction handles GET /api/v1/geo/brand/:brandId/actions/:actionId
func (s *Server) getAction(c *gin.Context) {
	brandID := c.Param("brandId")
	actionID := c.Param("actionId")
	if brandID == "" || actionID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand ID and Action ID are required")
		return
	}

	action, err := s.db.GetAction(c.Request.Context(), actionID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Action not found: "+err.Error())
		return
	}

	// Verify the action belongs to the brand
	if action.BrandID != brandID {
		s.errorResponse(c, http.StatusForbidden, "Action does not belong to this brand")
		return
	}

	// Check if action is in "preparing" status
	if action.Status == models.ActionStatusPreparing {
		response := models.GetActionResponse{
			Message: "Your action is being prepared by our AI agent. Please wait for some time.",
			Status:  "preparing",
		}
		s.successResponse(c, response)
		return
	}

	response := models.GetActionResponse{
		Action: action,
	}
	s.successResponse(c, response)
}

// updateActionStatus handles POST /api/v1/geo/brand/:brandId/actions/:actionId/status
func (s *Server) updateActionStatus(c *gin.Context) {
	brandID := c.Param("brandId")
	actionID := c.Param("actionId")
	if brandID == "" || actionID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Brand ID and Action ID are required")
		return
	}

	var req models.UpdateActionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Validate status
	validStatuses := []models.ActionStatus{
		models.ActionStatusReady,
		models.ActionStatusPending,
		models.ActionStatusInProgress,
		models.ActionStatusCompleted,
	}
	isValid := false
	for _, validStatus := range validStatuses {
		if req.Status == validStatus {
			isValid = true
			break
		}
	}
	if !isValid {
		s.errorResponse(c, http.StatusBadRequest, "Invalid status. Valid statuses: ready, pending, in_progress, completed")
		return
	}

	// Get existing action
	action, err := s.db.GetAction(c.Request.Context(), actionID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Action not found: "+err.Error())
		return
	}

	// Verify the action belongs to the brand
	if action.BrandID != brandID {
		s.errorResponse(c, http.StatusForbidden, "Action does not belong to this brand")
		return
	}

	// Update status
	action.Status = req.Status
	// Update assignee if provided
	if req.Assignee != nil {
		action.Assignee = *req.Assignee
	}
	action.UpdatedAt = time.Now()

	if err := s.db.UpdateAction(c.Request.Context(), action); err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to update action status: "+err.Error())
		return
	}

	response := models.UpdateActionStatusResponse{
		Action:  action,
		Message: "Action status updated successfully",
	}

	s.successResponse(c, response)
}

// Helper function to parse int with default value
func parseIntOrDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

// getLLMForBrand returns the LLM provider and model configured for a brand (from schedule)
func (s *Server) getLLMForBrand(ctx context.Context, brandID string) (llm.Provider, string, error) {
	// Get schedules for this brand to find the LLM config
	enabled := true
	schedules, err := s.db.ListSchedules(ctx, brandID, &enabled)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get schedules for brand: %w", err)
	}

	if len(schedules) == 0 {
		// Fallback to best available LLM
		return s.getBestLLM(ctx)
	}

	// Get the first schedule's LLM IDs
	schedule := schedules[0]
	if len(schedule.LLMIDs) == 0 {
		// Fallback to best available LLM
		return s.getBestLLM(ctx)
	}

	// Get the first LLM config
	llmID := schedule.LLMIDs[0]
	llmConfig, err := s.db.GetLLM(ctx, llmID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get LLM config: %w", err)
	}

	if llmConfig == nil || !llmConfig.Enabled {
		// Fallback to best available LLM
		return s.getBestLLM(ctx)
	}

	// Get the provider from registry
	provider, ok := s.llmRegistry.Get(llmConfig.Provider)
	if !ok {
		return nil, "", fmt.Errorf("LLM provider not found: %s", llmConfig.Provider)
	}

	fmt.Printf("🔧 Using LLM from schedule: provider=%s, model=%s\n", llmConfig.Provider, llmConfig.Model)
	return provider, llmConfig.Model, nil
}

// getBestLLM returns the best available LLM provider and model
func (s *Server) getBestLLM(ctx context.Context) (llm.Provider, string, error) {
	preferredProviders := []string{"google", "openai", "anthropic", "perplexity"}

	enabled := true
	llmConfigs, err := s.db.ListLLMs(ctx, &enabled)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list LLMs: %w", err)
	}

	if len(llmConfigs) == 0 {
		return nil, "", fmt.Errorf("no enabled LLMs configured")
	}

	// Find best LLM by provider preference
	var selectedConfig *models.LLMConfig
	for _, preferredProvider := range preferredProviders {
		for _, config := range llmConfigs {
			if strings.ToLower(config.Provider) == preferredProvider {
				selectedConfig = config
				break
			}
		}
		if selectedConfig != nil {
			break
		}
	}

	// If no preferred provider found, use first available
	if selectedConfig == nil {
		selectedConfig = llmConfigs[0]
	}

	provider, ok := s.llmRegistry.Get(selectedConfig.Provider)
	if !ok {
		return nil, "", fmt.Errorf("LLM provider not found: %s", selectedConfig.Provider)
	}

	return provider, selectedConfig.Model, nil
}
