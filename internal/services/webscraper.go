package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// WebScraperService handles website content extraction
type WebScraperService struct {
	client *http.Client
}

// NewWebScraperService creates a new web scraper service
func NewWebScraperService() *WebScraperService {
	return &WebScraperService{
		client: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// WebsiteContent represents extracted content from a website
type WebsiteContent struct {
	URL         string
	Title       string
	Description string
	MainContent string
	Keywords    []string
}

// ScrapeWebsite fetches and extracts key content from a website
func (s *WebScraperService) ScrapeWebsite(ctx context.Context, url string) (*WebsiteContent, error) {
	// Ensure URL has protocol
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent to avoid blocks
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GeoBot/1.0; +https://github.com/fissionx/gego)")

	// Fetch the page
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch website: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("website returned status %d", resp.StatusCode)
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse HTML
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	content := &WebsiteContent{
		URL: url,
	}

	// Extract metadata and content
	s.extractContent(doc, content)

	// Clean and limit content
	content.MainContent = s.cleanText(content.MainContent)
	if len(content.MainContent) > 2000 {
		content.MainContent = content.MainContent[:2000] + "..."
	}

	return content, nil
}

// extractContent recursively extracts text content from HTML
func (s *WebScraperService) extractContent(n *html.Node, content *WebsiteContent) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "title":
			if content.Title == "" {
				content.Title = s.getNodeText(n)
			}
		case "meta":
			s.extractMeta(n, content)
		case "h1", "h2", "h3":
			text := s.getNodeText(n)
			if text != "" {
				content.MainContent += "\n" + text + "\n"
			}
		case "p":
			text := s.getNodeText(n)
			if text != "" && len(text) > 20 {
				content.MainContent += text + " "
			}
		case "script", "style", "nav", "footer", "header":
			return // Skip these elements
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s.extractContent(c, content)
	}
}

// extractMeta extracts metadata from meta tags
func (s *WebScraperService) extractMeta(n *html.Node, content *WebsiteContent) {
	var name, property, metaContent string

	for _, attr := range n.Attr {
		switch attr.Key {
		case "name":
			name = attr.Val
		case "property":
			property = attr.Val
		case "content":
			metaContent = attr.Val
		}
	}

	switch {
	case name == "description" || property == "og:description":
		if content.Description == "" {
			content.Description = metaContent
		}
	case name == "keywords":
		keywords := strings.Split(metaContent, ",")
		for _, kw := range keywords {
			kw = strings.TrimSpace(kw)
			if kw != "" {
				content.Keywords = append(content.Keywords, kw)
			}
		}
	}
}

// getNodeText extracts text from a node and its children
func (s *WebScraperService) getNodeText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}

	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += s.getNodeText(c)
	}
	return text
}

// cleanText removes extra whitespace and cleans up text
func (s *WebScraperService) cleanText(text string) string {
	// Remove multiple spaces
	space := regexp.MustCompile(`\s+`)
	text = space.ReplaceAllString(text, " ")

	// Remove leading/trailing whitespace
	text = strings.TrimSpace(text)

	return text
}

// GetBrandContext creates a rich context string from scraped content
func (s *WebScraperService) GetBrandContext(content *WebsiteContent) string {
	var context strings.Builder

	if content.Title != "" {
		context.WriteString(fmt.Sprintf("Website: %s\n", content.Title))
	}

	if content.Description != "" {
		context.WriteString(fmt.Sprintf("Description: %s\n", content.Description))
	}

	if len(content.Keywords) > 0 {
		context.WriteString(fmt.Sprintf("Keywords: %s\n", strings.Join(content.Keywords, ", ")))
	}

	if content.MainContent != "" {
		context.WriteString(fmt.Sprintf("\nMain Content:\n%s", content.MainContent))
	}

	return context.String()
}

