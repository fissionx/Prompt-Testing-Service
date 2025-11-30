package shared

import (
	"fmt"
	"strings"
)

// StorageStats tracks storage optimization statistics
type StorageStats struct {
	OriginalSize   int64
	CompressedSize int64
	CompressionRate float64
}

// CalculateCompressionStats calculates compression statistics
func CalculateCompressionStats(original, compressed string) StorageStats {
	originalSize := int64(len(original))
	compressedSize := int64(len(compressed))
	
	var rate float64
	if originalSize > 0 {
		rate = (1.0 - float64(compressedSize)/float64(originalSize)) * 100
	}
	
	return StorageStats{
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		CompressionRate: rate,
	}
}

// OptimizeArrayField optimizes array fields by removing duplicates and trimming
func OptimizeArrayField(arr []string) []string {
	if len(arr) == 0 {
		return arr
	}
	
	seen := make(map[string]bool)
	result := make([]string, 0, len(arr))
	
	for _, item := range arr {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			result = append(result, trimmed)
		}
	}
	
	return result
}

// TruncateForStorage truncates strings for storage while preserving essential info
func TruncateForStorage(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	
	// Keep first 80% and last 10% to preserve context
	keepStart := int(float64(maxLength) * 0.8)
	keepEnd := int(float64(maxLength) * 0.1)
	
	if keepStart + keepEnd >= len(text) {
		return text[:maxLength]
	}
	
	return text[:keepStart] + "..." + text[len(text)-keepEnd:]
}

// EstimateDocumentSize estimates the size of a document in bytes
func EstimateDocumentSize(fields map[string]interface{}) int64 {
	var size int64
	
	for key, value := range fields {
		// Add key size
		size += int64(len(key))
		
		// Add value size based on type
		switch v := value.(type) {
		case string:
			size += int64(len(v))
		case []string:
			for _, s := range v {
				size += int64(len(s))
			}
		case int, int32, int64:
			size += 8
		case float32, float64:
			size += 8
		case bool:
			size += 1
		default:
			size += 50 // Estimate for complex types
		}
	}
	
	return size
}

// FormatStorageSize formats bytes into human-readable format
func FormatStorageSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

