package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fissionx/gego/internal/models"
)

// ==================== Cached GEO Insights Operations ====================

func (p *PostgreSQL) SaveCachedGEOInsights(ctx context.Context, insights *models.CachedGEOInsights) error {
	now := time.Now()
	if insights.CreatedAt.IsZero() {
		insights.CreatedAt = now
	}
	insights.UpdatedAt = now

	// Store the entire struct in JSONB for complex nested data
	dataJSON := structToJSONB(insights)

	query := `
		INSERT INTO cached_geo_insights (id, campaign_id, brand_id, org_id, brand, start_time, end_time, data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			campaign_id = EXCLUDED.campaign_id,
			brand_id = EXCLUDED.brand_id,
			org_id = EXCLUDED.org_id,
			brand = EXCLUDED.brand,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			data = EXCLUDED.data,
			updated_at = EXCLUDED.updated_at
	`

	_, err := p.db.ExecContext(ctx, query,
		insights.ID,
		insights.CampaignID,
		insights.BrandID,
		insights.OrgID,
		insights.Brand,
		insights.StartTime,
		insights.EndTime,
		dataJSON,
		insights.CreatedAt,
		insights.UpdatedAt,
	)

	return err
}

func (p *PostgreSQL) GetCachedGEOInsights(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedGEOInsights, error) {
	sqlQuery := `
		SELECT id, campaign_id, brand_id, org_id, brand, start_time, end_time, data, created_at, updated_at
		FROM cached_geo_insights
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if query.CampaignID != "" {
		sqlQuery += fmt.Sprintf(" AND campaign_id = $%d", argPos)
		args = append(args, query.CampaignID)
		argPos++
	}
	if query.BrandID != "" {
		sqlQuery += fmt.Sprintf(" AND brand_id = $%d", argPos)
		args = append(args, query.BrandID)
		argPos++
	}
	if query.StartTime != nil {
		sqlQuery += fmt.Sprintf(" AND start_time <= $%d", argPos)
		args = append(args, *query.StartTime)
		argPos++
	}
	if query.EndTime != nil {
		sqlQuery += fmt.Sprintf(" AND end_time >= $%d", argPos)
		args = append(args, *query.EndTime)
		argPos++
	}

	sqlQuery += " ORDER BY updated_at DESC LIMIT 1"

	var insights models.CachedGEOInsights
	var dataJSON string

	err := p.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&insights.ID,
		&insights.CampaignID,
		&insights.BrandID,
		&insights.OrgID,
		&insights.Brand,
		&insights.StartTime,
		&insights.EndTime,
		&dataJSON,
		&insights.CreatedAt,
		&insights.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached geo insights: %w", err)
	}

	// Deserialize the full struct from JSONB
	if err := jsonBToStruct(dataJSON, &insights); err != nil {
		return nil, fmt.Errorf("failed to deserialize cached geo insights: %w", err)
	}

	return &insights, nil
}

func (p *PostgreSQL) DeleteCachedGEOInsights(ctx context.Context, id string) error {
	query := "DELETE FROM cached_geo_insights WHERE id = $1"
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgreSQL) DeleteCachedGEOInsightsByBrand(ctx context.Context, brandID string) error {
	query := "DELETE FROM cached_geo_insights WHERE brand_id = $1"
	_, err := p.db.ExecContext(ctx, query, brandID)
	return err
}

// ==================== Cached Source Analytics Operations ====================

func (p *PostgreSQL) SaveCachedSourceAnalytics(ctx context.Context, analytics *models.CachedSourceAnalytics) error {
	now := time.Now()
	if analytics.CreatedAt.IsZero() {
		analytics.CreatedAt = now
	}
	analytics.UpdatedAt = now

	dataJSON := structToJSONB(analytics)

	query := `
		INSERT INTO cached_source_analytics (id, campaign_id, brand_id, org_id, brand, start_time, end_time, data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			campaign_id = EXCLUDED.campaign_id,
			brand_id = EXCLUDED.brand_id,
			org_id = EXCLUDED.org_id,
			brand = EXCLUDED.brand,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			data = EXCLUDED.data,
			updated_at = EXCLUDED.updated_at
	`

	_, err := p.db.ExecContext(ctx, query,
		analytics.ID,
		analytics.CampaignID,
		analytics.BrandID,
		analytics.OrgID,
		analytics.Brand,
		analytics.StartTime,
		analytics.EndTime,
		dataJSON,
		analytics.CreatedAt,
		analytics.UpdatedAt,
	)

	return err
}

func (p *PostgreSQL) GetCachedSourceAnalytics(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedSourceAnalytics, error) {
	sqlQuery := `
		SELECT id, campaign_id, brand_id, org_id, brand, start_time, end_time, data, created_at, updated_at
		FROM cached_source_analytics
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if query.CampaignID != "" {
		sqlQuery += fmt.Sprintf(" AND campaign_id = $%d", argPos)
		args = append(args, query.CampaignID)
		argPos++
	}
	if query.BrandID != "" {
		sqlQuery += fmt.Sprintf(" AND brand_id = $%d", argPos)
		args = append(args, query.BrandID)
		argPos++
	}
	if query.StartTime != nil {
		sqlQuery += fmt.Sprintf(" AND start_time <= $%d", argPos)
		args = append(args, *query.StartTime)
		argPos++
	}
	if query.EndTime != nil {
		sqlQuery += fmt.Sprintf(" AND end_time >= $%d", argPos)
		args = append(args, *query.EndTime)
		argPos++
	}

	sqlQuery += " ORDER BY updated_at DESC LIMIT 1"

	var analytics models.CachedSourceAnalytics
	var dataJSON string

	err := p.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&analytics.ID,
		&analytics.CampaignID,
		&analytics.BrandID,
		&analytics.OrgID,
		&analytics.Brand,
		&analytics.StartTime,
		&analytics.EndTime,
		&dataJSON,
		&analytics.CreatedAt,
		&analytics.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached source analytics: %w", err)
	}

	if err := jsonBToStruct(dataJSON, &analytics); err != nil {
		return nil, fmt.Errorf("failed to deserialize cached source analytics: %w", err)
	}

	return &analytics, nil
}

func (p *PostgreSQL) DeleteCachedSourceAnalytics(ctx context.Context, id string) error {
	query := "DELETE FROM cached_source_analytics WHERE id = $1"
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgreSQL) DeleteCachedSourceAnalyticsByBrand(ctx context.Context, brandID string) error {
	query := "DELETE FROM cached_source_analytics WHERE brand_id = $1"
	_, err := p.db.ExecContext(ctx, query, brandID)
	return err
}

// ==================== Cached Competitive Benchmark Operations ====================

func (p *PostgreSQL) SaveCachedCompetitiveBenchmark(ctx context.Context, benchmark *models.CachedCompetitiveBenchmark) error {
	now := time.Now()
	if benchmark.CreatedAt.IsZero() {
		benchmark.CreatedAt = now
	}
	benchmark.UpdatedAt = now

	dataJSON := structToJSONB(benchmark)

	query := `
		INSERT INTO cached_competitive_benchmark (id, campaign_id, brand_id, org_id, main_brand, start_time, end_time, data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			campaign_id = EXCLUDED.campaign_id,
			brand_id = EXCLUDED.brand_id,
			org_id = EXCLUDED.org_id,
			main_brand = EXCLUDED.main_brand,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			data = EXCLUDED.data,
			updated_at = EXCLUDED.updated_at
	`

	_, err := p.db.ExecContext(ctx, query,
		benchmark.ID,
		benchmark.CampaignID,
		benchmark.BrandID,
		benchmark.OrgID,
		benchmark.MainBrand,
		benchmark.StartTime,
		benchmark.EndTime,
		dataJSON,
		benchmark.CreatedAt,
		benchmark.UpdatedAt,
	)

	return err
}

func (p *PostgreSQL) GetCachedCompetitiveBenchmark(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedCompetitiveBenchmark, error) {
	sqlQuery := `
		SELECT id, campaign_id, brand_id, org_id, main_brand, start_time, end_time, data, created_at, updated_at
		FROM cached_competitive_benchmark
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if query.BrandID != "" {
		sqlQuery += fmt.Sprintf(" AND brand_id = $%d", argPos)
		args = append(args, query.BrandID)
		argPos++
	}

	if query.CampaignID != "" {
		sqlQuery += fmt.Sprintf(" AND campaign_id = $%d", argPos)
		args = append(args, query.CampaignID)
		argPos++
	} else {
		sqlQuery += " AND (campaign_id IS NULL OR campaign_id = '')"
	}

	if query.StartTime != nil && query.EndTime != nil {
		sqlQuery += fmt.Sprintf(" AND start_time <= $%d AND end_time >= $%d", argPos, argPos+1)
		args = append(args, *query.EndTime, *query.StartTime)
		argPos += 2
	} else if query.StartTime != nil {
		sqlQuery += fmt.Sprintf(" AND end_time >= $%d", argPos)
		args = append(args, *query.StartTime)
		argPos++
	} else if query.EndTime != nil {
		sqlQuery += fmt.Sprintf(" AND start_time <= $%d", argPos)
		args = append(args, *query.EndTime)
		argPos++
	}

	sqlQuery += " ORDER BY updated_at DESC LIMIT 1"

	var benchmark models.CachedCompetitiveBenchmark
	var dataJSON string

	err := p.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&benchmark.ID,
		&benchmark.CampaignID,
		&benchmark.BrandID,
		&benchmark.OrgID,
		&benchmark.MainBrand,
		&benchmark.StartTime,
		&benchmark.EndTime,
		&dataJSON,
		&benchmark.CreatedAt,
		&benchmark.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached competitive benchmark: %w", err)
	}

	if err := jsonBToStruct(dataJSON, &benchmark); err != nil {
		return nil, fmt.Errorf("failed to deserialize cached competitive benchmark: %w", err)
	}

	return &benchmark, nil
}

func (p *PostgreSQL) DeleteCachedCompetitiveBenchmark(ctx context.Context, id string) error {
	query := "DELETE FROM cached_competitive_benchmark WHERE id = $1"
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgreSQL) DeleteCachedCompetitiveBenchmarkByBrand(ctx context.Context, brandID string) error {
	query := "DELETE FROM cached_competitive_benchmark WHERE brand_id = $1"
	_, err := p.db.ExecContext(ctx, query, brandID)
	return err
}

// ==================== Cached Prompt Performance Operations ====================

func (p *PostgreSQL) SaveCachedPromptPerformance(ctx context.Context, performance *models.CachedPromptPerformance) error {
	now := time.Now()
	if performance.CreatedAt.IsZero() {
		performance.CreatedAt = now
	}
	performance.UpdatedAt = now

	dataJSON := structToJSONB(performance)

	query := `
		INSERT INTO cached_prompt_performance (id, campaign_id, brand_id, org_id, brand, start_time, end_time, data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			campaign_id = EXCLUDED.campaign_id,
			brand_id = EXCLUDED.brand_id,
			org_id = EXCLUDED.org_id,
			brand = EXCLUDED.brand,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			data = EXCLUDED.data,
			updated_at = EXCLUDED.updated_at
	`

	_, err := p.db.ExecContext(ctx, query,
		performance.ID,
		performance.CampaignID,
		performance.BrandID,
		performance.OrgID,
		performance.Brand,
		performance.StartTime,
		performance.EndTime,
		dataJSON,
		performance.CreatedAt,
		performance.UpdatedAt,
	)

	return err
}

func (p *PostgreSQL) GetCachedPromptPerformance(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedPromptPerformance, error) {
	sqlQuery := `
		SELECT id, campaign_id, brand_id, org_id, brand, start_time, end_time, data, created_at, updated_at
		FROM cached_prompt_performance
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if query.CampaignID != "" {
		sqlQuery += fmt.Sprintf(" AND campaign_id = $%d", argPos)
		args = append(args, query.CampaignID)
		argPos++
	}
	if query.BrandID != "" {
		sqlQuery += fmt.Sprintf(" AND brand_id = $%d", argPos)
		args = append(args, query.BrandID)
		argPos++
	}
	if query.StartTime != nil {
		sqlQuery += fmt.Sprintf(" AND start_time <= $%d", argPos)
		args = append(args, *query.StartTime)
		argPos++
	}
	if query.EndTime != nil {
		sqlQuery += fmt.Sprintf(" AND end_time >= $%d", argPos)
		args = append(args, *query.EndTime)
		argPos++
	}

	sqlQuery += " ORDER BY updated_at DESC LIMIT 1"

	var performance models.CachedPromptPerformance
	var dataJSON string

	err := p.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&performance.ID,
		&performance.CampaignID,
		&performance.BrandID,
		&performance.OrgID,
		&performance.Brand,
		&performance.StartTime,
		&performance.EndTime,
		&dataJSON,
		&performance.CreatedAt,
		&performance.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached prompt performance: %w", err)
	}

	if err := jsonBToStruct(dataJSON, &performance); err != nil {
		return nil, fmt.Errorf("failed to deserialize cached prompt performance: %w", err)
	}

	return &performance, nil
}

func (p *PostgreSQL) DeleteCachedPromptPerformance(ctx context.Context, id string) error {
	query := "DELETE FROM cached_prompt_performance WHERE id = $1"
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgreSQL) DeleteCachedPromptPerformanceByBrand(ctx context.Context, brandID string) error {
	query := "DELETE FROM cached_prompt_performance WHERE brand_id = $1"
	_, err := p.db.ExecContext(ctx, query, brandID)
	return err
}

// ==================== Scheduled Campaign Operations ====================

func (p *PostgreSQL) SaveScheduledCampaign(ctx context.Context, campaign *models.ScheduledCampaign) error {
	now := time.Now()
	if campaign.CreatedAt.IsZero() {
		campaign.CreatedAt = now
	}
	campaign.UpdatedAt = now

	query := `
		INSERT INTO scheduled_campaigns (id, brand_id, org_id, campaign_name, brand, prompt_ids, llm_ids, temperature, schedule_cron, status, total_runs, run_count, last_run_at, next_run_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			brand_id = EXCLUDED.brand_id,
			org_id = EXCLUDED.org_id,
			campaign_name = EXCLUDED.campaign_name,
			brand = EXCLUDED.brand,
			prompt_ids = EXCLUDED.prompt_ids,
			llm_ids = EXCLUDED.llm_ids,
			temperature = EXCLUDED.temperature,
			schedule_cron = EXCLUDED.schedule_cron,
			status = EXCLUDED.status,
			total_runs = EXCLUDED.total_runs,
			run_count = EXCLUDED.run_count,
			last_run_at = EXCLUDED.last_run_at,
			next_run_at = EXCLUDED.next_run_at,
			updated_at = EXCLUDED.updated_at
	`

	_, err := p.db.ExecContext(ctx, query,
		campaign.ID,
		campaign.BrandID,
		campaign.OrgID,
		campaign.CampaignName,
		campaign.Brand,
		sliceToJSON(campaign.PromptIDs),
		sliceToJSON(campaign.LLMIDs),
		campaign.Temperature,
		campaign.ScheduleCron,
		campaign.Status,
		campaign.TotalRuns,
		campaign.RunCount,
		campaign.LastRunAt,
		campaign.NextRunAt,
		campaign.CreatedAt,
		campaign.UpdatedAt,
	)

	return err
}

func (p *PostgreSQL) GetScheduledCampaign(ctx context.Context, id string) (*models.ScheduledCampaign, error) {
	query := `
		SELECT id, brand_id, org_id, campaign_name, brand, prompt_ids, llm_ids, temperature, schedule_cron, status, total_runs, run_count, last_run_at, next_run_at, created_at, updated_at
		FROM scheduled_campaigns
		WHERE id = $1
	`

	var campaign models.ScheduledCampaign
	var promptIDsJSON, llmIDsJSON string
	var lastRunAt, nextRunAt sql.NullTime

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&campaign.ID,
		&campaign.BrandID,
		&campaign.OrgID,
		&campaign.CampaignName,
		&campaign.Brand,
		&promptIDsJSON,
		&llmIDsJSON,
		&campaign.Temperature,
		&campaign.ScheduleCron,
		&campaign.Status,
		&campaign.TotalRuns,
		&campaign.RunCount,
		&lastRunAt,
		&nextRunAt,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	campaign.PromptIDs = jsonToSlice(promptIDsJSON)
	campaign.LLMIDs = jsonToSlice(llmIDsJSON)
	if lastRunAt.Valid {
		campaign.LastRunAt = &lastRunAt.Time
	}
	if nextRunAt.Valid {
		campaign.NextRunAt = &nextRunAt.Time
	}

	return &campaign, nil
}

func (p *PostgreSQL) GetScheduledCampaignByBrand(ctx context.Context, brand string) (*models.ScheduledCampaign, error) {
	query := `
		SELECT id, brand_id, org_id, campaign_name, brand, prompt_ids, llm_ids, temperature, schedule_cron, status, total_runs, run_count, last_run_at, next_run_at, created_at, updated_at
		FROM scheduled_campaigns
		WHERE brand = $1 AND status IN ('active', 'running')
		ORDER BY created_at DESC
		LIMIT 1
	`

	var campaign models.ScheduledCampaign
	var promptIDsJSON, llmIDsJSON string
	var lastRunAt, nextRunAt sql.NullTime

	err := p.db.QueryRowContext(ctx, query, brand).Scan(
		&campaign.ID,
		&campaign.BrandID,
		&campaign.OrgID,
		&campaign.CampaignName,
		&campaign.Brand,
		&promptIDsJSON,
		&llmIDsJSON,
		&campaign.Temperature,
		&campaign.ScheduleCron,
		&campaign.Status,
		&campaign.TotalRuns,
		&campaign.RunCount,
		&lastRunAt,
		&nextRunAt,
		&campaign.CreatedAt,
		&campaign.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	campaign.PromptIDs = jsonToSlice(promptIDsJSON)
	campaign.LLMIDs = jsonToSlice(llmIDsJSON)
	if lastRunAt.Valid {
		campaign.LastRunAt = &lastRunAt.Time
	}
	if nextRunAt.Valid {
		campaign.NextRunAt = &nextRunAt.Time
	}

	return &campaign, nil
}

func (p *PostgreSQL) ListScheduledCampaigns(ctx context.Context, status string) ([]*models.ScheduledCampaign, error) {
	query := `
		SELECT id, brand_id, org_id, campaign_name, brand, prompt_ids, llm_ids, temperature, schedule_cron, status, total_runs, run_count, last_run_at, next_run_at, created_at, updated_at
		FROM scheduled_campaigns
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, status)
		argPos++
	}

	query += " ORDER BY next_run_at ASC NULLS LAST"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var campaigns []*models.ScheduledCampaign
	for rows.Next() {
		var campaign models.ScheduledCampaign
		var promptIDsJSON, llmIDsJSON string
		var lastRunAt, nextRunAt sql.NullTime

		if err := rows.Scan(
			&campaign.ID,
			&campaign.BrandID,
			&campaign.OrgID,
			&campaign.CampaignName,
			&campaign.Brand,
			&promptIDsJSON,
			&llmIDsJSON,
			&campaign.Temperature,
			&campaign.ScheduleCron,
			&campaign.Status,
			&campaign.TotalRuns,
			&campaign.RunCount,
			&lastRunAt,
			&nextRunAt,
			&campaign.CreatedAt,
			&campaign.UpdatedAt,
		); err != nil {
			return nil, err
		}

		campaign.PromptIDs = jsonToSlice(promptIDsJSON)
		campaign.LLMIDs = jsonToSlice(llmIDsJSON)
		if lastRunAt.Valid {
			campaign.LastRunAt = &lastRunAt.Time
		}
		if nextRunAt.Valid {
			campaign.NextRunAt = &nextRunAt.Time
		}

		campaigns = append(campaigns, &campaign)
	}

	return campaigns, nil
}

func (p *PostgreSQL) UpdateScheduledCampaign(ctx context.Context, campaign *models.ScheduledCampaign) error {
	campaign.UpdatedAt = time.Now()

	query := `
		UPDATE scheduled_campaigns
		SET brand_id = $1, org_id = $2, campaign_name = $3, brand = $4, prompt_ids = $5, llm_ids = $6, temperature = $7,
			schedule_cron = $8, status = $9, total_runs = $10, run_count = $11,
			last_run_at = $12, next_run_at = $13, updated_at = $14
		WHERE id = $15
	`

	result, err := p.db.ExecContext(ctx, query,
		campaign.BrandID,
		campaign.OrgID,
		campaign.CampaignName,
		campaign.Brand,
		sliceToJSON(campaign.PromptIDs),
		sliceToJSON(campaign.LLMIDs),
		campaign.Temperature,
		campaign.ScheduleCron,
		campaign.Status,
		campaign.TotalRuns,
		campaign.RunCount,
		campaign.LastRunAt,
		campaign.NextRunAt,
		campaign.UpdatedAt,
		campaign.ID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("scheduled campaign not found: %s", campaign.ID)
	}

	return nil
}

func (p *PostgreSQL) DeleteScheduledCampaign(ctx context.Context, id string) error {
	query := "DELETE FROM scheduled_campaigns WHERE id = $1"
	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("scheduled campaign not found: %s", id)
	}

	return nil
}

// ==================== Cached Prompt Time Series Operations ====================

func (p *PostgreSQL) SaveCachedPromptTimeSeries(ctx context.Context, timeSeries *models.CachedPromptTimeSeries) error {
	now := time.Now()
	if timeSeries.CreatedAt.IsZero() {
		timeSeries.CreatedAt = now
	}
	timeSeries.UpdatedAt = now

	dataJSON := structToJSONB(timeSeries)

	query := `
		INSERT INTO cached_prompt_time_series (id, campaign_id, prompt_id, brand_id, org_id, brand, start_time, end_time, data, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			campaign_id = EXCLUDED.campaign_id,
			prompt_id = EXCLUDED.prompt_id,
			brand_id = EXCLUDED.brand_id,
			org_id = EXCLUDED.org_id,
			brand = EXCLUDED.brand,
			start_time = EXCLUDED.start_time,
			end_time = EXCLUDED.end_time,
			data = EXCLUDED.data,
			updated_at = EXCLUDED.updated_at
	`

	_, err := p.db.ExecContext(ctx, query,
		timeSeries.ID,
		timeSeries.CampaignID,
		timeSeries.PromptID,
		timeSeries.BrandID,
		timeSeries.OrgID,
		timeSeries.Brand,
		timeSeries.StartTime,
		timeSeries.EndTime,
		dataJSON,
		timeSeries.CreatedAt,
		timeSeries.UpdatedAt,
	)

	return err
}

func (p *PostgreSQL) GetCachedPromptTimeSeries(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedPromptTimeSeries, error) {
	sqlQuery := `
		SELECT id, campaign_id, prompt_id, brand_id, org_id, brand, start_time, end_time, data, created_at, updated_at
		FROM cached_prompt_time_series
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if query.PromptID != "" {
		sqlQuery += fmt.Sprintf(" AND prompt_id = $%d", argPos)
		args = append(args, query.PromptID)
		argPos++
	}
	if query.CampaignID != "" {
		sqlQuery += fmt.Sprintf(" AND campaign_id = $%d", argPos)
		args = append(args, query.CampaignID)
		argPos++
	}
	if query.BrandID != "" {
		sqlQuery += fmt.Sprintf(" AND brand_id = $%d", argPos)
		args = append(args, query.BrandID)
		argPos++
	}
	if query.StartTime != nil {
		sqlQuery += fmt.Sprintf(" AND start_time <= $%d", argPos)
		args = append(args, *query.StartTime)
		argPos++
	}
	if query.EndTime != nil {
		sqlQuery += fmt.Sprintf(" AND end_time >= $%d", argPos)
		args = append(args, *query.EndTime)
		argPos++
	}

	sqlQuery += " ORDER BY updated_at DESC LIMIT 1"

	var timeSeries models.CachedPromptTimeSeries
	var dataJSON string

	err := p.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&timeSeries.ID,
		&timeSeries.CampaignID,
		&timeSeries.PromptID,
		&timeSeries.BrandID,
		&timeSeries.OrgID,
		&timeSeries.Brand,
		&timeSeries.StartTime,
		&timeSeries.EndTime,
		&dataJSON,
		&timeSeries.CreatedAt,
		&timeSeries.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached prompt time series: %w", err)
	}

	if err := jsonBToStruct(dataJSON, &timeSeries); err != nil {
		return nil, fmt.Errorf("failed to deserialize cached prompt time series: %w", err)
	}

	return &timeSeries, nil
}

func (p *PostgreSQL) DeleteCachedPromptTimeSeries(ctx context.Context, id string) error {
	query := "DELETE FROM cached_prompt_time_series WHERE id = $1"
	_, err := p.db.ExecContext(ctx, query, id)
	return err
}

func (p *PostgreSQL) DeleteCachedPromptTimeSeriesByBrand(ctx context.Context, brandID string) error {
	query := "DELETE FROM cached_prompt_time_series WHERE brand_id = $1"
	_, err := p.db.ExecContext(ctx, query, brandID)
	return err
}

