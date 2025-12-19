package api

import (
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// BrandPromptsWithLatestResponse represents a prompt with its latest response
type BrandPromptsWithLatestResponse struct {
	Prompt   *models.Prompt   `json:"prompt"`
	Response *models.Response `json:"response,omitempty"`
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

	// For each prompt, get the latest response
	result := make([]BrandPromptsWithLatestResponse, 0)
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

		// Get the latest response
		var latestResponse *models.Response
		if len(responses) > 0 {
			latestResponse = responses[0]
		}

		result = append(result, BrandPromptsWithLatestResponse{
			Prompt:   prompt,
			Response: latestResponse,
		})
	}

	// Sort by prompt created_at descending
	sort.Slice(result, func(i, j int) bool {
		if result[i].Response != nil && result[j].Response != nil {
			return result[i].Response.CreatedAt.After(result[j].Response.CreatedAt)
		}
		if result[i].Response != nil {
			return true
		}
		if result[j].Response != nil {
			return false
		}
		return result[i].Prompt.CreatedAt.After(result[j].Prompt.CreatedAt)
	})

	s.successResponse(c, result)
}

// getPromptResponses handles GET /api/v1/prompts/:id/responses
// Returns the complete LLM response including all metadata for a specific prompt
// Returns the latest response by default, or all responses if ?all=true
func (s *Server) getPromptResponses(c *gin.Context) {
	promptID := c.Param("id")
	if promptID == "" {
		s.errorResponse(c, http.StatusBadRequest, "Prompt ID parameter is required")
		return
	}

	ctx := c.Request.Context()

	// Check if we should return all responses or just the latest
	all := c.Query("all") == "true"

	// Get the prompt to verify it exists
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

	// Sort by created_at descending
	sort.Slice(responses, func(i, j int) bool {
		return responses[i].CreatedAt.After(responses[j].CreatedAt)
	})

	// Prepare response data
	type PromptResponseData struct {
		Prompt    *models.Prompt    `json:"prompt"`
		Responses []*models.Response `json:"responses"`
		Latest    *models.Response   `json:"latest,omitempty"`
		Total     int                `json:"total"`
	}

	data := PromptResponseData{
		Prompt:    prompt,
		Responses: responses,
		Total:     len(responses),
	}

	if len(responses) > 0 {
		data.Latest = responses[0]
	}

	// If all=true, return all responses, otherwise return just the latest
	if !all {
		data.Responses = []*models.Response{}
		if data.Latest != nil {
			data.Responses = []*models.Response{data.Latest}
		}
	}

	s.successResponse(c, data)
}

