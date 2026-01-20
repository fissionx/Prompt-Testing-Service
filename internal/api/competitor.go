package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fissionx/gego/internal/middleware"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// saveCompetitors handles POST /api/v1/geo/brand/:brandId/competitors
// Adds new competitors to the existing list (does not replace)
// UI can send just the new competitor(s) to add - backend will merge with existing list
// Deduplicates automatically to prevent adding the same competitor twice
// Returns the same response format as GET /api/v1/geo/brand/:brandId/competitors
func (s *Server) saveCompetitors(c *gin.Context) {
	brandID := c.Param("brandId")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "BrandId is required")
		return
	}

	var req models.SaveCompetitorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.errorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Get brand info from external API
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), brandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	brandName := brandInfo.Name

	// Normalize competitor domains to www. format
	normalizedCompetitors := make([]models.Competitor, len(req.Competitors))
	for i, comp := range req.Competitors {
		normalizedCompetitors[i] = comp
		if comp.Domain != "" {
			normalizedCompetitors[i].Domain = shared.NormalizeDomainToURL(comp.Domain)
		}
	}

	if len(normalizedCompetitors) == 0 {
		s.errorResponse(c, http.StatusBadRequest, "At least one competitor is required")
		return
	}

	ctx := c.Request.Context()

	// Get orgId from context (set by auth middleware)
	orgID, err := middleware.GetOrgID(c)
	if err != nil {
		// If orgId is not available, use empty string (for backward compatibility)
		// In production, you might want to return an error instead
		orgID = ""
	}

	_, err = s.competitorService.SaveCompetitors(
		ctx,
		brandName,
		brandID,
		normalizedCompetitors,
		req.Source,
		orgID,
	)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to save competitors: "+err.Error())
		return
	}

	// Return the updated competitors list in the same format as GET endpoint
	s.getCompetitorsResponse(c, brandName, brandID, brandInfo.Domain, brandInfo.Description, brandInfo.Category, false)
}

// getCompetitors handles GET /api/v1/geo/brand/:brandId/competitors
// Merged endpoint that returns both saved competitors and suggested competitors
// If website parameter is provided, it will also fetch suggestions and filter out already saved competitors
func (s *Server) getCompetitors(c *gin.Context) {
	brandID := c.Param("brandId")
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
	forceRefresh := c.Query("forceRefresh") == "true"

	s.getCompetitorsResponse(c, brandInfo.Name, brandInfo.ID, brandInfo.Domain, brandInfo.Description, brandInfo.Category, forceRefresh)
}

// getCompetitorsResponse is a helper function that builds and returns the competitors response
// This is used by both GET and POST/DELETE endpoints to ensure consistent response format
func (s *Server) getCompetitorsResponse(c *gin.Context, brand, brandID, website, description, category string, forceRefresh bool) {
	ctx := c.Request.Context()

	// Get saved competitors
	response, err := s.competitorService.GetCompetitors(ctx, brandID)
	response.Brand = brand
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to get competitors: "+err.Error())
		return
	}

	// Build a map of saved competitor names for filtering (case-insensitive)
	savedCompetitorNames := make(map[string]bool)
	for _, comp := range response.Competitors {
		savedCompetitorNames[strings.ToLower(strings.TrimSpace(comp.Name))] = true
	}

	// Filter function to remove already saved competitors from suggestions
	filterSuggestions := func(suggestions []models.Competitor) []models.Competitor {
		filtered := make([]models.Competitor, 0)
		for _, suggested := range suggestions {
			suggestedName := strings.ToLower(strings.TrimSpace(suggested.Name))
			if !savedCompetitorNames[suggestedName] {
				filtered = append(filtered, suggested)
			}
		}
		return filtered
	}

	// If website is provided, ensure we get suggestions (from cache or LLM)
	// This is especially important when competitors list is empty - we need to populate suggestedList
	if website != "" {
		// When competitors list is empty and no cached suggestions, we MUST get fresh suggestions from LLM
		needsFreshSuggestions := len(response.Competitors) == 0 && len(response.SuggestedList) == 0

		// Get orgId from context (set by auth middleware)
		orgID, err := middleware.GetOrgID(c)
		if err != nil {
			// If orgId is not available, use empty string (for backward compatibility)
			orgID = ""
		}

		suggestResponse, err := s.competitorService.SuggestCompetitors(
			ctx,
			brand,
			brandID,
			website,
			description,
			category,
			forceRefresh || needsFreshSuggestions, // Force refresh if we need fresh suggestions
			orgID,
		)
		if err != nil {
			// If suggestion fails and we have no suggestions, return error
			if len(response.SuggestedList) == 0 {
				s.errorResponse(c, http.StatusInternalServerError,
					"Failed to get competitor suggestions: "+err.Error())
				return
			}
			// Otherwise, fall back to existing SuggestedList from database
			response.SuggestedList = filterSuggestions(response.SuggestedList)
		} else {
			// Filter the suggestions to exclude already saved competitors
			response.SuggestedList = filterSuggestions(suggestResponse.Competitors)

			// Include LLM details from the suggestion response
			response.LLMDetails = suggestResponse.LLMDetails

			// When competitors list is empty, ensure we have suggestions
			// If all were filtered out (shouldn't happen), use the unfiltered list
			if len(response.Competitors) == 0 && len(response.SuggestedList) == 0 && len(suggestResponse.Competitors) > 0 {
				// Edge case: all suggestions were filtered (shouldn't happen with empty competitors list)
				// But use unfiltered suggestions to ensure we return something
				response.SuggestedList = suggestResponse.Competitors
			}
		}
	} else {
		// No website provided, filter existing SuggestedList from database
		response.SuggestedList = filterSuggestions(response.SuggestedList)
	}

	message := "Competitors retrieved successfully"
	if len(response.Competitors) == 0 && len(response.SuggestedList) == 0 {
		message = "No competitors found for this brand"
	} else if len(response.Competitors) == 0 {
		message = "No competitors saved yet. Suggestions available."
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
		Message: message,
	})
}

// deleteCompetitors handles DELETE /api/v1/geo/brand/:brandId/competitors
// Supports two modes:
// 1. Delete all competitors: DELETE /api/v1/geo/brand/:brandId/competitors
// 2. Delete individual competitor: DELETE /api/v1/geo/brand/:brandId/competitors?name=Samsung
// Returns the same response format as GET /api/v1/geo/brand/:brandId/competitors
func (s *Server) deleteCompetitors(c *gin.Context) {
	brandID := c.Param("brandId")
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

	competitorName := c.Query("name")
	ctx := c.Request.Context()

	// If name is provided, delete individual competitor
	if competitorName != "" {
		_, err := s.competitorService.DeleteCompetitorByName(ctx, brandID, competitorName)
		if err != nil {
			// Check if it's a "not found" error
			if strings.Contains(err.Error(), "not found") {
				s.errorResponse(c, http.StatusNotFound, err.Error())
				return
			}
			s.errorResponse(c, http.StatusInternalServerError, "Failed to delete competitor: "+err.Error())
			return
		}

		// Return the updated competitors list in the same format as GET endpoint
		s.getCompetitorsResponse(c, brandName, brandID, brandInfo.Domain, brandInfo.Description, brandInfo.Category, false)
		return
	}

	// Otherwise, move all competitors to suggestedList (preserves data)
	err = s.competitorService.DeleteCompetitors(ctx, brandID)
	if err != nil {
		s.errorResponse(c, http.StatusInternalServerError, "Failed to clear competitors: "+err.Error())
		return
	}

	// Return the updated competitors list in the same format as GET endpoint
	s.getCompetitorsResponse(c, brandName, brandID, brandInfo.Domain, brandInfo.Description, brandInfo.Category, false)
}
