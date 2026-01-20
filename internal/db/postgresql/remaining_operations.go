package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fissionx/gego/internal/models"
)

// Helper to serialize any struct to JSONB
func structToJSONB(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// Helper to deserialize JSONB to struct
func jsonBToStruct(jsonStr string, v interface{}) error {
	if jsonStr == "" || jsonStr == "null" {
		return nil
	}
	return json.Unmarshal([]byte(jsonStr), v)
}

// ==================== Prompt Library Operations ====================

func (p *PostgreSQL) CreatePromptLibrary(ctx context.Context, library *models.PromptLibrary) error {
	start := time.Now()
	library.CreatedAt = time.Now()
	library.UpdatedAt = time.Now()

	query := `
		INSERT INTO prompt_library (id, brand, domain, category, prompt_ids, usage_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := p.db.ExecContext(ctx, query,
		library.ID,
		library.Brand,
		library.Domain,
		library.Category,
		sliceToJSON(library.PromptIDs),
		library.UsageCount,
		library.CreatedAt,
		library.UpdatedAt,
	)

	trackLatency(ctx, "CreatePromptLibrary", "prompt_library", start, err)
	return err
}

func (p *PostgreSQL) GetPromptLibrary(ctx context.Context, brand, domain, category string) (*models.PromptLibrary, error) {
	query := `
		SELECT id, brand, domain, category, prompt_ids, usage_count, created_at, updated_at
		FROM prompt_library
		WHERE domain = $1 AND category = $2
	`
	args := []interface{}{domain, category}

	if brand != "" {
		query += " AND brand = $3"
		args = append(args, brand)
	} else {
		query += " AND (brand IS NULL OR brand = '')"
	}

	var library models.PromptLibrary
	var promptIDsJSON string

	err := p.db.QueryRowContext(ctx, query, args...).Scan(
		&library.ID,
		&library.Brand,
		&library.Domain,
		&library.Category,
		&promptIDsJSON,
		&library.UsageCount,
		&library.CreatedAt,
		&library.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	library.PromptIDs = jsonToSlice(promptIDsJSON)
	return &library, nil
}

func (p *PostgreSQL) UpdatePromptLibrary(ctx context.Context, library *models.PromptLibrary) error {
	library.UpdatedAt = time.Now()

	query := `
		UPDATE prompt_library
		SET brand = $1, domain = $2, category = $3, prompt_ids = $4, usage_count = $5, updated_at = $6
		WHERE id = $7
	`

	result, err := p.db.ExecContext(ctx, query,
		library.Brand,
		library.Domain,
		library.Category,
		sliceToJSON(library.PromptIDs),
		library.UsageCount,
		library.UpdatedAt,
		library.ID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("prompt library not found: %s", library.ID)
	}

	return nil
}

func (p *PostgreSQL) ListPromptLibraries(ctx context.Context) ([]*models.PromptLibrary, error) {
	query := `
		SELECT id, brand, domain, category, prompt_ids, usage_count, created_at, updated_at
		FROM prompt_library
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var libraries []*models.PromptLibrary
	for rows.Next() {
		var library models.PromptLibrary
		var promptIDsJSON string

		if err := rows.Scan(
			&library.ID,
			&library.Brand,
			&library.Domain,
			&library.Category,
			&promptIDsJSON,
			&library.UsageCount,
			&library.CreatedAt,
			&library.UpdatedAt,
		); err != nil {
			return nil, err
		}

		library.PromptIDs = jsonToSlice(promptIDsJSON)
		libraries = append(libraries, &library)
	}

	return libraries, nil
}

// ==================== Brand Profile Operations ====================

func (p *PostgreSQL) CreateBrandProfile(ctx context.Context, profile *models.BrandProfile) error {
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	query := `
		INSERT INTO brand_profiles (id, brand_name, domain, category, website, description, competitors, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := p.db.ExecContext(ctx, query,
		profile.ID,
		profile.BrandName,
		profile.Domain,
		profile.Category,
		profile.Website,
		profile.Description,
		sliceToJSON(profile.Competitors),
		profile.CreatedAt,
		profile.UpdatedAt,
	)

	return err
}

func (p *PostgreSQL) GetBrandProfile(ctx context.Context, brandName string) (*models.BrandProfile, error) {
	query := `
		SELECT id, brand_name, domain, category, website, description, competitors, created_at, updated_at
		FROM brand_profiles
		WHERE brand_name = $1
	`

	var profile models.BrandProfile
	var competitorsJSON string

	err := p.db.QueryRowContext(ctx, query, brandName).Scan(
		&profile.ID,
		&profile.BrandName,
		&profile.Domain,
		&profile.Category,
		&profile.Website,
		&profile.Description,
		&competitorsJSON,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	profile.Competitors = jsonToSlice(competitorsJSON)
	return &profile, nil
}

func (p *PostgreSQL) UpdateBrandProfile(ctx context.Context, profile *models.BrandProfile) error {
	profile.UpdatedAt = time.Now()

	query := `
		UPDATE brand_profiles
		SET brand_name = $1, domain = $2, category = $3, website = $4, description = $5, competitors = $6, updated_at = $7
		WHERE id = $8
	`

	result, err := p.db.ExecContext(ctx, query,
		profile.BrandName,
		profile.Domain,
		profile.Category,
		profile.Website,
		profile.Description,
		sliceToJSON(profile.Competitors),
		profile.UpdatedAt,
		profile.ID,
	)

	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("brand profile not found: %s", profile.ID)
	}

	return nil
}

func (p *PostgreSQL) ListBrandProfiles(ctx context.Context) ([]*models.BrandProfile, error) {
	query := `
		SELECT id, brand_name, domain, category, website, description, competitors, created_at, updated_at
		FROM brand_profiles
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*models.BrandProfile
	for rows.Next() {
		var profile models.BrandProfile
		var competitorsJSON string

		if err := rows.Scan(
			&profile.ID,
			&profile.BrandName,
			&profile.Domain,
			&profile.Category,
			&profile.Website,
			&profile.Description,
			&competitorsJSON,
			&profile.CreatedAt,
			&profile.UpdatedAt,
		); err != nil {
			return nil, err
		}

		profile.Competitors = jsonToSlice(competitorsJSON)
		profiles = append(profiles, &profile)
	}

	return profiles, nil
}

// ==================== Brand Logo Operations ====================

func (p *PostgreSQL) SaveBrandLogo(ctx context.Context, logo *models.BrandLogoCache) error {
	logo.UpdatedAt = time.Now()
	if logo.CreatedAt.IsZero() {
		logo.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO brand_logos (id, brand_id, org_id, brand_name, domain, logo_url, fallback_logo_url, source, last_checked, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (brand_id) DO UPDATE SET
			org_id = EXCLUDED.org_id,
			brand_name = EXCLUDED.brand_name,
			domain = EXCLUDED.domain,
			logo_url = EXCLUDED.logo_url,
			fallback_logo_url = EXCLUDED.fallback_logo_url,
			source = EXCLUDED.source,
			last_checked = EXCLUDED.last_checked,
			updated_at = EXCLUDED.updated_at
	`

	_, err := p.db.ExecContext(ctx, query,
		logo.ID,
		logo.BrandID,
		logo.OrgID,
		logo.BrandName,
		logo.Domain,
		logo.LogoURL,
		logo.FallbackLogoURL,
		logo.Source,
		logo.LastChecked,
		logo.CreatedAt,
		logo.UpdatedAt,
	)

	return err
}

func (p *PostgreSQL) GetBrandLogo(ctx context.Context, brandID string) (*models.BrandLogoCache, error) {
	query := `
		SELECT id, brand_id, org_id, brand_name, domain, logo_url, fallback_logo_url, source, last_checked, created_at, updated_at
		FROM brand_logos
		WHERE brand_id = $1
	`

	var logo models.BrandLogoCache
	err := p.db.QueryRowContext(ctx, query, brandID).Scan(
		&logo.ID,
		&logo.BrandID,
		&logo.OrgID,
		&logo.BrandName,
		&logo.Domain,
		&logo.LogoURL,
		&logo.FallbackLogoURL,
		&logo.Source,
		&logo.LastChecked,
		&logo.CreatedAt,
		&logo.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &logo, nil
}

// ==================== Brand Competitors Operations ====================

func (p *PostgreSQL) SaveBrandCompetitors(ctx context.Context, competitors *models.BrandCompetitors) error {
	now := time.Now()
	if competitors.CreatedAt.IsZero() {
		competitors.CreatedAt = now
	}
	competitors.UpdatedAt = now

	query := `
		INSERT INTO brand_competitors (id, brand, brand_id, org_id, competitors, suggested_list, source, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (brand_id) DO UPDATE SET
			org_id = EXCLUDED.org_id,
			competitors = EXCLUDED.competitors,
			suggested_list = EXCLUDED.suggested_list,
			source = EXCLUDED.source,
			updated_at = EXCLUDED.updated_at
	`

	_, err := p.db.ExecContext(ctx, query,
		competitors.ID,
		competitors.Brand,
		competitors.BrandID,
		competitors.OrgID,
		sliceToJSON(competitors.Competitors),
		sliceToJSON(competitors.SuggestedList),
		competitors.Source,
		competitors.CreatedAt,
		competitors.UpdatedAt,
	)

	return err
}

func (p *PostgreSQL) GetBrandCompetitors(ctx context.Context, brandID string) (*models.BrandCompetitors, error) {
	query := `
		SELECT id, brand, brand_id, org_id, competitors, suggested_list, source, created_at, updated_at
		FROM brand_competitors
		WHERE brand_id = $1
	`

	var competitors models.BrandCompetitors
	var competitorsJSON, suggestedListJSON string

	err := p.db.QueryRowContext(ctx, query, brandID).Scan(
		&competitors.ID,
		&competitors.Brand,
		&competitors.BrandID,
		&competitors.OrgID,
		&competitorsJSON,
		&suggestedListJSON,
		&competitors.Source,
		&competitors.CreatedAt,
		&competitors.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	competitors.Competitors = jsonToSlice(competitorsJSON)
	competitors.SuggestedList = jsonToSlice(suggestedListJSON)
	return &competitors, nil
}

func (p *PostgreSQL) DeleteBrandCompetitors(ctx context.Context, brandID string) error {
	query := "DELETE FROM brand_competitors WHERE brand_id = $1"
	_, err := p.db.ExecContext(ctx, query, brandID)
	return err
}

func (p *PostgreSQL) ListBrandCompetitors(ctx context.Context) ([]*models.BrandCompetitors, error) {
	query := `
		SELECT id, brand, brand_id, org_id, competitors, suggested_list, source, created_at, updated_at
		FROM brand_competitors
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var competitorsList []*models.BrandCompetitors
	for rows.Next() {
		var competitors models.BrandCompetitors
		var competitorsJSON, suggestedListJSON string

		if err := rows.Scan(
			&competitors.ID,
			&competitors.Brand,
			&competitors.BrandID,
			&competitors.OrgID,
			&competitorsJSON,
			&suggestedListJSON,
			&competitors.Source,
			&competitors.CreatedAt,
			&competitors.UpdatedAt,
		); err != nil {
			return nil, err
		}

		competitors.Competitors = jsonToSlice(competitorsJSON)
		competitors.SuggestedList = jsonToSlice(suggestedListJSON)
		competitorsList = append(competitorsList, &competitors)
	}

	return competitorsList, nil
}

// ==================== Brand Prompts Operations ====================

func (p *PostgreSQL) SaveBrandPrompts(ctx context.Context, prompts *models.BrandPrompts) error {
	now := time.Now()
	if prompts.CreatedAt.IsZero() {
		prompts.CreatedAt = now
	}
	prompts.UpdatedAt = now

	query := `
		INSERT INTO brand_prompts (id, brand, brand_id, org_id, active_prompt_ids, suggested_prompt_ids, source, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (brand_id) DO UPDATE SET
			brand = EXCLUDED.brand,
			org_id = EXCLUDED.org_id,
			active_prompt_ids = EXCLUDED.active_prompt_ids,
			suggested_prompt_ids = EXCLUDED.suggested_prompt_ids,
			source = EXCLUDED.source,
			updated_at = EXCLUDED.updated_at
	`

	_, err := p.db.ExecContext(ctx, query,
		prompts.ID,
		prompts.Brand,
		prompts.BrandID,
		prompts.OrgID,
		sliceToJSON(prompts.ActivePromptIDs),
		sliceToJSON(prompts.SuggestedPromptIDs),
		prompts.Source,
		prompts.CreatedAt,
		prompts.UpdatedAt,
	)

	return err
}

func (p *PostgreSQL) GetBrandPrompts(ctx context.Context, brandID string) (*models.BrandPrompts, error) {
	query := `
		SELECT id, brand, brand_id, org_id, active_prompt_ids, suggested_prompt_ids, source, created_at, updated_at
		FROM brand_prompts
		WHERE brand_id = $1
	`

	var prompts models.BrandPrompts
	var activePromptIDsJSON, suggestedPromptIDsJSON string

	err := p.db.QueryRowContext(ctx, query, brandID).Scan(
		&prompts.ID,
		&prompts.Brand,
		&prompts.BrandID,
		&prompts.OrgID,
		&activePromptIDsJSON,
		&suggestedPromptIDsJSON,
		&prompts.Source,
		&prompts.CreatedAt,
		&prompts.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	prompts.ActivePromptIDs = jsonToSlice(activePromptIDsJSON)
	prompts.SuggestedPromptIDs = jsonToSlice(suggestedPromptIDsJSON)
	return &prompts, nil
}

// ==================== GEO Campaign Operations ====================

func (p *PostgreSQL) SaveGEOCampaign(ctx context.Context, campaign *models.GEOCampaign) error {
	now := time.Now()
	if campaign.CreatedAt.IsZero() {
		campaign.CreatedAt = now
	}
	campaign.UpdatedAt = now

	query := `
		INSERT INTO geo_campaigns (id, name, brand_id, org_id, brand, prompt_ids, llm_ids, status, total_runs, completed_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			brand_id = EXCLUDED.brand_id,
			org_id = EXCLUDED.org_id,
			brand = EXCLUDED.brand,
			prompt_ids = EXCLUDED.prompt_ids,
			llm_ids = EXCLUDED.llm_ids,
			status = EXCLUDED.status,
			total_runs = EXCLUDED.total_runs,
			completed_at = EXCLUDED.completed_at,
			updated_at = EXCLUDED.updated_at
	`

	_, err := p.db.ExecContext(ctx, query,
		campaign.ID,
		campaign.Name,
		campaign.BrandID,
		campaign.OrgID,
		campaign.Brand,
		sliceToJSON(campaign.PromptIDs),
		sliceToJSON(campaign.LLMIDs),
		campaign.Status,
		campaign.TotalRuns,
		campaign.CompletedAt,
		campaign.CreatedAt,
		campaign.UpdatedAt,
	)

	return err
}

func (p *PostgreSQL) GetGEOCampaign(ctx context.Context, id string) (*models.GEOCampaign, error) {
	query := `
		SELECT id, name, brand_id, org_id, brand, prompt_ids, llm_ids, status, total_runs, completed_at, created_at, updated_at
		FROM geo_campaigns
		WHERE id = $1
	`

	var campaign models.GEOCampaign
	var promptIDsJSON, llmIDsJSON string
	var completedAt sql.NullTime

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&campaign.ID,
		&campaign.Name,
		&campaign.BrandID,
		&campaign.OrgID,
		&campaign.Brand,
		&promptIDsJSON,
		&llmIDsJSON,
		&campaign.Status,
		&campaign.TotalRuns,
		&completedAt,
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
	if completedAt.Valid {
		campaign.CompletedAt = &completedAt.Time
	}

	return &campaign, nil
}

// GetGEOCampaignsByBrandID fetches all GEO campaigns for a specific brand ID
func (p *PostgreSQL) GetGEOCampaignsByBrandID(ctx context.Context, brandID string) ([]*models.GEOCampaign, error) {
	query := `
		SELECT id, name, brand_id, org_id, brand, prompt_ids, llm_ids, status, total_runs, completed_at, created_at, updated_at
		FROM geo_campaigns
		WHERE brand_id = $1
		ORDER BY created_at DESC
	`

	rows, err := p.db.QueryContext(ctx, query, brandID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var campaigns []*models.GEOCampaign
	for rows.Next() {
		var campaign models.GEOCampaign
		var promptIDsJSON, llmIDsJSON string
		var completedAt sql.NullTime

		err := rows.Scan(
			&campaign.ID,
			&campaign.Name,
			&campaign.BrandID,
			&campaign.OrgID,
			&campaign.Brand,
			&promptIDsJSON,
			&llmIDsJSON,
			&campaign.Status,
			&campaign.TotalRuns,
			&completedAt,
			&campaign.CreatedAt,
			&campaign.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		campaign.PromptIDs = jsonToSlice(promptIDsJSON)
		campaign.LLMIDs = jsonToSlice(llmIDsJSON)
		if completedAt.Valid {
			campaign.CompletedAt = &completedAt.Time
		}

		campaigns = append(campaigns, &campaign)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return campaigns, nil
}

func (p *PostgreSQL) GetRunningGEOCampaignByBrand(ctx context.Context, brand string) (*models.GEOCampaign, error) {
	query := `
		SELECT id, name, brand_id, org_id, brand, prompt_ids, llm_ids, status, total_runs, completed_at, created_at, updated_at
		FROM geo_campaigns
		WHERE brand = $1 AND status = 'running' AND completed_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1
	`

	var campaign models.GEOCampaign
	var promptIDsJSON, llmIDsJSON string
	var completedAt sql.NullTime

	err := p.db.QueryRowContext(ctx, query, brand).Scan(
		&campaign.ID,
		&campaign.Name,
		&campaign.BrandID,
		&campaign.OrgID,
		&campaign.Brand,
		&promptIDsJSON,
		&llmIDsJSON,
		&campaign.Status,
		&campaign.TotalRuns,
		&completedAt,
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
	if completedAt.Valid {
		campaign.CompletedAt = &completedAt.Time
	}

	return &campaign, nil
}

func (p *PostgreSQL) UpdateGEOCampaign(ctx context.Context, campaign *models.GEOCampaign) error {
	return p.SaveGEOCampaign(ctx, campaign)
}
