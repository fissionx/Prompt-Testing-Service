package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/fissionx/gego/internal/llm"
	"github.com/fissionx/gego/internal/models"
	"github.com/fissionx/gego/internal/utils"
)

// mapIntentTypeToPromptType maps the new intentType from LLM to PromptType enum
func mapIntentTypeToPromptType(intentType string) models.PromptType {
	intentTypeLower := strings.ToLower(intentType)

	// Map new intent types to existing PromptType enum
	if strings.Contains(intentTypeLower, "top") || strings.Contains(intentTypeLower, "best") {
		return models.PromptTypeTopBest
	}
	if strings.Contains(intentTypeLower, "vs") || strings.Contains(intentTypeLower, "comparison") || strings.Contains(intentTypeLower, "compare") {
		return models.PromptTypeComparison
	}
	if strings.Contains(intentTypeLower, "alternatives") || strings.Contains(intentTypeLower, "research") {
		return models.PromptTypeComparison
	}
	if strings.Contains(intentTypeLower, "features") || strings.Contains(intentTypeLower, "benefits") {
		return models.PromptTypeWhat
	}
	if strings.Contains(intentTypeLower, "quality") || strings.Contains(intentTypeLower, "credibility") ||
		strings.Contains(intentTypeLower, "pros") || strings.Contains(intentTypeLower, "cons") ||
		strings.Contains(intentTypeLower, "pricing") || strings.Contains(intentTypeLower, "value") ||
		strings.Contains(intentTypeLower, "support") || strings.Contains(intentTypeLower, "reliability") ||
		strings.Contains(intentTypeLower, "security") || strings.Contains(intentTypeLower, "compliance") {
		return models.PromptTypeBrand
	}

	// Default mappings for old format
	if strings.Contains(intentTypeLower, "unbranded") || strings.Contains(intentTypeLower, "discovery") {
		return models.PromptTypeWhat
	}
	if strings.Contains(intentTypeLower, "branded") {
		return models.PromptTypeBrand
	}

	// Default fallback
	return models.PromptTypeCustom
}

// PromptGenerationResult contains the full prompt data from LLM response
type PromptGenerationResult struct {
	Prompt                  string
	IntentType              string
	TargetingSearchKeywords []string
	SupportingFanoutQueries []string
}

// generatePromptsWithLLM is a shared function for generating prompts using the new BrandPromptGenerationPrompt system
// language is the brand's language for generated prompts (use utils.ResolveBrandLanguage; default English if empty).
func generatePromptsWithLLM(
	ctx context.Context,
	provider llm.Provider,
	model string,
	brand string,
	websiteContent *WebsiteContent,
	category string,
	description string,
	language string,
	count int,
	logPrefix string, // For logging context (e.g., "[BrandPromptService]" or "")
) ([]PromptGenerationResult, error) {
	// Build website content string for the new prompt format
	var websiteContentStr string
	if websiteContent != nil {
		var contentParts []string
		if websiteContent.Title != "" {
			contentParts = append(contentParts, fmt.Sprintf("Title: %s", websiteContent.Title))
		}
		if websiteContent.Description != "" {
			contentParts = append(contentParts, fmt.Sprintf("Description: %s", websiteContent.Description))
		}
		if len(websiteContent.Keywords) > 0 {
			contentParts = append(contentParts, fmt.Sprintf("Keywords: %s", strings.Join(websiteContent.Keywords, ", ")))
		}
		if websiteContent.MainContent != "" {
			content := websiteContent.MainContent
			if len(content) > 2000 {
				content = content[:2000] + "..."
			}
			contentParts = append(contentParts, fmt.Sprintf("Main Content: %s", content))
		}
		websiteContentStr = strings.Join(contentParts, "\n")
	}

	// Build product/service name from description or category
	productServiceName := ""
	if description != "" {
		// Extract first sentence or first 100 chars as product/service name
		sentences := strings.Split(description, ".")
		if len(sentences) > 0 && len(sentences[0]) > 0 {
			productServiceName = strings.TrimSpace(sentences[0])
			if len(productServiceName) > 100 {
				productServiceName = productServiceName[:100]
			}
		}
	}
	if productServiceName == "" && category != "" {
		productServiceName = category
	}

	// Use the new structured prompt generation
	generationPrompt := utils.BrandPromptGenerationPrompt(
		brand,              // businessName
		websiteContentStr,  // websiteContent
		productServiceName, // productServiceName
		"",                 // targetAudience (can be enhanced later)
		"",                 // region (can be enhanced later)
		language,           // language (brand's language for generated prompts)
		count,              // count
	)

	if logPrefix != "" {
		fmt.Printf("📝 %s Using new BrandPromptGenerationPrompt system prompt\n", logPrefix)
		fmt.Printf("📝 %s Requesting %d prompts\n", logPrefix, count)
	} else {
		fmt.Printf("📝 Using new BrandPromptGenerationPrompt system prompt\n")
		fmt.Printf("📝 Prompt length: %d characters\n", len(generationPrompt))
		fmt.Printf("📝 Requesting %d prompts\n", count)
	}

	response, err := provider.Generate(ctx, generationPrompt, llm.Config{
		Model:       model,
		Temperature: 0.8, // Balanced creativity for realistic queries
		MaxTokens:   4096,
	})
	if err != nil {
		if logPrefix != "" {
			fmt.Printf("❌ %s LLM generation error: %v\n", logPrefix, err)
		} else {
			fmt.Printf("❌ LLM generation error: %v\n", err)
		}
		return nil, err
	}

	if logPrefix != "" {
		fmt.Printf("✅ %s LLM response received (length: %d chars, tokens: %d)\n", logPrefix, len(response.Text), response.TokensUsed)
	} else {
		fmt.Printf("✅ LLM response received (length: %d chars, tokens: %d)\n", len(response.Text), response.TokensUsed)
	}

	// Parse JSON response
	results, err := parseBrandPromptGenerationResponse(response.Text, logPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated prompts: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no prompts generated from LLM response")
	}

	return results, nil
}

// parseBrandPromptGenerationResponse parses the JSON response from the new prompt generation format
// This is a shared function used by both PromptGenerationService and BrandPromptService
// Returns full prompt data including intentType, keywords, and fanout queries
func parseBrandPromptGenerationResponse(responseText string, logPrefix string) ([]PromptGenerationResult, error) {
	// Log the raw response for debugging
	if logPrefix != "" {
		fmt.Printf("🔍 %s Raw LLM response (first 500 chars): %s\n", logPrefix, truncateString(responseText, 500))
	} else {
		fmt.Printf("🔍 Raw LLM response (first 500 chars): %s\n", truncateString(responseText, 500))
	}

	// Clean up the response - remove markdown code blocks if present
	responseText = strings.TrimSpace(responseText)
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	// Try to find JSON array in the response
	// Look for the start of JSON array
	startIdx := strings.Index(responseText, "[")
	if startIdx == -1 {
		// Try to find JSON object with array inside
		objStartIdx := strings.Index(responseText, "{")
		if objStartIdx != -1 {
			// Try to extract array from object
			arrayKeyIdx := strings.Index(responseText, `"prompts"`)
			if arrayKeyIdx == -1 {
				arrayKeyIdx = strings.Index(responseText, `"queries"`)
			}
			if arrayKeyIdx != -1 {
				// Find the array after the key
				startIdx = strings.Index(responseText[arrayKeyIdx:], "[")
				if startIdx != -1 {
					startIdx += arrayKeyIdx
				}
			}
		}
		if startIdx == -1 {
			prefix := logPrefix
			if prefix == "" {
				prefix = ""
			}
			fmt.Printf("❌ %s No JSON array found in response. Response preview: %s\n", prefix, truncateString(responseText, 200))
			return nil, fmt.Errorf("no JSON array found in response")
		}
	}

	// Find the matching closing bracket
	bracketCount := 0
	endIdx := -1
	inString := false
	escapeNext := false

	for i := startIdx; i < len(responseText); i++ {
		char := responseText[i]

		if escapeNext {
			escapeNext = false
			continue
		}

		if char == '\\' {
			escapeNext = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if char == '[' {
				bracketCount++
			} else if char == ']' {
				bracketCount--
				if bracketCount == 0 {
					endIdx = i
					break
				}
			}
		}
	}

	if endIdx == -1 {
		prefix := logPrefix
		if prefix == "" {
			prefix = ""
		}
		fmt.Printf("❌ %s Malformed JSON array - could not find closing bracket. Response preview: %s\n", prefix, truncateString(responseText, 200))
		return nil, fmt.Errorf("malformed JSON array in response")
	}

	jsonStr := responseText[startIdx : endIdx+1]
	if logPrefix != "" {
		fmt.Printf("✅ %s Extracted JSON array (length: %d)\n", logPrefix, len(jsonStr))
	} else {
		fmt.Printf("✅ Extracted JSON array (length: %d)\n", len(jsonStr))
	}

	// Parse JSON array
	var items []models.BrandPromptGenerationItem
	if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
		prefix := logPrefix
		if prefix == "" {
			prefix = ""
		}
		fmt.Printf("❌ %s JSON unmarshal error: %v\n", prefix, err)
		fmt.Printf("❌ %s JSON string (first 500 chars): %s\n", prefix, truncateString(jsonStr, 500))

		// Try to extract prompts even if JSON structure is imperfect
		// Look for "prompt" fields in the text
		prompts := extractPromptsFromText(responseText)
		if len(prompts) > 0 {
			if logPrefix != "" {
				fmt.Printf("⚠️ %s Using fallback extraction, found %d prompts\n", logPrefix, len(prompts))
			} else {
				fmt.Printf("⚠️ Using fallback extraction, found %d prompts\n", len(prompts))
			}
			// Convert to PromptGenerationResult format
			results := make([]PromptGenerationResult, len(prompts))
			for i, p := range prompts {
				results[i] = PromptGenerationResult{
					Prompt:                  p,
					IntentType:              "", // Unknown in fallback
					TargetingSearchKeywords: []string{},
					SupportingFanoutQueries: []string{},
				}
			}
			return results, nil
		}

		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if logPrefix != "" {
		fmt.Printf("✅ %s Successfully parsed %d items from JSON\n", logPrefix, len(items))
	} else {
		fmt.Printf("✅ Successfully parsed %d items from JSON\n", len(items))
	}

	// Extract full prompt data from items
	var results []PromptGenerationResult
	for i, item := range items {
		if item.Prompt != "" {
			result := PromptGenerationResult{
				Prompt:                  item.Prompt,
				IntentType:              item.IntentType,
				TargetingSearchKeywords: item.TargetingSearchKeywords,
				SupportingFanoutQueries: item.SupportingFanoutQueries,
			}
			results = append(results, result)
			if logPrefix != "" {
				fmt.Printf("  [%d] Intent: %s, Prompt: %s\n", i+1, item.IntentType, truncateString(item.Prompt, 80))
			} else {
				fmt.Printf("  [%d] Intent: %s, Prompt: %s\n", i+1, item.IntentType, truncateString(item.Prompt, 80))
			}
		} else {
			if logPrefix != "" {
				fmt.Printf("  [%d] WARNING: Empty prompt in item\n", i+1)
			} else {
				fmt.Printf("  [%d] WARNING: Empty prompt in item\n", i+1)
			}
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no valid prompts found in parsed items")
	}

	return results, nil
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractPromptsFromText attempts to extract prompts from text even if JSON is malformed
func extractPromptsFromText(text string) []string {
	var prompts []string

	// Try to find "prompt" fields in JSON-like structures
	// Look for patterns like "prompt": "..." or "prompt":"..."
	re := regexp.MustCompile(`"prompt"\s*:\s*"([^"]+)"`)
	matches := re.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) > 1 && match[1] != "" {
			// Unescape JSON string
			prompt := strings.ReplaceAll(match[1], "\\n", "\n")
			prompt = strings.ReplaceAll(prompt, "\\\"", "\"")
			prompt = strings.ReplaceAll(prompt, "\\\\", "\\")
			prompts = append(prompts, prompt)
		}
	}

	return prompts
}
