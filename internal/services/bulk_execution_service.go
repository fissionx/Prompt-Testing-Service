package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/AI2HU/gego/internal/db"
	"github.com/AI2HU/gego/internal/llm"
	"github.com/AI2HU/gego/internal/models"
)

// BulkExecutionService handles batch execution of prompts across multiple LLMs
type BulkExecutionService struct {
	db          db.Database
	llmRegistry *llm.Registry
}

// NewBulkExecutionService creates a new bulk execution service
func NewBulkExecutionService(database db.Database, registry *llm.Registry) *BulkExecutionService {
	return &BulkExecutionService{
		db:          database,
		llmRegistry: registry,
	}
}

// ExecuteCampaign executes all prompts across all LLMs for a GEO campaign
func (s *BulkExecutionService) ExecuteCampaign(ctx context.Context, campaignName, brand string, promptIDs, llmIDs []string, temperature float64) (*models.GEOCampaign, error) {
	if temperature == 0 {
		temperature = 0.7
	}

	// Create campaign
	campaign := &models.GEOCampaign{
		ID:        uuid.New().String(),
		Name:      campaignName,
		Brand:     brand,
		PromptIDs: promptIDs,
		LLMIDs:    llmIDs,
		Status:    "running",
		TotalRuns: len(promptIDs) * len(llmIDs),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Start execution in background
	go s.executeInBackground(context.Background(), campaign, temperature)

	return campaign, nil
}

// executeInBackground runs the campaign execution asynchronously
func (s *BulkExecutionService) executeInBackground(ctx context.Context, campaign *models.GEOCampaign, temperature float64) {
	log.Printf("========== STARTING CAMPAIGN: %s ==========", campaign.Name)
	log.Printf("Brand: %s, Prompts: %d, LLMs: %d, Total Runs: %d", 
		campaign.Brand, len(campaign.PromptIDs), len(campaign.LLMIDs), campaign.TotalRuns)

	// Fetch prompts and LLMs
	prompts, err := s.getPrompts(ctx, campaign.PromptIDs)
	if err != nil {
		log.Printf("Failed to fetch prompts: %v", err)
		campaign.Status = "failed"
		return
	}

	llms, err := s.getLLMs(ctx, campaign.LLMIDs)
	if err != nil {
		log.Printf("Failed to fetch LLMs: %v", err)
		campaign.Status = "failed"
		return
	}

	// Execute with concurrency control
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3) // Max 3 concurrent executions
	completed := 0
	mu := sync.Mutex{}

	for _, prompt := range prompts {
		for _, llmConfig := range llms {
			wg.Add(1)
			
			go func(p *models.Prompt, llm *models.LLMConfig) {
				defer wg.Done()
				
				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				// Execute single prompt-LLM pair
				err := s.executeSingle(ctx, p, llm, campaign.Brand, temperature)
				
				mu.Lock()
				completed++
				if completed%10 == 0 || completed == campaign.TotalRuns {
					log.Printf("Campaign %s: %d/%d completed", campaign.Name, completed, campaign.TotalRuns)
				}
				mu.Unlock()

				if err != nil {
					log.Printf("Execution failed for prompt %s with LLM %s: %v", p.ID, llm.ID, err)
				}
			}(prompt, llmConfig)
		}
	}

	wg.Wait()

	completedTime := time.Now()
	campaign.Status = "completed"
	campaign.CompletedAt = &completedTime
	campaign.UpdatedAt = completedTime

	log.Printf("========== CAMPAIGN COMPLETED: %s ==========", campaign.Name)
	log.Printf("Total executions: %d", completed)
}

// executeSingle executes a single prompt with a single LLM
func (s *BulkExecutionService) executeSingle(ctx context.Context, prompt *models.Prompt, llmConfig *models.LLMConfig, brand string, temperature float64) error {
	// Create LLM provider
	provider, ok := s.llmRegistry.Get(llmConfig.Provider)
	if !ok || provider == nil {
		return fmt.Errorf("provider not available: %s", llmConfig.Provider)
	}

	// Execute prompt
	response, err := provider.Generate(ctx, prompt.Template, llm.Config{
		Model:       llmConfig.Model,
		Temperature: temperature,
		MaxTokens:   4096,
		Brand:       brand,
	})
	if err != nil {
		// Save error response
		errorResponse := &models.Response{
			ID:           uuid.New().String(),
			PromptID:     prompt.ID,
			PromptText:   prompt.Template,
			LLMID:        llmConfig.ID,
			LLMName:      llmConfig.Name,
			LLMProvider:  llmConfig.Provider,
			LLMModel:     llmConfig.Model,
			Brand:        brand,
			Temperature:  temperature,
			Error:        err.Error(),
			CreatedAt:    time.Now(),
		}
		s.db.CreateResponse(ctx, errorResponse)
		return err
	}

	// Parse GEO analysis from response (if brand was provided)
	responseModel := &models.Response{
		ID:           uuid.New().String(),
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

	// Save response
	return s.db.CreateResponse(ctx, responseModel)
}

// getPrompts fetches prompts by IDs
func (s *BulkExecutionService) getPrompts(ctx context.Context, promptIDs []string) ([]*models.Prompt, error) {
	var prompts []*models.Prompt
	for _, id := range promptIDs {
		prompt, err := s.db.GetPrompt(ctx, id)
		if err != nil {
			continue
		}
		prompts = append(prompts, prompt)
	}
	if len(prompts) == 0 {
		return nil, fmt.Errorf("no valid prompts found")
	}
	return prompts, nil
}

// getLLMs fetches LLM configs by IDs
func (s *BulkExecutionService) getLLMs(ctx context.Context, llmIDs []string) ([]*models.LLMConfig, error) {
	llmService := NewLLMService(s.db)
	var llms []*models.LLMConfig
	for _, id := range llmIDs {
		llmConfig, err := llmService.GetLLM(ctx, id)
		if err != nil {
			continue
		}
		if llmConfig.Enabled {
			llms = append(llms, llmConfig)
		}
	}
	if len(llms) == 0 {
		return nil, fmt.Errorf("no valid enabled LLMs found")
	}
	return llms, nil
}

