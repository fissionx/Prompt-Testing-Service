package models

import (
	"time"
)

// TimeSeriesPoint represents a point in time-series data
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int       `json:"count"`
}

// KeywordCount represents a keyword and its mention count
type KeywordCount struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

// PromptStats represents aggregated statistics for a prompt
type PromptStats struct {
	PromptID       string         `json:"promptId"`
	TotalResponses int            `json:"totalResponses"`
	UniqueLLMs     int            `json:"uniqueLlms"`
	LLMCounts      map[string]int `json:"llmCounts"`
	AvgTokens      float64        `json:"avgTokens"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// LLMStats represents aggregated statistics for an LLM
type LLMStats struct {
	LLMID          string         `json:"llmId"`
	TotalResponses int            `json:"totalResponses"`
	UniquePrompts  int            `json:"uniquePrompts"`
	PromptCounts   map[string]int `json:"promptCounts"`
	AvgTokens      float64        `json:"avgTokens"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

// KeywordStats represents on-demand calculated statistics for a keyword search
type KeywordStats struct {
	Keyword       string         `json:"keyword"`
	TotalMentions int            `json:"totalMentions"`
	UniquePrompts int            `json:"uniquePrompts"`
	UniqueLLMs    int            `json:"uniqueLlms"`
	ByPrompt      map[string]int `json:"byPrompt"`
	ByLLM         map[string]int `json:"byLlm"`
	ByProvider    map[string]int `json:"byProvider"`
	FirstSeen     time.Time      `json:"firstSeen"`
	LastSeen      time.Time      `json:"lastSeen"`
}
