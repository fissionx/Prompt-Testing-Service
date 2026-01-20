package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/fissionx/gego/internal/models"
)

// ==================== Schedule Operations ====================

func (p *PostgreSQL) CreateSchedule(ctx context.Context, schedule *models.Schedule) error {
	now := time.Now()
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = now
	}
	schedule.UpdatedAt = now

	query := `
		INSERT INTO schedules (id, brand_id, org_id, name, prompt_ids, llm_ids, cron_expr, temperature, enabled, last_run, next_run, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			brand_id = EXCLUDED.brand_id,
			org_id = EXCLUDED.org_id,
			name = EXCLUDED.name,
			prompt_ids = EXCLUDED.prompt_ids,
			llm_ids = EXCLUDED.llm_ids,
			cron_expr = EXCLUDED.cron_expr,
			temperature = EXCLUDED.temperature,
			enabled = EXCLUDED.enabled,
			last_run = EXCLUDED.last_run,
			next_run = EXCLUDED.next_run,
			updated_at = EXCLUDED.updated_at
	`

	_, err := p.db.ExecContext(ctx, query,
		schedule.ID,
		schedule.BrandID,
		schedule.OrgID,
		schedule.Name,
		sliceToJSON(schedule.PromptIDs),
		sliceToJSON(schedule.LLMIDs),
		schedule.CronExpr,
		schedule.Temperature,
		schedule.Enabled,
		schedule.LastRun,
		schedule.NextRun,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	)

	return err
}

func (p *PostgreSQL) GetSchedule(ctx context.Context, id string) (*models.Schedule, error) {
	query := `
		SELECT id, brand_id, org_id, name, prompt_ids, llm_ids, cron_expr, temperature, enabled, last_run, next_run, created_at, updated_at
		FROM schedules WHERE id = $1
	`

	var schedule models.Schedule
	var promptIDsJSON, llmIDsJSON string
	var lastRun, nextRun sql.NullTime

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&schedule.ID,
		&schedule.BrandID,
		&schedule.OrgID,
		&schedule.Name,
		&promptIDsJSON,
		&llmIDsJSON,
		&schedule.CronExpr,
		&schedule.Temperature,
		&schedule.Enabled,
		&lastRun,
		&nextRun,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	schedule.PromptIDs = jsonToSlice(promptIDsJSON)
	schedule.LLMIDs = jsonToSlice(llmIDsJSON)
	if lastRun.Valid {
		schedule.LastRun = &lastRun.Time
	}
	if nextRun.Valid {
		schedule.NextRun = &nextRun.Time
	}

	return &schedule, nil
}

func (p *PostgreSQL) ListSchedules(ctx context.Context, brandID string, enabled *bool) ([]*models.Schedule, error) {
	query := `
		SELECT id, brand_id, org_id, name, prompt_ids, llm_ids, cron_expr, temperature, enabled, last_run, next_run, created_at, updated_at
		FROM schedules WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if brandID != "" {
		query += fmt.Sprintf(" AND brand_id = $%d", argPos)
		args = append(args, brandID)
		argPos++
	}

	if enabled != nil {
		query += fmt.Sprintf(" AND enabled = $%d", argPos)
		args = append(args, *enabled)
		argPos++
	}

	query += " ORDER BY created_at DESC"

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []*models.Schedule
	for rows.Next() {
		var schedule models.Schedule
		var promptIDsJSON, llmIDsJSON string
		var lastRun, nextRun sql.NullTime

		err := rows.Scan(
			&schedule.ID,
			&schedule.BrandID,
			&schedule.OrgID,
			&schedule.Name,
			&promptIDsJSON,
			&llmIDsJSON,
			&schedule.CronExpr,
			&schedule.Temperature,
			&schedule.Enabled,
			&lastRun,
			&nextRun,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		schedule.PromptIDs = jsonToSlice(promptIDsJSON)
		schedule.LLMIDs = jsonToSlice(llmIDsJSON)
		if lastRun.Valid {
			schedule.LastRun = &lastRun.Time
		}
		if nextRun.Valid {
			schedule.NextRun = &nextRun.Time
		}

		schedules = append(schedules, &schedule)
	}

	return schedules, nil
}

func (p *PostgreSQL) UpdateSchedule(ctx context.Context, schedule *models.Schedule) error {
	schedule.UpdatedAt = time.Now()

	query := `
		UPDATE schedules 
		SET brand_id = $1, org_id = $2, name = $3, prompt_ids = $4, llm_ids = $5, cron_expr = $6, temperature = $7, enabled = $8, last_run = $9, next_run = $10, updated_at = $11
		WHERE id = $12
	`

	result, err := p.db.ExecContext(ctx, query,
		schedule.BrandID,
		schedule.OrgID,
		schedule.Name,
		sliceToJSON(schedule.PromptIDs),
		sliceToJSON(schedule.LLMIDs),
		schedule.CronExpr,
		schedule.Temperature,
		schedule.Enabled,
		schedule.LastRun,
		schedule.NextRun,
		schedule.UpdatedAt,
		schedule.ID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found: %s", schedule.ID)
	}

	return nil
}

func (p *PostgreSQL) DeleteSchedule(ctx context.Context, id string) error {
	query := "DELETE FROM schedules WHERE id = $1"
	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found: %s", id)
	}

	return nil
}

func (p *PostgreSQL) DeleteAllSchedules(ctx context.Context) (int, error) {
	query := "DELETE FROM schedules"
	result, err := p.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}
