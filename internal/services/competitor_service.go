package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/llm"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
	"github.com/google/uuid"
)

// CompetitorService handles competitor discovery and insights
type CompetitorService struct {
	db          db.Database
	logoService *LogoService
	llmRegistry *llm.Registry
	scraper     *WebScraperService
}

// NewCompetitorService creates a new competitor service
func NewCompetitorService(database db.Database) *CompetitorService {
	return &CompetitorService{
		db:          database,
		logoService: NewLogoService(database),
		scraper:     NewWebScraperService(),
	}
}

// NewCompetitorServiceWithLLM creates a new competitor service with LLM support
func NewCompetitorServiceWithLLM(database db.Database, llmRegistry *llm.Registry) *CompetitorService {
	return &CompetitorService{
		db:          database,
		logoService: NewLogoService(database),
		llmRegistry: llmRegistry,
		scraper:     NewWebScraperService(),
	}
}

// DiscoverCompetitors discovers competitors from response data
func (s *CompetitorService) DiscoverCompetitors(
	ctx context.Context,
	brand string,
	startTime, endTime *time.Time,
	limit int,
) (*models.DiscoverCompetitorsResponse, error) {
	if limit <= 0 {
		limit = 20
	}

	// Fetch all responses for the brand
	filter := shared.ResponseFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}

	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch responses: %w", err)
	}

	// Filter responses for the main brand
	var brandResponses []*models.Response
	for _, resp := range allResponses {
		if strings.EqualFold(resp.Brand, brand) {
			brandResponses = append(brandResponses, resp)
		}
	}

	// Count competitor mentions
	competitorCounts := make(map[string]int)
	for _, resp := range brandResponses {
		for _, comp := range resp.CompetitorsMention {
			normalized := strings.TrimSpace(comp)
			if normalized != "" && !strings.EqualFold(normalized, brand) {
				competitorCounts[normalized]++
			}
		}
	}

	// Sort by mention count
	type competitorWithCount struct {
		name  string
		count int
	}

	var sortedCompetitors []competitorWithCount
	for name, count := range competitorCounts {
		sortedCompetitors = append(sortedCompetitors, competitorWithCount{name, count})
	}

	sort.Slice(sortedCompetitors, func(i, j int) bool {
		return sortedCompetitors[i].count > sortedCompetitors[j].count
	})

	// Limit results
	if len(sortedCompetitors) > limit {
		sortedCompetitors = sortedCompetitors[:limit]
	}

	// Build competitor list with logos
	competitors := make([]models.Competitor, 0, len(sortedCompetitors))
	for _, c := range sortedCompetitors {
		logo := s.logoService.GetBrandLogo(ctx, c.name, "")
		competitors = append(competitors, models.Competitor{
			Name:            c.name,
			Domain:          s.logoService.extractDomain(c.name, ""),
			Website:         "https://" + s.logoService.extractDomain(c.name, ""),
			LogoURL:         logo.LogoURL,
			FallbackLogoURL: logo.FallbackLogoURL,
			MentionCount:    c.count,
		})
	}

	// Get brand logo
	brandLogo := s.logoService.GetBrandLogo(ctx, brand, "")

	return &models.DiscoverCompetitorsResponse{
		Brand:           brand,
		LogoURL:         brandLogo.LogoURL,
		FallbackLogoURL: brandLogo.FallbackLogoURL,
		Competitors:     competitors,
		TotalDiscovered: len(competitors),
		DiscoveredFrom:  len(brandResponses),
		AnalyzedAt:      time.Now(),
	}, nil
}

// SuggestCompetitors uses LLM to automatically suggest competitors based on brand info
func (s *CompetitorService) SuggestCompetitors(
	ctx context.Context,
	req *models.SuggestCompetitorsRequest,
) (*models.SuggestCompetitorsResponse, error) {
	if s.llmRegistry == nil {
		return nil, fmt.Errorf("LLM registry not configured for competitor suggestions")
	}

	brand := req.Brand
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	// Get brand logo first
	brandLogo := s.logoService.GetBrandLogo(ctx, brand, req.Website)

	// Scrape website for additional context if provided
	var websiteContext string
	industry := req.Industry
	if req.Website != "" {
		content, err := s.scraper.ScrapeWebsite(ctx, req.Website)
		if err == nil && content != nil {
			websiteContext = s.buildWebsiteContext(content)
			// If industry not provided, try to infer from website
			if industry == "" && content.Description != "" {
				industry = s.inferIndustryFromContent(content)
			}
		}
	}

	// Get an LLM provider for suggestion
	provider, err := s.getLLMProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM provider: %w", err)
	}

	// Build the prompt for competitor suggestion
	prompt := s.buildCompetitorSuggestionPrompt(brand, req.Website, industry, req.Country, websiteContext, limit)

	// Call LLM
	response, err := provider.Generate(ctx, prompt, llm.Config{
		Temperature: 0.3, // Low temperature for consistent results
		MaxTokens:   2048,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse LLM response
	suggestedCompetitors, parsedIndustry := s.parseCompetitorSuggestions(response.Text)

	// If we got industry from LLM and don't have one, use it
	if industry == "" && parsedIndustry != "" {
		industry = parsedIndustry
	}

	// Enrich competitors with logos
	enrichedCompetitors := make([]models.SuggestedCompetitor, 0, len(suggestedCompetitors))
	for _, comp := range suggestedCompetitors {
		if len(enrichedCompetitors) >= limit {
			break
		}

		// Get logo for competitor
		logo := s.logoService.GetBrandLogo(ctx, comp.Name, comp.Website)

		enrichedCompetitors = append(enrichedCompetitors, models.SuggestedCompetitor{
			Name:            comp.Name,
			Website:         comp.Website,
			Domain:          s.logoService.extractDomain(comp.Name, comp.Website),
			LogoURL:         logo.LogoURL,
			FallbackLogoURL: logo.FallbackLogoURL,
			Description:     comp.Description,
			Reason:          comp.Reason,
			Relevance:       comp.Relevance,
			MarketPosition:  comp.MarketPosition,
		})
	}

	return &models.SuggestCompetitorsResponse{
		Brand:           brand,
		LogoURL:         brandLogo.LogoURL,
		FallbackLogoURL: brandLogo.FallbackLogoURL,
		Industry:        industry,
		Competitors:     enrichedCompetitors,
		TotalSuggested:  len(enrichedCompetitors),
		Source:          "llm",
		SuggestedAt:     time.Now(),
	}, nil
}

// getLLMProvider returns the best available LLM provider
func (s *CompetitorService) getLLMProvider() (llm.Provider, error) {
	if s.llmRegistry == nil {
		return nil, fmt.Errorf("LLM registry not configured")
	}

	// Prefer Google for latest information
	if provider, ok := s.llmRegistry.Get("google"); ok {
		return provider, nil
	}

	// Fallback to OpenAI
	if provider, ok := s.llmRegistry.Get("openai"); ok {
		return provider, nil
	}

	// Try any available provider
	providers := s.llmRegistry.List()
	if len(providers) == 0 {
		return nil, fmt.Errorf("no LLM providers available")
	}

	provider, _ := s.llmRegistry.Get(providers[0])
	return provider, nil
}

// buildWebsiteContext builds context from scraped website content
func (s *CompetitorService) buildWebsiteContext(content *WebsiteContent) string {
	var parts []string

	if content.Title != "" {
		parts = append(parts, fmt.Sprintf("Website Title: %s", content.Title))
	}
	if content.Description != "" {
		parts = append(parts, fmt.Sprintf("Description: %s", content.Description))
	}
	if len(content.Keywords) > 0 {
		parts = append(parts, fmt.Sprintf("Keywords: %s", strings.Join(content.Keywords, ", ")))
	}
	if content.MainContent != "" {
		mainContent := content.MainContent
		if len(mainContent) > 500 {
			mainContent = mainContent[:500] + "..."
		}
		parts = append(parts, fmt.Sprintf("Content Summary: %s", mainContent))
	}

	return strings.Join(parts, "\n")
}

// inferIndustryFromContent tries to infer industry from website content
func (s *CompetitorService) inferIndustryFromContent(content *WebsiteContent) string {
	text := strings.ToLower(content.Title + " " + content.Description)

	industryKeywords := map[string][]string{
		"Payment Processing":   {"payment", "fintech", "transaction", "checkout", "billing"},
		"E-commerce":           {"ecommerce", "e-commerce", "online store", "shopping", "retail"},
		"SaaS":                  {"saas", "software", "platform", "cloud", "subscription"},
		"Healthcare":           {"health", "medical", "patient", "clinic", "hospital"},
		"Education":            {"education", "learning", "course", "university", "school"},
		"Finance":              {"finance", "banking", "investment", "trading", "insurance"},
		"Marketing":            {"marketing", "advertising", "seo", "analytics", "campaign"},
		"Technology":           {"technology", "tech", "software", "digital", "innovation"},
		"Food & Delivery":      {"food", "restaurant", "delivery", "order", "dining"},
		"Travel":               {"travel", "booking", "hotel", "flight", "vacation"},
		"Real Estate":          {"real estate", "property", "home", "rental", "mortgage"},
		"Cybersecurity":        {"security", "cyber", "protection", "privacy", "encryption"},
		"AI & Machine Learning": {"ai", "artificial intelligence", "machine learning", "ml", "deep learning"},
	}

	for industry, keywords := range industryKeywords {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				return industry
			}
		}
	}

	return ""
}

// buildCompetitorSuggestionPrompt builds the prompt for LLM
func (s *CompetitorService) buildCompetitorSuggestionPrompt(brand, website, industry, country, websiteContext string, limit int) string {
	var contextParts []string
	contextParts = append(contextParts, fmt.Sprintf("Brand: %s", brand))

	if website != "" {
		contextParts = append(contextParts, fmt.Sprintf("Website: %s", website))
	}
	if industry != "" {
		contextParts = append(contextParts, fmt.Sprintf("Industry: %s", industry))
	}
	if country != "" {
		contextParts = append(contextParts, fmt.Sprintf("Target Market: %s", country))
	}
	if websiteContext != "" {
		contextParts = append(contextParts, fmt.Sprintf("\nWebsite Information:\n%s", websiteContext))
	}

	brandContext := strings.Join(contextParts, "\n")

	return fmt.Sprintf(`Analyze this brand and suggest their top %d competitors.

%s

TASK: Identify the main competitors of "%s" based on the information provided.

IMPORTANT REQUIREMENTS:
1. Suggest REAL, EXISTING companies that compete directly with this brand
2. Include a mix of:
   - Direct competitors (same product/service)
   - Indirect competitors (alternative solutions)
   - Emerging competitors (growing players)
3. For each competitor, provide their ACTUAL website domain
4. Provide concise but specific descriptions

Respond in VALID JSON format with this EXACT structure:
{
  "industry": "<detected industry>",
  "competitors": [
    {
      "name": "<company name>",
      "website": "<actual website URL like https://example.com>",
      "description": "<brief description of what they do>",
      "reason": "<why they compete with the brand>",
      "relevance": "<direct|indirect|emerging>",
      "marketPosition": "<leader|challenger|niche>"
    }
  ]
}

CRITICAL: 
- Use REAL company websites (not made-up domains)
- Include major players that anyone in this industry would know
- JSON must be valid and parseable
- Suggest exactly %d competitors

Respond with ONLY the JSON, no explanations or markdown.`, limit, brandContext, brand, limit)
}

// competitorSuggestionResult represents the LLM response structure
type competitorSuggestionResult struct {
	Industry    string `json:"industry"`
	Competitors []struct {
		Name           string `json:"name"`
		Website        string `json:"website"`
		Description    string `json:"description"`
		Reason         string `json:"reason"`
		Relevance      string `json:"relevance"`
		MarketPosition string `json:"marketPosition"`
	} `json:"competitors"`
}

// parseCompetitorSuggestions parses LLM response to extract competitor suggestions
func (s *CompetitorService) parseCompetitorSuggestions(response string) ([]models.SuggestedCompetitor, string) {
	// Clean up response - remove markdown code blocks if present
	cleaned := strings.TrimSpace(response)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var result competitorSuggestionResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		fmt.Printf("Warning: failed to parse competitor suggestions JSON: %v\n", err)
		// Try to extract competitors manually if JSON parsing fails
		return s.parseCompetitorsManually(response), ""
	}

	competitors := make([]models.SuggestedCompetitor, 0, len(result.Competitors))
	for _, c := range result.Competitors {
		// Clean up website URL
		website := c.Website
		if !strings.HasPrefix(website, "http") && website != "" {
			website = "https://" + website
		}

		competitors = append(competitors, models.SuggestedCompetitor{
			Name:           c.Name,
			Website:        website,
			Description:    c.Description,
			Reason:         c.Reason,
			Relevance:      c.Relevance,
			MarketPosition: c.MarketPosition,
		})
	}

	return competitors, result.Industry
}

// parseCompetitorsManually tries to extract competitors from non-JSON response
func (s *CompetitorService) parseCompetitorsManually(response string) []models.SuggestedCompetitor {
	// Fallback parsing - look for company names and websites
	competitors := make([]models.SuggestedCompetitor, 0)

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines
		if line == "" || strings.HasPrefix(line, "{") || strings.HasPrefix(line, "}") {
			continue
		}

		// Try to find company name mentions
		// This is a very basic fallback
		if strings.Contains(line, "competitor") || strings.Contains(line, "Company") {
			// Extract potential company names - this is limited but better than nothing
			continue
		}
	}

	return competitors
}

// ListCompetitors returns the list of competitors for a brand
func (s *CompetitorService) ListCompetitors(
	ctx context.Context,
	brand string,
	includeCustom bool,
	limit int,
) (*models.ListCompetitorsResponse, error) {
	if limit <= 0 {
		limit = 50
	}

	// First, try to get saved competitor list
	savedList, err := s.getSavedCompetitorList(ctx, brand)

	var competitors []models.Competitor
	var customCount int

	if err == nil && savedList != nil {
		// Use saved list
		competitors = savedList.Competitors
		if includeCustom {
			competitors = append(competitors, savedList.CustomAdded...)
			customCount = len(savedList.CustomAdded)
		}
	} else {
		// Discover competitors from data
		discovered, err := s.DiscoverCompetitors(ctx, brand, nil, nil, limit)
		if err != nil {
			return nil, err
		}
		competitors = discovered.Competitors
	}

	// Limit results
	if len(competitors) > limit {
		competitors = competitors[:limit]
	}

	// Get brand logo
	brandLogo := s.logoService.GetBrandLogo(ctx, brand, "")

	return &models.ListCompetitorsResponse{
		Brand:             brand,
		LogoURL:           brandLogo.LogoURL,
		FallbackLogoURL:   brandLogo.FallbackLogoURL,
		Competitors:       competitors,
		TotalDiscovered:   len(competitors),
		CustomCompetitors: customCount,
	}, nil
}

// AddCustomCompetitors adds user-defined competitors to the list
func (s *CompetitorService) AddCustomCompetitors(
	ctx context.Context,
	brand string,
	customCompetitors []models.CustomCompetitor,
) (*models.AddCustomCompetitorsResponse, error) {
	// Get or create saved competitor list
	savedList, err := s.getSavedCompetitorList(ctx, brand)
	if err != nil || savedList == nil {
		savedList = &models.SavedCompetitorList{
			ID:        uuid.New().String(),
			Brand:     brand,
			CreatedAt: time.Now(),
		}
	}

	// Add custom competitors
	addedCompetitors := make([]models.Competitor, 0, len(customCompetitors))
	for _, cc := range customCompetitors {
		logo := s.logoService.GetBrandLogo(ctx, cc.Name, cc.Website)
		domain := s.logoService.extractDomain(cc.Name, cc.Website)

		website := cc.Website
		if website == "" && domain != "" {
			website = "https://" + domain
		}

		competitor := models.Competitor{
			Name:            cc.Name,
			Website:         website,
			Domain:          domain,
			LogoURL:         logo.LogoURL,
			FallbackLogoURL: logo.FallbackLogoURL,
			Description:     cc.Description,
			Industry:        cc.Industry,
			IsCustom:        true,
		}
		addedCompetitors = append(addedCompetitors, competitor)
		savedList.CustomAdded = append(savedList.CustomAdded, competitor)
	}

	savedList.UpdatedAt = time.Now()

	// Save the updated list
	if err := s.saveCompetitorList(ctx, savedList); err != nil {
		return nil, fmt.Errorf("failed to save competitor list: %w", err)
	}

	return &models.AddCustomCompetitorsResponse{
		Brand:            brand,
		AddedCompetitors: addedCompetitors,
		TotalCompetitors: len(savedList.Competitors) + len(savedList.CustomAdded),
	}, nil
}

// GetCompetitorInsights returns comprehensive competitor insights
func (s *CompetitorService) GetCompetitorInsights(
	ctx context.Context,
	req *models.CompetitorInsightsRequest,
) (*models.CompetitorInsightsResponse, error) {
	brand := req.Brand

	// Fetch all responses for the brand
	filter := shared.ResponseFilter{
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Limit:     10000,
	}

	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch responses: %w", err)
	}

	// Filter responses for the main brand
	var brandResponses []*models.Response
	for _, resp := range allResponses {
		if strings.EqualFold(resp.Brand, brand) {
			// Apply additional filters
			if len(req.PromptIDs) > 0 && !containsStr(req.PromptIDs, resp.PromptID) {
				continue
			}
			if len(req.LLMIDs) > 0 && !containsStr(req.LLMIDs, resp.LLMID) {
				continue
			}
			brandResponses = append(brandResponses, resp)
		}
	}

	if len(brandResponses) == 0 {
		return nil, fmt.Errorf("no responses found for brand %s", brand)
	}

	// Determine which competitors to analyze
	competitorsToAnalyze := s.determineCompetitors(ctx, brand, req, brandResponses)

	// Calculate your (main brand) metrics
	yourMetrics := s.calculateBrandMetrics(brandResponses, brand, true)

	// Calculate competitor metrics and insights
	competitorInsights := s.analyzeCompetitors(ctx, brandResponses, competitorsToAnalyze)

	// Generate head-to-head comparisons
	headToHead := s.generateHeadToHead(brandResponses, brand, competitorInsights, yourMetrics)

	// Determine market leader and rankings
	allBrands := append([]models.DetailedCompetitorInsight{{
		Competitor: models.Competitor{Name: brand},
		Metrics:    yourMetrics,
	}}, competitorInsights...)

	sort.Slice(allBrands, func(i, j int) bool {
		return allBrands[i].Metrics.MentionRate > allBrands[j].Metrics.MentionRate
	})

	marketLeader := allBrands[0].Competitor.Name
	yourRank := 1
	for i, b := range allBrands {
		if strings.EqualFold(b.Competitor.Name, brand) {
			yourRank = i + 1
			break
		}
	}

	// Generate recommendations and strategic insights
	recommendations := s.generateRecommendations(brand, yourMetrics, competitorInsights, headToHead)
	strategicInsights := s.generateStrategicInsights(brand, yourMetrics, competitorInsights)

	// Get brand logo
	brandLogo := s.logoService.GetBrandLogo(ctx, brand, "")

	// Format period
	period := formatPeriod(req.StartTime, req.EndTime)

	return &models.CompetitorInsightsResponse{
		Brand:             brand,
		LogoURL:           brandLogo.LogoURL,
		FallbackLogoURL:   brandLogo.FallbackLogoURL,
		Period:            period,
		YourMetrics:       yourMetrics,
		Competitors:       competitorInsights,
		HeadToHead:        headToHead,
		MarketLeader:      marketLeader,
		YourRank:          yourRank,
		TotalBrands:       len(allBrands),
		Recommendations:   recommendations,
		StrategicInsights: strategicInsights,
		AnalyzedAt:        time.Now(),
	}, nil
}

// determineCompetitors determines which competitors to analyze
func (s *CompetitorService) determineCompetitors(
	ctx context.Context,
	brand string,
	req *models.CompetitorInsightsRequest,
	responses []*models.Response,
) []models.Competitor {
	competitors := make([]models.Competitor, 0)
	seenCompetitors := make(map[string]bool)

	// Add explicitly requested competitors
	for _, name := range req.Competitors {
		normalized := strings.TrimSpace(name)
		if normalized != "" && !strings.EqualFold(normalized, brand) && !seenCompetitors[strings.ToLower(normalized)] {
			logo := s.logoService.GetBrandLogo(ctx, normalized, "")
			competitors = append(competitors, models.Competitor{
				Name:            normalized,
				LogoURL:         logo.LogoURL,
				FallbackLogoURL: logo.FallbackLogoURL,
				Domain:          s.logoService.extractDomain(normalized, ""),
				Website:         "https://" + s.logoService.extractDomain(normalized, ""),
			})
			seenCompetitors[strings.ToLower(normalized)] = true
		}
	}

	// Add custom competitors from request
	for _, cc := range req.CustomCompetitors {
		normalized := strings.TrimSpace(cc.Name)
		if normalized != "" && !strings.EqualFold(normalized, brand) && !seenCompetitors[strings.ToLower(normalized)] {
			logo := s.logoService.GetBrandLogo(ctx, normalized, cc.Website)
			domain := s.logoService.extractDomain(normalized, cc.Website)
			website := cc.Website
			if website == "" && domain != "" {
				website = "https://" + domain
			}
			competitors = append(competitors, models.Competitor{
				Name:            normalized,
				Website:         website,
				Domain:          domain,
				LogoURL:         logo.LogoURL,
				FallbackLogoURL: logo.FallbackLogoURL,
				Description:     cc.Description,
				Industry:        cc.Industry,
				IsCustom:        true,
			})
			seenCompetitors[strings.ToLower(normalized)] = true
		}
	}

	// If includeAll or no specific competitors, discover from data
	if req.IncludeAll || (len(req.Competitors) == 0 && len(req.CustomCompetitors) == 0) {
		// Discover from response data
		competitorCounts := make(map[string]int)
		for _, resp := range responses {
			for _, comp := range resp.CompetitorsMention {
				normalized := strings.TrimSpace(comp)
				if normalized != "" && !strings.EqualFold(normalized, brand) && !seenCompetitors[strings.ToLower(normalized)] {
					competitorCounts[normalized]++
				}
			}
		}

		// Sort by count and add top competitors
		type compCount struct {
			name  string
			count int
		}
		var sorted []compCount
		for name, count := range competitorCounts {
			sorted = append(sorted, compCount{name, count})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].count > sorted[j].count
		})

		// Add top 10 discovered competitors
		for i, cc := range sorted {
			if i >= 10 {
				break
			}
			logo := s.logoService.GetBrandLogo(ctx, cc.name, "")
			competitors = append(competitors, models.Competitor{
				Name:            cc.name,
				LogoURL:         logo.LogoURL,
				FallbackLogoURL: logo.FallbackLogoURL,
				Domain:          s.logoService.extractDomain(cc.name, ""),
				Website:         "https://" + s.logoService.extractDomain(cc.name, ""),
				MentionCount:    cc.count,
			})
		}
	}

	return competitors
}

// calculateBrandMetrics calculates metrics for a brand
func (s *CompetitorService) calculateBrandMetrics(
	responses []*models.Response,
	brand string,
	isMainBrand bool,
) models.CompetitorInsightMetrics {
	if len(responses) == 0 {
		return models.CompetitorInsightMetrics{}
	}

	var (
		mentionCount     int
		totalVisibility  float64
		totalPosition    float64
		positionCount    int
		topPositionCount int
		groundingCount   int
		sentimentSum     float64
		sentimentCount   int
		firstMentioned   time.Time
		lastMentioned    time.Time
	)

	for _, resp := range responses {
		mentioned := false

		if isMainBrand {
			// For main brand, use the analyzed fields
			if resp.BrandMentioned {
				mentioned = true
				mentionCount++
				totalVisibility += float64(resp.VisibilityScore)

				if resp.BrandPosition > 0 {
					totalPosition += float64(resp.BrandPosition)
					positionCount++
					if resp.BrandPosition <= 3 {
						topPositionCount++
					}
				}

				if resp.InGroundingSources {
					groundingCount++
				}

				if resp.Sentiment != "" {
					sentimentSum += calculateSentimentScore(resp.Sentiment)
					sentimentCount++
				}
			}
		} else {
			// For competitors, check if mentioned in competitors list
			for _, comp := range resp.CompetitorsMention {
				if strings.EqualFold(strings.TrimSpace(comp), brand) {
					mentioned = true
					mentionCount++
					break
				}
			}
		}

		if mentioned {
			if firstMentioned.IsZero() || resp.CreatedAt.Before(firstMentioned) {
				firstMentioned = resp.CreatedAt
			}
			if lastMentioned.IsZero() || resp.CreatedAt.After(lastMentioned) {
				lastMentioned = resp.CreatedAt
			}
		}
	}

	metrics := models.CompetitorInsightMetrics{
		ResponseCount: len(responses),
	}

	if len(responses) > 0 {
		metrics.MentionRate = float64(mentionCount) / float64(len(responses)) * 100
	}

	if mentionCount > 0 {
		metrics.AvgVisibility = totalVisibility / float64(mentionCount)
	}

	if positionCount > 0 {
		metrics.AvgPosition = totalPosition / float64(positionCount)
		metrics.TopPositionRate = float64(topPositionCount) / float64(positionCount) * 100
	}

	if mentionCount > 0 {
		metrics.GroundingRate = float64(groundingCount) / float64(mentionCount) * 100
	}

	if sentimentCount > 0 {
		metrics.SentimentScore = sentimentSum / float64(sentimentCount)
	}

	if !firstMentioned.IsZero() {
		metrics.FirstMentioned = firstMentioned.Format("2006-01-02")
	}
	if !lastMentioned.IsZero() {
		metrics.LastMentioned = lastMentioned.Format("2006-01-02")
	}

	return metrics
}

// analyzeCompetitors generates insights for each competitor
func (s *CompetitorService) analyzeCompetitors(
	ctx context.Context,
	responses []*models.Response,
	competitors []models.Competitor,
) []models.DetailedCompetitorInsight {
	insights := make([]models.DetailedCompetitorInsight, 0, len(competitors))

	// Calculate total mentions for market share
	totalMentions := 0
	competitorMentions := make(map[string]int)

	for _, resp := range responses {
		for _, comp := range resp.CompetitorsMention {
			normalized := strings.ToLower(strings.TrimSpace(comp))
			competitorMentions[normalized]++
			totalMentions++
		}
	}

	for _, comp := range competitors {
		metrics := s.calculateBrandMetrics(responses, comp.Name, false)
		metrics.MentionCount = competitorMentions[strings.ToLower(comp.Name)]

		// Calculate market share
		if totalMentions > 0 {
			metrics.MarketShare = float64(metrics.MentionCount) / float64(totalMentions) * 100
		}

		// Generate strengths and weaknesses
		strengths, weaknesses := s.generateStrengthsWeaknesses(metrics)

		// Calculate by LLM
		byLLM := s.calculateByLLM(responses, comp.Name)

		// Calculate by category
		byCategory := s.calculateByCategory(ctx, responses, comp.Name)

		insights = append(insights, models.DetailedCompetitorInsight{
			Competitor: comp,
			Metrics:    metrics,
			Strengths:  strengths,
			Weaknesses: weaknesses,
			ByLLM:      byLLM,
			ByCategory: byCategory,
		})
	}

	// Sort by mention rate
	sort.Slice(insights, func(i, j int) bool {
		return insights[i].Metrics.MentionRate > insights[j].Metrics.MentionRate
	})

	return insights
}

// calculateByLLM calculates visibility by LLM for a competitor
func (s *CompetitorService) calculateByLLM(responses []*models.Response, competitor string) map[string]float64 {
	llmMentions := make(map[string]int)
	llmTotal := make(map[string]int)

	for _, resp := range responses {
		llmTotal[resp.LLMName]++
		for _, comp := range resp.CompetitorsMention {
			if strings.EqualFold(strings.TrimSpace(comp), competitor) {
				llmMentions[resp.LLMName]++
				break
			}
		}
	}

	result := make(map[string]float64)
	for llm, total := range llmTotal {
		if total > 0 {
			result[llm] = float64(llmMentions[llm]) / float64(total) * 100
		}
	}

	return result
}

// calculateByCategory calculates visibility by category for a competitor
func (s *CompetitorService) calculateByCategory(ctx context.Context, responses []*models.Response, competitor string) map[string]float64 {
	categoryMentions := make(map[string]int)
	categoryTotal := make(map[string]int)

	for _, resp := range responses {
		// Get prompt category
		prompt, err := s.db.GetPrompt(ctx, resp.PromptID)
		category := "unknown"
		if err == nil && prompt != nil && prompt.Category != "" {
			category = prompt.Category
		}

		categoryTotal[category]++
		for _, comp := range resp.CompetitorsMention {
			if strings.EqualFold(strings.TrimSpace(comp), competitor) {
				categoryMentions[category]++
				break
			}
		}
	}

	result := make(map[string]float64)
	for cat, total := range categoryTotal {
		if total > 0 {
			result[cat] = float64(categoryMentions[cat]) / float64(total) * 100
		}
	}

	return result
}

// generateStrengthsWeaknesses generates strengths and weaknesses based on metrics
func (s *CompetitorService) generateStrengthsWeaknesses(metrics models.CompetitorInsightMetrics) ([]string, []string) {
	var strengths, weaknesses []string

	// High mention rate
	if metrics.MentionRate > 50 {
		strengths = append(strengths, "High visibility across AI responses")
	} else if metrics.MentionRate < 20 {
		weaknesses = append(weaknesses, "Low visibility in AI responses")
	}

	// Position
	if metrics.AvgPosition > 0 && metrics.AvgPosition <= 2 {
		strengths = append(strengths, "Consistently ranked in top positions")
	} else if metrics.AvgPosition > 5 {
		weaknesses = append(weaknesses, "Often ranked in lower positions")
	}

	// Top position rate
	if metrics.TopPositionRate > 60 {
		strengths = append(strengths, "Frequently appears in top 3 results")
	}

	// Sentiment
	if metrics.SentimentScore > 0.5 {
		strengths = append(strengths, "Positive sentiment in AI responses")
	} else if metrics.SentimentScore < -0.2 {
		weaknesses = append(weaknesses, "Negative sentiment detected")
	}

	// Grounding
	if metrics.GroundingRate > 50 {
		strengths = append(strengths, "Well-cited in source references")
	} else if metrics.GroundingRate < 20 && metrics.MentionRate > 30 {
		weaknesses = append(weaknesses, "Mentioned but rarely cited in sources")
	}

	return strengths, weaknesses
}

// generateHeadToHead generates head-to-head comparisons
func (s *CompetitorService) generateHeadToHead(
	responses []*models.Response,
	mainBrand string,
	competitorInsights []models.DetailedCompetitorInsight,
	yourMetrics models.CompetitorInsightMetrics,
) []models.CompetitorComparison {
	comparisons := make([]models.CompetitorComparison, 0, len(competitorInsights))

	for _, insight := range competitorInsights {
		comp := insight.Competitor
		theirMetrics := insight.Metrics

		// Calculate co-mention rate
		coMentionCount := 0
		yourWins := 0
		theirWins := 0

		for _, resp := range responses {
			yourMentioned := resp.BrandMentioned
			theirMentioned := false

			for _, mentioned := range resp.CompetitorsMention {
				if strings.EqualFold(strings.TrimSpace(mentioned), comp.Name) {
					theirMentioned = true
					break
				}
			}

			if yourMentioned && theirMentioned {
				coMentionCount++
			}

			// Determine winner based on visibility
			if yourMentioned && !theirMentioned {
				yourWins++
			} else if !yourMentioned && theirMentioned {
				theirWins++
			} else if yourMentioned && theirMentioned {
				// Both mentioned - compare position
				if resp.BrandPosition > 0 && resp.BrandPosition <= 3 {
					yourWins++
				} else {
					theirWins++
				}
			}
		}

		totalContests := yourWins + theirWins
		winRate := 0.0
		if totalContests > 0 {
			winRate = float64(yourWins) / float64(totalContests) * 100
		}

		coMentionRate := 0.0
		if len(responses) > 0 {
			coMentionRate = float64(coMentionCount) / float64(len(responses)) * 100
		}

		visibilityGap := yourMetrics.AvgVisibility - theirMetrics.AvgVisibility
		mentionGap := yourMetrics.MentionRate - theirMetrics.MentionRate

		status := "even"
		if mentionGap > 10 {
			status = "leading"
		} else if mentionGap < -10 {
			status = "trailing"
		}

		comparisons = append(comparisons, models.CompetitorComparison{
			Competitor:       comp,
			YourVisibility:   yourMetrics.AvgVisibility,
			TheirVisibility:  theirMetrics.AvgVisibility,
			VisibilityGap:    visibilityGap,
			YourMentionRate:  yourMetrics.MentionRate,
			TheirMentionRate: theirMetrics.MentionRate,
			MentionGap:       mentionGap,
			WinRate:          winRate,
			CoMentionRate:    coMentionRate,
			Status:           status,
		})
	}

	// Sort by mention gap (biggest threats first)
	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].MentionGap < comparisons[j].MentionGap
	})

	return comparisons
}

// generateRecommendations generates actionable recommendations
func (s *CompetitorService) generateRecommendations(
	brand string,
	yourMetrics models.CompetitorInsightMetrics,
	competitors []models.DetailedCompetitorInsight,
	headToHead []models.CompetitorComparison,
) []models.Recommendation {
	recommendations := make([]models.Recommendation, 0)

	// Find biggest threats (competitors beating you)
	for _, comparison := range headToHead {
		if comparison.Status == "trailing" && comparison.MentionGap < -20 {
			recommendations = append(recommendations, models.Recommendation{
				Type:        "competitive",
				Priority:    "high",
				Title:       fmt.Sprintf("Address %s competitive gap", comparison.Competitor.Name),
				Description: fmt.Sprintf("%s has %.1f%% higher visibility than you", comparison.Competitor.Name, -comparison.MentionGap),
				Action:      fmt.Sprintf("Analyze content strategy of %s and optimize your messaging", comparison.Competitor.Name),
				Impact:      "Could improve visibility by 20-30%",
			})
		}
	}

	// Low visibility recommendation
	if yourMetrics.MentionRate < 30 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "visibility",
			Priority:    "high",
			Title:       "Improve AI visibility",
			Description: fmt.Sprintf("Your current mention rate is only %.1f%%", yourMetrics.MentionRate),
			Action:      "Optimize content for AI crawlers, improve technical SEO, and create more comprehensive content",
			Impact:      "Could double your visibility in AI responses",
		})
	}

	// Position recommendation
	if yourMetrics.AvgPosition > 3 && yourMetrics.MentionRate > 20 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "position",
			Priority:    "medium",
			Title:       "Improve ranking position",
			Description: fmt.Sprintf("Your average position is %.1f (lower is better)", yourMetrics.AvgPosition),
			Action:      "Focus on authoritative content, build more backlinks, and establish thought leadership",
			Impact:      "Moving to top 3 positions increases CTR by 50%",
		})
	}

	// Sentiment recommendation
	if yourMetrics.SentimentScore < 0.3 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "sentiment",
			Priority:    "medium",
			Title:       "Improve brand sentiment",
			Description: "AI responses about your brand have neutral or negative sentiment",
			Action:      "Address common complaints, improve customer experience, and generate positive coverage",
			Impact:      "Better sentiment increases conversion rates",
		})
	}

	// Grounding recommendation
	if yourMetrics.GroundingRate < 30 && yourMetrics.MentionRate > 20 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "sources",
			Priority:    "medium",
			Title:       "Increase source citations",
			Description: fmt.Sprintf("Only %.1f%% of responses cite your sources", yourMetrics.GroundingRate),
			Action:      "Create more citable content, publish research, and build authoritative resources",
			Impact:      "Source citations increase trust and authority",
		})
	}

	return recommendations
}

// generateStrategicInsights generates strategic insights
func (s *CompetitorService) generateStrategicInsights(
	brand string,
	yourMetrics models.CompetitorInsightMetrics,
	competitors []models.DetailedCompetitorInsight,
) []models.StrategicInsight {
	insights := make([]models.StrategicInsight, 0)

	// Find opportunities (competitors with weaknesses)
	for _, comp := range competitors {
		if comp.Metrics.MentionRate > yourMetrics.MentionRate && len(comp.Weaknesses) > 0 {
			insights = append(insights, models.StrategicInsight{
				Type:        "opportunity",
				Priority:    "high",
				Title:       fmt.Sprintf("Exploit %s weakness", comp.Competitor.Name),
				Description: fmt.Sprintf("%s has weakness: %s", comp.Competitor.Name, comp.Weaknesses[0]),
				Competitor:  comp.Competitor.Name,
				Action:      "Create content addressing this gap",
				Impact:      "Could capture market share",
			})
		}
	}

	// Find threats
	for _, comp := range competitors {
		if comp.Metrics.MentionRate > yourMetrics.MentionRate*2 {
			insights = append(insights, models.StrategicInsight{
				Type:        "threat",
				Priority:    "high",
				Title:       fmt.Sprintf("%s dominates visibility", comp.Competitor.Name),
				Description: fmt.Sprintf("%s has %.1fx your visibility", comp.Competitor.Name, comp.Metrics.MentionRate/yourMetrics.MentionRate),
				Competitor:  comp.Competitor.Name,
				Action:      "Develop differentiated messaging strategy",
				Impact:      "Critical for market positioning",
			})
		}
	}

	// Identify your strengths
	if yourMetrics.TopPositionRate > 50 {
		insights = append(insights, models.StrategicInsight{
			Type:        "strength",
			Priority:    "medium",
			Title:       "Strong ranking performance",
			Description: fmt.Sprintf("You appear in top 3 positions %.1f%% of the time", yourMetrics.TopPositionRate),
			Action:      "Maintain and expand content strategy",
			Impact:      "Sustain competitive advantage",
		})
	}

	return insights
}

// Helper functions

func (s *CompetitorService) getSavedCompetitorList(ctx context.Context, brand string) (*models.SavedCompetitorList, error) {
	// This would retrieve from database - for now return nil
	// Could be implemented with a new database method
	return nil, nil
}

func (s *CompetitorService) saveCompetitorList(ctx context.Context, list *models.SavedCompetitorList) error {
	// This would save to database
	// Could be implemented with a new database method
	return nil
}

func formatPeriod(startTime, endTime *time.Time) string {
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()

	if startTime != nil {
		start = *startTime
	}
	if endTime != nil {
		end = *endTime
	}

	return start.Format("Jan 2, 2006") + " - " + end.Format("Jan 2, 2006")
}

func containsStr(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

