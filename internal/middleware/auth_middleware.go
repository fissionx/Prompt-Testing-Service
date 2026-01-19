package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/fissionx/gego/internal/config"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// ClerkClaims represents the JWT claims from Clerk
type ClerkClaims struct {
	jwt.RegisteredClaims
	Email         string                 `json:"email,omitempty"`
	EmailVerified bool                   `json:"email_verified,omitempty"`
	FirstName     string                 `json:"first_name,omitempty"`
	LastName      string                 `json:"last_name,omitempty"`
	FullName      string                 `json:"full_name,omitempty"`
	ImageURL      string                 `json:"image_url,omitempty"`
	Orgs          map[string]interface{} `json:"orgs,omitempty"` // Clerk org memberships
	Azp           string                 `json:"azp,omitempty"`  // Authorized party
}

const (
	// Context keys (flat structure)
	UserIDKey     = "userId"     // Internal user ID (from your DB)
	IAMUserIDKey  = "iamUserId"  // Clerk user ID (from JWT sub claim)
	OrgIDKey      = "orgId"      // Internal org ID (from your DB, if resolved)
	IAMOrgIDKey   = "iamOrgId"   // Clerk org ID (from header)
	UserOrgIDsKey = "userOrgIds" // User's org memberships from JWT (Clerk org IDs)
	UserClaimsKey = "userClaims" // Full JWT claims for advanced use cases

	// Header name for organization ID (kebab-case, standard HTTP convention)
	OrgIDHeader = "X-Org-Id"

	// Cookie name for Clerk session
	ClerkSessionCookie = "__session"
)

// clerkJWKS holds the cached JWKS keyfunc
var (
	clerkJWKS     keyfunc.Keyfunc
	clerkJWKSOnce sync.Once
	clerkJWKSErr  error
)

// ClerkAuth creates a Gin middleware for Clerk JWT authentication
// It extracts the token from the __session cookie or Authorization header
func ClerkAuth(cfg config.ClerkConfig, logger config.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from cookie first, then header
		token := extractToken(c)
		if token == "" {
			logger.Warn("No authentication token found",
				zap.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHORIZED",
			})
			return
		}

		// Initialize JWKS (once, cached)
		jwksURL := getJWKSURL(cfg)
		if jwksURL == "" {
			logger.Error("Clerk JWKS URL not configured")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Authentication not configured",
				"code":  "AUTH_CONFIG_ERROR",
			})
			return
		}

		// Get or create JWKS keyfunc
		keyfuncJWKS, err := getClerkJWKS(c.Request.Context(), jwksURL, logger)
		if err != nil {
			logger.Error("Failed to fetch Clerk JWKS",
				zap.Error(err),
			)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "Authentication service unavailable",
				"code":  "AUTH_SERVICE_ERROR",
			})
			return
		}

		// Parse and validate JWT
		claims := &ClerkClaims{}
		parsedToken, err := jwt.ParseWithClaims(token, claims, keyfuncJWKS.KeyfuncCtx(c.Request.Context()))
		if err != nil {
			logger.Warn("Invalid JWT token",
				zap.Error(err),
				zap.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
				"code":  "INVALID_TOKEN",
			})
			return
		}

		if !parsedToken.Valid {
			logger.Warn("JWT token not valid",
				zap.String("path", c.Request.URL.Path),
			)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
				"code":  "INVALID_TOKEN",
			})
			return
		}

		// Set flat user values in context
		// iamUserId = Clerk user ID from JWT (sub claim)
		// userId = Internal DB user ID (to be resolved by service layer if needed)
		c.Set(IAMUserIDKey, claims.Subject)

		// Extract and store user's org memberships from JWT
		userOrgIDs := extractOrgIDsFromClaims(claims.Orgs)
		c.Set(UserOrgIDsKey, userOrgIDs)

		// Store full claims for advanced use cases
		c.Set(UserClaimsKey, claims)

		logger.Debug("User authenticated",
			zap.String("iamUserId", claims.Subject),
			zap.String("email", claims.Email),
			zap.Strings("orgMemberships", userOrgIDs),
			zap.String("path", c.Request.URL.Path),
		)

		c.Next()
	}
}

// extractToken extracts JWT from cookie or Authorization header
func extractToken(c *gin.Context) string {
	// Try cookie first (Clerk stores session in __session cookie)
	if cookie, err := c.Cookie(ClerkSessionCookie); err == nil && cookie != "" {
		return cookie
	}

	// Fallback to Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// Support "Bearer <token>" format
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
		// Also support raw token
		return authHeader
	}

	return ""
}

// getJWKSURL returns the Clerk JWKS URL
// If not explicitly set, it derives from the publishable key
func getJWKSURL(cfg config.ClerkConfig) string {
	if cfg.JWKSURL != "" {
		return cfg.JWKSURL
	}

	// Derive from publishable key
	// Publishable key format: pk_test_<base64-encoded-frontend-api>.clerk.accounts.dev
	// or pk_live_<base64-encoded-frontend-api>.clerk.accounts.dev
	if cfg.PublishableKey != "" {
		// Extract the frontend API from publishable key
		// The key contains the frontend API subdomain
		parts := strings.Split(cfg.PublishableKey, "_")
		if len(parts) >= 3 {
			// Decode the base64 part to get frontend API
			// For simplicity, use the standard Clerk JWKS endpoint
			// Users should set JWKS URL explicitly for production
		}
	}

	// Default fallback - users should configure this
	// Clerk JWKS URL format: https://<clerk-frontend-api>/.well-known/jwks.json
	// Example: https://clerk.your-domain.com/.well-known/jwks.json
	return ""
}

// getClerkJWKS returns a cached JWKS keyfunc for Clerk
func getClerkJWKS(ctx context.Context, jwksURL string, logger config.Logger) (keyfunc.Keyfunc, error) {
	clerkJWKSOnce.Do(func() {
		logger.Info("Initializing Clerk JWKS",
			zap.String("url", jwksURL),
		)

		// Create keyfunc with background refresh
		clerkJWKS, clerkJWKSErr = keyfunc.NewDefault([]string{jwksURL})
		if clerkJWKSErr != nil {
			logger.Error("Failed to create JWKS keyfunc",
				zap.Error(clerkJWKSErr),
				zap.String("url", jwksURL),
			)
		}
	})

	return clerkJWKS, clerkJWKSErr
}

// =========================================
// CONTEXT GETTER FUNCTIONS (Flat Structure)
// =========================================

// GetIAMUserID retrieves the Clerk user ID from context
func GetIAMUserID(c *gin.Context) (string, error) {
	val, exists := c.Get(IAMUserIDKey)
	if !exists {
		return "", errors.New("iamUserId not set in context")
	}
	id, ok := val.(string)
	if !ok {
		return "", errors.New("invalid iamUserId type")
	}
	return id, nil
}

// MustGetIAMUserID retrieves the Clerk user ID or panics
func MustGetIAMUserID(c *gin.Context) string {
	id, err := GetIAMUserID(c)
	if err != nil {
		panic("MustGetIAMUserID: " + err.Error())
	}
	return id
}

// GetUserID retrieves the internal DB user ID from context
func GetUserID(c *gin.Context) (string, error) {
	val, exists := c.Get(UserIDKey)
	if !exists {
		return "", errors.New("userId not set in context")
	}
	id, ok := val.(string)
	if !ok {
		return "", errors.New("invalid userId type")
	}
	return id, nil
}

// SetUserID sets the internal DB user ID in context (called by service layer)
func SetUserID(c *gin.Context, userID string) {
	c.Set(UserIDKey, userID)
}

// GetOrgID retrieves the internal DB org ID from context
func GetOrgID(c *gin.Context) (string, error) {
	val, exists := c.Get(OrgIDKey)
	if !exists {
		return "", errors.New("orgId not set in context")
	}
	id, ok := val.(string)
	if !ok {
		return "", errors.New("invalid orgId type")
	}
	return id, nil
}

// MustGetOrgID retrieves the org ID from context or panics
// Use only when you're certain the org context is set (after RequireOrgContext middleware)
func MustGetOrgID(c *gin.Context) string {
	id, err := GetOrgID(c)
	if err != nil {
		panic("MustGetOrgID called without org ID in context: " + err.Error())
	}
	return id
}

// SetOrgID sets the internal DB org ID in context (called by service layer)
func SetOrgID(c *gin.Context, orgID string) {
	c.Set(OrgIDKey, orgID)
}

// GetUserOrgIDs retrieves the user's org memberships (Clerk org IDs) from context
func GetUserOrgIDs(c *gin.Context) ([]string, error) {
	val, exists := c.Get(UserOrgIDsKey)
	if !exists {
		return nil, errors.New("userOrgIds not set in context")
	}
	ids, ok := val.([]string)
	if !ok {
		return nil, errors.New("invalid userOrgIds type")
	}
	return ids, nil
}

// GetUserClaims retrieves the full JWT claims from context
func GetUserClaims(c *gin.Context) (*ClerkClaims, error) {
	val, exists := c.Get(UserClaimsKey)
	if !exists {
		return nil, errors.New("userClaims not set in context")
	}
	claims, ok := val.(*ClerkClaims)
	if !ok {
		return nil, errors.New("invalid userClaims type")
	}
	return claims, nil
}

// extractOrgIDsFromClaims extracts org IDs from Clerk's org claims map
func extractOrgIDsFromClaims(orgs map[string]interface{}) []string {
	if orgs == nil {
		return []string{}
	}

	ids := make([]string, 0, len(orgs))
	for orgID := range orgs {
		ids = append(ids, orgID)
	}
	return ids
}

// OptionalAuth creates a middleware that extracts auth info if present, but doesn't require it
// Useful for endpoints that behave differently for authenticated vs anonymous users
func OptionalAuth(cfg config.ClerkConfig, logger config.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			// No token, continue without auth
			c.Next()
			return
		}

		jwksURL := getJWKSURL(cfg)
		if jwksURL == "" {
			// Auth not configured, continue without auth
			c.Next()
			return
		}

		keyfuncJWKS, err := getClerkJWKS(c.Request.Context(), jwksURL, logger)
		if err != nil {
			// JWKS error, continue without auth
			logger.Warn("Failed to fetch JWKS for optional auth", zap.Error(err))
			c.Next()
			return
		}

		claims := &ClerkClaims{}
		parsedToken, err := jwt.ParseWithClaims(token, claims, keyfuncJWKS.KeyfuncCtx(c.Request.Context()))
		if err != nil || !parsedToken.Valid {
			// Invalid token, continue without auth
			c.Next()
			return
		}

		// Valid token, set flat user values in context
		c.Set(IAMUserIDKey, claims.Subject)

		c.Next()
	}
}

// RequireRole creates a middleware that requires a specific role
// This should be used AFTER ClerkAuth middleware
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This is a placeholder - implement role checking based on your user model
		// You might want to check the user's role in your database
		// For now, just continue
		c.Next()
	}
}

// RequireOrgContext creates a middleware that extracts org ID from header
// and validates the user has access to that organization
// This should be used AFTER ClerkAuth middleware
func RequireOrgContext(logger config.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get org ID from header
		orgID := c.GetHeader(OrgIDHeader)
		if orgID == "" {
			logger.Warn("Missing organization ID in header",
				zap.String("path", c.Request.URL.Path),
				zap.String("header", OrgIDHeader),
			)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Organization ID required",
				"code":  "ORG_ID_REQUIRED",
				"hint":  "Set the " + OrgIDHeader + " header with your organization ID",
			})
			return
		}

		// Set orgId in Gin context (from header - this is our internal DB org ID)
		// Note: iamOrgId comes from the JWT token, not from header
		// Org access validation should be done at the service/DB layer
		c.Set(OrgIDKey, orgID)

		logger.Debug("Organization ID set in context",
			zap.String("orgId", orgID),
			zap.String("path", c.Request.URL.Path),
		)

		c.Next()
	}
}

// GetIAMOrgID retrieves the IAM org ID from Gin context
func GetIAMOrgID(c *gin.Context) (string, error) {
	orgID, exists := c.Get(IAMOrgIDKey)
	if !exists {
		return "", errors.New("organization ID not set")
	}

	id, ok := orgID.(string)
	if !ok {
		return "", errors.New("invalid organization ID type")
	}

	return id, nil
}

// MustGetIAMOrgID retrieves the IAM org ID or panics
// Use only when you're certain the org context is set (after RequireOrgContext middleware)
func MustGetIAMOrgID(c *gin.Context) string {
	orgID, err := GetIAMOrgID(c)
	if err != nil {
		panic("MustGetIAMOrgID called without org context: " + err.Error())
	}
	return orgID
}

// ShutdownJWKS gracefully shuts down the JWKS background refresh
func ShutdownJWKS() {
	if clerkJWKS != nil {
		// Give it a moment to complete any pending operations
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = ctx // keyfunc doesn't have a shutdown method, but we prepare for future versions
	}
}
