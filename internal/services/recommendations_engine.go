package services

import (
	"fmt"
	"sort"
	"strings"

	"github.com/AI2HU/gego/internal/models"
)

// RecommendationsEngine generates actionable insights from GEO data
type RecommendationsEngine struct{}

// NewRecommendationsEngine creates a new recommendations engine
func NewRecommendationsEngine() *RecommendationsEngine {
	return &RecommendationsEngine{}
}

// GenerateSourceRecommendations generates recommendations based on source citation analysis
func (e *RecommendationsEngine) GenerateSourceRecommendations(
	brand string,
	topSources []models.SourceInsight,
	totalCitations int,
	brandInSources bool,
) []models.Recommendation {
	var recommendations []models.Recommendation
	
	// Analyze top sources and generate recommendations
	for i, source := range topSources {
		if i >= 5 { // Only analyze top 5 sources
			break
		}
		
		categories := categorizeSource(source.Domain)
		
		// Review sites recommendations
		if containsCategory(categories, "review_site") {
			if source.CitationCount >= 3 {
				recommendations = append(recommendations, models.Recommendation{
					Type:        "source_opportunity",
					Priority:    "high",
					Title:       fmt.Sprintf("Optimize presence on %s", source.Domain),
					Description: fmt.Sprintf("The review site %s is frequently cited (%d times, %.1f%% of responses). This is a high-value source for AI visibility.", source.Domain, source.CitationCount, source.MentionRate),
					Action:      fmt.Sprintf("Create or optimize your profile on %s. Encourage customers to leave reviews. Ensure your listing is complete and up-to-date.", source.Domain),
					Impact:      "high",
				})
			}
		}
		
		// Social media recommendations
		if containsCategory(categories, "social_media") {
			if strings.Contains(source.Domain, "reddit") {
				recommendations = append(recommendations, models.Recommendation{
					Type:        "content_opportunity",
					Priority:    "medium",
					Title:       "Engage in Reddit discussions",
					Description: fmt.Sprintf("Reddit (%s) appears frequently in citations (%d times). AI models value community discussions.", source.Domain, source.CitationCount),
					Action:      "Participate authentically in relevant subreddit discussions. Share helpful insights without overtly promoting. Consider doing an AMA if appropriate.",
					Impact:      "medium",
				})
			} else if strings.Contains(source.Domain, "linkedin") {
				recommendations = append(recommendations, models.Recommendation{
					Type:        "content_opportunity",
					Priority:    "medium",
					Title:       "Increase LinkedIn presence",
					Description: fmt.Sprintf("LinkedIn is cited %d times. Professional networks are valued by AI models.", source.CitationCount),
					Action:      "Publish thought leadership articles on LinkedIn. Engage with industry discussions. Ensure company page is complete.",
					Impact:      "medium",
				})
			}
		}
		
		// News and publications recommendations
		if containsCategory(categories, "news") || containsCategory(categories, "publication") {
			recommendations = append(recommendations, models.Recommendation{
				Type:        "pr_opportunity",
				Priority:    "high",
				Title:       fmt.Sprintf("Digital PR: Target %s", source.Domain),
				Description: fmt.Sprintf("Editorial site %s is cited %d times by AI models. Getting featured here significantly boosts visibility.", source.Domain, source.CitationCount),
				Action:      fmt.Sprintf("Consider digital PR campaigns targeting %s. Pitch newsworthy stories, data releases, or expert commentary.", source.Domain),
				Impact:      "high",
			})
		}
	}
	
	// General recommendations based on overall performance
	if !brandInSources && len(topSources) > 0 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "source_opportunity",
			Priority:    "critical",
			Title:       "Brand not appearing in cited sources",
			Description: "Your brand is not appearing in the sources that AI models cite. This severely limits visibility.",
			Action:      fmt.Sprintf("Focus on getting mentioned in these high-value sources: %s, %s, %s", getTopDomainNames(topSources, 3)),
			Impact:      "critical",
		})
	}
	
	// Citation diversity recommendation
	if len(topSources) > 0 && topSources[0].CitationCount > totalCitations/2 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "diversity_warning",
			Priority:    "medium",
			Title:       "Low citation diversity",
			Description: fmt.Sprintf("%.1f%% of citations come from a single source (%s). This creates dependency risk.", topSources[0].MentionRate, topSources[0].Domain),
			Action:      "Diversify your presence across multiple high-quality sources to reduce risk and increase overall visibility.",
			Impact:      "medium",
		})
	}
	
	return recommendations
}

// GenerateCompetitiveRecommendations generates recommendations based on competitive analysis
func (e *RecommendationsEngine) GenerateCompetitiveRecommendations(
	mainBrand models.BrandPerformance,
	competitors []models.BrandPerformance,
	marketLeader string,
) []models.Recommendation {
	var recommendations []models.Recommendation
	
	// Sort competitors by visibility
	sort.Slice(competitors, func(i, j int) bool {
		return competitors[i].Visibility > competitors[j].Visibility
	})
	
	// If we're not the market leader
	if marketLeader != mainBrand.Brand {
		topCompetitor := competitors[0]
		visibilityGap := topCompetitor.Visibility - mainBrand.Visibility
		
		if visibilityGap > 3.0 {
			recommendations = append(recommendations, models.Recommendation{
				Type:        "competitor_threat",
				Priority:    "critical",
				Title:       fmt.Sprintf("Significant visibility gap with %s", topCompetitor.Brand),
				Description: fmt.Sprintf("Your visibility (%.1f) is %.1f points behind market leader %s (%.1f). This represents a %.1f%% gap.", mainBrand.Visibility, visibilityGap, topCompetitor.Brand, topCompetitor.Visibility, (visibilityGap/topCompetitor.Visibility)*100),
				Action:      "Analyze what content and sources drive competitor visibility. Focus on getting mentioned in their key citation sources.",
				Impact:      "critical",
			})
		} else if visibilityGap > 1.0 {
			recommendations = append(recommendations, models.Recommendation{
				Type:        "competitive_opportunity",
				Priority:    "high",
				Title:       "Close visibility gap with market leader",
				Description: fmt.Sprintf("You're %.1f points behind %s. The gap is closeable with focused effort.", visibilityGap, topCompetitor.Brand),
				Action:      "Focus on improving position in list-based prompts. Ensure strong presence in review sites and communities.",
				Impact:      "high",
			})
		}
	} else {
		// We are the market leader
		recommendations = append(recommendations, models.Recommendation{
			Type:        "maintain_leadership",
			Priority:    "medium",
			Title:       "Maintain market leadership position",
			Description: fmt.Sprintf("You're the market leader with %.1f visibility. Focus on maintaining and extending this lead.", mainBrand.Visibility),
			Action:      "Continue your current strategy. Monitor emerging competitors. Expand to new prompt categories and regions.",
			Impact:      "medium",
		})
	}
	
	// Position analysis
	if mainBrand.AveragePosition > 3.0 && mainBrand.AveragePosition > 0 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "position_improvement",
			Priority:    "high",
			Title:       "Improve average position in lists",
			Description: fmt.Sprintf("Your average position is %.1f (where 1 is best). You're often mentioned but not at the top.", mainBrand.AveragePosition),
			Action:      "Focus on being the 'best' or 'top choice' in content. Improve review scores and ratings. Get more positive testimonials.",
			Impact:      "high",
		})
	}
	
	// Sentiment comparison
	avgCompetitorSentiment := 0.0
	for _, comp := range competitors {
		avgCompetitorSentiment += comp.SentimentScore
	}
	if len(competitors) > 0 {
		avgCompetitorSentiment /= float64(len(competitors))
		
		if mainBrand.SentimentScore < avgCompetitorSentiment-0.2 {
			recommendations = append(recommendations, models.Recommendation{
				Type:        "sentiment_warning",
				Priority:    "high",
				Title:       "Sentiment below competitors",
				Description: fmt.Sprintf("Your sentiment score (%.2f) is lower than competitors' average (%.2f).", mainBrand.SentimentScore, avgCompetitorSentiment),
				Action:      "Address negative feedback and reviews. Improve customer satisfaction. Highlight positive case studies.",
				Impact:      "high",
			})
		}
	}
	
	// Grounding rate analysis
	if mainBrand.GroundingRate < 30.0 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "source_opportunity",
			Priority:    "critical",
			Title:       "Low source citation rate",
			Description: fmt.Sprintf("Your brand appears in cited sources only %.1f%% of the time. This limits credibility.", mainBrand.GroundingRate),
			Action:      "Focus on getting your website and brand mentioned in authoritative sources that AI models cite (review sites, industry publications, news).",
			Impact:      "critical",
		})
	}
	
	return recommendations
}

// GeneratePositionRecommendations generates recommendations based on position analysis
func (e *RecommendationsEngine) GeneratePositionRecommendations(
	brand string,
	avgPosition float64,
	topPositionRate float64,
	totalMentions int,
) []models.Recommendation {
	var recommendations []models.Recommendation
	
	if totalMentions == 0 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "visibility_critical",
			Priority:    "critical",
			Title:       "Brand not appearing in AI responses",
			Description: "Your brand is not being mentioned by AI models at all.",
			Action:      "Focus on foundational GEO: create authoritative content, get listed in industry directories, obtain reviews, build citations.",
			Impact:      "critical",
		})
		return recommendations
	}
	
	if avgPosition > 5.0 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "position_improvement",
			Priority:    "high",
			Title:       "Average position needs improvement",
			Description: fmt.Sprintf("Average position is %.1f. You're mentioned but ranked low in lists.", avgPosition),
			Action:      "Improve your 'best in class' signals: higher review scores, more positive testimonials, authoritative endorsements.",
			Impact:      "high",
		})
	}
	
	if topPositionRate < 20.0 && totalMentions > 10 {
		recommendations = append(recommendations, models.Recommendation{
			Type:        "position_improvement",
			Priority:    "medium",
			Title:       "Rarely appearing in top positions",
			Description: fmt.Sprintf("Only %.1f%% of mentions are in top 3 positions. Top positions get most attention.", topPositionRate),
			Action:      "Focus on superlatives in content ('best', 'top', 'leading'). Improve comparative advantages vs competitors.",
			Impact:      "medium",
		})
	}
	
	return recommendations
}

// Helper functions

func containsCategory(categories []string, category string) bool {
	for _, c := range categories {
		if c == category {
			return true
		}
	}
	return false
}

func getTopDomainNames(sources []models.SourceInsight, n int) string {
	var domains []string
	for i := 0; i < n && i < len(sources); i++ {
		domains = append(domains, sources[i].Domain)
	}
	return strings.Join(domains, ", ")
}

