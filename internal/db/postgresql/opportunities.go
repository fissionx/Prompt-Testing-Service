package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fissionx/gego/internal/models"
)

// CreateOpportunity creates a new opportunity
func (p *PostgreSQL) CreateOpportunity(ctx context.Context, opportunity *models.Opportunity) error {
	start := time.Now()
	opportunity.CreatedAt = time.Now()
	opportunity.UpdatedAt = time.Now()

	query := `
		INSERT INTO opportunities (
			id, brand_id, prompt_id, response_id, type, title, description,
			impact_score, status, content_hash, action_id, metadata, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (prompt_id, content_hash) DO UPDATE SET
			response_id = EXCLUDED.response_id,
			impact_score = EXCLUDED.impact_score,
			updated_at = EXCLUDED.updated_at
		WHERE opportunities.status = 'new'
	`

	metadataJSON := mapToJSON(opportunity.Metadata)

	_, err := p.db.ExecContext(ctx, query,
		opportunity.ID,
		opportunity.BrandID,
		opportunity.PromptID,
		nullString(opportunity.ResponseID),
		string(opportunity.Type),
		opportunity.Title,
		opportunity.Description,
		opportunity.ImpactScore,
		string(opportunity.Status),
		opportunity.ContentHash,
		nullString(opportunity.ActionID),
		metadataJSON,
		opportunity.CreatedAt,
		opportunity.UpdatedAt,
	)

	trackLatency(ctx, "CreateOpportunity", "opportunities", start, err)
	return err
}

// GetOpportunity retrieves an opportunity by ID
func (p *PostgreSQL) GetOpportunity(ctx context.Context, id string) (*models.Opportunity, error) {
	start := time.Now()
	query := `
		SELECT id, brand_id, prompt_id, response_id, type, title, description,
		       impact_score, status, content_hash, action_id, metadata, created_at, updated_at
		FROM opportunities WHERE id = $1
	`

	var opportunity models.Opportunity
	var responseID, actionID sql.NullString
	var metadataJSON string
	var typeStr, statusStr string

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&opportunity.ID,
		&opportunity.BrandID,
		&opportunity.PromptID,
		&responseID,
		&typeStr,
		&opportunity.Title,
		&opportunity.Description,
		&opportunity.ImpactScore,
		&statusStr,
		&opportunity.ContentHash,
		&actionID,
		&metadataJSON,
		&opportunity.CreatedAt,
		&opportunity.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		trackLatency(ctx, "GetOpportunity", "opportunities", start, fmt.Errorf("opportunity not found: %s", id))
		return nil, fmt.Errorf("opportunity not found: %s", id)
	}
	if err != nil {
		trackLatency(ctx, "GetOpportunity", "opportunities", start, err)
		return nil, err
	}

	opportunity.ResponseID = responseID.String
	opportunity.ActionID = actionID.String
	opportunity.Type = models.OpportunityType(typeStr)
	opportunity.Status = models.OpportunityStatus(statusStr)
	opportunity.Metadata = jsonToMap(metadataJSON)

	trackLatency(ctx, "GetOpportunity", "opportunities", start, nil)
	return &opportunity, nil
}

// ListOpportunities lists opportunities with filters
func (p *PostgreSQL) ListOpportunities(ctx context.Context, filter models.OpportunityFilter) ([]*models.Opportunity, error) {
	start := time.Now()

	query := `
		SELECT id, brand_id, prompt_id, response_id, type, title, description,
		       impact_score, status, content_hash, action_id, metadata, created_at, updated_at
		FROM opportunities WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if filter.BrandID != "" {
		query += fmt.Sprintf(" AND brand_id = $%d", argPos)
		args = append(args, filter.BrandID)
		argPos++
	}

	if filter.PromptID != "" {
		query += fmt.Sprintf(" AND prompt_id = $%d", argPos)
		args = append(args, filter.PromptID)
		argPos++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filter.Status)
		argPos++
	}

	if filter.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argPos)
		args = append(args, filter.Type)
		argPos++
	}

	if filter.MinImpact > 0 {
		query += fmt.Sprintf(" AND impact_score >= $%d", argPos)
		args = append(args, filter.MinImpact)
		argPos++
	}

	query += " ORDER BY impact_score DESC, created_at DESC"

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

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		trackLatency(ctx, "ListOpportunities", "opportunities", start, err)
		return nil, err
	}
	defer rows.Close()

	var opportunities []*models.Opportunity
	for rows.Next() {
		var opportunity models.Opportunity
		var responseID, actionID sql.NullString
		var metadataJSON string
		var typeStr, statusStr string

		err := rows.Scan(
			&opportunity.ID,
			&opportunity.BrandID,
			&opportunity.PromptID,
			&responseID,
			&typeStr,
			&opportunity.Title,
			&opportunity.Description,
			&opportunity.ImpactScore,
			&statusStr,
			&opportunity.ContentHash,
			&actionID,
			&metadataJSON,
			&opportunity.CreatedAt,
			&opportunity.UpdatedAt,
		)
		if err != nil {
			trackLatency(ctx, "ListOpportunities", "opportunities", start, err)
			return nil, err
		}

		opportunity.ResponseID = responseID.String
		opportunity.ActionID = actionID.String
		opportunity.Type = models.OpportunityType(typeStr)
		opportunity.Status = models.OpportunityStatus(statusStr)
		opportunity.Metadata = jsonToMap(metadataJSON)

		opportunities = append(opportunities, &opportunity)
	}

	trackLatency(ctx, "ListOpportunities", "opportunities", start, nil)
	return opportunities, nil
}

// CountOpportunities counts opportunities with filters
func (p *PostgreSQL) CountOpportunities(ctx context.Context, filter models.OpportunityFilter) (int64, error) {
	start := time.Now()

	query := `SELECT COUNT(*) FROM opportunities WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	if filter.BrandID != "" {
		query += fmt.Sprintf(" AND brand_id = $%d", argPos)
		args = append(args, filter.BrandID)
		argPos++
	}

	if filter.PromptID != "" {
		query += fmt.Sprintf(" AND prompt_id = $%d", argPos)
		args = append(args, filter.PromptID)
		argPos++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filter.Status)
		argPos++
	}

	if filter.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argPos)
		args = append(args, filter.Type)
		argPos++
	}

	if filter.MinImpact > 0 {
		query += fmt.Sprintf(" AND impact_score >= $%d", argPos)
		args = append(args, filter.MinImpact)
		argPos++
	}

	var count int64
	err := p.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		trackLatency(ctx, "CountOpportunities", "opportunities", start, err)
		return 0, err
	}

	trackLatency(ctx, "CountOpportunities", "opportunities", start, nil)
	return count, nil
}

// UpdateOpportunity updates an existing opportunity
func (p *PostgreSQL) UpdateOpportunity(ctx context.Context, opportunity *models.Opportunity) error {
	start := time.Now()
	opportunity.UpdatedAt = time.Now()

	query := `
		UPDATE opportunities 
		SET brand_id = $1, prompt_id = $2, response_id = $3, type = $4, title = $5,
		    description = $6, impact_score = $7, status = $8, content_hash = $9,
		    action_id = $10, metadata = $11, updated_at = $12
		WHERE id = $13
	`

	metadataJSON := mapToJSON(opportunity.Metadata)

	result, err := p.db.ExecContext(ctx, query,
		opportunity.BrandID,
		opportunity.PromptID,
		nullString(opportunity.ResponseID),
		string(opportunity.Type),
		opportunity.Title,
		opportunity.Description,
		opportunity.ImpactScore,
		string(opportunity.Status),
		opportunity.ContentHash,
		nullString(opportunity.ActionID),
		metadataJSON,
		opportunity.UpdatedAt,
		opportunity.ID,
	)

	if err != nil {
		trackLatency(ctx, "UpdateOpportunity", "opportunities", start, err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		trackLatency(ctx, "UpdateOpportunity", "opportunities", start, err)
		return err
	}

	if rowsAffected == 0 {
		err := fmt.Errorf("opportunity not found: %s", opportunity.ID)
		trackLatency(ctx, "UpdateOpportunity", "opportunities", start, err)
		return err
	}

	trackLatency(ctx, "UpdateOpportunity", "opportunities", start, nil)
	return nil
}

// DeleteOpportunity deletes an opportunity by ID
func (p *PostgreSQL) DeleteOpportunity(ctx context.Context, id string) error {
	start := time.Now()
	query := "DELETE FROM opportunities WHERE id = $1"
	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		trackLatency(ctx, "DeleteOpportunity", "opportunities", start, err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		trackLatency(ctx, "DeleteOpportunity", "opportunities", start, err)
		return err
	}

	if rowsAffected == 0 {
		err := fmt.Errorf("opportunity not found: %s", id)
		trackLatency(ctx, "DeleteOpportunity", "opportunities", start, err)
		return err
	}

	trackLatency(ctx, "DeleteOpportunity", "opportunities", start, nil)
	return nil
}

// GetOpportunityByContentHash finds an opportunity by prompt ID and content hash
func (p *PostgreSQL) GetOpportunityByContentHash(ctx context.Context, promptID, contentHash string) (*models.Opportunity, error) {
	start := time.Now()
	query := `
		SELECT id, brand_id, prompt_id, response_id, type, title, description,
		       impact_score, status, content_hash, action_id, metadata, created_at, updated_at
		FROM opportunities WHERE prompt_id = $1 AND content_hash = $2
	`

	var opportunity models.Opportunity
	var responseID, actionID sql.NullString
	var metadataJSON string
	var typeStr, statusStr string

	err := p.db.QueryRowContext(ctx, query, promptID, contentHash).Scan(
		&opportunity.ID,
		&opportunity.BrandID,
		&opportunity.PromptID,
		&responseID,
		&typeStr,
		&opportunity.Title,
		&opportunity.Description,
		&opportunity.ImpactScore,
		&statusStr,
		&opportunity.ContentHash,
		&actionID,
		&metadataJSON,
		&opportunity.CreatedAt,
		&opportunity.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		trackLatency(ctx, "GetOpportunityByContentHash", "opportunities", start, nil)
		return nil, nil // Not found is not an error for this function
	}
	if err != nil {
		trackLatency(ctx, "GetOpportunityByContentHash", "opportunities", start, err)
		return nil, err
	}

	opportunity.ResponseID = responseID.String
	opportunity.ActionID = actionID.String
	opportunity.Type = models.OpportunityType(typeStr)
	opportunity.Status = models.OpportunityStatus(statusStr)
	opportunity.Metadata = jsonToMap(metadataJSON)

	trackLatency(ctx, "GetOpportunityByContentHash", "opportunities", start, nil)
	return &opportunity, nil
}

// GetOpportunitySummary returns a summary of opportunities for a brand
func (p *PostgreSQL) GetOpportunitySummary(ctx context.Context, brandID string) (*models.OpportunitySummary, error) {
	start := time.Now()

	summary := &models.OpportunitySummary{
		ByType: make(map[string]int),
	}

	// Get counts by status
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM opportunities
		WHERE brand_id = $1
		GROUP BY status
	`
	rows, err := p.db.QueryContext(ctx, statusQuery, brandID)
	if err != nil {
		trackLatency(ctx, "GetOpportunitySummary", "opportunities", start, err)
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			trackLatency(ctx, "GetOpportunitySummary", "opportunities", start, err)
			return nil, err
		}
		summary.TotalOpportunities += count
		switch status {
		case "new":
			summary.NewOpportunities = count
		case "in_progress":
			summary.InProgressOpportunities = count
		case "completed":
			summary.CompletedOpportunities = count
		case "archived":
			summary.ArchivedOpportunities = count
		}
	}

	// Get counts by type
	typeQuery := `
		SELECT type, COUNT(*) as count
		FROM opportunities
		WHERE brand_id = $1 AND status != 'archived'
		GROUP BY type
	`
	typeRows, err := p.db.QueryContext(ctx, typeQuery, brandID)
	if err != nil {
		trackLatency(ctx, "GetOpportunitySummary", "opportunities", start, err)
		return nil, err
	}
	defer typeRows.Close()

	for typeRows.Next() {
		var oppType string
		var count int
		if err := typeRows.Scan(&oppType, &count); err != nil {
			trackLatency(ctx, "GetOpportunitySummary", "opportunities", start, err)
			return nil, err
		}
		summary.ByType[oppType] = count
	}

	// Get average impact score
	avgQuery := `
		SELECT COALESCE(AVG(impact_score), 0)
		FROM opportunities
		WHERE brand_id = $1 AND status != 'archived'
	`
	err = p.db.QueryRowContext(ctx, avgQuery, brandID).Scan(&summary.AvgImpactScore)
	if err != nil {
		trackLatency(ctx, "GetOpportunitySummary", "opportunities", start, err)
		return nil, err
	}

	// Get top 5 opportunities by impact score
	topQuery := `
		SELECT id, brand_id, prompt_id, response_id, type, title, description,
		       impact_score, status, content_hash, action_id, metadata, created_at, updated_at
		FROM opportunities
		WHERE brand_id = $1 AND status NOT IN ('completed', 'archived')
		ORDER BY impact_score DESC
		LIMIT 5
	`
	topRows, err := p.db.QueryContext(ctx, topQuery, brandID)
	if err != nil {
		trackLatency(ctx, "GetOpportunitySummary", "opportunities", start, err)
		return nil, err
	}
	defer topRows.Close()

	for topRows.Next() {
		var opportunity models.Opportunity
		var responseID, actionID sql.NullString
		var metadataJSON string
		var typeStr, statusStr string

		err := topRows.Scan(
			&opportunity.ID,
			&opportunity.BrandID,
			&opportunity.PromptID,
			&responseID,
			&typeStr,
			&opportunity.Title,
			&opportunity.Description,
			&opportunity.ImpactScore,
			&statusStr,
			&opportunity.ContentHash,
			&actionID,
			&metadataJSON,
			&opportunity.CreatedAt,
			&opportunity.UpdatedAt,
		)
		if err != nil {
			trackLatency(ctx, "GetOpportunitySummary", "opportunities", start, err)
			return nil, err
		}

		opportunity.ResponseID = responseID.String
		opportunity.ActionID = actionID.String
		opportunity.Type = models.OpportunityType(typeStr)
		opportunity.Status = models.OpportunityStatus(statusStr)
		opportunity.Metadata = jsonToMap(metadataJSON)

		summary.TopOpportunities = append(summary.TopOpportunities, opportunity)
	}

	trackLatency(ctx, "GetOpportunitySummary", "opportunities", start, nil)
	return summary, nil
}

// ==================== Action Operations ====================

// CreateAction creates a new action
func (p *PostgreSQL) CreateAction(ctx context.Context, action *models.Action) error {
	start := time.Now()
	action.CreatedAt = time.Now()
	action.UpdatedAt = time.Now()

	stepsJSON, err := json.Marshal(action.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}

	resourcesJSON := sliceToJSON(action.Resources)

	query := `
		INSERT INTO actions (
			id, opportunity_id, brand_id, title, description, steps,
			estimated_effort, resources, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = p.db.ExecContext(ctx, query,
		action.ID,
		action.OpportunityID,
		action.BrandID,
		action.Title,
		action.Description,
		string(stepsJSON),
		action.EstimatedEffort,
		resourcesJSON,
		string(action.Status),
		action.CreatedAt,
		action.UpdatedAt,
	)

	trackLatency(ctx, "CreateAction", "actions", start, err)
	return err
}

// GetAction retrieves an action by ID
func (p *PostgreSQL) GetAction(ctx context.Context, id string) (*models.Action, error) {
	start := time.Now()
	query := `
		SELECT id, opportunity_id, brand_id, title, description, steps,
		       estimated_effort, resources, status, created_at, updated_at
		FROM actions WHERE id = $1
	`

	var action models.Action
	var stepsJSON, resourcesJSON string
	var statusStr string

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&action.ID,
		&action.OpportunityID,
		&action.BrandID,
		&action.Title,
		&action.Description,
		&stepsJSON,
		&action.EstimatedEffort,
		&resourcesJSON,
		&statusStr,
		&action.CreatedAt,
		&action.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		trackLatency(ctx, "GetAction", "actions", start, fmt.Errorf("action not found: %s", id))
		return nil, fmt.Errorf("action not found: %s", id)
	}
	if err != nil {
		trackLatency(ctx, "GetAction", "actions", start, err)
		return nil, err
	}

	action.Status = models.ActionStatus(statusStr)
	action.Resources = jsonToSlice(resourcesJSON)

	// Parse steps
	if stepsJSON != "" && stepsJSON != "[]" {
		if err := json.Unmarshal([]byte(stepsJSON), &action.Steps); err != nil {
			trackLatency(ctx, "GetAction", "actions", start, err)
			return nil, fmt.Errorf("failed to parse steps: %w", err)
		}
	}

	trackLatency(ctx, "GetAction", "actions", start, nil)
	return &action, nil
}

// GetActionByOpportunityID retrieves an action by opportunity ID
func (p *PostgreSQL) GetActionByOpportunityID(ctx context.Context, opportunityID string) (*models.Action, error) {
	start := time.Now()
	query := `
		SELECT id, opportunity_id, brand_id, title, description, steps,
		       estimated_effort, resources, status, created_at, updated_at
		FROM actions WHERE opportunity_id = $1
	`

	var action models.Action
	var stepsJSON, resourcesJSON string
	var statusStr string

	err := p.db.QueryRowContext(ctx, query, opportunityID).Scan(
		&action.ID,
		&action.OpportunityID,
		&action.BrandID,
		&action.Title,
		&action.Description,
		&stepsJSON,
		&action.EstimatedEffort,
		&resourcesJSON,
		&statusStr,
		&action.CreatedAt,
		&action.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		trackLatency(ctx, "GetActionByOpportunityID", "actions", start, nil)
		return nil, nil // Not found is not an error
	}
	if err != nil {
		trackLatency(ctx, "GetActionByOpportunityID", "actions", start, err)
		return nil, err
	}

	action.Status = models.ActionStatus(statusStr)
	action.Resources = jsonToSlice(resourcesJSON)

	if stepsJSON != "" && stepsJSON != "[]" {
		if err := json.Unmarshal([]byte(stepsJSON), &action.Steps); err != nil {
			trackLatency(ctx, "GetActionByOpportunityID", "actions", start, err)
			return nil, fmt.Errorf("failed to parse steps: %w", err)
		}
	}

	trackLatency(ctx, "GetActionByOpportunityID", "actions", start, nil)
	return &action, nil
}

// ListActions lists actions with filters
func (p *PostgreSQL) ListActions(ctx context.Context, filter models.ActionFilter) ([]*models.Action, error) {
	start := time.Now()

	query := `
		SELECT id, opportunity_id, brand_id, title, description, steps,
		       estimated_effort, resources, status, created_at, updated_at
		FROM actions WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	if filter.BrandID != "" {
		query += fmt.Sprintf(" AND brand_id = $%d", argPos)
		args = append(args, filter.BrandID)
		argPos++
	}

	if filter.OpportunityID != "" {
		query += fmt.Sprintf(" AND opportunity_id = $%d", argPos)
		args = append(args, filter.OpportunityID)
		argPos++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filter.Status)
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

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		trackLatency(ctx, "ListActions", "actions", start, err)
		return nil, err
	}
	defer rows.Close()

	var actions []*models.Action
	for rows.Next() {
		var action models.Action
		var stepsJSON, resourcesJSON string
		var statusStr string

		err := rows.Scan(
			&action.ID,
			&action.OpportunityID,
			&action.BrandID,
			&action.Title,
			&action.Description,
			&stepsJSON,
			&action.EstimatedEffort,
			&resourcesJSON,
			&statusStr,
			&action.CreatedAt,
			&action.UpdatedAt,
		)
		if err != nil {
			trackLatency(ctx, "ListActions", "actions", start, err)
			return nil, err
		}

		action.Status = models.ActionStatus(statusStr)
		action.Resources = jsonToSlice(resourcesJSON)

		if stepsJSON != "" && stepsJSON != "[]" {
			if err := json.Unmarshal([]byte(stepsJSON), &action.Steps); err != nil {
				trackLatency(ctx, "ListActions", "actions", start, err)
				return nil, fmt.Errorf("failed to parse steps: %w", err)
			}
		}

		actions = append(actions, &action)
	}

	trackLatency(ctx, "ListActions", "actions", start, nil)
	return actions, nil
}

// CountActions counts actions with filters
func (p *PostgreSQL) CountActions(ctx context.Context, filter models.ActionFilter) (int64, error) {
	start := time.Now()

	query := `SELECT COUNT(*) FROM actions WHERE 1=1`
	args := []interface{}{}
	argPos := 1

	if filter.BrandID != "" {
		query += fmt.Sprintf(" AND brand_id = $%d", argPos)
		args = append(args, filter.BrandID)
		argPos++
	}

	if filter.OpportunityID != "" {
		query += fmt.Sprintf(" AND opportunity_id = $%d", argPos)
		args = append(args, filter.OpportunityID)
		argPos++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filter.Status)
		argPos++
	}

	var count int64
	err := p.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		trackLatency(ctx, "CountActions", "actions", start, err)
		return 0, err
	}

	trackLatency(ctx, "CountActions", "actions", start, nil)
	return count, nil
}

// UpdateAction updates an existing action
func (p *PostgreSQL) UpdateAction(ctx context.Context, action *models.Action) error {
	start := time.Now()
	action.UpdatedAt = time.Now()

	stepsJSON, err := json.Marshal(action.Steps)
	if err != nil {
		return fmt.Errorf("failed to marshal steps: %w", err)
	}

	resourcesJSON := sliceToJSON(action.Resources)

	query := `
		UPDATE actions 
		SET opportunity_id = $1, brand_id = $2, title = $3, description = $4,
		    steps = $5, estimated_effort = $6, resources = $7, status = $8, updated_at = $9
		WHERE id = $10
	`

	result, err := p.db.ExecContext(ctx, query,
		action.OpportunityID,
		action.BrandID,
		action.Title,
		action.Description,
		string(stepsJSON),
		action.EstimatedEffort,
		resourcesJSON,
		string(action.Status),
		action.UpdatedAt,
		action.ID,
	)

	if err != nil {
		trackLatency(ctx, "UpdateAction", "actions", start, err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		trackLatency(ctx, "UpdateAction", "actions", start, err)
		return err
	}

	if rowsAffected == 0 {
		err := fmt.Errorf("action not found: %s", action.ID)
		trackLatency(ctx, "UpdateAction", "actions", start, err)
		return err
	}

	trackLatency(ctx, "UpdateAction", "actions", start, nil)
	return nil
}

// DeleteAction deletes an action by ID
func (p *PostgreSQL) DeleteAction(ctx context.Context, id string) error {
	start := time.Now()
	query := "DELETE FROM actions WHERE id = $1"
	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		trackLatency(ctx, "DeleteAction", "actions", start, err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		trackLatency(ctx, "DeleteAction", "actions", start, err)
		return err
	}

	if rowsAffected == 0 {
		err := fmt.Errorf("action not found: %s", id)
		trackLatency(ctx, "DeleteAction", "actions", start, err)
		return err
	}

	trackLatency(ctx, "DeleteAction", "actions", start, nil)
	return nil
}

// Helper function for nullable strings
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// normalizeForHash normalizes a string for hashing (lowercase, trim, remove extra spaces)
func normalizeForHash(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// Replace multiple spaces with single space
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}
