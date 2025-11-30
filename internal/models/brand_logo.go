package models

import (
	"time"
)

// BrandLogoCache represents a cached brand logo in the database
type BrandLogoCache struct {
	ID              string    `json:"id" bson:"_id"`
	BrandName       string    `json:"brandName" bson:"brand_name"`        // Normalized brand name
	Domain          string    `json:"domain" bson:"domain"`               // Associated domain
	LogoURL         string    `json:"logoUrl" bson:"logo_url"`            // Primary logo URL
	FallbackLogoURL string    `json:"fallbackLogoUrl" bson:"fallback_logo_url"` // Fallback logo URL
	Source          string    `json:"source" bson:"source"`               // clearbit, google, placeholder
	LastChecked     time.Time `json:"lastChecked" bson:"last_checked"`    // When logo was last verified
	CreatedAt       time.Time `json:"createdAt" bson:"created_at"`
	UpdatedAt       time.Time `json:"updatedAt" bson:"updated_at"`
}

// BrandWithLogo represents brand information with logo
type BrandWithLogo struct {
	Brand           string `json:"brand"`
	LogoURL         string `json:"logoUrl,omitempty"`
	FallbackLogoURL string `json:"fallbackLogoUrl,omitempty"`
}

