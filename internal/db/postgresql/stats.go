package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fissionx/gego/internal/models"
)

// GetPromptStats calculates prompt statistics on-demand from responses
func (p *PostgreSQL) GetPromptStats(ctx context.Context, promptID string) (*models.PromptStats, error) {
	start := time.Now()
	// Get aggregate stats
	query := `
		SELECT 
			COUNT(*) as total_responses,
			COALESCE(AVG(tokens_used), 0) as avg_tokens,
			COUNT(DISTINCT llm_id) as unique_llms
		FROM responses
		WHERE prompt_id = $1
	`

	var stats models.PromptStats
	stats.PromptID = promptID

	err := p.db.QueryRowContext(ctx, query, promptID).Scan(
		&stats.TotalResponses,
		&stats.AvgTokens,
		&stats.UniqueLLMs,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get prompt stats: %w", err)
	}

	// Get LLM counts
	llmCounts, err := p.getLLMCountsForPrompt(ctx, promptID)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM counts: %w", err)
	}

	stats.LLMCounts = llmCounts
	stats.UpdatedAt = time.Now()

	trackLatency(ctx, "GetPromptStats", "responses", start, nil)
	return &stats, nil
}

// GetLLMStats calculates LLM statistics on-demand from responses
func (p *PostgreSQL) GetLLMStats(ctx context.Context, llmID string) (*models.LLMStats, error) {
	start := time.Now()
	// Get aggregate stats
	query := `
		SELECT 
			COUNT(*) as total_responses,
			COALESCE(AVG(tokens_used), 0) as avg_tokens,
			COUNT(DISTINCT prompt_id) as unique_prompts
		FROM responses
		WHERE llm_id = $1
	`

	var stats models.LLMStats
	stats.LLMID = llmID

	err := p.db.QueryRowContext(ctx, query, llmID).Scan(
		&stats.TotalResponses,
		&stats.AvgTokens,
		&stats.UniquePrompts,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get LLM stats: %w", err)
	}

	// Get prompt counts
	promptCounts, err := p.getPromptCountsForLLM(ctx, llmID)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt counts: %w", err)
	}

	stats.PromptCounts = promptCounts
	stats.UpdatedAt = time.Now()

	trackLatency(ctx, "GetLLMStats", "responses", start, nil)
	return &stats, nil
}

// getLLMCountsForPrompt gets the count of responses by LLM for a specific prompt
func (p *PostgreSQL) getLLMCountsForPrompt(ctx context.Context, promptID string) (map[string]int, error) {
	query := `
		SELECT llm_id, COUNT(*) as count
		FROM responses
		WHERE prompt_id = $1 AND llm_id IS NOT NULL
		GROUP BY llm_id
	`

	rows, err := p.db.QueryContext(ctx, query, promptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var llmID string
		var count int
		if err := rows.Scan(&llmID, &count); err != nil {
			continue
		}
		counts[llmID] = count
	}

	return counts, nil
}

// getPromptCountsForLLM gets the count of responses by prompt for a specific LLM
func (p *PostgreSQL) getPromptCountsForLLM(ctx context.Context, llmID string) (map[string]int, error) {
	query := `
		SELECT prompt_id, COUNT(*) as count
		FROM responses
		WHERE llm_id = $1 AND prompt_id IS NOT NULL
		GROUP BY prompt_id
	`

	rows, err := p.db.QueryContext(ctx, query, llmID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var promptID string
		var count int
		if err := rows.Scan(&promptID, &count); err != nil {
			continue
		}
		counts[promptID] = count
	}

	return counts, nil
}

