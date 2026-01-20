package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/fissionx/gego/internal/models"
)

// getBrand handles GET /api/v1/brands/:id
func (s *Server) getBrand(c *gin.Context) {
	brandID := c.Param("id")
	if brandID == "" {
		s.errorResponse(c, http.StatusBadRequest, "brand ID is required")
		return
	}

	// Get brand info from brand service
	brandInfo, err := s.brandService.GetBrandInfo(c.Request.Context(), brandID)
	if err != nil {
		s.errorResponse(c, http.StatusNotFound, "Failed to fetch brand info: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    brandInfo,
		Message: "Brand info retrieved successfully",
	})
}

