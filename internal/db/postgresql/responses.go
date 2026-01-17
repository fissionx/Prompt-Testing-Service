package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// CreateResponse creates a new response
func (p *PostgreSQL) CreateResponse(ctx context.Context, response *models.Response) error {
	start := time.Now()
	response.CreatedAt = time.Now()

	// Compress large text fields to save storage space
	compressedResponseText := response.ResponseText
	compressedPromptText := response.PromptText

	if shared.ShouldCompress(response.ResponseText) {
		if compressed, err := shared.CompressString(response.ResponseText); err == nil {
			compressedResponseText = compressed
		}
	}

	if shared.ShouldCompress(response.PromptText) {
		if compressed, err := shared.CompressString(response.PromptText); err == nil {
			compressedPromptText = compressed
		}
	}

	query := `
		INSERT INTO responses (
			id, prompt_id, prompt_text, llm_id, llm_name, llm_provider, llm_model,
			response_text, brand, temperature, metadata, schedule_id, tokens_used,
			latency_ms, error, visibility_score, brand_mentioned, in_grounding_sources,
			grounding_sources, sentiment, competitors_mention, brand_position,
			total_brands_listed, grounding_domains, web_search_queries, web_search_calls,
			search_answer, week, month, quarter, region, language, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30,
			$31, $32, $33
		)
	`

	metadataJSON := "{}"
	if response.Metadata != nil {
		metadataJSON = mapToJSON(response.Metadata)
	}

	_, err := p.db.ExecContext(ctx, query,
		response.ID,
		response.PromptID,
		compressedPromptText,
		response.LLMID,
		response.LLMName,
		response.LLMProvider,
		response.LLMModel,
		compressedResponseText,
		response.Brand,
		response.Temperature,
		metadataJSON,
		response.ScheduleID,
		response.TokensUsed,
		response.LatencyMs,
		response.Error,
		response.VisibilityScore,
		response.BrandMentioned,
		response.InGroundingSources,
		sliceToJSON(response.GroundingSources),
		response.Sentiment,
		sliceToJSON(response.CompetitorsMention),
		response.BrandPosition,
		response.TotalBrandsListed,
		sliceToJSON(response.GroundingDomains),
		sliceToJSON(response.WebSearchQueries),
		webSearchCallsToJSON(response.WebSearchCalls),
		response.SearchAnswer,
		response.Week,
		response.Month,
		response.Quarter,
		response.Region,
		response.Language,
		response.CreatedAt,
	)

	trackLatency(ctx, "CreateResponse", "responses", start, err)
	return err
}

// Helper to convert WebSearchCallDetails slice to JSON
func webSearchCallsToJSON(calls []models.WebSearchCallDetails) string {
	if len(calls) == 0 {
		return "[]"
	}
	data, err := json.Marshal(calls)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// Helper to convert JSON to WebSearchCallDetails slice
func jsonToWebSearchCalls(jsonStr string) []models.WebSearchCallDetails {
	if jsonStr == "" || jsonStr == "[]" || jsonStr == "null" {
		return []models.WebSearchCallDetails{}
	}
	var result []models.WebSearchCallDetails
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return []models.WebSearchCallDetails{}
	}
	return result
}

// GetResponse retrieves a response by ID
func (p *PostgreSQL) GetResponse(ctx context.Context, id string) (*models.Response, error) {
	start := time.Now()
	query := `
		SELECT id, prompt_id, prompt_text, llm_id, llm_name, llm_provider, llm_model,
			response_text, brand, temperature, metadata, schedule_id, tokens_used,
			latency_ms, error, visibility_score, brand_mentioned, in_grounding_sources,
			grounding_sources, sentiment, competitors_mention, brand_position,
			total_brands_listed, grounding_domains, web_search_queries, web_search_calls,
			search_answer, week, month, quarter, region, language, created_at
		FROM responses WHERE id = $1
	`

	var response models.Response
	var metadataJSON, groundingSourcesJSON, competitorsMentionJSON, groundingDomainsJSON,
		webSearchQueriesJSON, webSearchCallsJSON sql.NullString

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&response.ID,
		&response.PromptID,
		&response.PromptText,
		&response.LLMID,
		&response.LLMName,
		&response.LLMProvider,
		&response.LLMModel,
		&response.ResponseText,
		&response.Brand,
		&response.Temperature,
		&metadataJSON,
		&response.ScheduleID,
		&response.TokensUsed,
		&response.LatencyMs,
		&response.Error,
		&response.VisibilityScore,
		&response.BrandMentioned,
		&response.InGroundingSources,
		&groundingSourcesJSON,
		&response.Sentiment,
		&competitorsMentionJSON,
		&response.BrandPosition,
		&response.TotalBrandsListed,
		&groundingDomainsJSON,
		&webSearchQueriesJSON,
		&webSearchCallsJSON,
		&response.SearchAnswer,
		&response.Week,
		&response.Month,
		&response.Quarter,
		&response.Region,
		&response.Language,
		&response.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("response not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	if metadataJSON.Valid {
		response.Metadata = jsonToMap(metadataJSON.String)
	}
	if groundingSourcesJSON.Valid {
		response.GroundingSources = jsonToSlice(groundingSourcesJSON.String)
	}
	if competitorsMentionJSON.Valid {
		response.CompetitorsMention = jsonToSlice(competitorsMentionJSON.String)
	}
	if groundingDomainsJSON.Valid {
		response.GroundingDomains = jsonToSlice(groundingDomainsJSON.String)
	}
	if webSearchQueriesJSON.Valid {
		response.WebSearchQueries = jsonToSlice(webSearchQueriesJSON.String)
	}
	if webSearchCallsJSON.Valid {
		response.WebSearchCalls = jsonToWebSearchCalls(webSearchCallsJSON.String)
	}

	// Decompress text fields if they were compressed
	if decompressed, err := shared.DecompressString(response.ResponseText); err == nil {
		response.ResponseText = decompressed
	}
	if decompressed, err := shared.DecompressString(response.PromptText); err == nil {
		response.PromptText = decompressed
	}

	trackLatency(ctx, "GetResponse", "responses", start, nil)
	return &response, nil
}

// ListResponses lists responses with filtering
func (p *PostgreSQL) ListResponses(ctx context.Context, filter shared.ResponseFilter) ([]*models.Response, error) {
	start := time.Now()
	query := `SELECT id, prompt_id, prompt_text, llm_id, llm_name, llm_provider, llm_model, 
		response_text, brand, temperature, metadata, schedule_id, tokens_used, latency_ms, error, 
		visibility_score, brand_mentioned, in_grounding_sources, grounding_sources, sentiment, 
		competitors_mention, brand_position, total_brands_listed, grounding_domains, web_search_queries, 
		web_search_calls, search_answer, week, month, quarter, region, language, created_at 
		FROM responses WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	// Handle prompt ID filtering - prefer batch (PromptIDs) over single (PromptID)
	if len(filter.PromptIDs) > 0 {
		query += fmt.Sprintf(" AND prompt_id = ANY($%d)", argPos)
		args = append(args, pq.Array(filter.PromptIDs))
		argPos++
	} else if filter.PromptID != "" {
		query += fmt.Sprintf(" AND prompt_id = $%d", argPos)
		args = append(args, filter.PromptID)
		argPos++
	}

	// Handle LLM ID filtering - prefer batch (LLMIDs) over single (LLMID)
	if len(filter.LLMIDs) > 0 {
		query += fmt.Sprintf(" AND llm_id = ANY($%d)", argPos)
		args = append(args, pq.Array(filter.LLMIDs))
		argPos++
	} else if filter.LLMID != "" {
		query += fmt.Sprintf(" AND llm_id = $%d", argPos)
		args = append(args, filter.LLMID)
		argPos++
	}

	// Filter by brand if provided
	if filter.Brand != "" {
		query += fmt.Sprintf(" AND brand = $%d", argPos)
		args = append(args, filter.Brand)
		argPos++
	}

	if filter.ScheduleID != "" {
		query += fmt.Sprintf(" AND schedule_id = $%d", argPos)
		args = append(args, filter.ScheduleID)
		argPos++
	}
	if filter.Keyword != "" {
		query += fmt.Sprintf(" AND to_tsvector('english', response_text) @@ plainto_tsquery('english', $%d)", argPos)
		args = append(args, filter.Keyword)
		argPos++
	}
	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, *filter.StartTime)
		argPos++
	}
	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, *filter.EndTime)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argPos)
		args = append(args, filter.Offset)
		argPos++
	}

	// Use QueryContext directly without prepared statement caching issues
	// Ensure args is properly formatted for the query
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var responses []*models.Response
	for rows.Next() {
		var response models.Response
		var metadataJSON, groundingSourcesJSON, competitorsMentionJSON, groundingDomainsJSON,
			webSearchQueriesJSON, webSearchCallsJSON sql.NullString

		err := rows.Scan(
			&response.ID,
			&response.PromptID,
			&response.PromptText,
			&response.LLMID,
			&response.LLMName,
			&response.LLMProvider,
			&response.LLMModel,
			&response.ResponseText,
			&response.Brand,
			&response.Temperature,
			&metadataJSON,
			&response.ScheduleID,
			&response.TokensUsed,
			&response.LatencyMs,
			&response.Error,
			&response.VisibilityScore,
			&response.BrandMentioned,
			&response.InGroundingSources,
			&groundingSourcesJSON,
			&response.Sentiment,
			&competitorsMentionJSON,
			&response.BrandPosition,
			&response.TotalBrandsListed,
			&groundingDomainsJSON,
			&webSearchQueriesJSON,
			&webSearchCallsJSON,
			&response.SearchAnswer,
			&response.Week,
			&response.Month,
			&response.Quarter,
			&response.Region,
			&response.Language,
			&response.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if metadataJSON.Valid {
			response.Metadata = jsonToMap(metadataJSON.String)
		}
		if groundingSourcesJSON.Valid {
			response.GroundingSources = jsonToSlice(groundingSourcesJSON.String)
		}
		if competitorsMentionJSON.Valid {
			response.CompetitorsMention = jsonToSlice(competitorsMentionJSON.String)
		}
		if groundingDomainsJSON.Valid {
			response.GroundingDomains = jsonToSlice(groundingDomainsJSON.String)
		}
		if webSearchQueriesJSON.Valid {
			response.WebSearchQueries = jsonToSlice(webSearchQueriesJSON.String)
		}
		if webSearchCallsJSON.Valid {
			response.WebSearchCalls = jsonToWebSearchCalls(webSearchCallsJSON.String)
		}

		// Decompress text fields if they were compressed
		if decompressed, err := shared.DecompressString(response.ResponseText); err == nil {
			response.ResponseText = decompressed
		}
		if decompressed, err := shared.DecompressString(response.PromptText); err == nil {
			response.PromptText = decompressed
		}

		responses = append(responses, &response)
	}

	trackLatency(ctx, "ListResponses", "responses", start, nil)
	return responses, nil
}

// CountResponses counts responses matching the filter
func (p *PostgreSQL) CountResponses(ctx context.Context, filter shared.ResponseFilter) (int64, error) {
	start := time.Now()
	query := "SELECT COUNT(*) FROM responses WHERE 1=1"
	args := []interface{}{}
	argPos := 1

	// Handle prompt ID filtering - prefer batch (PromptIDs) over single (PromptID)
	if len(filter.PromptIDs) > 0 {
		query += fmt.Sprintf(" AND prompt_id = ANY($%d)", argPos)
		args = append(args, pq.Array(filter.PromptIDs))
		argPos++
	} else if filter.PromptID != "" {
		query += fmt.Sprintf(" AND prompt_id = $%d", argPos)
		args = append(args, filter.PromptID)
		argPos++
	}

	// Handle LLM ID filtering - prefer batch (LLMIDs) over single (LLMID)
	if len(filter.LLMIDs) > 0 {
		query += fmt.Sprintf(" AND llm_id = ANY($%d)", argPos)
		args = append(args, pq.Array(filter.LLMIDs))
		argPos++
	} else if filter.LLMID != "" {
		query += fmt.Sprintf(" AND llm_id = $%d", argPos)
		args = append(args, filter.LLMID)
		argPos++
	}

	// Filter by brand if provided
	if filter.Brand != "" {
		query += fmt.Sprintf(" AND brand = $%d", argPos)
		args = append(args, filter.Brand)
		argPos++
	}

	if filter.ScheduleID != "" {
		query += fmt.Sprintf(" AND schedule_id = $%d", argPos)
		args = append(args, filter.ScheduleID)
		argPos++
	}
	if filter.Keyword != "" {
		query += fmt.Sprintf(" AND to_tsvector('english', response_text) @@ plainto_tsquery('english', $%d)", argPos)
		args = append(args, filter.Keyword)
		argPos++
	}
	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, *filter.StartTime)
		argPos++
	}
	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, *filter.EndTime)
		argPos++
	}

	var count int64
	err := p.db.QueryRowContext(ctx, query, args...).Scan(&count)
	trackLatency(ctx, "CountResponses", "responses", start, err)
	return count, err
}

// DeleteAllResponses deletes all responses from the database
func (p *PostgreSQL) DeleteAllResponses(ctx context.Context) (int, error) {
	start := time.Now()
	query := "DELETE FROM responses"
	result, err := p.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		trackLatency(ctx, "DeleteAllResponses", "responses", start, err)
		return 0, err
	}

	trackLatency(ctx, "DeleteAllResponses", "responses", start, nil)
	return int(rowsAffected), nil
}

