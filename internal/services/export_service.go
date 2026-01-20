package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fissionx/gego/internal/db"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/shared"
)

// ExportService provides data export functionality
type ExportService struct {
	db                          db.Database
	geoAnalyticsService         *GEOAnalyticsService
	sourceAnalyticsService      *SourceAnalyticsService
	competitiveBenchmarkService *CompetitiveBenchmarkService
	promptPerformanceService    *PromptPerformanceService
}

// NewExportService creates a new export service
func NewExportService(database db.Database) *ExportService {
	return &ExportService{
		db:                          database,
		geoAnalyticsService:         NewGEOAnalyticsService(database),
		sourceAnalyticsService:      NewSourceAnalyticsService(database),
		competitiveBenchmarkService: NewCompetitiveBenchmarkService(database),
		promptPerformanceService:    NewPromptPerformanceService(database),
	}
}

// Export exports data in the requested format
func (s *ExportService) Export(
	ctx context.Context,
	req models.ExportRequest,
) (*models.ExportResponse, error) {
	if req.Format == "" {
		req.Format = "csv"
	}

	switch req.ExportType {
	case "insights":
		return s.exportInsights(ctx, req)
	case "prompts":
		return s.exportPrompts(ctx, req)
	case "responses":
		return s.exportResponses(ctx, req)
	case "sources":
		return s.exportSources(ctx, req)
	case "competitive":
		return s.exportCompetitive(ctx, req)
	default:
		return nil, fmt.Errorf("unsupported export type: %s", req.ExportType)
	}
}

func (s *ExportService) exportInsights(ctx context.Context, req models.ExportRequest) (*models.ExportResponse, error) {
	// Note: orgID is not available in ExportRequest, passing empty string
	// The logo service will still work but won't store orgID
	insights, err := s.geoAnalyticsService.GetGEOInsights(ctx, req.BrandID, "", req.Brand, req.StartTime, req.EndTime)
	if err != nil {
		return nil, err
	}

	if req.Format == "json" {
		return s.exportJSON(insights, "insights")
	}

	// Export as CSV
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Header
	writer.Write([]string{"Brand", "Average Visibility", "Mention Rate %", "Grounding Rate %", "Total Responses"})

	// Data
	writer.Write([]string{
		insights.Brand,
		fmt.Sprintf("%.2f", insights.AverageVisibility),
		fmt.Sprintf("%.2f", insights.MentionRate),
		fmt.Sprintf("%.2f", insights.GroundingRate),
		fmt.Sprintf("%d", insights.TotalResponses),
	})

	// LLM Performance
	writer.Write([]string{})
	writer.Write([]string{"LLM Performance"})
	writer.Write([]string{"LLM Name", "Provider", "Visibility", "Mention Rate %", "Response Count"})
	for _, llm := range insights.PerformanceByLLM {
		writer.Write([]string{
			llm.LLMName,
			llm.LLMProvider,
			fmt.Sprintf("%.2f", llm.Visibility),
			fmt.Sprintf("%.2f", llm.MentionRate),
			fmt.Sprintf("%d", llm.ResponseCount),
		})
	}

	// Competitors
	writer.Write([]string{})
	writer.Write([]string{"Top Competitors"})
	writer.Write([]string{"Name", "Mention Count"})
	for _, comp := range insights.TopCompetitors {
		writer.Write([]string{
			comp.Name,
			fmt.Sprintf("%d", comp.MentionCount),
		})
	}

	writer.Flush()

	return &models.ExportResponse{
		FileName:    fmt.Sprintf("insights_%s_%s.csv", req.Brand, time.Now().Format("20060102")),
		ContentType: "text/csv",
		Data:        base64.StdEncoding.EncodeToString(buf.Bytes()),
		RecordCount: 1,
	}, nil
}

func (s *ExportService) exportPrompts(ctx context.Context, req models.ExportRequest) (*models.ExportResponse, error) {
	prompts, err := s.db.ListPrompts(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Filter by prompt IDs if specified
	if len(req.PromptIDs) > 0 {
		var filtered []*models.Prompt
		for _, p := range prompts {
			for _, id := range req.PromptIDs {
				if p.ID == id {
					filtered = append(filtered, p)
					break
				}
			}
		}
		prompts = filtered
	}

	if req.Format == "json" {
		return s.exportJSON(prompts, "prompts")
	}

	// Export as CSV
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Header
	writer.Write([]string{"ID", "Template", "Type", "Category", "Brand", "Enabled", "Created At"})

	// Data
	for _, prompt := range prompts {
		writer.Write([]string{
			prompt.ID,
			prompt.Template,
			string(prompt.PromptType),
			prompt.Category,
			prompt.Brand,
			fmt.Sprintf("%t", prompt.Enabled),
			prompt.CreatedAt.Format(time.RFC3339),
		})
	}

	writer.Flush()

	return &models.ExportResponse{
		FileName:    fmt.Sprintf("prompts_%s.csv", time.Now().Format("20060102")),
		ContentType: "text/csv",
		Data:        base64.StdEncoding.EncodeToString(buf.Bytes()),
		RecordCount: len(prompts),
	}, nil
}

func (s *ExportService) exportResponses(ctx context.Context, req models.ExportRequest) (*models.ExportResponse, error) {
	filter := shared.ResponseFilter{
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Limit:     10000,
	}

	allResponses, err := s.db.ListResponses(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Filter for brand
	var responses []*models.Response
	for _, resp := range allResponses {
		if resp.Brand == req.Brand {
			responses = append(responses, resp)
		}
	}

	if req.Format == "json" {
		return s.exportJSON(responses, "responses")
	}

	// Export as CSV
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Header
	writer.Write([]string{
		"ID", "Prompt Text", "LLM Name", "Provider", "Model",
		"Brand", "Visibility Score", "Brand Mentioned", "Position",
		"Sentiment", "In Grounding Sources", "Competitors",
		"Created At",
	})

	// Data
	for _, resp := range responses {
		writer.Write([]string{
			resp.ID,
			truncate(resp.PromptText, 200),
			resp.LLMName,
			resp.LLMProvider,
			resp.LLMModel,
			resp.Brand,
			fmt.Sprintf("%d", resp.VisibilityScore),
			fmt.Sprintf("%t", resp.BrandMentioned),
			fmt.Sprintf("%d", resp.BrandPosition),
			resp.Sentiment,
			fmt.Sprintf("%t", resp.InGroundingSources),
			strings.Join(resp.CompetitorsMention, ", "),
			resp.CreatedAt.Format(time.RFC3339),
		})
	}

	writer.Flush()

	return &models.ExportResponse{
		FileName:    fmt.Sprintf("responses_%s_%s.csv", req.Brand, time.Now().Format("20060102")),
		ContentType: "text/csv",
		Data:        base64.StdEncoding.EncodeToString(buf.Bytes()),
		RecordCount: len(responses),
	}, nil
}

func (s *ExportService) exportSources(ctx context.Context, req models.ExportRequest) (*models.ExportResponse, error) {
	// Note: orgID is not available in ExportRequest, passing empty string
	analytics, err := s.sourceAnalyticsService.GetSourceAnalytics(ctx, req.BrandID, "", req.Brand, req.StartTime, req.EndTime, 100)
	if err != nil {
		return nil, err
	}

	if req.Format == "json" {
		return s.exportJSON(analytics, "sources")
	}

	// Export as CSV
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Header
	writer.Write([]string{"Domain", "Citation Count", "Mention Rate %", "Categories"})

	// Data
	for _, source := range analytics.TopSources {
		writer.Write([]string{
			source.Domain,
			fmt.Sprintf("%d", source.CitationCount),
			fmt.Sprintf("%.2f", source.MentionRate),
			strings.Join(source.Categories, ", "),
		})
	}

	writer.Flush()

	return &models.ExportResponse{
		FileName:    fmt.Sprintf("sources_%s_%s.csv", req.Brand, time.Now().Format("20060102")),
		ContentType: "text/csv",
		Data:        base64.StdEncoding.EncodeToString(buf.Bytes()),
		RecordCount: len(analytics.TopSources),
	}, nil
}

func (s *ExportService) exportCompetitive(ctx context.Context, req models.ExportRequest) (*models.ExportResponse, error) {
	benchmark, err := s.competitiveBenchmarkService.GetCompetitiveBenchmark(
		ctx, req.Brand, nil, req.PromptIDs, req.LLMIDs, req.StartTime, req.EndTime, "", nil,
	)
	if err != nil {
		return nil, err
	}

	if req.Format == "json" {
		return s.exportJSON(benchmark, "competitive")
	}

	// Export as CSV
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)

	// Header
	writer.Write([]string{
		"Brand", "Visibility", "Mention Rate %", "Grounding Rate %",
		"Average Position", "Top Position Rate %", "Sentiment Score",
		"Response Count", "Market Share %",
	})

	// Main brand
	mb := benchmark.MainBrand
	writer.Write([]string{
		mb.Brand + " (YOU)",
		fmt.Sprintf("%.2f", mb.Visibility),
		fmt.Sprintf("%.2f", mb.MentionRate),
		fmt.Sprintf("%.2f", mb.GroundingRate),
		fmt.Sprintf("%.2f", mb.AveragePosition),
		fmt.Sprintf("%.2f", mb.TopPositionRate),
		fmt.Sprintf("%.2f", mb.SentimentScore),
		fmt.Sprintf("%d", mb.ResponseCount),
		fmt.Sprintf("%.2f", mb.MarketSharePct),
	})

	// Competitors
	for _, comp := range benchmark.Competitors {
		writer.Write([]string{
			comp.Brand,
			fmt.Sprintf("%.2f", comp.Visibility),
			fmt.Sprintf("%.2f", comp.MentionRate),
			fmt.Sprintf("%.2f", comp.GroundingRate),
			fmt.Sprintf("%.2f", comp.AveragePosition),
			fmt.Sprintf("%.2f", comp.TopPositionRate),
			fmt.Sprintf("%.2f", comp.SentimentScore),
			fmt.Sprintf("%d", comp.ResponseCount),
			fmt.Sprintf("%.2f", comp.MarketSharePct),
		})
	}

	writer.Flush()

	return &models.ExportResponse{
		FileName:    fmt.Sprintf("competitive_%s_%s.csv", req.Brand, time.Now().Format("20060102")),
		ContentType: "text/csv",
		Data:        base64.StdEncoding.EncodeToString(buf.Bytes()),
		RecordCount: len(benchmark.Competitors) + 1,
	}, nil
}

func (s *ExportService) exportJSON(data interface{}, exportType string) (*models.ExportResponse, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return &models.ExportResponse{
		FileName:    fmt.Sprintf("%s_%s.json", exportType, time.Now().Format("20060102")),
		ContentType: "application/json",
		Data:        string(jsonData),
		RecordCount: 1,
	}, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
