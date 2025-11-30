package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AI2HU/gego/internal/db"
	"github.com/AI2HU/gego/internal/models"
	"github.com/google/uuid"
)

// LogoService provides company logo URLs with database caching
type LogoService struct {
	db db.Database
}

// NewLogoService creates a new logo service
func NewLogoService(database db.Database) *LogoService {
	return &LogoService{
		db: database,
	}
}

// GetBrandLogo returns logo information for a brand (checks DB first, then fetches)
func (s *LogoService) GetBrandLogo(ctx context.Context, brandName string, website string) models.BrandWithLogo {
	// Normalize brand name for lookup
	normalized := strings.ToLower(strings.TrimSpace(brandName))
	
	// Try to get from database first
	cached, err := s.db.GetBrandLogo(ctx, normalized)
	if err == nil && cached != nil {
		// Check if cache is still fresh (refresh after 30 days)
		if time.Since(cached.LastChecked) < 30*24*time.Hour {
			return models.BrandWithLogo{
				Brand:           brandName,
				LogoURL:         cached.LogoURL,
				FallbackLogoURL: cached.FallbackLogoURL,
			}
		}
	}
	
	// Not in cache or stale - fetch new
	domain := s.extractDomain(brandName, website)
	logoURL := ""
	fallbackURL := ""
	source := "placeholder"
	
	if domain != "" {
		logoURL = fmt.Sprintf("https://logo.clearbit.com/%s", domain)
		fallbackURL = fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=128", domain)
		source = "clearbit"
	} else {
		logoURL = s.getPlaceholderLogo(brandName)
		fallbackURL = logoURL
		source = "placeholder"
	}
	
	// Save to database
	logoCache := &models.BrandLogoCache{
		ID:              uuid.New().String(),
		BrandName:       normalized,
		Domain:          domain,
		LogoURL:         logoURL,
		FallbackLogoURL: fallbackURL,
		Source:          source,
		LastChecked:     time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	
	// Save asynchronously (don't block on save)
	go func() {
		if err := s.db.SaveBrandLogo(context.Background(), logoCache); err != nil {
			// Log error but don't fail the request
			fmt.Printf("Warning: failed to cache logo for %s: %v\n", brandName, err)
		}
	}()
	
	return models.BrandWithLogo{
		Brand:           brandName,
		LogoURL:         logoURL,
		FallbackLogoURL: fallbackURL,
	}
}

// GetMultipleLogos returns logo information for multiple brands
func (s *LogoService) GetMultipleLogos(ctx context.Context, brands []BrandLogoRequest) []models.BrandWithLogo {
	logos := make([]models.BrandWithLogo, 0, len(brands))
	
	for _, brand := range brands {
		logos = append(logos, s.GetBrandLogo(ctx, brand.Name, brand.Website))
	}
	
	return logos
}

// extractDomain extracts a domain from brand name or website
func (s *LogoService) extractDomain(brandName string, website string) string {
	// If website is provided, extract domain from it
	if website != "" {
		domain := s.cleanDomain(website)
		if domain != "" {
			return domain
		}
	}
	
	// Try to infer domain from brand name
	brandLower := strings.ToLower(strings.TrimSpace(brandName))
	
	// Common brand name to domain mappings
	knownDomains := map[string]string{
		"amazon":         "amazon.com",
		"ebay":           "ebay.com",
		"walmart":        "walmart.com",
		"target":         "target.com",
		"best buy":       "bestbuy.com",
		"etsy":           "etsy.com",
		"shopify":        "shopify.com",
		"alibaba":        "alibaba.com",
		"wayfair":        "wayfair.com",
		"costco":         "costco.com",
		"kroger":         "kroger.com",
		"home depot":     "homedepot.com",
		"the home depot": "homedepot.com",
		"apple":          "apple.com",
		"nike":           "nike.com",
		"samsung":        "samsung.com",
		"temu":           "temu.com",
		"shein":          "shein.com",
		"chewy":          "chewy.com",
		"google":         "google.com",
		"microsoft":      "microsoft.com",
		"facebook":       "facebook.com",
		"meta":           "meta.com",
		"netflix":        "netflix.com",
		"spotify":        "spotify.com",
		"twitter":        "twitter.com",
		"x":              "x.com",
		"linkedin":       "linkedin.com",
		"reddit":         "reddit.com",
		"youtube":        "youtube.com",
		"instagram":      "instagram.com",
		"tiktok":         "tiktok.com",
		"salesforce":     "salesforce.com",
		"oracle":         "oracle.com",
		"adobe":          "adobe.com",
		"ibm":            "ibm.com",
		"cisco":          "cisco.com",
		"uber":           "uber.com",
		"lyft":           "lyft.com",
		"airbnb":         "airbnb.com",
		"booking":        "booking.com",
		"expedia":        "expedia.com",
	}
	
	if domain, ok := knownDomains[brandLower]; ok {
		return domain
	}
	
	// Try simple heuristic: brandname.com
	// Remove spaces and special characters
	simpleDomain := strings.ReplaceAll(brandLower, " ", "")
	simpleDomain = strings.ReplaceAll(simpleDomain, ".", "")
	simpleDomain = strings.ReplaceAll(simpleDomain, "-", "")
	
	if simpleDomain != "" {
		return simpleDomain + ".com"
	}
	
	return ""
}

// cleanDomain extracts clean domain from URL
func (s *LogoService) cleanDomain(urlOrDomain string) string {
	// Remove protocol
	domain := strings.TrimPrefix(urlOrDomain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "www.")
	
	// Remove path
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}
	
	// Remove port
	if idx := strings.Index(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}
	
	return strings.TrimSpace(domain)
}

// getFallbackLogo returns a fallback logo URL (Google Favicon)
func (s *LogoService) getFallbackLogo(brandName string, website string) string {
	domain := s.extractDomain(brandName, website)
	
	if domain == "" {
		return s.getPlaceholderLogo(brandName)
	}
	
	// Google Favicon service (always returns something, even if generic)
	return fmt.Sprintf("https://www.google.com/s2/favicons?domain=%s&sz=128", domain)
}

// getPlaceholderLogo returns a placeholder logo based on brand initials
func (s *LogoService) getPlaceholderLogo(brandName string) string {
	// UI Avatars - generates placeholder with initials
	// Example: https://ui-avatars.com/api/?name=Amazon&size=128&background=0D8ABC&color=fff
	initials := s.getInitials(brandName)
	return fmt.Sprintf("https://ui-avatars.com/api/?name=%s&size=128&background=0D8ABC&color=fff&bold=true", initials)
}

// getInitials extracts initials from brand name
func (s *LogoService) getInitials(brandName string) string {
	parts := strings.Fields(strings.TrimSpace(brandName))
	
	if len(parts) == 0 {
		return "?"
	}
	
	if len(parts) == 1 {
		// Single word - take first 2 characters
		if len(parts[0]) >= 2 {
			return strings.ToUpper(parts[0][:2])
		}
		return strings.ToUpper(parts[0])
	}
	
	// Multiple words - take first letter of first two words
	initials := string(parts[0][0]) + string(parts[1][0])
	return strings.ToUpper(initials)
}

// BrandLogoRequest represents a request for a brand logo
type BrandLogoRequest struct {
	Name    string
	Website string
}

