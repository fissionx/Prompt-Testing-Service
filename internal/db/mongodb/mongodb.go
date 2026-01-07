package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// trackLatency accumulates MongoDB operation latency in context for summary logging
func trackLatency(ctx context.Context, operation, collection string, start time.Time, err error) {
	duration := time.Since(start)
	
	// Accumulate time in context if available (from API requests)
	if mongoTotalTime, ok := ctx.Value("mongo_total_time").(*time.Duration); ok {
		*mongoTotalTime += duration
	}
}

// MongoDB implements the Database interface for MongoDB
type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
	config   *models.Config
}

const (
	collPrompts          = "prompts"
	collResponses        = "responses"
	collPromptLibrary    = "prompt_library"
	collBrandProfiles    = "brand_profiles"
	collBrandLogos       = "brand_logos"
	collBrandCompetitors = "brand_competitors"
	collBrandPrompts     = "brand_prompts"
	collGEOCampaigns     = "geo_campaigns"
)

// New creates a new MongoDB database instance
func New(config *models.Config) (*MongoDB, error) {
	return &MongoDB{
		config: config,
	}, nil
}

// Connect establishes connection to MongoDB
func (m *MongoDB) Connect(ctx context.Context) error {
	clientOptions := options.Client().ApplyURI(m.config.URI)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	m.client = client
	m.database = client.Database(m.config.Database)

	if err := m.createIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Create analytics cache indexes
	if err := m.createAnalyticsCacheIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create analytics cache indexes: %w", err)
	}

	return nil
}

// Disconnect closes the MongoDB connection
func (m *MongoDB) Disconnect(ctx context.Context) error {
	if m.client != nil {
		return m.client.Disconnect(ctx)
	}
	return nil
}

// Ping checks the database connection
func (m *MongoDB) Ping(ctx context.Context) error {
	if m.client == nil {
		return fmt.Errorf("not connected to database")
	}
	return m.client.Ping(ctx, nil)
}

// createIndexes creates necessary indexes for optimal query performance
func (m *MongoDB) createIndexes(ctx context.Context) error {
	// Optimize indexes for storage and performance
	responseIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "prompt_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "created_at", Value: -1},
			},
		},
		{
			Keys: bson.D{
				{Key: "brand", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "brand", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
		// Add sparse index for schedule_id (only when present)
		{
			Keys: bson.D{
				{Key: "schedule_id", Value: 1},
			},
			Options: options.Index().SetSparse(true),
		},
		// Add sparse index for llm_id
		{
			Keys: bson.D{
				{Key: "llm_id", Value: 1},
			},
			Options: options.Index().SetSparse(true),
		},
	}

	_, err := m.database.Collection(collResponses).Indexes().CreateMany(ctx, responseIndexes)
	if err != nil {
		return fmt.Errorf("failed to create response indexes: %w", err)
	}

	// Create indexes for prompts collection (brand lookup for efficient filtering)
	promptIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "brand", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "brand", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
	}

	_, err = m.database.Collection(collPrompts).Indexes().CreateMany(ctx, promptIndexes)
	if err != nil {
		return fmt.Errorf("failed to create prompt indexes: %w", err)
	}

	// Create index for prompt library (domain + category lookup for cross-brand reuse)
	libraryIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "domain", Value: 1},
				{Key: "category", Value: 1},
			},
		},
	}

	_, err = m.database.Collection(collPromptLibrary).Indexes().CreateMany(ctx, libraryIndexes)
	if err != nil {
		return fmt.Errorf("failed to create prompt library indexes: %w", err)
	}

	// Create index for brand profiles (brand_name lookup)
	profileIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "brand_name", Value: 1},
			},
		},
	}

	_, err = m.database.Collection(collBrandProfiles).Indexes().CreateMany(ctx, profileIndexes)
	if err != nil {
		return fmt.Errorf("failed to create brand profile indexes: %w", err)
	}

	// Create index for brand logos (brand_name lookup)
	logoIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "brand_name", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
	}

	_, err = m.database.Collection(collBrandLogos).Indexes().CreateMany(ctx, logoIndexes)
	if err != nil {
		return fmt.Errorf("failed to create brand logo indexes: %w", err)
	}

	return nil
}

// CreatePrompt creates a new prompt
func (m *MongoDB) CreatePrompt(ctx context.Context, prompt *models.Prompt) error {
	start := time.Now()
	prompt.CreatedAt = time.Now()
	prompt.UpdatedAt = time.Now()

	// Compress template if large
	template := prompt.Template
	if shared.ShouldCompress(template) {
		if compressed, err := shared.CompressString(template); err == nil {
			template = compressed
		}
	}

	doc := bson.M{
		"_id":         prompt.ID,
		"template":    template,
		"prompt_type": prompt.PromptType,
		"tags":        prompt.Tags,
		"category":    prompt.Category,
		"domain":      prompt.Domain,
		"brand":       prompt.Brand,
		"source_id":   prompt.SourceID,
		"generated":   prompt.Generated,
		"enabled":     prompt.Enabled,
		"created_at":  prompt.CreatedAt,
		"updated_at":  prompt.UpdatedAt,
	}

	_, err := m.database.Collection(collPrompts).InsertOne(ctx, doc)
	trackLatency(ctx, "CreatePrompt", collPrompts, start, err)
	return err
}

// GetPrompt retrieves a prompt by ID
func (m *MongoDB) GetPrompt(ctx context.Context, id string) (*models.Prompt, error) {
	start := time.Now()
	var doc bson.M
	err := m.database.Collection(collPrompts).FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		trackLatency(ctx, "GetPrompt", collPrompts, start, err)
		return nil, fmt.Errorf("prompt not found: %s", id)
	}
	if err != nil {
		trackLatency(ctx, "GetPrompt", collPrompts, start, err)
		return nil, err
	}

	var promptID string
	if id, ok := doc["_id"].(string); ok {
		promptID = id
	} else if objectID, ok := doc["_id"].(primitive.ObjectID); ok {
		promptID = objectID.Hex()
	} else {
		return nil, fmt.Errorf("invalid _id type in document")
	}

	// Decompress template if it was compressed
	template := getString(doc, "template")
	if decompressed, err := shared.DecompressString(template); err == nil {
		template = decompressed
	}

	prompt := &models.Prompt{
		ID:         promptID,
		Template:   template,
		PromptType: models.PromptType(getString(doc, "prompt_type")),
		Category:   getString(doc, "category"),
		Domain:     getString(doc, "domain"),
		Brand:      getString(doc, "brand"),
		SourceID:   getString(doc, "source_id"),
		Generated:  getBool(doc, "generated"),
		Enabled:    getBool(doc, "enabled"),
		CreatedAt:  getTime(doc, "created_at"),
		UpdatedAt:  getTime(doc, "updated_at"),
	}

		if tags, ok := doc["tags"].([]interface{}); ok {
			for _, t := range tags {
				if str, ok := t.(string); ok {
					prompt.Tags = append(prompt.Tags, str)
				}
			}
		}

	trackLatency(ctx, "GetPrompt", collPrompts, start, nil)
	return prompt, nil
}

// ListPrompts lists all prompts, optionally filtered by enabled status
func (m *MongoDB) ListPrompts(ctx context.Context, enabled *bool) ([]*models.Prompt, error) {
	start := time.Now()
	filter := bson.M{}
	if enabled != nil {
		filter["enabled"] = *enabled
	}

	cursor, err := m.database.Collection(collPrompts).Find(ctx, filter)
	if err != nil {
		trackLatency(ctx, "ListPrompts", collPrompts, start, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var prompts []*models.Prompt
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}

		// Convert BSON document to Prompt struct
		var promptID string
		if id, ok := doc["_id"].(string); ok {
			promptID = id
		} else if objectID, ok := doc["_id"].(primitive.ObjectID); ok {
			promptID = objectID.Hex()
		} else {
			return nil, fmt.Errorf("invalid _id type in document")
		}

		// Decompress template if it was compressed
		template := getString(doc, "template")
		if decompressed, err := shared.DecompressString(template); err == nil {
			template = decompressed
		}

		prompt := &models.Prompt{
			ID:         promptID,
			Template:   template,
			PromptType: models.PromptType(getString(doc, "prompt_type")),
			Category:   getString(doc, "category"),
			Domain:     getString(doc, "domain"),
			Brand:      getString(doc, "brand"),
			SourceID:   getString(doc, "source_id"),
			Generated:  getBool(doc, "generated"),
			Enabled:    getBool(doc, "enabled"),
			CreatedAt:  getTime(doc, "created_at"),
			UpdatedAt:  getTime(doc, "updated_at"),
		}

		// Handle optional fields
		if tags, ok := doc["tags"].([]interface{}); ok {
			for _, t := range tags {
				if str, ok := t.(string); ok {
					prompt.Tags = append(prompt.Tags, str)
				}
			}
		}

		prompts = append(prompts, prompt)
	}

	trackLatency(ctx, "ListPrompts", collPrompts, start, nil)
	return prompts, nil
}

// UpdatePrompt updates an existing prompt
func (m *MongoDB) UpdatePrompt(ctx context.Context, prompt *models.Prompt) error {
	start := time.Now()
	prompt.UpdatedAt = time.Now()

	// Compress template if large
	template := prompt.Template
	if shared.ShouldCompress(template) {
		if compressed, err := shared.CompressString(template); err == nil {
			template = compressed
		}
	}

	// Convert to BSON document with explicit _id field
	doc := bson.M{
		"_id":         prompt.ID,
		"template":    template,
		"prompt_type": prompt.PromptType,
		"tags":        prompt.Tags,
		"category":    prompt.Category,
		"domain":      prompt.Domain,
		"brand":       prompt.Brand,
		"source_id":   prompt.SourceID,
		"generated":   prompt.Generated,
		"enabled":     prompt.Enabled,
		"created_at":  prompt.CreatedAt,
		"updated_at":  prompt.UpdatedAt,
	}

	result, err := m.database.Collection(collPrompts).ReplaceOne(
		ctx,
		bson.M{"_id": prompt.ID},
		doc,
	)

	if err != nil {
		trackLatency(ctx, "UpdatePrompt", collPrompts, start, err)
		return err
	}

	if result.MatchedCount == 0 {
		err := fmt.Errorf("prompt not found: %s", prompt.ID)
		trackLatency(ctx, "UpdatePrompt", collPrompts, start, err)
		return err
	}

	trackLatency(ctx, "UpdatePrompt", collPrompts, start, nil)
	return nil
}

// DeletePrompt deletes a prompt by ID
func (m *MongoDB) DeletePrompt(ctx context.Context, id string) error {
	start := time.Now()
	var filter bson.M
	if objectID, err := primitive.ObjectIDFromHex(id); err == nil {
		filter = bson.M{"_id": objectID}
	} else {
		filter = bson.M{"_id": id}
	}

	result, err := m.database.Collection(collPrompts).DeleteOne(ctx, filter)
	if err != nil {
		trackLatency(ctx, "DeletePrompt", collPrompts, start, err)
		return err
	}

	if result.DeletedCount == 0 {
		err := fmt.Errorf("prompt not found: %s", id)
		trackLatency(ctx, "DeletePrompt", collPrompts, start, err)
		return err
	}

	trackLatency(ctx, "DeletePrompt", collPrompts, start, nil)
	return nil
}

// DeleteAllPrompts deletes all prompts
func (m *MongoDB) DeleteAllPrompts(ctx context.Context) (int, error) {
	start := time.Now()
	result, err := m.database.Collection(collPrompts).DeleteMany(ctx, bson.M{})
	trackLatency(ctx, "DeleteAllPrompts", collPrompts, start, err)
	if err != nil {
		return 0, err
	}
	return int(result.DeletedCount), nil
}

// DeletePromptsByBrand deletes all prompts for a specific brand
func (m *MongoDB) DeletePromptsByBrand(ctx context.Context, brand string) (int, error) {
	start := time.Now()
	filter := bson.M{"brand": bson.M{"$regex": primitive.Regex{Pattern: "^" + brand + "$", Options: "i"}}}
	result, err := m.database.Collection(collPrompts).DeleteMany(ctx, filter)
	trackLatency(ctx, "DeletePromptsByBrand", collPrompts, start, err)
	if err != nil {
		return 0, err
	}
	return int(result.DeletedCount), nil
}

// CreateResponse creates a new response
func (m *MongoDB) CreateResponse(ctx context.Context, response *models.Response) error {
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

	doc := bson.M{
		"_id":           response.ID,
		"prompt_id":     response.PromptID,
		"prompt_text":   compressedPromptText,
		"llm_id":        response.LLMID,
		"llm_name":      response.LLMName,
		"llm_provider":  response.LLMProvider,
		"llm_model":     response.LLMModel,
		"response_text": compressedResponseText,
		"schedule_id":   response.ScheduleID,
		"tokens_used":   response.TokensUsed,
		"temperature":   response.Temperature,
		"latency_ms":    response.LatencyMs,
		"error":         response.Error,
		"created_at":    response.CreatedAt,

		// GEO Analytics Fields
		"brand":                response.Brand,
		"visibility_score":     response.VisibilityScore,
		"brand_mentioned":      response.BrandMentioned,
		"in_grounding_sources": response.InGroundingSources,
		"sentiment":            response.Sentiment,
		"competitors_mention":  response.CompetitorsMention,
		"grounding_sources":    response.GroundingSources,
		"grounding_domains":    response.GroundingDomains,

		// Position/Ranking Fields
		"brand_position":      response.BrandPosition,
		"total_brands_listed": response.TotalBrandsListed,

		// Time-Series Fields
		"week":    response.Week,
		"month":   response.Month,
		"quarter": response.Quarter,

		// Regional Fields
		"region":   response.Region,
		"language": response.Language,

		// Web Search Metadata (for ChatGPT/Gemini-like experience)
		"web_search_queries": response.WebSearchQueries,
		"web_search_calls":   response.WebSearchCalls,
		"search_answer":      response.SearchAnswer,
	}

	if response.Metadata != nil {
		doc["metadata"] = response.Metadata
	}

	_, err := m.database.Collection(collResponses).InsertOne(ctx, doc)
	trackLatency(ctx, "CreateResponse", collResponses, start, err)
	return err
}

// GetResponse retrieves a response by ID
func (m *MongoDB) GetResponse(ctx context.Context, id string) (*models.Response, error) {
	start := time.Now()
	var response models.Response
	err := m.database.Collection(collResponses).FindOne(ctx, bson.M{"_id": id}).Decode(&response)
	if err == mongo.ErrNoDocuments {
		trackLatency(ctx, "GetResponse", collResponses, start, err)
		return nil, fmt.Errorf("response not found: %s", id)
	}
	if err != nil {
		trackLatency(ctx, "GetResponse", collResponses, start, err)
		return nil, err
	}

	// Decompress text fields if they were compressed
	if decompressed, err := shared.DecompressString(response.ResponseText); err == nil {
		response.ResponseText = decompressed
	}
	if decompressed, err := shared.DecompressString(response.PromptText); err == nil {
		response.PromptText = decompressed
	}

	trackLatency(ctx, "GetResponse", collResponses, start, nil)
	return &response, nil
}

// ListResponses lists responses with filtering
func (m *MongoDB) ListResponses(ctx context.Context, filter shared.ResponseFilter) ([]*models.Response, error) {
	start := time.Now()
	query := bson.M{}

	if filter.PromptID != "" {
		query["prompt_id"] = filter.PromptID
	}
	if filter.LLMID != "" {
		query["llm_id"] = filter.LLMID
	}
	if filter.ScheduleID != "" {
		query["schedule_id"] = filter.ScheduleID
	}
	if filter.Keyword != "" {
		query["response_text"] = bson.M{
			"$regex":   filter.Keyword,
			"$options": "i",
		}
	}
	if filter.StartTime != nil || filter.EndTime != nil {
		timeQuery := bson.M{}
		if filter.StartTime != nil {
			timeQuery["$gte"] = *filter.StartTime
		}
		if filter.EndTime != nil {
			timeQuery["$lte"] = *filter.EndTime
		}
		query["created_at"] = timeQuery
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}
	if filter.Offset > 0 {
		opts.SetSkip(int64(filter.Offset))
	}

	cursor, err := m.database.Collection(collResponses).Find(ctx, query, opts)
	if err != nil {
		trackLatency(ctx, "ListResponses", collResponses, start, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var responses []*models.Response
	if err := cursor.All(ctx, &responses); err != nil {
		trackLatency(ctx, "ListResponses", collResponses, start, err)
		return nil, err
	}

	// Decompress text fields for all responses
	for _, response := range responses {
		if decompressed, err := shared.DecompressString(response.ResponseText); err == nil {
			response.ResponseText = decompressed
		}
		if decompressed, err := shared.DecompressString(response.PromptText); err == nil {
			response.PromptText = decompressed
		}
	}

	trackLatency(ctx, "ListResponses", collResponses, start, nil)
	return responses, nil
}

// CountResponses counts responses matching the filter without fetching all documents
func (m *MongoDB) CountResponses(ctx context.Context, filter shared.ResponseFilter) (int64, error) {
	start := time.Now()
	query := bson.M{}

	if filter.PromptID != "" {
		query["prompt_id"] = filter.PromptID
	}
	if filter.LLMID != "" {
		query["llm_id"] = filter.LLMID
	}
	if filter.ScheduleID != "" {
		query["schedule_id"] = filter.ScheduleID
	}
	if filter.Keyword != "" {
		query["response_text"] = bson.M{
			"$regex":   filter.Keyword,
			"$options": "i",
		}
	}
	if filter.StartTime != nil || filter.EndTime != nil {
		timeQuery := bson.M{}
		if filter.StartTime != nil {
			timeQuery["$gte"] = *filter.StartTime
		}
		if filter.EndTime != nil {
			timeQuery["$lte"] = *filter.EndTime
		}
		query["created_at"] = timeQuery
	}

	count, err := m.database.Collection(collResponses).CountDocuments(ctx, query)
	trackLatency(ctx, "CountResponses", collResponses, start, err)
	return count, err
}

// GetDatabase returns the underlying MongoDB database instance
func (m *MongoDB) GetDatabase() *mongo.Database {
	return m.database
}

// Helper functions for safe field extraction
func getString(doc bson.M, key string) string {
	if val, ok := doc[key]; ok && val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBool(doc bson.M, key string) bool {
	if val, ok := doc[key]; ok && val != nil {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func getTime(doc bson.M, key string) time.Time {
	if val, ok := doc[key]; ok && val != nil {
		if t, ok := val.(time.Time); ok {
			return t
		}
		if dt, ok := val.(primitive.DateTime); ok {
			return dt.Time()
		}
		if ts, ok := val.(int64); ok {
			return time.Unix(ts, 0)
		}
		if ts, ok := val.(float64); ok {
			return time.Unix(int64(ts), 0)
		}
	}
	return time.Time{}
}

// DeleteAllResponses deletes all responses from the database
func (m *MongoDB) DeleteAllResponses(ctx context.Context) (int, error) {
	start := time.Now()
	result, err := m.database.Collection(collResponses).DeleteMany(ctx, bson.M{})
	trackLatency(ctx, "DeleteAllResponses", collResponses, start, err)
	if err != nil {
		return 0, err
	}
	return int(result.DeletedCount), nil
}

// GetPromptStats calculates prompt statistics on-demand from responses
func (m *MongoDB) GetPromptStats(ctx context.Context, promptID string) (*models.PromptStats, error) {
	start := time.Now()
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"prompt_id": promptID,
			},
		},
		{
			"$group": bson.M{
				"_id":             nil,
				"total_responses": bson.M{"$sum": 1},
				"avg_tokens": bson.M{
					"$avg": "$tokens_used",
				},
				"unique_llms": bson.M{"$addToSet": "$llm_id"},
			},
		},
		{
			"$project": bson.M{
				"total_responses": 1,
				"avg_tokens":      1,
				"unique_llms":     bson.M{"$size": "$unique_llms"},
			},
		},
	}

	cursor, err := m.database.Collection(collResponses).Aggregate(ctx, pipeline)
	if err != nil {
		trackLatency(ctx, "GetPromptStats", collResponses, start, err)
		return nil, fmt.Errorf("failed to aggregate prompt stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalResponses int     `bson:"total_responses"`
		AvgTokens      float64 `bson:"avg_tokens"`
		UniqueLLMs     int     `bson:"unique_llms"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode prompt stats: %w", err)
		}
	}

	llmCounts, err := m.getLLMCountsForPrompt(ctx, promptID)
	if err != nil {
		trackLatency(ctx, "GetPromptStats", collResponses, start, err)
		return nil, fmt.Errorf("failed to get LLM counts: %w", err)
	}

	trackLatency(ctx, "GetPromptStats", collResponses, start, nil)
	return &models.PromptStats{
		PromptID:       promptID,
		TotalResponses: result.TotalResponses,
		UniqueLLMs:     result.UniqueLLMs,
		LLMCounts:      llmCounts,
		AvgTokens:      result.AvgTokens,
		UpdatedAt:      time.Now(),
	}, nil
}

// GetLLMStats calculates LLM statistics on-demand from responses
func (m *MongoDB) GetLLMStats(ctx context.Context, llmID string) (*models.LLMStats, error) {
	start := time.Now()
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"llm_id": llmID,
			},
		},
		{
			"$group": bson.M{
				"_id":             nil,
				"total_responses": bson.M{"$sum": 1},
				"avg_tokens": bson.M{
					"$avg": "$tokens_used",
				},
				"unique_prompts": bson.M{"$addToSet": "$prompt_id"},
			},
		},
		{
			"$project": bson.M{
				"total_responses": 1,
				"avg_tokens":      1,
				"unique_prompts":  bson.M{"$size": "$unique_prompts"},
			},
		},
	}

	cursor, err := m.database.Collection(collResponses).Aggregate(ctx, pipeline)
	if err != nil {
		trackLatency(ctx, "GetLLMStats", collResponses, start, err)
		return nil, fmt.Errorf("failed to aggregate LLM stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalResponses int     `bson:"total_responses"`
		AvgTokens      float64 `bson:"avg_tokens"`
		UniquePrompts  int     `bson:"unique_prompts"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			trackLatency(ctx, "GetLLMStats", collResponses, start, err)
			return nil, fmt.Errorf("failed to decode LLM stats: %w", err)
		}
	}

	promptCounts, err := m.getPromptCountsForLLM(ctx, llmID)
	if err != nil {
		trackLatency(ctx, "GetLLMStats", collResponses, start, err)
		return nil, fmt.Errorf("failed to get prompt counts: %w", err)
	}

	trackLatency(ctx, "GetLLMStats", collResponses, start, nil)
	return &models.LLMStats{
		LLMID:          llmID,
		TotalResponses: result.TotalResponses,
		UniquePrompts:  result.UniquePrompts,
		PromptCounts:   promptCounts,
		AvgTokens:      result.AvgTokens,
		UpdatedAt:      time.Now(),
	}, nil
}

// getLLMCountsForPrompt gets the count of responses by LLM for a specific prompt
func (m *MongoDB) getLLMCountsForPrompt(ctx context.Context, promptID string) (map[string]int, error) {
	start := time.Now()
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"prompt_id": promptID,
			},
		},
		{
			"$group": bson.M{
				"_id":   "$llm_id",
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := m.database.Collection(collResponses).Aggregate(ctx, pipeline)
	if err != nil {
		trackLatency(ctx, "getLLMCountsForPrompt", collResponses, start, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	counts := make(map[string]int)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		counts[result.ID] = result.Count
	}

	trackLatency(ctx, "getLLMCountsForPrompt", collResponses, start, nil)
	return counts, nil
}

// getPromptCountsForLLM gets the count of responses by prompt for a specific LLM
func (m *MongoDB) getPromptCountsForLLM(ctx context.Context, llmID string) (map[string]int, error) {
	start := time.Now()
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"llm_id": llmID,
			},
		},
		{
			"$group": bson.M{
				"_id":   "$prompt_id",
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := m.database.Collection(collResponses).Aggregate(ctx, pipeline)
	if err != nil {
		trackLatency(ctx, "getPromptCountsForLLM", collResponses, start, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	counts := make(map[string]int)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		counts[result.ID] = result.Count
	}

	trackLatency(ctx, "getPromptCountsForLLM", collResponses, start, nil)
	return counts, nil
}

// CreatePromptLibrary creates a new prompt library entry
func (m *MongoDB) CreatePromptLibrary(ctx context.Context, library *models.PromptLibrary) error {
	start := time.Now()
	library.CreatedAt = time.Now()
	library.UpdatedAt = time.Now()

	doc := bson.M{
		"_id":         library.ID,
		"brand":       library.Brand,
		"domain":      library.Domain,
		"category":    library.Category,
		"prompt_ids":  library.PromptIDs,
		"usage_count": library.UsageCount,
		"created_at":  library.CreatedAt,
		"updated_at":  library.UpdatedAt,
	}

	_, err := m.database.Collection(collPromptLibrary).InsertOne(ctx, doc)
	trackLatency(ctx, "CreatePromptLibrary", collPromptLibrary, start, err)
	return err
}

// GetPromptLibrary retrieves a prompt library by brand, domain, and category
// If brand is empty, it searches by domain/category only (for cross-brand reuse)
func (m *MongoDB) GetPromptLibrary(ctx context.Context, brand, domain, category string) (*models.PromptLibrary, error) {
	start := time.Now()
	filter := bson.M{
		"domain":   domain,
		"category": category,
	}

	// If brand is specified, include it in the filter (exact match)
	if brand != "" {
		filter["brand"] = brand
	}

	var library models.PromptLibrary
	err := m.database.Collection(collPromptLibrary).FindOne(ctx, filter).Decode(&library)
	if err == mongo.ErrNoDocuments {
		trackLatency(ctx, "GetPromptLibrary", collPromptLibrary, start, nil)
		return nil, nil // Return nil if not found (not an error)
	}
	if err != nil {
		trackLatency(ctx, "GetPromptLibrary", collPromptLibrary, start, err)
		return nil, fmt.Errorf("failed to find prompt library: %w", err)
	}

	trackLatency(ctx, "GetPromptLibrary", collPromptLibrary, start, nil)
	return &library, nil
}

// UpdatePromptLibrary updates an existing prompt library
func (m *MongoDB) UpdatePromptLibrary(ctx context.Context, library *models.PromptLibrary) error {
	start := time.Now()
	library.UpdatedAt = time.Now()

	doc := bson.M{
		"_id":         library.ID,
		"brand":       library.Brand,
		"domain":      library.Domain,
		"category":    library.Category,
		"prompt_ids":  library.PromptIDs,
		"usage_count": library.UsageCount,
		"created_at":  library.CreatedAt,
		"updated_at":  library.UpdatedAt,
	}

	result, err := m.database.Collection(collPromptLibrary).ReplaceOne(
		ctx,
		bson.M{"_id": library.ID},
		doc,
	)

	if err != nil {
		trackLatency(ctx, "UpdatePromptLibrary", collPromptLibrary, start, err)
		return fmt.Errorf("failed to update prompt library: %w", err)
	}

	if result.MatchedCount == 0 {
		err := fmt.Errorf("prompt library not found: %s", library.ID)
		trackLatency(ctx, "UpdatePromptLibrary", collPromptLibrary, start, err)
		return err
	}

	trackLatency(ctx, "UpdatePromptLibrary", collPromptLibrary, start, nil)
	return nil
}

// ListPromptLibraries lists all prompt libraries
func (m *MongoDB) ListPromptLibraries(ctx context.Context) ([]*models.PromptLibrary, error) {
	start := time.Now()
	cursor, err := m.database.Collection(collPromptLibrary).Find(ctx, bson.M{})
	if err != nil {
		trackLatency(ctx, "ListPromptLibraries", collPromptLibrary, start, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var libraries []*models.PromptLibrary
	if err := cursor.All(ctx, &libraries); err != nil {
		trackLatency(ctx, "ListPromptLibraries", collPromptLibrary, start, err)
		return nil, err
	}

	trackLatency(ctx, "ListPromptLibraries", collPromptLibrary, start, nil)
	return libraries, nil
}

// CreateBrandProfile creates a new brand profile
func (m *MongoDB) CreateBrandProfile(ctx context.Context, profile *models.BrandProfile) error {
	start := time.Now()
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = time.Now()

	doc := bson.M{
		"_id":         profile.ID,
		"brand_name":  profile.BrandName,
		"domain":      profile.Domain,
		"category":    profile.Category,
		"website":     profile.Website,
		"description": profile.Description,
		"competitors": profile.Competitors,
		"created_at":  profile.CreatedAt,
		"updated_at":  profile.UpdatedAt,
	}

	_, err := m.database.Collection(collBrandProfiles).InsertOne(ctx, doc)
	trackLatency(ctx, "CreateBrandProfile", collBrandProfiles, start, err)
	return err
}

// GetBrandProfile retrieves a brand profile by brand name
func (m *MongoDB) GetBrandProfile(ctx context.Context, brandName string) (*models.BrandProfile, error) {
	start := time.Now()
	var profile models.BrandProfile
	err := m.database.Collection(collBrandProfiles).FindOne(ctx, bson.M{"brand_name": brandName}).Decode(&profile)
	if err == mongo.ErrNoDocuments {
		trackLatency(ctx, "GetBrandProfile", collBrandProfiles, start, nil)
		return nil, nil // Return nil if not found (not an error)
	}
	if err != nil {
		trackLatency(ctx, "GetBrandProfile", collBrandProfiles, start, err)
		return nil, fmt.Errorf("failed to find brand profile: %w", err)
	}

	trackLatency(ctx, "GetBrandProfile", collBrandProfiles, start, nil)
	return &profile, nil
}

// UpdateBrandProfile updates an existing brand profile
func (m *MongoDB) UpdateBrandProfile(ctx context.Context, profile *models.BrandProfile) error {
	start := time.Now()
	profile.UpdatedAt = time.Now()

	doc := bson.M{
		"_id":         profile.ID,
		"brand_name":  profile.BrandName,
		"domain":      profile.Domain,
		"category":    profile.Category,
		"website":     profile.Website,
		"description": profile.Description,
		"competitors": profile.Competitors,
		"created_at":  profile.CreatedAt,
		"updated_at":  profile.UpdatedAt,
	}

	result, err := m.database.Collection(collBrandProfiles).ReplaceOne(
		ctx,
		bson.M{"_id": profile.ID},
		doc,
	)

	if err != nil {
		trackLatency(ctx, "UpdateBrandProfile", collBrandProfiles, start, err)
		return fmt.Errorf("failed to update brand profile: %w", err)
	}

	if result.MatchedCount == 0 {
		err := fmt.Errorf("brand profile not found: %s", profile.ID)
		trackLatency(ctx, "UpdateBrandProfile", collBrandProfiles, start, err)
		return err
	}

	trackLatency(ctx, "UpdateBrandProfile", collBrandProfiles, start, nil)
	return nil
}

// ListBrandProfiles lists all brand profiles
func (m *MongoDB) ListBrandProfiles(ctx context.Context) ([]*models.BrandProfile, error) {
	start := time.Now()
	cursor, err := m.database.Collection(collBrandProfiles).Find(ctx, bson.M{})
	if err != nil {
		trackLatency(ctx, "ListBrandProfiles", collBrandProfiles, start, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var profiles []*models.BrandProfile
	if err := cursor.All(ctx, &profiles); err != nil {
		trackLatency(ctx, "ListBrandProfiles", collBrandProfiles, start, err)
		return nil, err
	}

	trackLatency(ctx, "ListBrandProfiles", collBrandProfiles, start, nil)
	return profiles, nil
}

// SaveBrandLogo saves or updates a brand logo in the cache
func (m *MongoDB) SaveBrandLogo(ctx context.Context, logo *models.BrandLogoCache) error {
	start := time.Now()
	logo.UpdatedAt = time.Now()

	doc := bson.M{
		"_id":               logo.ID,
		"brand_name":        logo.BrandName,
		"domain":            logo.Domain,
		"logo_url":          logo.LogoURL,
		"fallback_logo_url": logo.FallbackLogoURL,
		"source":            logo.Source,
		"last_checked":      logo.LastChecked,
		"created_at":        logo.CreatedAt,
		"updated_at":        logo.UpdatedAt,
	}

	// Upsert - update if exists, insert if not
	opts := options.Replace().SetUpsert(true)
	_, err := m.database.Collection(collBrandLogos).ReplaceOne(
		ctx,
		bson.M{"brand_name": logo.BrandName},
		doc,
		opts,
	)

	trackLatency(ctx, "SaveBrandLogo", collBrandLogos, start, err)
	return err
}

// GetBrandLogo retrieves a cached brand logo by brand name
func (m *MongoDB) GetBrandLogo(ctx context.Context, brandName string) (*models.BrandLogoCache, error) {
	start := time.Now()
	var logo models.BrandLogoCache
	err := m.database.Collection(collBrandLogos).FindOne(ctx, bson.M{"brand_name": brandName}).Decode(&logo)
	if err == mongo.ErrNoDocuments {
		trackLatency(ctx, "GetBrandLogo", collBrandLogos, start, nil)
		return nil, nil // Not found - not an error
	}
	if err != nil {
		trackLatency(ctx, "GetBrandLogo", collBrandLogos, start, err)
		return nil, fmt.Errorf("failed to find brand logo: %w", err)
	}

	trackLatency(ctx, "GetBrandLogo", collBrandLogos, start, nil)
	return &logo, nil
}

// ==================== Brand Competitors Operations ====================

// SaveBrandCompetitors saves or updates brand competitors
func (m *MongoDB) SaveBrandCompetitors(ctx context.Context, competitors *models.BrandCompetitors) error {
	start := time.Now()
	now := time.Now()
	if competitors.CreatedAt.IsZero() {
		competitors.CreatedAt = now
	}
	competitors.UpdatedAt = now

	doc := bson.M{
		"_id":            competitors.ID,
		"brand":          competitors.Brand,
		"competitors":    competitors.Competitors,
		"suggested_list": competitors.SuggestedList,
		"source":         competitors.Source,
		"created_at":     competitors.CreatedAt,
		"updated_at":     competitors.UpdatedAt,
	}

	// Upsert by brand - one competitor list per brand
	opts := options.Replace().SetUpsert(true)
	_, err := m.database.Collection(collBrandCompetitors).ReplaceOne(
		ctx,
		bson.M{"brand": competitors.Brand},
		doc,
		opts,
	)

	trackLatency(ctx, "SaveBrandCompetitors", collBrandCompetitors, start, err)
	return err
}

// GetBrandCompetitors retrieves competitors for a specific brand
func (m *MongoDB) GetBrandCompetitors(ctx context.Context, brand string) (*models.BrandCompetitors, error) {
	start := time.Now()
	var competitors models.BrandCompetitors
	err := m.database.Collection(collBrandCompetitors).FindOne(ctx, bson.M{"brand": brand}).Decode(&competitors)
	if err == mongo.ErrNoDocuments {
		trackLatency(ctx, "GetBrandCompetitors", collBrandCompetitors, start, nil)
		return nil, nil // Not found - not an error
	}
	if err != nil {
		trackLatency(ctx, "GetBrandCompetitors", collBrandCompetitors, start, err)
		return nil, fmt.Errorf("failed to get brand competitors: %w", err)
	}

	trackLatency(ctx, "GetBrandCompetitors", collBrandCompetitors, start, nil)
	return &competitors, nil
}

// DeleteBrandCompetitors deletes competitors for a specific brand
func (m *MongoDB) DeleteBrandCompetitors(ctx context.Context, brand string) error {
	start := time.Now()
	_, err := m.database.Collection(collBrandCompetitors).DeleteOne(ctx, bson.M{"brand": brand})
	trackLatency(ctx, "DeleteBrandCompetitors", collBrandCompetitors, start, err)
	return err
}

// ListBrandCompetitors lists all brand competitors
func (m *MongoDB) ListBrandCompetitors(ctx context.Context) ([]*models.BrandCompetitors, error) {
	start := time.Now()
	cursor, err := m.database.Collection(collBrandCompetitors).Find(ctx, bson.M{})
	if err != nil {
		trackLatency(ctx, "ListBrandCompetitors", collBrandCompetitors, start, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var competitorsList []*models.BrandCompetitors
	if err := cursor.All(ctx, &competitorsList); err != nil {
		trackLatency(ctx, "ListBrandCompetitors", collBrandCompetitors, start, err)
		return nil, err
	}

	trackLatency(ctx, "ListBrandCompetitors", collBrandCompetitors, start, nil)
	return competitorsList, nil
}

// SaveBrandPrompts saves or updates brand prompts
func (m *MongoDB) SaveBrandPrompts(ctx context.Context, prompts *models.BrandPrompts) error {
	start := time.Now()
	now := time.Now()
	if prompts.CreatedAt.IsZero() {
		prompts.CreatedAt = now
	}
	prompts.UpdatedAt = now

	doc := bson.M{
		"_id":                  prompts.ID,
		"brand":                prompts.Brand,
		"active_prompt_ids":    prompts.ActivePromptIDs,
		"suggested_prompt_ids": prompts.SuggestedPromptIDs,
		"source":               prompts.Source,
		"created_at":           prompts.CreatedAt,
		"updated_at":           prompts.UpdatedAt,
	}

	// Upsert by brand - one prompt list per brand
	opts := options.Replace().SetUpsert(true)
	_, err := m.database.Collection(collBrandPrompts).ReplaceOne(
		ctx,
		bson.M{"brand": prompts.Brand},
		doc,
		opts,
	)

	trackLatency(ctx, "SaveBrandPrompts", collBrandPrompts, start, err)
	return err
}

// GetBrandPrompts retrieves prompts for a specific brand
func (m *MongoDB) GetBrandPrompts(ctx context.Context, brand string) (*models.BrandPrompts, error) {
	start := time.Now()
	var prompts models.BrandPrompts
	err := m.database.Collection(collBrandPrompts).FindOne(ctx, bson.M{"brand": brand}).Decode(&prompts)
	if err == mongo.ErrNoDocuments {
		trackLatency(ctx, "GetBrandPrompts", collBrandPrompts, start, nil)
		return nil, nil // Not found - not an error
	}
	if err != nil {
		trackLatency(ctx, "GetBrandPrompts", collBrandPrompts, start, err)
		return nil, fmt.Errorf("failed to get brand prompts: %w", err)
	}

	trackLatency(ctx, "GetBrandPrompts", collBrandPrompts, start, nil)
	return &prompts, nil
}

// SaveGEOCampaign saves or updates a GEO campaign
func (m *MongoDB) SaveGEOCampaign(ctx context.Context, campaign *models.GEOCampaign) error {
	start := time.Now()
	now := time.Now()
	if campaign.CreatedAt.IsZero() {
		campaign.CreatedAt = now
	}
	campaign.UpdatedAt = now

	doc := bson.M{
		"_id":        campaign.ID,
		"name":       campaign.Name,
		"brand_id":   campaign.BrandID,
		"brand":      campaign.Brand,
		"prompt_ids": campaign.PromptIDs,
		"llm_ids":    campaign.LLMIDs,
		"status":     campaign.Status,
		"total_runs": campaign.TotalRuns,
		"created_at": campaign.CreatedAt,
		"updated_at": campaign.UpdatedAt,
	}

	if campaign.CompletedAt != nil {
		doc["completed_at"] = campaign.CompletedAt
	}

	// Upsert by ID
	filter := bson.M{"_id": campaign.ID}
	opts := options.Replace().SetUpsert(true)
	_, err := m.database.Collection(collGEOCampaigns).ReplaceOne(ctx, filter, doc, opts)
	trackLatency(ctx, "SaveGEOCampaign", collGEOCampaigns, start, err)
	return err
}

// GetGEOCampaign retrieves a GEO campaign by ID
func (m *MongoDB) GetGEOCampaign(ctx context.Context, id string) (*models.GEOCampaign, error) {
	start := time.Now()
	var campaign models.GEOCampaign
	err := m.database.Collection(collGEOCampaigns).FindOne(ctx, bson.M{"_id": id}).Decode(&campaign)
	if err == mongo.ErrNoDocuments {
		trackLatency(ctx, "GetGEOCampaign", collGEOCampaigns, start, nil)
		return nil, nil
	}
	if err != nil {
		trackLatency(ctx, "GetGEOCampaign", collGEOCampaigns, start, err)
		return nil, fmt.Errorf("failed to get GEO campaign: %w", err)
	}
	trackLatency(ctx, "GetGEOCampaign", collGEOCampaigns, start, nil)
	return &campaign, nil
}

// GetRunningGEOCampaignByBrand retrieves the most recent running GEO campaign for a brand
// A campaign is considered "running" only if:
// 1. Status is "running" AND
// 2. CompletedAt is nil (not set) - meaning it hasn't completed yet
func (m *MongoDB) GetRunningGEOCampaignByBrand(ctx context.Context, brand string) (*models.GEOCampaign, error) {
	start := time.Now()
	filter := bson.M{
		"brand":        brand,
		"status":       "running",
		"completed_at": nil, // Only return campaigns that haven't completed
	}

	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})

	var campaign models.GEOCampaign
	err := m.database.Collection(collGEOCampaigns).FindOne(ctx, filter, opts).Decode(&campaign)
	if err == mongo.ErrNoDocuments {
		trackLatency(ctx, "GetRunningGEOCampaignByBrand", collGEOCampaigns, start, nil)
		return nil, nil
	}
	if err != nil {
		trackLatency(ctx, "GetRunningGEOCampaignByBrand", collGEOCampaigns, start, err)
		return nil, fmt.Errorf("failed to get running GEO campaign by brand: %w", err)
	}
	trackLatency(ctx, "GetRunningGEOCampaignByBrand", collGEOCampaigns, start, nil)
	return &campaign, nil
}

// UpdateGEOCampaign updates a GEO campaign
func (m *MongoDB) UpdateGEOCampaign(ctx context.Context, campaign *models.GEOCampaign) error {
	return m.SaveGEOCampaign(ctx, campaign)
}
