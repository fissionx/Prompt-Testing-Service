package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// SearchKeyword searches for a keyword in all responses and calculates stats on-the-fly
func (p *PostgreSQL) SearchKeyword(ctx context.Context, keyword string, startTime, endTime *time.Time) (*models.KeywordStats, error) {
	start := time.Now()
	query := `
		SELECT prompt_id, llm_id, llm_provider, response_text, created_at
		FROM responses
		WHERE to_tsvector('english', response_text) @@ plainto_tsquery('english', $1)
	`
	args := []interface{}{keyword}
	argPos := 2

	if startTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, *startTime)
		argPos++
	}
	if endTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, *endTime)
		argPos++
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := &models.KeywordStats{
		Keyword:    keyword,
		ByPrompt:   make(map[string]int),
		ByLLM:      make(map[string]int),
		ByProvider: make(map[string]int),
	}

	promptsSeen := make(map[string]bool)
	llmsSeen := make(map[string]bool)

	for rows.Next() {
		var promptID, llmID, llmProvider sql.NullString
		var responseText string
		var createdAt time.Time

		if err := rows.Scan(&promptID, &llmID, &llmProvider, &responseText, &createdAt); err != nil {
			continue
		}

		// Decompress if needed
		if decompressed, err := shared.DecompressString(responseText); err == nil {
			responseText = decompressed
		}

		pID := ""
		if promptID.Valid {
			pID = promptID.String
		}

		lID := ""
		if llmID.Valid {
			lID = llmID.String
		}

		lProvider := ""
		if llmProvider.Valid {
			lProvider = llmProvider.String
		}

		count := shared.CountOccurrences(responseText, keyword)
		stats.TotalMentions += count

		if pID != "" {
			stats.ByPrompt[pID] += count
			promptsSeen[pID] = true
		}

		if lID != "" {
			stats.ByLLM[lID] += count
			llmsSeen[lID] = true
		}

		if lProvider != "" {
			stats.ByProvider[lProvider] += count
		}

		if stats.FirstSeen.IsZero() || createdAt.Before(stats.FirstSeen) {
			stats.FirstSeen = createdAt
		}
		if stats.LastSeen.IsZero() || createdAt.After(stats.LastSeen) {
			stats.LastSeen = createdAt
		}
	}

	stats.UniquePrompts = len(promptsSeen)
	stats.UniqueLLMs = len(llmsSeen)

	trackLatency(ctx, "SearchKeyword", "responses", start, nil)
	return stats, nil
}

// GetTopKeywords returns the most common keywords across all responses
func (p *PostgreSQL) GetTopKeywords(ctx context.Context, limit int, startTime, endTime *time.Time) ([]models.KeywordCount, error) {
	start := time.Now()
	query := "SELECT response_text, created_at FROM responses WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	if startTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, *startTime)
		argPos++
	}
	if endTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, *endTime)
		argPos++
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	wordCounts := make(map[string]int)
	for rows.Next() {
		var responseText string
		var createdAt time.Time

		if err := rows.Scan(&responseText, &createdAt); err != nil {
			continue
		}

		// Decompress response text if needed
		if decompressed, err := shared.DecompressString(responseText); err == nil {
			responseText = decompressed
		}

		words := shared.ExtractCapitalizedWords(responseText)
		for _, word := range words {
			wordCounts[word]++
		}
	}

	// Sort by count
	type kv struct {
		keyword string
		count   int
	}

	var sorted []kv
	for k, v := range wordCounts {
		sorted = append(sorted, kv{k, v})
	}

	// Simple bubble sort (for small datasets)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].count > sorted[i].count {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	var results []models.KeywordCount
	for i := 0; i < len(sorted) && i < limit; i++ {
		results = append(results, models.KeywordCount{
			Keyword: sorted[i].keyword,
			Count:   sorted[i].count,
		})
	}

	trackLatency(ctx, "GetTopKeywords", "responses", start, nil)
	return results, nil
}

