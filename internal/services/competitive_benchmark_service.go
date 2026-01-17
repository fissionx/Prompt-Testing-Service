package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// CompetitiveBenchmarkService provides competitive analysis
type CompetitiveBenchmarkService struct {
	db                    db.Database
	recommendationsEngine *RecommendationsEngine
	logoService           *LogoService
}

// NewCompetitiveBenchmarkService creates a new competitive benchmark service
func NewCompetitiveBenchmarkService(database db.Database) *CompetitiveBenchmarkService {
	return &CompetitiveBenchmarkService{
		db:                    database,
		recommendationsEngine: NewRecommendationsEngine(),
		logoService:           NewLogoService(database),
	}
}

// GetCompetitiveBenchmark performs competitive analysis
// Analyzes how your brand and competitors appear in the SAME responses/prompts
func (s *CompetitiveBenchmarkService) GetCompetitiveBenchmark(
	ctx context.Context,
	mainBrand string,
	competitors []string,
	promptIDs, llmIDs []string,
	startTime, endTime *time.Time,
	region string,
	competitorMap map[string]string, // name -> domain mapping
) (*models.CompetitiveBenchmarkResponse, error) {
	// Fetch responses - optimized: filter at database level instead of in-memory
	filter := shared.ResponseFilter{
		Brand:     mainBrand, // Filter by brand at database level
		PromptIDs: promptIDs, // Filter by prompt IDs at database level (if provided)
		LLMIDs:    llmIDs,    // Filter by LLM IDs at database level (if provided)
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}

	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch responses: %w", err)
	}

	// Filter by region (region filtering still done in-memory as it's not in ResponseFilter yet)
	var responses []*models.Response
	for _, resp := range allResponses {
		// Region filter: only apply if BOTH are specified
		// If response has no region, or request has no region filter, include it
		if region != "" && resp.Region != "" && !strings.EqualFold(resp.Region, region) {
			continue
		}

		responses = append(responses, resp)
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("no responses found for brand %s", mainBrand)
	}

	// If competitors not specified, check for saved competitors first
	if len(competitors) == 0 {
		// Try to get saved competitors
		savedCompetitors, err := s.db.GetBrandCompetitors(ctx, mainBrand)
		if err == nil && savedCompetitors != nil && len(savedCompetitors.Competitors) > 0 {
			// Use saved competitors - parse "name|domain" format
			parsedCompetitors, parsedMap := parseCompetitorStrings(savedCompetitors.Competitors)
			competitors = parsedCompetitors
			// Merge parsed domains into competitorMap
			if competitorMap == nil {
				competitorMap = make(map[string]string)
			}
			for name, domain := range parsedMap {
				competitorMap[name] = domain
			}
		} else {
			// Fall back to auto-detection from responses
			competitorSet := make(map[string]bool)
			for _, resp := range responses {
				for _, comp := range resp.CompetitorsMention {
					// Normalize competitor name
					normalized := strings.TrimSpace(comp)
					if normalized != "" && !strings.EqualFold(normalized, mainBrand) {
						competitorSet[normalized] = true
					}
				}
			}
			for comp := range competitorSet {
				competitors = append(competitors, comp)
			}
		}
	} else {
		// Parse competitors if they're in "name|domain" format
		parsedCompetitors, parsedMap := parseCompetitorStrings(competitors)
		competitors = parsedCompetitors
		// Merge parsed domains into competitorMap
		if competitorMap == nil {
			competitorMap = make(map[string]string)
		}
		for name, domain := range parsedMap {
			competitorMap[name] = domain
		}
	}

	// Collect all unique competitors mentioned across all responses
	allMentionedCompetitors := make(map[string]bool)
	for _, resp := range responses {
		for _, mentioned := range resp.CompetitorsMention {
			normalized := strings.TrimSpace(mentioned)
			if normalized != "" && !strings.EqualFold(normalized, mainBrand) {
				allMentionedCompetitors[normalized] = true
			}
		}
	}

	// Identify tracked brands (main brand + explicitly tracked competitors)
	trackedBrands := make(map[string]bool)
	trackedBrands[strings.ToLower(mainBrand)] = true
	for _, comp := range competitors {
		trackedBrands[strings.ToLower(comp)] = true
	}

	// Find untracked competitors (mentioned but not in tracked list)
	var untrackedCompetitors []string
	for mentioned := range allMentionedCompetitors {
		if !trackedBrands[strings.ToLower(mentioned)] {
			untrackedCompetitors = append(untrackedCompetitors, mentioned)
		}
	}

	// Analyze ALL brands mentioned across all responses
	// This gives us the real "share of voice" in AI responses
	allBrands := append([]string{mainBrand}, competitors...)
	brandStats := make(map[string]*brandMentionStats)

	for _, brand := range allBrands {
		brandStats[brand] = &brandMentionStats{
			brand:           brand,
			mentionCount:    0,
			totalVisibility: 0,
			totalPosition:   0,
			positionCount:   0,
			sentimentScores: []float64{},
		}
	}

	// Initialize stats for untracked competitors
	untrackedStats := make(map[string]*brandMentionStats)
	for _, brand := range untrackedCompetitors {
		untrackedStats[brand] = &brandMentionStats{
			brand:           brand,
			mentionCount:    0,
			totalVisibility: 0,
			totalPosition:   0,
			positionCount:   0,
			sentimentScores: []float64{},
		}
	}

	// Analyze each response
	for _, resp := range responses {
		// Main brand stats (from the actual analysis)
		if stats, ok := brandStats[mainBrand]; ok {
			if resp.BrandMentioned {
				stats.mentionCount++
				stats.totalVisibility += float64(resp.VisibilityScore)

				if resp.BrandPosition > 0 {
					stats.totalPosition += float64(resp.BrandPosition)
					stats.positionCount++
				}

				if resp.Sentiment != "" {
					stats.sentimentScores = append(stats.sentimentScores, calculateSentimentScore(resp.Sentiment))
				}
			}
		}

		// Competitor stats (from competitors_mention field)
		for _, mentioned := range resp.CompetitorsMention {
			normalized := strings.TrimSpace(mentioned)
			if normalized == "" {
				continue
			}

			// Check against tracked competitors (case-insensitive)
			found := false
			for compName, stats := range brandStats {
				if compName != mainBrand && strings.EqualFold(normalized, compName) {
					stats.mentionCount++
					found = true
					break
				}
			}

			// If not found in tracked competitors, add to untracked stats
			if !found && !strings.EqualFold(normalized, mainBrand) {
				if stats, ok := untrackedStats[normalized]; ok {
					stats.mentionCount++
				} else {
					// New untracked competitor found, initialize it
					untrackedStats[normalized] = &brandMentionStats{
						brand:           normalized,
						mentionCount:    1,
						totalVisibility: 0,
						totalPosition:   0,
						positionCount:   0,
						sentimentScores: []float64{},
					}
				}
			}
		}
	}

	// Build performance objects for all brands
	var allPerformances []models.BrandPerformance
	totalMentions := 0

	// Calculate total mentions including tracked brands
	for _, brand := range allBrands {
		stats := brandStats[brand]
		totalMentions += stats.mentionCount
	}

	// Add untracked competitors to total mentions for accurate share of voice
	for _, stats := range untrackedStats {
		totalMentions += stats.mentionCount
	}

	// Get logo URLs for all brands
	logoRequests := make([]BrandLogoRequest, 0, len(allBrands))
	for _, brand := range allBrands {
		logoRequests = append(logoRequests, BrandLogoRequest{
			Name:    brand,
			Website: "", // Service will infer from name or use cached data
		})
	}
	brandLogos := s.logoService.GetMultipleLogos(ctx, logoRequests)
	logoMap := make(map[string]models.BrandWithLogo)
	for _, logo := range brandLogos {
		logoMap[logo.Brand] = logo
	}

	for _, brand := range allBrands {
		stats := brandStats[brand]
		logo := logoMap[brand]

		// Get domain from map or derive it
		domain := ""
		if competitorMap != nil {
			if d, ok := competitorMap[brand]; ok {
				domain = d
			}
		}
		if domain == "" {
			domain = deriveCompetitorDomainFromName(brand)
		}

		perf := models.BrandPerformance{
			Brand:           brand,
			Domain:          domain,
			LogoURL:         logo.LogoURL,
			FallbackLogoURL: logo.FallbackLogoURL,
			ResponseCount:   stats.mentionCount,
		}

		// Calculate rates
		if len(responses) > 0 {
			perf.MentionRate = float64(stats.mentionCount) / float64(len(responses)) * 100
		}

		// Market share (share of total mentions)
		if totalMentions > 0 {
			perf.MarketSharePct = float64(stats.mentionCount) / float64(totalMentions) * 100
		}

		// Main brand gets additional metrics from actual analysis
		if brand == mainBrand && stats.mentionCount > 0 {
			perf.Visibility = stats.totalVisibility / float64(stats.mentionCount)

			if stats.positionCount > 0 {
				perf.AveragePosition = stats.totalPosition / float64(stats.positionCount)
			}

			if len(stats.sentimentScores) > 0 {
				sum := 0.0
				for _, s := range stats.sentimentScores {
					sum += s
				}
				perf.SentimentScore = sum / float64(len(stats.sentimentScores))
			}
		}

		allPerformances = append(allPerformances, perf)
	}

	// Sort by mention count to determine rankings
	sort.Slice(allPerformances, func(i, j int) bool {
		if allPerformances[i].ResponseCount != allPerformances[j].ResponseCount {
			return allPerformances[i].ResponseCount > allPerformances[j].ResponseCount
		}
		return allPerformances[i].MentionRate > allPerformances[j].MentionRate
	})

	// Find main brand and competitors
	var mainBrandPerf models.BrandPerformance
	var competitorPerfs []models.BrandPerformance
	mainBrandRank := 1

	for i, perf := range allPerformances {
		if perf.Brand == mainBrand {
			mainBrandPerf = perf
			mainBrandRank = i + 1
		} else {
			competitorPerfs = append(competitorPerfs, perf)
		}
	}

	// Market leader
	marketLeader := allPerformances[0].Brand

	// Calculate metrics for untracked competitors
	var otherBrandPerfs []models.BrandPerformance
	if len(untrackedStats) > 0 {
		// Get logo URLs for untracked competitors
		otherLogoRequests := make([]BrandLogoRequest, 0, len(untrackedStats))
		for brand := range untrackedStats {
			otherLogoRequests = append(otherLogoRequests, BrandLogoRequest{
				Name:    brand,
				Website: "",
			})
		}
		otherBrandLogos := s.logoService.GetMultipleLogos(ctx, otherLogoRequests)
		otherLogoMap := make(map[string]models.BrandWithLogo)
		for _, logo := range otherBrandLogos {
			otherLogoMap[logo.Brand] = logo
		}

		// Build performance objects for untracked competitors
		for brand, stats := range untrackedStats {
			if stats.mentionCount == 0 {
				continue // Skip if not actually mentioned
			}

			logo := otherLogoMap[brand]
			domain := deriveCompetitorDomainFromName(brand)

			perf := models.BrandPerformance{
				Brand:           brand,
				Domain:          domain,
				LogoURL:         logo.LogoURL,
				FallbackLogoURL: logo.FallbackLogoURL,
				ResponseCount:   stats.mentionCount,
			}

			// Calculate rates
			if len(responses) > 0 {
				perf.MentionRate = float64(stats.mentionCount) / float64(len(responses)) * 100
			}

			// Market share (share of total mentions - already includes untracked in totalMentions)
			if totalMentions > 0 {
				perf.MarketSharePct = float64(stats.mentionCount) / float64(totalMentions) * 100
			}

			otherBrandPerfs = append(otherBrandPerfs, perf)
		}

		// Sort untracked competitors by mention count
		sort.Slice(otherBrandPerfs, func(i, j int) bool {
			if otherBrandPerfs[i].ResponseCount != otherBrandPerfs[j].ResponseCount {
				return otherBrandPerfs[i].ResponseCount > otherBrandPerfs[j].ResponseCount
			}
			return otherBrandPerfs[i].MentionRate > otherBrandPerfs[j].MentionRate
		})
	}

	// Generate prompt-level breakdown (pass untracked competitors for share of voice calculation)
	promptBreakdown := s.generatePromptBreakdown(responses, mainBrand, competitors, untrackedStats)

	return &models.CompetitiveBenchmarkResponse{
		MainBrand:       mainBrandPerf,
		Competitors:     competitorPerfs,
		OtherBrands:     otherBrandPerfs,
		MarketLeader:    marketLeader,
		YourRank:        mainBrandRank,
		TotalBrands:     len(allPerformances) + len(otherBrandPerfs),
		PromptBreakdown: promptBreakdown,
		AnalyzedAt:      time.Now(),
	}, nil
}

// generatePromptBreakdown creates per-prompt competitive analysis
func (s *CompetitiveBenchmarkService) generatePromptBreakdown(
	responses []*models.Response,
	mainBrand string,
	competitors []string,
	untrackedStats map[string]*brandMentionStats,
) []models.PromptCompetitiveAnalysis {
	var breakdown []models.PromptCompetitiveAnalysis

	for _, resp := range responses {
		// Main brand result
		mainResult := models.PromptBrandResult{
			Mentioned:       resp.BrandMentioned,
			VisibilityScore: resp.VisibilityScore,
			Position:        resp.BrandPosition,
			Sentiment:       resp.Sentiment,
			InSources:       resp.InGroundingSources,
		}

		// Competitor mentions
		var competitorMentions []models.PromptCompetitorMention
		mentionedBrands := make(map[string]bool)
		trackedBrandsSet := make(map[string]bool)

		// Build set of tracked brands for quick lookup
		for _, comp := range competitors {
			trackedBrandsSet[strings.ToLower(comp)] = true
		}
		trackedBrandsSet[strings.ToLower(mainBrand)] = true

		for _, mentioned := range resp.CompetitorsMention {
			mentionedBrands[strings.ToLower(mentioned)] = true
		}

		for _, comp := range competitors {
			isMentioned := mentionedBrands[strings.ToLower(comp)]
			competitorMentions = append(competitorMentions, models.PromptCompetitorMention{
				Brand:     comp,
				Mentioned: isMentioned,
			})
		}

		// Count total brands mentioned (main brand + all competitors including untracked)
		// This needs to be calculated before we use it in share of voice calculations
		totalBrands := 0
		if resp.BrandMentioned {
			totalBrands++
		}
		totalBrands += len(resp.CompetitorsMention)

		// Calculate share of voice for main brand at prompt level
		// Share of voice = (1 if mentioned else 0) / total brands mentioned * 100
		mainBrandShareOfVoice := 0.0
		if totalBrands > 0 {
			if resp.BrandMentioned {
				mainBrandShareOfVoice = (1.0 / float64(totalBrands)) * 100.0
			}
		}
		mainResult.ShareOfVoice = mainBrandShareOfVoice

		// Identify untracked competitors mentioned (not in tracked list) with metrics
		var untrackedCompetitorsMentioned []models.UntrackedCompetitorMention
		competitorPositionMap := make(map[string]int) // Track position of each competitor
		
		// Build position map: earlier mentions = better position
		// Position 1 = main brand (if mentioned), then competitors start at 2
		position := 2 // Start at 2 (position 1 is for main brand if mentioned)
		if !resp.BrandMentioned {
			position = 1 // If main brand not mentioned, competitors start at 1
		}
		for _, mentioned := range resp.CompetitorsMention {
			normalized := strings.TrimSpace(mentioned)
			if normalized != "" {
				competitorPositionMap[normalized] = position
				position++
			}
		}

		// Check which competitors are in grounding sources
		competitorsInSources := make(map[string]bool)
		for _, source := range resp.GroundingSources {
			// Extract domain/brand from source URL if possible
			// For now, we'll check if competitor name appears in source
			for _, mentioned := range resp.CompetitorsMention {
				if strings.Contains(strings.ToLower(source), strings.ToLower(mentioned)) {
					competitorsInSources[mentioned] = true
				}
			}
		}

		for _, mentioned := range resp.CompetitorsMention {
			normalized := strings.TrimSpace(mentioned)
			if normalized != "" && !trackedBrandsSet[strings.ToLower(normalized)] {
				// Calculate position (earlier in list = better position)
				compPosition := competitorPositionMap[normalized]
				if compPosition == 0 {
					compPosition = position // If not in map, assign last position
				}

				// Estimate visibility score based on position
				// Position 1-2: high visibility (8-10), Position 3-5: medium (5-7), Position 6+: low (3-5)
				visibilityScore := 6 // Default
				if compPosition <= 2 {
					visibilityScore = 8 + (3 - compPosition) // 8-10
				} else if compPosition <= 5 {
					visibilityScore = 7 - (compPosition - 3) // 5-7
				} else {
					posOffset := compPosition - 6
					if posOffset > 2 {
						posOffset = 2
					}
					visibilityScore = 5 - posOffset // 3-5
					if visibilityScore < 3 {
						visibilityScore = 3
					}
				}

				// Default sentiment (we don't have per-competitor sentiment, so use positive as default)
				sentiment := "positive"

				// Check if in sources
				inSources := competitorsInSources[normalized]

				// Calculate share of voice
				shareOfVoice := 0.0
				if totalBrands > 0 {
					shareOfVoice = (1.0 / float64(totalBrands)) * 100.0
				}

				untrackedCompetitorsMentioned = append(untrackedCompetitorsMentioned, models.UntrackedCompetitorMention{
					Brand:           normalized,
					VisibilityScore: visibilityScore,
					Position:        compPosition,
					Sentiment:       sentiment,
					InSources:       inSources,
					ShareOfVoice:    shareOfVoice,
				})
			}
		}

		// Determine winner (brand with best position or visibility)
		winner := ""
		if resp.BrandMentioned {
			winner = mainBrand
			// If main brand has lower position (worse), a competitor might be winning
			if resp.BrandPosition > 1 {
				// Check if any competitors were mentioned (they might be ahead)
				for _, comp := range resp.CompetitorsMention {
					winner = comp // First competitor mentioned might be the leader
					break
				}
			}
		} else if len(resp.CompetitorsMention) > 0 {
			winner = resp.CompetitorsMention[0] // First mentioned competitor wins
		}

		// Get prompt type from database
		promptType := ""
		if prompt, err := s.db.GetPrompt(context.Background(), resp.PromptID); err == nil {
			promptType = string(prompt.PromptType)
		}

		// Generate insights for this prompt
		insights := s.generatePromptInsights(mainResult, competitorMentions, mainBrand, winner)

		breakdown = append(breakdown, models.PromptCompetitiveAnalysis{
			PromptID:                      resp.PromptID,
			PromptText:                    resp.PromptText,
			PromptType:                    promptType,
			MainBrandResult:               mainResult,
			TrackedCompetitorsMentioned:   competitorMentions,
			UntrackedCompetitorsMentioned: untrackedCompetitorsMentioned,
			Winner:                        winner,
			TotalBrandsMentioned:          totalBrands,
			ExecutedAt:                    resp.CreatedAt,
			Insights:                      insights,
		})
	}

	return breakdown
}

// generatePromptInsights analyzes the gap between main brand and competitors for a specific prompt
// Returns gap percentage (0-100) and categorical reason for dashboard visualization
func (s *CompetitiveBenchmarkService) generatePromptInsights(
	mainResult models.PromptBrandResult,
	competitorMentions []models.PromptCompetitorMention,
	mainBrand string,
	winner string,
) *models.PromptInsights {
	// Count how many competitors were mentioned
	competitorsMentionedCount := 0
	leadingCompetitor := ""
	for _, comp := range competitorMentions {
		if comp.Mentioned {
			competitorsMentionedCount++
			if leadingCompetitor == "" {
				leadingCompetitor = comp.Brand
			}
		}
	}

	// If no competitors mentioned, no gap to analyze
	if competitorsMentionedCount == 0 {
		return &models.PromptInsights{
			GapPercentage:  0,
			ReasonCategory: "none",
			Severity:       "none",
			Recommendations: []string{"Maintain your current positioning", "Continue monitoring for competitive mentions"},
		}
	}

	// Track the most critical gap
	var maxGapPercentage float64
	var primaryReasonCategory string
	var severity string

	// Gap 1: Not mentioned at all while competitors are (CRITICAL - 100% gap)
	if !mainResult.Mentioned && competitorsMentionedCount > 0 {
		maxGapPercentage = 100.0
		primaryReasonCategory = "no_mention"
		severity = "critical"
		if leadingCompetitor == "" && winner != "" && winner != mainBrand {
			leadingCompetitor = winner
		}
		return &models.PromptInsights{
			GapPercentage:   maxGapPercentage,
			ReasonCategory:  primaryReasonCategory,
			Severity:        severity,
			LeadingCompetitor: leadingCompetitor,
			Recommendations: s.getRecommendationsForCategory(primaryReasonCategory, leadingCompetitor),
		}
	}

	// If brand is mentioned, analyze other gaps
	if mainResult.Mentioned {
		// Gap 2: Poor position (high position number = worse)
		// Position 1 = 0% gap, Position 10+ = 90% gap
		if mainResult.Position > 0 {
			positionGap := 0.0
			if mainResult.Position == 1 {
				positionGap = 0.0
			} else if mainResult.Position <= 3 {
				// Positions 2-3: 20-40% gap
				positionGap = float64((mainResult.Position-1) * 20)
			} else if mainResult.Position <= 5 {
				// Positions 4-5: 50-70% gap
				positionGap = 50.0 + float64((mainResult.Position-4)*10)
			} else {
				// Positions 6+: 80-90% gap
				posOffset := mainResult.Position - 6
				if posOffset > 4 {
					posOffset = 4
				}
				positionGap = 80.0 + float64(posOffset)*2.5
				if positionGap > 90.0 {
					positionGap = 90.0
				}
			}

			if positionGap > maxGapPercentage {
				maxGapPercentage = positionGap
				primaryReasonCategory = "poor_position"
				if positionGap >= 70.0 {
					severity = "high"
				} else if positionGap >= 40.0 {
					severity = "medium"
				} else {
					severity = "low"
				}
			}
		}

		// Gap 3: Low visibility score (0-10 scale)
		// Score 0-2 = 80-100% gap, Score 3-4 = 60-70% gap, Score 5-6 = 40-50% gap, Score 7+ = 0-30% gap
		if mainResult.VisibilityScore < 7 {
			visibilityGap := 0.0
			if mainResult.VisibilityScore <= 2 {
				visibilityGap = 100.0 - float64(mainResult.VisibilityScore*10)
			} else if mainResult.VisibilityScore <= 4 {
				visibilityGap = 70.0 - float64((mainResult.VisibilityScore-2)*5)
			} else {
				visibilityGap = 50.0 - float64((mainResult.VisibilityScore-4)*5)
			}

			if visibilityGap > maxGapPercentage {
				maxGapPercentage = visibilityGap
				primaryReasonCategory = "low_visibility"
				if visibilityGap >= 70.0 {
					severity = "high"
				} else if visibilityGap >= 40.0 {
					severity = "medium"
				} else {
					severity = "low"
				}
			}
		}

		// Gap 4: Missing from citation sources
		if !mainResult.InSources {
			citationGap := 40.0 // Missing citations = 40% gap
			if citationGap > maxGapPercentage {
				maxGapPercentage = citationGap
				primaryReasonCategory = "missing_citations"
				severity = "medium"
			}
		}

		// Gap 5: Negative sentiment
		if mainResult.Sentiment == "negative" {
			sentimentGap := 60.0 // Negative sentiment = 60% gap
			if sentimentGap > maxGapPercentage {
				maxGapPercentage = sentimentGap
				primaryReasonCategory = "negative_sentiment"
				severity = "high"
			}
		}

		// Gap 6: Topic not discussed (inferred from low visibility + competitors mentioned)
		if mainResult.VisibilityScore < 5 && competitorsMentionedCount > 0 {
			topicGap := 50.0
			if topicGap > maxGapPercentage && primaryReasonCategory != "low_visibility" {
				maxGapPercentage = topicGap
				primaryReasonCategory = "topic_not_discussed"
				severity = "medium"
			}
		}

		// Gap 7: Blog/content missing (inferred from not in sources + competitors mentioned)
		if !mainResult.InSources && competitorsMentionedCount > 0 {
			blogGap := 35.0
			if blogGap > maxGapPercentage && primaryReasonCategory != "missing_citations" {
				maxGapPercentage = blogGap
				primaryReasonCategory = "blog_missing"
				severity = "medium"
			}
		}

		// Gap 8: Low reviews/ratings (inferred from negative sentiment or poor position)
		if mainResult.Sentiment == "negative" || (mainResult.Position > 3 && mainResult.Sentiment != "positive") {
			reviewGap := 45.0
			if reviewGap > maxGapPercentage && primaryReasonCategory != "negative_sentiment" && primaryReasonCategory != "poor_position" {
				maxGapPercentage = reviewGap
				primaryReasonCategory = "low_review"
				severity = "medium"
			}
		}
	}

	// If winner is a competitor, update leading competitor
	if winner != "" && winner != mainBrand {
		leadingCompetitor = winner
		// If no other gap detected, this is the primary reason
		if maxGapPercentage == 0 {
			maxGapPercentage = 30.0
			primaryReasonCategory = "topic_not_discussed"
			severity = "low"
		}
	}

	// Default: no significant gap
	if maxGapPercentage == 0 {
		primaryReasonCategory = "none"
		severity = "none"
	}

	return &models.PromptInsights{
		GapPercentage:    maxGapPercentage,
		ReasonCategory:  primaryReasonCategory,
		Severity:        severity,
		LeadingCompetitor: leadingCompetitor,
		Recommendations: s.getRecommendationsForCategory(primaryReasonCategory, leadingCompetitor),
	}
}

// getRecommendationsForCategory returns recommendations based on the reason category
func (s *CompetitiveBenchmarkService) getRecommendationsForCategory(category string, leadingCompetitor string) []string {
	switch category {
	case "no_mention":
		recs := []string{
			"Improve content marketing and SEO to increase brand visibility for this query type",
			"Analyze competitor content strategies and identify keywords they're ranking for",
			"Create more authoritative content that addresses this specific query",
			"Build more backlinks and citations from reputable sources",
		}
		if leadingCompetitor != "" {
			recs = append(recs, fmt.Sprintf("Study %s's content approach for this query type", leadingCompetitor))
		}
		return recs

	case "poor_position":
		recs := []string{
			"Optimize content to be more directly relevant to this query type",
			"Improve brand authority through thought leadership and expert content",
			"Analyze competitor positioning strategies and adapt your approach",
			"Focus on being mentioned earlier in responses by improving content quality and relevance",
		}
		if leadingCompetitor != "" {
			recs = append(recs, fmt.Sprintf("Analyze why %s ranks higher and adapt similar strategies", leadingCompetitor))
		}
		return recs

	case "low_visibility":
		return []string{
			"Enhance brand content depth and authority to improve visibility scores",
			"Focus on creating comprehensive, well-structured content that LLMs prefer to cite",
			"Improve technical SEO and structured data to help LLMs understand your brand better",
			"Increase content frequency and coverage of relevant topics",
		}

	case "missing_citations":
		return []string{
			"Create more authoritative, well-sourced content that LLMs can cite",
			"Improve technical SEO and structured data (Schema.org) to help LLMs identify your content",
			"Build more high-quality backlinks from reputable sources",
			"Ensure your content is easily discoverable and properly formatted for LLM consumption",
		}

	case "negative_sentiment":
		return []string{
			"Address any negative brand associations through reputation management",
			"Increase positive brand mentions and reviews",
			"Improve customer experience to generate more positive sentiment",
			"Monitor and respond to negative feedback proactively",
		}

	case "topic_not_discussed":
		recs := []string{
			"Create comprehensive content covering this topic in depth",
			"Develop thought leadership content that addresses this query type",
			"Ensure your content directly answers the query being asked",
		}
		if leadingCompetitor != "" {
			recs = append(recs, fmt.Sprintf("Review how %s addresses this topic and create differentiated content", leadingCompetitor))
		}
		return recs

	case "blog_missing":
		return []string{
			"Create blog content that addresses this query type",
			"Develop a content strategy that covers topics relevant to this query",
			"Ensure your blog content is optimized for LLM discovery and citation",
			"Build a content library that establishes authority in this domain",
		}

	case "low_review":
		return []string{
			"Improve product/service quality to generate better reviews",
			"Encourage satisfied customers to leave positive reviews",
			"Address common complaints mentioned in negative reviews",
			"Build a stronger online reputation through consistent quality delivery",
		}

	default:
		return []string{
			"Continue monitoring competitive performance",
			"Maintain current content strategy",
		}
	}
}

// brandMentionStats tracks mention statistics for a brand
type brandMentionStats struct {
	brand           string
	mentionCount    int
	totalVisibility float64
	totalPosition   float64
	positionCount   int
	sentimentScores []float64
}

// analyzeBrandPerformance analyzes performance for a single brand
func (s *CompetitiveBenchmarkService) analyzeBrandPerformance(
	ctx context.Context,
	brand string,
	promptIDs, llmIDs []string,
	startTime, endTime *time.Time,
	region string,
) (models.BrandPerformance, error) {
	// Fetch responses
	filter := shared.ResponseFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     10000,
	}

	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return models.BrandPerformance{}, err
	}

	// Filter responses
	var filteredResponses []*models.Response
	for _, resp := range allResponses {
		// Filter by brand
		if resp.Brand != brand {
			continue
		}

		// Filter by region if specified
		if region != "" && resp.Region != region {
			continue
		}

		// Filter by prompt IDs if specified
		if len(promptIDs) > 0 && !contains(promptIDs, resp.PromptID) {
			continue
		}

		// Filter by LLM IDs if specified
		if len(llmIDs) > 0 && !contains(llmIDs, resp.LLMID) {
			continue
		}

		filteredResponses = append(filteredResponses, resp)
	}

	if len(filteredResponses) == 0 {
		return models.BrandPerformance{
			Brand:         brand,
			ResponseCount: 0,
		}, nil
	}

	// Calculate metrics
	totalVisibility := 0.0
	mentionCount := 0
	groundingCount := 0
	totalPosition := 0.0
	positionCount := 0
	topPositionCount := 0
	sentimentSum := 0.0
	sentimentCount := 0

	for _, resp := range filteredResponses {
		totalVisibility += float64(resp.VisibilityScore)

		if resp.BrandMentioned {
			mentionCount++
		}

		if resp.InGroundingSources {
			groundingCount++
		}

		if resp.BrandPosition > 0 {
			totalPosition += float64(resp.BrandPosition)
			positionCount++

			if resp.BrandPosition <= 3 {
				topPositionCount++
			}
		}

		if resp.Sentiment != "" {
			sentimentSum += calculateSentimentScore(resp.Sentiment)
			sentimentCount++
		}
	}

	perf := models.BrandPerformance{
		Brand:         brand,
		Visibility:    totalVisibility / float64(len(filteredResponses)),
		MentionRate:   float64(mentionCount) / float64(len(filteredResponses)) * 100,
		GroundingRate: float64(groundingCount) / float64(len(filteredResponses)) * 100,
		ResponseCount: len(filteredResponses),
	}

	if positionCount > 0 {
		perf.AveragePosition = totalPosition / float64(positionCount)
		perf.TopPositionRate = float64(topPositionCount) / float64(positionCount) * 100
	}

	if sentimentCount > 0 {
		perf.SentimentScore = sentimentSum / float64(sentimentCount)
	}

	return perf, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

