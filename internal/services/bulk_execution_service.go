package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/llm"
	"github.com/fissionx/gego/internal/models"
)

// BulkExecutionService handles batch execution of prompts across multiple LLMs
type BulkExecutionService struct {
	db                 db.Database
	llmRegistry        *llm.Registry
	opportunityService *OpportunityService
}

// NewBulkExecutionService creates a new bulk execution service
func NewBulkExecutionService(database db.Database, registry *llm.Registry) *BulkExecutionService {
	return &BulkExecutionService{
		db:                 database,
		llmRegistry:        registry,
		opportunityService: NewOpportunityService(database, registry),
	}
}

// ExecuteCampaign executes all prompts across all LLMs for a GEO campaign
func (s *BulkExecutionService) ExecuteCampaign(ctx context.Context, campaignName, brandID, orgID, brand string, promptIDs, llmIDs []string, temperature float64, totalRuns int) (*models.GEOCampaign, error) {
	if temperature == 0 {
		temperature = 0.7
	}

	if totalRuns == 0 {
		totalRuns = 1
	}

	// Create campaign
	campaign := &models.GEOCampaign{
		ID:        uuid.New().String(),
		Name:      campaignName,
		BrandID:   brandID,
		OrgID:     orgID,
		Brand:     brand,
		PromptIDs: promptIDs,
		LLMIDs:    llmIDs,
		Status:    "running",
		TotalRuns: len(promptIDs) * len(llmIDs) * totalRuns,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save campaign to database immediately
	if err := s.db.SaveGEOCampaign(ctx, campaign); err != nil {
		log.Printf("Failed to save campaign: %v", err)
		// Continue anyway - campaign will still run
	}

	// Start execution in background
	go s.executeInBackground(context.Background(), campaign, temperature, totalRuns)

	return campaign, nil
}

// executeInBackground runs the campaign execution asynchronously
func (s *BulkExecutionService) executeInBackground(ctx context.Context, campaign *models.GEOCampaign, temperature float64, totalRuns int) {
	log.Printf("========== STARTING CAMPAIGN: %s ==========", campaign.Name)
	log.Printf("Brand: %s, Prompts: %d, LLMs: %d, Runs per prompt: %d, Total Runs: %d",
		campaign.Brand, len(campaign.PromptIDs), len(campaign.LLMIDs), totalRuns, campaign.TotalRuns)

	// Fetch prompts and LLMs
	prompts, err := s.getPrompts(ctx, campaign.PromptIDs)
	if err != nil {
		log.Printf("Failed to fetch prompts: %v", err)
		campaign.Status = "failed"
		campaign.UpdatedAt = time.Now()
		s.db.UpdateGEOCampaign(ctx, campaign)
		return
	}

	llms, err := s.getLLMs(ctx, campaign.LLMIDs)
	if err != nil {
		log.Printf("Failed to fetch LLMs: %v", err)
		campaign.Status = "failed"
		campaign.UpdatedAt = time.Now()
		s.db.UpdateGEOCampaign(ctx, campaign)
		return
	}

	// Execute with concurrency control
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3) // Max 3 concurrent executions
	completed := 0
	mu := sync.Mutex{}

	// Execute each prompt-LLM combination multiple times
	for _, prompt := range prompts {
		for _, llmConfig := range llms {
			for run := 0; run < totalRuns; run++ {
				wg.Add(1)

				go func(p *models.Prompt, llm *models.LLMConfig, runNum int) {
					defer wg.Done()

					// Acquire semaphore
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					// Execute single prompt-LLM pair
					err := s.executeSingle(ctx, p, llm, campaign.BrandID, campaign.OrgID, campaign.Brand, temperature)

					mu.Lock()
					completed++
					if completed%10 == 0 || completed == campaign.TotalRuns {
						log.Printf("Campaign %s: %d/%d completed", campaign.Name, completed, campaign.TotalRuns)
					}
					mu.Unlock()

					if err != nil {
						log.Printf("Execution failed for prompt %s with LLM %s (run %d/%d): %v", p.ID, llm.ID, runNum+1, totalRuns, err)
					}
				}(prompt, llmConfig, run)
			}
		}
	}

	wg.Wait()

	completedTime := time.Now()
	campaign.Status = "completed"
	campaign.CompletedAt = &completedTime
	campaign.UpdatedAt = completedTime

	// Update campaign status in database
	if err := s.db.UpdateGEOCampaign(ctx, campaign); err != nil {
		log.Printf("Failed to update campaign status: %v", err)
	}

	log.Printf("========== CAMPAIGN COMPLETED: %s ==========", campaign.Name)
	log.Printf("Total executions: %d", completed)
}

// executeSingle executes a single prompt with a single LLM
func (s *BulkExecutionService) executeSingle(ctx context.Context, prompt *models.Prompt, llmConfig *models.LLMConfig, brandID string, orgID string, brand string, temperature float64) error {
	// Create LLM provider
	provider, ok := s.llmRegistry.Get(llmConfig.Provider)
	if !ok || provider == nil {
		return fmt.Errorf("provider not available: %s", llmConfig.Provider)
	}

	// Execute prompt
	response, err := provider.Generate(ctx, prompt.Template, llm.Config{
		Model:       llmConfig.Model,
		Temperature: temperature,
		MaxTokens:   4096, // TODO: Make this configurable
		Brand:       brand,
	})
	if err != nil {
		// Save error response
		errorResponse := &models.Response{
			ID:          uuid.New().String(),
			BrandID:     brandID,
			OrgID:       orgID,
			PromptID:    prompt.ID,
			PromptText:  prompt.Template,
			LLMID:       llmConfig.ID,
			LLMName:     llmConfig.Name,
			LLMProvider: llmConfig.Provider,
			LLMModel:    llmConfig.Model,
			Brand:       brand,
			Temperature: temperature,
			Error:       err.Error(),
			CreatedAt:   time.Now(),
		}
		s.db.CreateResponse(ctx, errorResponse)
		return err
	}

	// Parse GEO analysis from response (if brand was provided)
	responseModel := &models.Response{
		ID:           uuid.New().String(),
		BrandID:      brandID,
		OrgID:        orgID,
		PromptID:     prompt.ID,
		PromptText:   prompt.Template,
		LLMID:        llmConfig.ID,
		LLMName:      llmConfig.Name,
		LLMProvider:  llmConfig.Provider,
		LLMModel:     llmConfig.Model,
		ResponseText: response.Text,
		Brand:        brand,
		Temperature:  temperature,
		TokensUsed:   response.TokensUsed,
		LatencyMs:    response.LatencyMs,
		CreatedAt:    time.Now(),
	}

	// Store web search metadata (for ChatGPT/Gemini-like experience)
	responseModel.WebSearchQueries = response.WebSearchQueries
	responseModel.GroundingSources = response.GroundingSources

	// Store original search answer (before GEO analysis, if applicable)
	if response.SearchAnswer != "" {
		responseModel.SearchAnswer = response.SearchAnswer
	} else {
		// If no SearchAnswer provided, use ResponseText as fallback
		responseModel.SearchAnswer = response.Text
	}

	// Parse GEO metrics if brand was provided
	if brand != "" {
		// Always extract domains from grounding sources (even if JSON parsing fails)
		if len(response.GroundingSources) > 0 {
			responseModel.GroundingDomains = ExtractDomainsFromSources(response.GroundingSources)

			// Check if brand appears in grounding sources (even if JSON parsing fails)
			brandLower := strings.ToLower(brand)
			brandDomain := strings.ReplaceAll(brandLower, " ", "")
			for _, source := range response.GroundingSources {
				sourceLower := strings.ToLower(source)
				if strings.Contains(sourceLower, brandDomain) ||
					strings.Contains(sourceLower, strings.ReplaceAll(brandLower, ".", "")) {
					responseModel.InGroundingSources = true
					break
				}
			}
		}

		geoAnalysis := parseGEOAnalysis(response.Text)
		if geoAnalysis != nil {
			// Access nested GEOAnalysis struct
			geo := geoAnalysis.GEOAnalysis
			responseModel.VisibilityScore = geo.VisibilityScore
			responseModel.BrandMentioned = geo.BrandMentioned
			// Override InGroundingSources from JSON if available (more accurate)
			responseModel.InGroundingSources = geo.InGroundingSources
			responseModel.Sentiment = geo.Sentiment
			responseModel.CompetitorsMention = geo.Competitors

			// Extract position/ranking from the search_answer text
			searchAnswer := geoAnalysis.SearchAnswer
			if searchAnswer == "" {
				searchAnswer = responseModel.SearchAnswer // Use stored search answer
			}

			if geo.BrandMentioned {
				position, totalBrands := ExtractBrandPosition(searchAnswer, brand)
				responseModel.BrandPosition = position
				responseModel.TotalBrandsListed = totalBrands
			}
		} else {
			// If JSON parsing failed, still try to extract basic metrics from response text
			responseTextLower := strings.ToLower(responseModel.ResponseText)
			brandLower := strings.ToLower(brand)
			if strings.Contains(responseTextLower, brandLower) {
				responseModel.BrandMentioned = true
			}
		}
	}

	// Add time-series fields
	now := time.Now()
	responseModel.Week = now.Format("2006-W02")
	responseModel.Month = now.Format("2006-01")
	quarter := (int(now.Month())-1)/3 + 1
	responseModel.Quarter = fmt.Sprintf("%d-Q%d", now.Year(), quarter)

	// Save response
	if err := s.db.CreateResponse(ctx, responseModel); err != nil {
		return err
	}

	// Generate opportunities from the response (async, don't block execution)
	// Pass the same LLM provider that was used for executing the prompt
	if brand != "" && s.opportunityService != nil {
		go s.generateOpportunitiesAsync(context.Background(), prompt.OrgID, brandID, brand, prompt.ID, responseModel.ID, prompt.Template, responseModel.SearchAnswer, responseModel.GroundingSources, provider, llmConfig.ID, llmConfig.Model)
	}

	return nil
}

// generateOpportunitiesAsync generates opportunities in the background using the same LLM that executed the prompt
func (s *BulkExecutionService) generateOpportunitiesAsync(ctx context.Context, orgID, brandID, brandName, promptID, responseID, searchQuery, searchAnswer string, groundingSources []string, llmProvider llm.Provider, llmID, llmModel string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in generateOpportunitiesAsync: %v", r)
		}
	}()

	log.Printf("🔍 Starting opportunity generation for prompt %s (brand: %s) using LLM: %s", promptID, brandName, llmModel)

	// Skip if no search answer
	if searchAnswer == "" {
		log.Printf("⚠️ Skipping opportunity generation - no search answer for prompt %s", promptID)
		return
	}

	log.Printf("📝 Search answer length: %d chars for prompt %s", len(searchAnswer), promptID)

	// Get competitors for the brand (if available)
	var competitors []string
	brandCompetitors, err := s.db.GetBrandCompetitors(ctx, brandID)
	if err == nil && brandCompetitors != nil {
		competitors = brandCompetitors.Competitors
	}

	// Build sources info string
	sourcesInfo := ""
	if len(groundingSources) > 0 {
		sourcesInfo = fmt.Sprintf("\n\nGROUNDING SOURCES (URLs cited by the AI):\n%s", strings.Join(groundingSources, "\n"))
	}

	// Generate opportunities using the service with the same LLM that executed the prompt
	_, opportunities, err := s.opportunityService.AnalyzeAndGenerateOpportunities(
		ctx,
		orgID,
		brandID,
		brandName,
		promptID,
		responseID,
		searchQuery,
		searchAnswer,
		sourcesInfo,
		competitors,
		llmProvider,
		llmID,
		llmModel,
	)
	if err != nil {
		log.Printf("Failed to generate opportunities for prompt %s: %v", promptID, err)
		return
	}

	if len(opportunities) > 0 {
		log.Printf("✅ Generated %d opportunities for prompt %s", len(opportunities), promptID)
	}
}

// getPrompts fetches prompts by IDs - optimized: batch fetch instead of N queries
func (s *BulkExecutionService) getPrompts(ctx context.Context, promptIDs []string) ([]*models.Prompt, error) {
	prompts, err := s.db.GetPromptsByIDs(ctx, promptIDs)
	if err != nil {
		// Fall back to individual fetches for backward compatibility
		prompts = []*models.Prompt{}
		for _, id := range promptIDs {
			prompt, err := s.db.GetPrompt(ctx, id)
			if err != nil {
				continue
			}
			prompts = append(prompts, prompt)
		}
	}
	if len(prompts) == 0 {
		return nil, fmt.Errorf("no valid prompts found")
	}
	return prompts, nil
}

// getLLMs fetches LLM configs by IDs - optimized: batch fetch instead of N queries
func (s *BulkExecutionService) getLLMs(ctx context.Context, llmIDs []string) ([]*models.LLMConfig, error) {
	if len(llmIDs) == 0 {
		return nil, fmt.Errorf("no LLM IDs provided")
	}

	// Batch fetch all LLMs
	llmRecords, err := s.db.GetLLMsByIDs(ctx, llmIDs)
	if err != nil {
		// Fall back to individual fetches for backward compatibility
		llmService := NewLLMService(s.db)
		llmRecords = []*models.LLMConfig{}
		for _, id := range llmIDs {
			llmConfig, err := llmService.GetLLM(ctx, id)
			if err != nil {
				continue
			}
			llmRecords = append(llmRecords, llmConfig)
		}
	}

	// Filter enabled LLMs and track disabled/not found
	var llms []*models.LLMConfig
	var notFound []string
	var disabled []string
	foundMap := make(map[string]bool)

	for _, llmConfig := range llmRecords {
		foundMap[llmConfig.ID] = true
		if !llmConfig.Enabled {
			disabled = append(disabled, fmt.Sprintf("%s (%s)", llmConfig.ID, llmConfig.Name))
			log.Printf("LLM %s (%s) is disabled", llmConfig.ID, llmConfig.Name)
			continue
		}
		llms = append(llms, llmConfig)
	}

	// Track which LLMs were not found
	for _, id := range llmIDs {
		if !foundMap[id] {
			notFound = append(notFound, id)
			log.Printf("LLM %s not found", id)
		}
	}

	if len(llms) == 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString("no valid enabled LLMs found")
		if len(notFound) > 0 {
			errorMsg.WriteString(fmt.Sprintf(". Not found: %v", notFound))
		}
		if len(disabled) > 0 {
			errorMsg.WriteString(fmt.Sprintf(". Disabled: %v", disabled))
		}
		return nil, fmt.Errorf("%s", errorMsg.String())
	}

	// Log warning if some LLMs were skipped
	if len(notFound) > 0 || len(disabled) > 0 {
		log.Printf("Warning: %d LLMs skipped (not found: %d, disabled: %d), using %d enabled LLMs",
			len(notFound)+len(disabled), len(notFound), len(disabled), len(llms))
	}

	return llms, nil
}

// GEOAnalysisResult represents the parsed GEO analysis from LLM
type GEOAnalysisResult struct {
	SearchAnswer string `json:"search_answer"`
	GEOAnalysis  struct {
		VisibilityScore    int      `json:"visibility_score"`
		BrandMentioned     bool     `json:"brand_mentioned"`
		InGroundingSources bool     `json:"in_grounding_sources"`
		MentionStatus      string   `json:"mention_status"`
		Reason             string   `json:"reason"`
		Sentiment          string   `json:"sentiment"`
		Competitors        []string `json:"competitors"`
		Insights           []string `json:"insights"`
		Actions            []string `json:"actions"`
		CompetitorInfo     string   `json:"competitor_info"`
	} `json:"geo_analysis"`
}

// parseGEOAnalysis extracts and parses JSON from the LLM response
func parseGEOAnalysis(text string) *GEOAnalysisResult {
	if text == "" {
		return nil
	}

	// Clean up the response - remove markdown code blocks if present
	cleanedText := strings.TrimSpace(text)

	// Remove markdown code block wrappers (```json ... ``` or ``` ... ```)
	jsonBlockRegex := regexp.MustCompile("(?s)```(?:json)?\\s*(.+?)\\s*```")
	if matches := jsonBlockRegex.FindStringSubmatch(cleanedText); len(matches) > 1 {
		cleanedText = strings.TrimSpace(matches[1])
	} else {
		// Try simple prefix/suffix removal
		cleanedText = strings.TrimPrefix(cleanedText, "```json")
		cleanedText = strings.TrimPrefix(cleanedText, "```")
		cleanedText = strings.TrimSuffix(cleanedText, "```")
		cleanedText = strings.TrimSpace(cleanedText)
	}

	// Try to find JSON object in the text if it's mixed with other content
	if !strings.HasPrefix(cleanedText, "{") {
		jsonStartIdx := strings.Index(cleanedText, "{")
		jsonEndIdx := strings.LastIndex(cleanedText, "}")
		if jsonStartIdx != -1 && jsonEndIdx != -1 && jsonEndIdx > jsonStartIdx {
			cleanedText = cleanedText[jsonStartIdx : jsonEndIdx+1]
		} else {
			// If no JSON found, return nil
			log.Printf("⚠️ No JSON object found in response text")
			return nil
		}
	}

	// Validate that we have a non-empty JSON string
	if cleanedText == "" || cleanedText == "{}" {
		log.Printf("⚠️ Empty or invalid JSON string")
		return nil
	}

	var result GEOAnalysisResult
	if err := json.Unmarshal([]byte(cleanedText), &result); err != nil {
		log.Printf("❌ Failed to parse GEO analysis JSON: %v", err)
		log.Printf("Cleaned text length: %d chars", len(cleanedText))
		log.Printf("Cleaned text (first 500 chars): %s", truncateForLog(cleanedText, 500))
		if len(cleanedText) > 500 {
			log.Printf("Cleaned text (last 200 chars): %s", truncateForLog(cleanedText[len(cleanedText)-200:], 200))
		}
		return nil
	}

	log.Printf("✅ Parsed GEO: Score=%d, Mentioned=%v, InSources=%v, Sentiment=%s, Competitors=%v",
		result.GEOAnalysis.VisibilityScore,
		result.GEOAnalysis.BrandMentioned,
		result.GEOAnalysis.InGroundingSources,
		result.GEOAnalysis.Sentiment,
		result.GEOAnalysis.Competitors)

	return &result
}

// truncateForLog truncates a string for logging
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
