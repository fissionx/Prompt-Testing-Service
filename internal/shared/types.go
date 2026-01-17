package shared

import (
	"time"
)

// ResponseFilter provides filtering options for listing responses
type ResponseFilter struct {
	PromptID   string   // Single prompt ID (for backward compatibility)
	PromptIDs  []string // Multiple prompt IDs (batch filtering)
	LLMID      string   // Single LLM ID (for backward compatibility)
	LLMIDs     []string // Multiple LLM IDs (batch filtering)
	Brand      string   // Filter by brand name
	ScheduleID string
	Keyword    string
	StartTime  *time.Time
	EndTime    *time.Time
	Limit      int
	Offset     int
}
