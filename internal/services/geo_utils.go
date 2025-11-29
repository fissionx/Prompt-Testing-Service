package services

import (
	"net/url"
	"regexp"
	"strings"
)

// ExtractBrandPosition analyzes response text to find brand's position in list-based responses
func ExtractBrandPosition(responseText, brand string) (position int, totalBrands int) {
	responseText = strings.ToLower(responseText)
	brand = strings.ToLower(brand)
	
	// Split into lines and paragraphs
	lines := strings.Split(responseText, "\n")
	
	position = 0
	totalBrands = 0
	brandFound := false
	
	// Look for numbered lists or bullet points
	numberPattern := regexp.MustCompile(`^\s*(\d+)[\.\)]\s+(.+)`)
	bulletPattern := regexp.MustCompile(`^\s*[-*•]\s+(.+)`)
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Check if line is a numbered item
		if matches := numberPattern.FindStringSubmatch(line); matches != nil {
			totalBrands++
			content := strings.ToLower(matches[2])
			if strings.Contains(content, brand) && !brandFound {
				position = totalBrands
				brandFound = true
			}
		} else if matches := bulletPattern.FindStringSubmatch(line); matches != nil {
			// Bullet point list
			totalBrands++
			content := strings.ToLower(matches[1])
			if strings.Contains(content, brand) && !brandFound {
				position = totalBrands
				brandFound = true
			}
		} else if i > 0 && strings.Contains(line, brand) && !brandFound {
			// Check if this looks like part of a list (starts with brand name or capitalized word)
			if strings.Contains(line, ":") || (len(line) > 0 && line[0] >= 'A' && line[0] <= 'Z') {
				totalBrands++
				position = totalBrands
				brandFound = true
			}
		}
	}
	
	// If we didn't find a structured list but found the brand, set position to 1
	if position == 0 && strings.Contains(responseText, brand) {
		position = 1
		totalBrands = countBrandMentions(responseText)
	}
	
	return position, totalBrands
}

// countBrandMentions counts approximate number of distinct brands mentioned
func countBrandMentions(text string) int {
	// Simple heuristic: count capitalized words that look like brand names
	pattern := regexp.MustCompile(`\b([A-Z][a-z]+(?:[A-Z][a-z]+)*)\b`)
	matches := pattern.FindAllString(text, -1)
	
	// Deduplicate
	seen := make(map[string]bool)
	for _, match := range matches {
		seen[strings.ToLower(match)] = true
	}
	
	// Return reasonable count (max 20 brands in a response)
	count := len(seen)
	if count > 20 {
		count = 20
	}
	return count
}

// ExtractDomainFromURL extracts the domain from a URL
func ExtractDomainFromURL(urlStr string) string {
	// Handle URLs without scheme
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}
	
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	
	// Get the hostname and remove www. prefix
	domain := parsedURL.Hostname()
	domain = strings.TrimPrefix(domain, "www.")
	
	return domain
}

// ExtractDomainsFromSources extracts unique domains from source URLs
func ExtractDomainsFromSources(sources []string) []string {
	domainMap := make(map[string]bool)
	var domains []string
	
	for _, source := range sources {
		domain := ExtractDomainFromURL(source)
		if domain != "" && !domainMap[domain] {
			domainMap[domain] = true
			domains = append(domains, domain)
		}
	}
	
	return domains
}

// categorizeSource categorizes a source domain into types
func categorizeSource(domain string) []string {
	categories := []string{}
	
	// Review sites
	reviewSites := []string{"g2.com", "capterra.com", "trustpilot.com", "yelp.com", "tripadvisor.com"}
	for _, site := range reviewSites {
		if strings.Contains(domain, site) {
			categories = append(categories, "review_site")
			break
		}
	}
	
	// Social media
	socialMedia := []string{"reddit.com", "twitter.com", "x.com", "linkedin.com", "facebook.com", "instagram.com", "youtube.com"}
	for _, site := range socialMedia {
		if strings.Contains(domain, site) {
			categories = append(categories, "social_media")
			break
		}
	}
	
	// News outlets
	newsOutlets := []string{"nytimes.com", "wsj.com", "bbc.com", "cnn.com", "reuters.com", "bloomberg.com", "techcrunch.com", "theverge.com"}
	for _, site := range newsOutlets {
		if strings.Contains(domain, site) {
			categories = append(categories, "news")
			break
		}
	}
	
	// Industry publications
	industryPubs := []string{"forbes.com", "inc.com", "entrepreneur.com", "wired.com", "arstechnica.com"}
	for _, site := range industryPubs {
		if strings.Contains(domain, site) {
			categories = append(categories, "publication")
			break
		}
	}
	
	// Company websites (if categories is empty, it's likely a company site)
	if len(categories) == 0 {
		categories = append(categories, "company_website")
	}
	
	return categories
}

// calculateSentimentScore converts sentiment string to numeric score
func calculateSentimentScore(sentiment string) float64 {
	switch strings.ToLower(sentiment) {
	case "positive":
		return 1.0
	case "neutral":
		return 0.0
	case "negative":
		return -1.0
	default:
		return 0.0
	}
}

// getWeekString formats time to ISO week string
func getWeekString(t interface{}) string {
	// Will be implemented when used
	return ""
}

// getMonthString formats time to month string
func getMonthString(t interface{}) string {
	// Will be implemented when used
	return ""
}

// getQuarterString formats time to quarter string
func getQuarterString(t interface{}) string {
	// Will be implemented when used
	return ""
}

