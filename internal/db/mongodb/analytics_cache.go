package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/fissionx/gego/internal/models"
)

// Collection names for cached analytics
const (
	collCachedGEOInsights          = "cached_geo_insights"
	collCachedSourceAnalytics      = "cached_source_analytics"
	collCachedCompetitiveBenchmark = "cached_competitive_benchmark"
	collCachedPromptPerformance    = "cached_prompt_performance"
	collCachedPromptTimeSeries     = "cached_prompt_time_series"
	collScheduledCampaigns         = "scheduled_campaigns"
)

// createAnalyticsCacheIndexes creates indexes for analytics cache collections
func (m *MongoDB) createAnalyticsCacheIndexes(ctx context.Context) error {
	// Indexes for cached GEO insights
	geoInsightsIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "brand", Value: 1},
				{Key: "campaign_id", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "brand", Value: 1},
				{Key: "start_time", Value: -1},
				{Key: "end_time", Value: -1},
			},
		},
	}

	_, err := m.database.Collection(collCachedGEOInsights).Indexes().CreateMany(ctx, geoInsightsIndexes)
	if err != nil {
		return fmt.Errorf("failed to create cached geo insights indexes: %w", err)
	}

	// Indexes for cached source analytics
	sourceAnalyticsIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "brand", Value: 1},
				{Key: "campaign_id", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "brand", Value: 1},
				{Key: "start_time", Value: -1},
				{Key: "end_time", Value: -1},
			},
		},
	}

	_, err = m.database.Collection(collCachedSourceAnalytics).Indexes().CreateMany(ctx, sourceAnalyticsIndexes)
	if err != nil {
		return fmt.Errorf("failed to create cached source analytics indexes: %w", err)
	}

	// Indexes for cached competitive benchmark
	competitiveBenchmarkIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "main_brand", Value: 1},
				{Key: "campaign_id", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "main_brand", Value: 1},
				{Key: "start_time", Value: -1},
				{Key: "end_time", Value: -1},
			},
		},
	}

	_, err = m.database.Collection(collCachedCompetitiveBenchmark).Indexes().CreateMany(ctx, competitiveBenchmarkIndexes)
	if err != nil {
		return fmt.Errorf("failed to create cached competitive benchmark indexes: %w", err)
	}

	// Indexes for cached prompt performance
	promptPerformanceIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "brand", Value: 1},
				{Key: "campaign_id", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "brand", Value: 1},
				{Key: "start_time", Value: -1},
				{Key: "end_time", Value: -1},
			},
		},
	}

	_, err = m.database.Collection(collCachedPromptPerformance).Indexes().CreateMany(ctx, promptPerformanceIndexes)
	if err != nil {
		return fmt.Errorf("failed to create cached prompt performance indexes: %w", err)
	}

	// Indexes for scheduled campaigns
	scheduledCampaignIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "brand", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "next_run_at", Value: 1},
			},
			Options: options.Index().SetSparse(true),
		},
	}

	_, err = m.database.Collection(collScheduledCampaigns).Indexes().CreateMany(ctx, scheduledCampaignIndexes)
	if err != nil {
		return fmt.Errorf("failed to create scheduled campaigns indexes: %w", err)
	}

	// Indexes for cached prompt time series
	promptTimeSeriesIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "prompt_id", Value: 1},
				{Key: "brand", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "prompt_id", Value: 1},
				{Key: "campaign_id", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "prompt_id", Value: 1},
				{Key: "start_time", Value: -1},
				{Key: "end_time", Value: -1},
			},
		},
	}

	_, err = m.database.Collection(collCachedPromptTimeSeries).Indexes().CreateMany(ctx, promptTimeSeriesIndexes)
	if err != nil {
		return fmt.Errorf("failed to create cached prompt time series indexes: %w", err)
	}

	return nil
}

// SaveCachedGEOInsights saves or updates cached GEO insights
func (m *MongoDB) SaveCachedGEOInsights(ctx context.Context, insights *models.CachedGEOInsights) error {
	now := time.Now()
	if insights.CreatedAt.IsZero() {
		insights.CreatedAt = now
	}
	insights.UpdatedAt = now

	// Upsert by campaign_id and brand
	filter := bson.M{
		"campaign_id": insights.CampaignID,
		"brand":       insights.Brand,
	}

	opts := options.Replace().SetUpsert(true)
	_, err := m.database.Collection(collCachedGEOInsights).ReplaceOne(ctx, filter, insights, opts)
	return err
}

// GetCachedGEOInsights retrieves cached GEO insights based on query
func (m *MongoDB) GetCachedGEOInsights(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedGEOInsights, error) {
	filter := bson.M{}

	if query.CampaignID != "" {
		filter["campaign_id"] = query.CampaignID
	}
	if query.Brand != "" {
		filter["brand"] = query.Brand
	}
	if query.StartTime != nil {
		filter["start_time"] = bson.M{"$lte": *query.StartTime}
	}
	if query.EndTime != nil {
		filter["end_time"] = bson.M{"$gte": *query.EndTime}
	}

	// Get the most recent cached data
	opts := options.FindOne().SetSort(bson.D{{Key: "updated_at", Value: -1}})

	var insights models.CachedGEOInsights
	err := m.database.Collection(collCachedGEOInsights).FindOne(ctx, filter, opts).Decode(&insights)
	if err == mongo.ErrNoDocuments {
		return nil, nil // Not found, not an error
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached geo insights: %w", err)
	}

	return &insights, nil
}

// DeleteCachedGEOInsights deletes cached GEO insights by ID
func (m *MongoDB) DeleteCachedGEOInsights(ctx context.Context, id string) error {
	_, err := m.database.Collection(collCachedGEOInsights).DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// DeleteCachedGEOInsightsByBrand deletes all cached GEO insights for a brand
func (m *MongoDB) DeleteCachedGEOInsightsByBrand(ctx context.Context, brand string) error {
	_, err := m.database.Collection(collCachedGEOInsights).DeleteMany(ctx, bson.M{"brand": brand})
	return err
}

// SaveCachedSourceAnalytics saves or updates cached source analytics
func (m *MongoDB) SaveCachedSourceAnalytics(ctx context.Context, analytics *models.CachedSourceAnalytics) error {
	now := time.Now()
	if analytics.CreatedAt.IsZero() {
		analytics.CreatedAt = now
	}
	analytics.UpdatedAt = now

	// Upsert by campaign_id and brand
	filter := bson.M{
		"campaign_id": analytics.CampaignID,
		"brand":       analytics.Brand,
	}

	opts := options.Replace().SetUpsert(true)
	_, err := m.database.Collection(collCachedSourceAnalytics).ReplaceOne(ctx, filter, analytics, opts)
	return err
}

// GetCachedSourceAnalytics retrieves cached source analytics based on query
func (m *MongoDB) GetCachedSourceAnalytics(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedSourceAnalytics, error) {
	filter := bson.M{}

	if query.CampaignID != "" {
		filter["campaign_id"] = query.CampaignID
	}
	if query.Brand != "" {
		filter["brand"] = query.Brand
	}
	if query.StartTime != nil {
		filter["start_time"] = bson.M{"$lte": *query.StartTime}
	}
	if query.EndTime != nil {
		filter["end_time"] = bson.M{"$gte": *query.EndTime}
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "updated_at", Value: -1}})

	var analytics models.CachedSourceAnalytics
	err := m.database.Collection(collCachedSourceAnalytics).FindOne(ctx, filter, opts).Decode(&analytics)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached source analytics: %w", err)
	}

	return &analytics, nil
}

// DeleteCachedSourceAnalytics deletes cached source analytics by ID
func (m *MongoDB) DeleteCachedSourceAnalytics(ctx context.Context, id string) error {
	_, err := m.database.Collection(collCachedSourceAnalytics).DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// DeleteCachedSourceAnalyticsByBrand deletes all cached source analytics for a brand
func (m *MongoDB) DeleteCachedSourceAnalyticsByBrand(ctx context.Context, brand string) error {
	_, err := m.database.Collection(collCachedSourceAnalytics).DeleteMany(ctx, bson.M{"brand": brand})
	return err
}

// SaveCachedCompetitiveBenchmark saves or updates cached competitive benchmark
func (m *MongoDB) SaveCachedCompetitiveBenchmark(ctx context.Context, benchmark *models.CachedCompetitiveBenchmark) error {
	now := time.Now()
	if benchmark.CreatedAt.IsZero() {
		benchmark.CreatedAt = now
	}
	benchmark.UpdatedAt = now

	// Upsert by campaign_id and main_brand
	filter := bson.M{
		"campaign_id": benchmark.CampaignID,
		"main_brand":  benchmark.MainBrand,
	}

	opts := options.Replace().SetUpsert(true)
	_, err := m.database.Collection(collCachedCompetitiveBenchmark).ReplaceOne(ctx, filter, benchmark, opts)
	return err
}

// GetCachedCompetitiveBenchmark retrieves cached competitive benchmark based on query
func (m *MongoDB) GetCachedCompetitiveBenchmark(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedCompetitiveBenchmark, error) {
	filter := bson.M{}

	// Always filter by brand
	if query.Brand != "" {
		filter["main_brand"] = query.Brand
	}

	// For campaign-specific queries, filter by campaign_id
	// For general queries (no campaign_id), only get entries with empty campaign_id
	if query.CampaignID != "" {
		filter["campaign_id"] = query.CampaignID
	} else {
		// For general cache, only get entries with empty campaign_id
		filter["campaign_id"] = ""
	}

	// Time range matching: cache should overlap with query range
	// Cache is valid if: cache.start_time <= query.end_time AND cache.end_time >= query.start_time
	if query.StartTime != nil && query.EndTime != nil {
		filter["start_time"] = bson.M{"$lte": *query.EndTime}
		filter["end_time"] = bson.M{"$gte": *query.StartTime}
	} else if query.StartTime != nil {
		filter["end_time"] = bson.M{"$gte": *query.StartTime}
	} else if query.EndTime != nil {
		filter["start_time"] = bson.M{"$lte": *query.EndTime}
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "updated_at", Value: -1}})

	var benchmark models.CachedCompetitiveBenchmark
	err := m.database.Collection(collCachedCompetitiveBenchmark).FindOne(ctx, filter, opts).Decode(&benchmark)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached competitive benchmark: %w", err)
	}

	return &benchmark, nil
}

// DeleteCachedCompetitiveBenchmark deletes cached competitive benchmark by ID
func (m *MongoDB) DeleteCachedCompetitiveBenchmark(ctx context.Context, id string) error {
	_, err := m.database.Collection(collCachedCompetitiveBenchmark).DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// DeleteCachedCompetitiveBenchmarkByBrand deletes all cached competitive benchmark for a brand
func (m *MongoDB) DeleteCachedCompetitiveBenchmarkByBrand(ctx context.Context, brand string) error {
	_, err := m.database.Collection(collCachedCompetitiveBenchmark).DeleteMany(ctx, bson.M{"main_brand": brand})
	return err
}

// SaveCachedPromptPerformance saves or updates cached prompt performance
func (m *MongoDB) SaveCachedPromptPerformance(ctx context.Context, performance *models.CachedPromptPerformance) error {
	now := time.Now()
	if performance.CreatedAt.IsZero() {
		performance.CreatedAt = now
	}
	performance.UpdatedAt = now

	// Upsert by campaign_id and brand
	filter := bson.M{
		"campaign_id": performance.CampaignID,
		"brand":       performance.Brand,
	}

	opts := options.Replace().SetUpsert(true)
	_, err := m.database.Collection(collCachedPromptPerformance).ReplaceOne(ctx, filter, performance, opts)
	return err
}

// GetCachedPromptPerformance retrieves cached prompt performance based on query
func (m *MongoDB) GetCachedPromptPerformance(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedPromptPerformance, error) {
	filter := bson.M{}

	if query.CampaignID != "" {
		filter["campaign_id"] = query.CampaignID
	}
	if query.Brand != "" {
		filter["brand"] = query.Brand
	}
	if query.StartTime != nil {
		filter["start_time"] = bson.M{"$lte": *query.StartTime}
	}
	if query.EndTime != nil {
		filter["end_time"] = bson.M{"$gte": *query.EndTime}
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "updated_at", Value: -1}})

	var performance models.CachedPromptPerformance
	err := m.database.Collection(collCachedPromptPerformance).FindOne(ctx, filter, opts).Decode(&performance)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached prompt performance: %w", err)
	}

	return &performance, nil
}

// DeleteCachedPromptPerformance deletes cached prompt performance by ID
func (m *MongoDB) DeleteCachedPromptPerformance(ctx context.Context, id string) error {
	_, err := m.database.Collection(collCachedPromptPerformance).DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// DeleteCachedPromptPerformanceByBrand deletes all cached prompt performance for a brand
func (m *MongoDB) DeleteCachedPromptPerformanceByBrand(ctx context.Context, brand string) error {
	_, err := m.database.Collection(collCachedPromptPerformance).DeleteMany(ctx, bson.M{"brand": brand})
	return err
}

// SaveScheduledCampaign saves or updates a scheduled campaign
func (m *MongoDB) SaveScheduledCampaign(ctx context.Context, campaign *models.ScheduledCampaign) error {
	now := time.Now()
	if campaign.CreatedAt.IsZero() {
		campaign.CreatedAt = now
	}
	campaign.UpdatedAt = now

	// Upsert by ID
	filter := bson.M{"_id": campaign.ID}

	opts := options.Replace().SetUpsert(true)
	_, err := m.database.Collection(collScheduledCampaigns).ReplaceOne(ctx, filter, campaign, opts)
	return err
}

// GetScheduledCampaign retrieves a scheduled campaign by ID
func (m *MongoDB) GetScheduledCampaign(ctx context.Context, id string) (*models.ScheduledCampaign, error) {
	var campaign models.ScheduledCampaign
	err := m.database.Collection(collScheduledCampaigns).FindOne(ctx, bson.M{"_id": id}).Decode(&campaign)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled campaign: %w", err)
	}

	return &campaign, nil
}

// GetScheduledCampaignByBrand retrieves a scheduled campaign by brand
func (m *MongoDB) GetScheduledCampaignByBrand(ctx context.Context, brand string) (*models.ScheduledCampaign, error) {
	filter := bson.M{
		"brand":  brand,
		"status": bson.M{"$in": []string{"active", "running"}},
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})

	var campaign models.ScheduledCampaign
	err := m.database.Collection(collScheduledCampaigns).FindOne(ctx, filter, opts).Decode(&campaign)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled campaign by brand: %w", err)
	}

	return &campaign, nil
}

// ListScheduledCampaigns lists all scheduled campaigns with optional status filter
func (m *MongoDB) ListScheduledCampaigns(ctx context.Context, status string) ([]*models.ScheduledCampaign, error) {
	filter := bson.M{}
	if status != "" {
		filter["status"] = status
	}

	opts := options.Find().SetSort(bson.D{{Key: "next_run_at", Value: 1}})

	cursor, err := m.database.Collection(collScheduledCampaigns).Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list scheduled campaigns: %w", err)
	}
	defer cursor.Close(ctx)

	var campaigns []*models.ScheduledCampaign
	if err := cursor.All(ctx, &campaigns); err != nil {
		return nil, fmt.Errorf("failed to decode scheduled campaigns: %w", err)
	}

	return campaigns, nil
}

// UpdateScheduledCampaign updates a scheduled campaign
func (m *MongoDB) UpdateScheduledCampaign(ctx context.Context, campaign *models.ScheduledCampaign) error {
	campaign.UpdatedAt = time.Now()

	result, err := m.database.Collection(collScheduledCampaigns).ReplaceOne(
		ctx,
		bson.M{"_id": campaign.ID},
		campaign,
	)
	if err != nil {
		return fmt.Errorf("failed to update scheduled campaign: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("scheduled campaign not found: %s", campaign.ID)
	}

	return nil
}

// DeleteScheduledCampaign deletes a scheduled campaign by ID
func (m *MongoDB) DeleteScheduledCampaign(ctx context.Context, id string) error {
	result, err := m.database.Collection(collScheduledCampaigns).DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete scheduled campaign: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("scheduled campaign not found: %s", id)
	}

	return nil
}

// SaveCachedPromptTimeSeries saves or updates cached prompt time series
func (m *MongoDB) SaveCachedPromptTimeSeries(ctx context.Context, timeSeries *models.CachedPromptTimeSeries) error {
	now := time.Now()
	if timeSeries.CreatedAt.IsZero() {
		timeSeries.CreatedAt = now
	}
	timeSeries.UpdatedAt = now

	// Upsert by prompt_id and brand
	filter := bson.M{
		"prompt_id": timeSeries.PromptID,
		"brand":     timeSeries.Brand,
	}

	opts := options.Replace().SetUpsert(true)
	_, err := m.database.Collection(collCachedPromptTimeSeries).ReplaceOne(ctx, filter, timeSeries, opts)
	return err
}

// GetCachedPromptTimeSeries retrieves cached prompt time series based on query
func (m *MongoDB) GetCachedPromptTimeSeries(ctx context.Context, query models.AnalyticsCacheQuery) (*models.CachedPromptTimeSeries, error) {
	filter := bson.M{}

	if query.PromptID != "" {
		filter["prompt_id"] = query.PromptID
	}
	if query.CampaignID != "" {
		filter["campaign_id"] = query.CampaignID
	}
	if query.Brand != "" {
		filter["brand"] = query.Brand
	}
	if query.StartTime != nil {
		filter["start_time"] = bson.M{"$lte": *query.StartTime}
	}
	if query.EndTime != nil {
		filter["end_time"] = bson.M{"$gte": *query.EndTime}
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "updated_at", Value: -1}})

	var timeSeries models.CachedPromptTimeSeries
	err := m.database.Collection(collCachedPromptTimeSeries).FindOne(ctx, filter, opts).Decode(&timeSeries)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached prompt time series: %w", err)
	}

	return &timeSeries, nil
}

// DeleteCachedPromptTimeSeries deletes cached prompt time series by ID
func (m *MongoDB) DeleteCachedPromptTimeSeries(ctx context.Context, id string) error {
	_, err := m.database.Collection(collCachedPromptTimeSeries).DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// DeleteCachedPromptTimeSeriesByBrand deletes all cached prompt time series for a brand
func (m *MongoDB) DeleteCachedPromptTimeSeriesByBrand(ctx context.Context, brand string) error {
	_, err := m.database.Collection(collCachedPromptTimeSeries).DeleteMany(ctx, bson.M{"brand": brand})
	return err
}
