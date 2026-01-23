package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

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
		Status:        string(models.OpportunityStatusArchived),
		SuppressedAt:  opportunity.UpdatedAt,
		Message:       "Opportunity suppressed successfully",
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

	action, err := opportunityService.ConvertToAction(c.Request.Context(), opportunityID, req.AdditionalContext)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to convert opportunity to action: "+err.Error())
		return
	}

	response := models.ConvertToActionResponse{
		OpportunityID: opportunityID,
		Action:        action,
		Message:       "Action plan generated successfully",
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
