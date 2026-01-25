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
			id, org_id, brand_id, prompt_id, response_id, llm_id, type, title, description,
			current_state, source_evidence, impact_score, urgency, effort_estimate,
			status, content_hash, action_id, is_archived, metadata, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
		ON CONFLICT (prompt_id, content_hash) DO UPDATE SET
			response_id = EXCLUDED.response_id,
			llm_id = EXCLUDED.llm_id,
			impact_score = EXCLUDED.impact_score,
			current_state = EXCLUDED.current_state,
			source_evidence = EXCLUDED.source_evidence,
			urgency = EXCLUDED.urgency,
			effort_estimate = EXCLUDED.effort_estimate,
			updated_at = EXCLUDED.updated_at
		WHERE opportunities.status = 'open'
	`

	metadataJSON := mapToJSON(opportunity.Metadata)

	_, err := p.db.ExecContext(ctx, query,
		opportunity.ID,
		nullString(opportunity.OrgID),
		opportunity.BrandID,
		opportunity.PromptID,
		nullString(opportunity.ResponseID),
		nullString(opportunity.LLMID),
		string(opportunity.Type),
		opportunity.Title,
		opportunity.Description,
		nullString(opportunity.CurrentState),
		nullString(opportunity.SourceEvidence),
		opportunity.ImpactScore,
		nullString(opportunity.Urgency),
		nullString(opportunity.EffortEstimate),
		string(opportunity.Status),
		opportunity.ContentHash,
		nullString(opportunity.ActionID),
		opportunity.IsArchived,
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
		SELECT id, org_id, brand_id, prompt_id, response_id, llm_id, type, title, description,
		       current_state, source_evidence, impact_score, urgency, effort_estimate,
		       status, content_hash, action_id, is_archived, metadata, created_at, updated_at
		FROM opportunities WHERE id = $1
	`

	var opportunity models.Opportunity
	var orgID, responseID, llmID, actionID sql.NullString
	var currentState, sourceEvidence, urgency, effortEstimate sql.NullString
	var metadataJSON string
	var typeStr, statusStr string

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&opportunity.ID,
		&orgID,
		&opportunity.BrandID,
		&opportunity.PromptID,
		&responseID,
		&llmID,
		&typeStr,
		&opportunity.Title,
		&opportunity.Description,
		&currentState,
		&sourceEvidence,
		&opportunity.ImpactScore,
		&urgency,
		&effortEstimate,
		&statusStr,
		&opportunity.ContentHash,
		&actionID,
		&opportunity.IsArchived,
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

	opportunity.OrgID = orgID.String
	opportunity.ResponseID = responseID.String
	opportunity.LLMID = llmID.String
	opportunity.ActionID = actionID.String
	opportunity.CurrentState = currentState.String
	opportunity.SourceEvidence = sourceEvidence.String
	opportunity.Urgency = urgency.String
	opportunity.EffortEstimate = effortEstimate.String
	opportunity.Type = models.OpportunityType(typeStr)
	opportunity.Status = models.OpportunityStatus(statusStr)
	opportunity.Metadata = jsonToMap(metadataJSON)

	trackLatency(ctx, "GetOpportunity", "opportunities", start, nil)
	return &opportunity, nil
}

// ListOpportunities lists opportunities with filters
// Returns both active and archived opportunities (filtered by isArchived if specified)
func (p *PostgreSQL) ListOpportunities(ctx context.Context, filter models.OpportunityFilter) ([]*models.Opportunity, error) {
	start := time.Now()

	query := `
		SELECT id, org_id, brand_id, prompt_id, response_id, llm_id, type, title, description,
		       current_state, source_evidence, impact_score, urgency, effort_estimate,
		       status, content_hash, action_id, is_archived, metadata, created_at, updated_at
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

	query += " ORDER BY is_archived ASC, impact_score DESC, created_at DESC"

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
		var orgID, responseID, llmID, actionID sql.NullString
		var currentState, sourceEvidence, urgency, effortEstimate sql.NullString
		var metadataJSON string
		var typeStr, statusStr string

		err := rows.Scan(
			&opportunity.ID,
			&orgID,
			&opportunity.BrandID,
			&opportunity.PromptID,
			&responseID,
			&llmID,
			&typeStr,
			&opportunity.Title,
			&opportunity.Description,
			&currentState,
			&sourceEvidence,
			&opportunity.ImpactScore,
			&urgency,
			&effortEstimate,
			&statusStr,
			&opportunity.ContentHash,
			&actionID,
			&opportunity.IsArchived,
			&metadataJSON,
			&opportunity.CreatedAt,
			&opportunity.UpdatedAt,
		)
		if err != nil {
			trackLatency(ctx, "ListOpportunities", "opportunities", start, err)
			return nil, err
		}

		opportunity.OrgID = orgID.String
		opportunity.ResponseID = responseID.String
		opportunity.LLMID = llmID.String
		opportunity.ActionID = actionID.String
		opportunity.CurrentState = currentState.String
		opportunity.SourceEvidence = sourceEvidence.String
		opportunity.Urgency = urgency.String
		opportunity.EffortEstimate = effortEstimate.String
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
		SET org_id = $1, brand_id = $2, prompt_id = $3, response_id = $4, llm_id = $5, type = $6, title = $7,
		    description = $8, current_state = $9, source_evidence = $10, impact_score = $11,
		    urgency = $12, effort_estimate = $13, status = $14, content_hash = $15,
		    action_id = $16, is_archived = $17, metadata = $18, updated_at = $19
		WHERE id = $20
	`

	metadataJSON := mapToJSON(opportunity.Metadata)

	result, err := p.db.ExecContext(ctx, query,
		nullString(opportunity.OrgID),
		opportunity.BrandID,
		opportunity.PromptID,
		nullString(opportunity.ResponseID),
		nullString(opportunity.LLMID),
		string(opportunity.Type),
		opportunity.Title,
		opportunity.Description,
		nullString(opportunity.CurrentState),
		nullString(opportunity.SourceEvidence),
		opportunity.ImpactScore,
		nullString(opportunity.Urgency),
		nullString(opportunity.EffortEstimate),
		string(opportunity.Status),
		opportunity.ContentHash,
		nullString(opportunity.ActionID),
		opportunity.IsArchived,
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
		SELECT id, org_id, brand_id, prompt_id, response_id, llm_id, type, title, description,
		       current_state, source_evidence, impact_score, urgency, effort_estimate,
		       status, content_hash, action_id, is_archived, metadata, created_at, updated_at
		FROM opportunities WHERE prompt_id = $1 AND content_hash = $2
	`

	var opportunity models.Opportunity
	var orgID, responseID, llmID, actionID sql.NullString
	var currentState, sourceEvidence, urgency, effortEstimate sql.NullString
	var metadataJSON string
	var typeStr, statusStr string

	err := p.db.QueryRowContext(ctx, query, promptID, contentHash).Scan(
		&opportunity.ID,
		&orgID,
		&opportunity.BrandID,
		&opportunity.PromptID,
		&responseID,
		&llmID,
		&typeStr,
		&opportunity.Title,
		&opportunity.Description,
		&currentState,
		&sourceEvidence,
		&opportunity.ImpactScore,
		&urgency,
		&effortEstimate,
		&statusStr,
		&opportunity.ContentHash,
		&actionID,
		&opportunity.IsArchived,
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

	opportunity.OrgID = orgID.String
	opportunity.ResponseID = responseID.String
	opportunity.LLMID = llmID.String
	opportunity.ActionID = actionID.String
	opportunity.CurrentState = currentState.String
	opportunity.SourceEvidence = sourceEvidence.String
	opportunity.Urgency = urgency.String
	opportunity.EffortEstimate = effortEstimate.String
	opportunity.Type = models.OpportunityType(typeStr)
	opportunity.Status = models.OpportunityStatus(statusStr)
	opportunity.Metadata = jsonToMap(metadataJSON)

	trackLatency(ctx, "GetOpportunityByContentHash", "opportunities", start, nil)
	return &opportunity, nil
}

// GetOpportunityByPromptAndType finds an opportunity by prompt ID and type
// Returns nil if not found (used for checking if type already exists for this prompt)
// Only returns non-archived opportunities for the "one per type per prompt" rule
func (p *PostgreSQL) GetOpportunityByPromptAndType(ctx context.Context, promptID, oppType string) (*models.Opportunity, error) {
	start := time.Now()
	query := `
		SELECT id, org_id, brand_id, prompt_id, response_id, llm_id, type, title, description,
		       current_state, source_evidence, impact_score, urgency, effort_estimate,
		       status, content_hash, action_id, is_archived, metadata, created_at, updated_at
		FROM opportunities WHERE prompt_id = $1 AND type = $2 AND is_archived = FALSE
		ORDER BY created_at DESC LIMIT 1
	`

	var opportunity models.Opportunity
	var orgID, responseID, llmID, actionID sql.NullString
	var currentState, sourceEvidence, urgency, effortEstimate sql.NullString
	var metadataJSON string
	var typeStr, statusStr string

	err := p.db.QueryRowContext(ctx, query, promptID, oppType).Scan(
		&opportunity.ID,
		&orgID,
		&opportunity.BrandID,
		&opportunity.PromptID,
		&responseID,
		&llmID,
		&typeStr,
		&opportunity.Title,
		&opportunity.Description,
		&currentState,
		&sourceEvidence,
		&opportunity.ImpactScore,
		&urgency,
		&effortEstimate,
		&statusStr,
		&opportunity.ContentHash,
		&actionID,
		&opportunity.IsArchived,
		&metadataJSON,
		&opportunity.CreatedAt,
		&opportunity.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		trackLatency(ctx, "GetOpportunityByPromptAndType", "opportunities", start, nil)
		return nil, nil // Not found is not an error for this function
	}
	if err != nil {
		trackLatency(ctx, "GetOpportunityByPromptAndType", "opportunities", start, err)
		return nil, err
	}

	opportunity.OrgID = orgID.String
	opportunity.ResponseID = responseID.String
	opportunity.LLMID = llmID.String
	opportunity.ActionID = actionID.String
	opportunity.CurrentState = currentState.String
	opportunity.SourceEvidence = sourceEvidence.String
	opportunity.Urgency = urgency.String
	opportunity.EffortEstimate = effortEstimate.String
	opportunity.Type = models.OpportunityType(typeStr)
	opportunity.Status = models.OpportunityStatus(statusStr)
	opportunity.Metadata = jsonToMap(metadataJSON)

	trackLatency(ctx, "GetOpportunityByPromptAndType", "opportunities", start, nil)
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
		case "open":
			summary.OpenOpportunities = count
		case "in_progress":
			summary.InProgressOpportunities = count
		case "completed":
			summary.CompletedOpportunities = count
		case "dismissed":
			summary.DismissedOpportunities = count
		}
	}

	// Get counts by type
	typeQuery := `
		SELECT type, COUNT(*) as count
		FROM opportunities
		WHERE brand_id = $1 AND status != 'dismissed'
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
			id, org_id, opportunity_id, brand_id, title, description, steps,
			estimated_effort, resources, status, assignee, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = p.db.ExecContext(ctx, query,
		action.ID,
		action.OrgID,
		action.OpportunityID,
		action.BrandID,
		action.Title,
		action.Description,
		string(stepsJSON),
		action.EstimatedEffort,
		resourcesJSON,
		string(action.Status),
		action.Assignee,
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
		SELECT id, org_id, opportunity_id, brand_id, title, description, steps,
		       estimated_effort, resources, status, assignee, created_at, updated_at
		FROM actions WHERE id = $1
	`

	var action models.Action
	var stepsJSON, resourcesJSON string
	var statusStr string
	var assignee sql.NullString

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&action.ID,
		&action.OrgID,
		&action.OpportunityID,
		&action.BrandID,
		&action.Title,
		&action.Description,
		&stepsJSON,
		&action.EstimatedEffort,
		&resourcesJSON,
		&statusStr,
		&assignee,
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
	if assignee.Valid {
		action.Assignee = assignee.String
	}

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
		SELECT id, org_id, opportunity_id, brand_id, title, description, steps,
		       estimated_effort, resources, status, assignee, created_at, updated_at
		FROM actions WHERE opportunity_id = $1
	`

	var action models.Action
	var stepsJSON, resourcesJSON string
	var statusStr string
	var assignee sql.NullString

	err := p.db.QueryRowContext(ctx, query, opportunityID).Scan(
		&action.ID,
		&action.OrgID,
		&action.OpportunityID,
		&action.BrandID,
		&action.Title,
		&action.Description,
		&stepsJSON,
		&action.EstimatedEffort,
		&resourcesJSON,
		&statusStr,
		&assignee,
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
	if assignee.Valid {
		action.Assignee = assignee.String
	}

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
		SELECT id, org_id, opportunity_id, brand_id, title, description, steps,
		       estimated_effort, resources, status, assignee, created_at, updated_at
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
		var assignee sql.NullString

		err := rows.Scan(
			&action.ID,
			&action.OrgID,
			&action.OpportunityID,
			&action.BrandID,
			&action.Title,
			&action.Description,
			&stepsJSON,
			&action.EstimatedEffort,
			&resourcesJSON,
			&statusStr,
			&assignee,
			&action.CreatedAt,
			&action.UpdatedAt,
		)
		if err != nil {
			trackLatency(ctx, "ListActions", "actions", start, err)
			return nil, err
		}

		action.Status = models.ActionStatus(statusStr)
		action.Resources = jsonToSlice(resourcesJSON)
		if assignee.Valid {
			action.Assignee = assignee.String
		}

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
		SET org_id = $1, opportunity_id = $2, brand_id = $3, title = $4, description = $5,
		    steps = $6, estimated_effort = $7, resources = $8, status = $9, assignee = $10, updated_at = $11
		WHERE id = $12
	`

	result, err := p.db.ExecContext(ctx, query,
		action.OrgID,
		action.OpportunityID,
		action.BrandID,
		action.Title,
		action.Description,
		string(stepsJSON),
		action.EstimatedEffort,
		resourcesJSON,
		string(action.Status),
		action.Assignee,
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

// ============================================================================
// OPPORTUNITY EMBEDDING OPERATIONS
// ============================================================================

// OpportunityEmbedding represents a stored opportunity embedding for semantic matching
type OpportunityEmbedding struct {
	ID              string    `json:"id"`
	BrandID         string    `json:"brandId"`
	OpportunityID   string    `json:"opportunityId"`
	OpportunityType string    `json:"opportunityType"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	NormalizedText  string    `json:"normalizedText"`
	KeyConcepts     []string  `json:"keyConcepts"`
	ActionVerbs     []string  `json:"actionVerbs"`
	TargetEntities  []string  `json:"targetEntities"`
	EmbeddingVector []float64 `json:"embeddingVector"`
	CreatedAt       time.Time `json:"createdAt"`
}

// CreateOpportunityEmbedding stores an opportunity embedding for semantic deduplication
func (p *PostgreSQL) CreateOpportunityEmbedding(ctx context.Context, emb *OpportunityEmbedding) error {
	start := time.Now()
	emb.CreatedAt = time.Now()

	query := `
		INSERT INTO opportunity_embeddings (
			id, brand_id, opportunity_id, opportunity_type, title, description,
			normalized_text, key_concepts, action_verbs, target_entities, embedding_vector, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			normalized_text = EXCLUDED.normalized_text,
			key_concepts = EXCLUDED.key_concepts,
			action_verbs = EXCLUDED.action_verbs,
			target_entities = EXCLUDED.target_entities,
			embedding_vector = EXCLUDED.embedding_vector
	`

	_, err := p.db.ExecContext(ctx, query,
		emb.ID,
		emb.BrandID,
		emb.OpportunityID,
		emb.OpportunityType,
		emb.Title,
		emb.Description,
		emb.NormalizedText,
		sliceToPostgresArray(emb.KeyConcepts),
		sliceToPostgresArray(emb.ActionVerbs),
		sliceToPostgresArray(emb.TargetEntities),
		float64SliceToPostgresArray(emb.EmbeddingVector),
		emb.CreatedAt,
	)

	trackLatency(ctx, "CreateOpportunityEmbedding", "opportunity_embeddings", start, err)
	return err
}

// ListOpportunityEmbeddingsByBrand retrieves all embeddings for a brand
func (p *PostgreSQL) ListOpportunityEmbeddingsByBrand(ctx context.Context, brandID string, limit int) ([]*OpportunityEmbedding, error) {
	start := time.Now()

	query := `
		SELECT id, brand_id, opportunity_id, opportunity_type, title, description,
		       normalized_text, key_concepts, action_verbs, target_entities, embedding_vector, created_at
		FROM opportunity_embeddings
		WHERE brand_id = $1
		ORDER BY created_at DESC
	`

	args := []interface{}{brandID}
	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		trackLatency(ctx, "ListOpportunityEmbeddingsByBrand", "opportunity_embeddings", start, err)
		return nil, err
	}
	defer rows.Close()

	var embeddings []*OpportunityEmbedding
	for rows.Next() {
		emb := &OpportunityEmbedding{}
		var keyConcepts, actionVerbs, targetEntities, embeddingVector string

		err := rows.Scan(
			&emb.ID,
			&emb.BrandID,
			&emb.OpportunityID,
			&emb.OpportunityType,
			&emb.Title,
			&emb.Description,
			&emb.NormalizedText,
			&keyConcepts,
			&actionVerbs,
			&targetEntities,
			&embeddingVector,
			&emb.CreatedAt,
		)
		if err != nil {
			trackLatency(ctx, "ListOpportunityEmbeddingsByBrand", "opportunity_embeddings", start, err)
			return nil, err
		}

		emb.KeyConcepts = postgresArrayToSlice(keyConcepts)
		emb.ActionVerbs = postgresArrayToSlice(actionVerbs)
		emb.TargetEntities = postgresArrayToSlice(targetEntities)
		emb.EmbeddingVector = postgresArrayToFloat64Slice(embeddingVector)

		embeddings = append(embeddings, emb)
	}

	trackLatency(ctx, "ListOpportunityEmbeddingsByBrand", "opportunity_embeddings", start, nil)
	return embeddings, nil
}

// DeleteOpportunityEmbedding deletes an embedding by ID
func (p *PostgreSQL) DeleteOpportunityEmbedding(ctx context.Context, id string) error {
	start := time.Now()
	query := "DELETE FROM opportunity_embeddings WHERE id = $1"
	_, err := p.db.ExecContext(ctx, query, id)
	trackLatency(ctx, "DeleteOpportunityEmbedding", "opportunity_embeddings", start, err)
	return err
}

// DeleteOpportunityEmbeddingsByBrand deletes all embeddings for a brand
func (p *PostgreSQL) DeleteOpportunityEmbeddingsByBrand(ctx context.Context, brandID string) error {
	start := time.Now()
	query := "DELETE FROM opportunity_embeddings WHERE brand_id = $1"
	_, err := p.db.ExecContext(ctx, query, brandID)
	trackLatency(ctx, "DeleteOpportunityEmbeddingsByBrand", "opportunity_embeddings", start, err)
	return err
}

// Helper function to convert string slice to PostgreSQL array format
func sliceToPostgresArray(arr []string) string {
	if len(arr) == 0 {
		return "{}"
	}

	escaped := make([]string, len(arr))
	for i, s := range arr {
		// Escape special characters for PostgreSQL array
		s = strings.ReplaceAll(s, "\\", "\\\\")
		s = strings.ReplaceAll(s, "\"", "\\\"")
		escaped[i] = "\"" + s + "\""
	}
	return "{" + strings.Join(escaped, ",") + "}"
}

// Helper function to convert float64 slice to PostgreSQL array format
func float64SliceToPostgresArray(arr []float64) string {
	if len(arr) == 0 {
		return "{}"
	}

	parts := make([]string, len(arr))
	for i, v := range arr {
		parts[i] = fmt.Sprintf("%v", v)
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// Helper function to convert PostgreSQL array to string slice
func postgresArrayToSlice(pgArray string) []string {
	if pgArray == "" || pgArray == "{}" {
		return nil
	}

	// Remove braces
	pgArray = strings.TrimPrefix(pgArray, "{")
	pgArray = strings.TrimSuffix(pgArray, "}")

	if pgArray == "" {
		return nil
	}

	// Parse the array - handle quoted elements
	var result []string
	var current strings.Builder
	inQuote := false
	escape := false

	for _, c := range pgArray {
		if escape {
			current.WriteRune(c)
			escape = false
			continue
		}

		switch c {
		case '\\':
			escape = true
		case '"':
			inQuote = !inQuote
		case ',':
			if inQuote {
				current.WriteRune(c)
			} else {
				if s := current.String(); s != "" {
					result = append(result, s)
				}
				current.Reset()
			}
		default:
			current.WriteRune(c)
		}
	}

	if s := current.String(); s != "" {
		result = append(result, s)
	}

	return result
}

// Helper function to convert PostgreSQL array to float64 slice
func postgresArrayToFloat64Slice(pgArray string) []float64 {
	if pgArray == "" || pgArray == "{}" {
		return nil
	}

	pgArray = strings.TrimPrefix(pgArray, "{")
	pgArray = strings.TrimSuffix(pgArray, "}")

	if pgArray == "" {
		return nil
	}

	parts := strings.Split(pgArray, ",")
	result := make([]float64, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		var v float64
		fmt.Sscanf(p, "%f", &v)
		result = append(result, v)
	}

	return result
}
