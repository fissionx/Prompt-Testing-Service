package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// BrandInfo represents brand information from the external API
type BrandInfo struct {
	ID        string    `json:"id"` // UUID string
	Name      string    `json:"name"`
	Domain    string    `json:"domain"`
	Language  string    `json:"language"`
	Category  string    `json:"category"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// BrandService handles brand information retrieval and caching
type BrandService struct {
	baseURL    string
	client     *http.Client
	cache      map[string]*BrandInfo
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
}

// NewBrandService creates a new brand service
func NewBrandService(baseURL string) *BrandService {
	if baseURL == "" {
		baseURL = "https://fissionx-geo-backend-service.fly.dev/api/v1"
	}

	return &BrandService{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:    make(map[string]*BrandInfo),
		cacheTTL: 30 * time.Minute, // Cache for 30 minutes
	}
}

// GetBrandInfo retrieves brand information by ID (UUID string) with caching
func (s *BrandService) GetBrandInfo(ctx context.Context, brandID string) (*BrandInfo, error) {
	// Check cache first
	s.cacheMutex.RLock()
	if cached, exists := s.cache[brandID]; exists {
		s.cacheMutex.RUnlock()
		return cached, nil
	}
	s.cacheMutex.RUnlock()

	// Fetch from API
	brandInfo, err := s.fetchBrandInfo(ctx, brandID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch brand info: %w", err)
	}

	// Store in cache
	s.cacheMutex.Lock()
	s.cache[brandID] = brandInfo
	s.cacheMutex.Unlock()

	return brandInfo, nil
}

// fetchBrandInfo makes the actual API call to get brand information
func (s *BrandService) fetchBrandInfo(ctx context.Context, brandID string) (*BrandInfo, error) {
	url := fmt.Sprintf("%s/brands/%s", s.baseURL, brandID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var brandInfo BrandInfo
	if err := json.NewDecoder(resp.Body).Decode(&brandInfo); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &brandInfo, nil
}

// ClearCache clears the brand info cache (useful for testing or forced refresh)
func (s *BrandService) ClearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.cache = make(map[string]*BrandInfo)
}

// ClearCacheForBrand clears cache for a specific brand ID (UUID string)
func (s *BrandService) ClearCacheForBrand(brandID string) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	delete(s.cache, brandID)
}
