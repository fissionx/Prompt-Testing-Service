package utils

import (
	"testing"
)

func TestTextEmbedder_Tokenize(t *testing.T) {
	embedder := NewTextEmbedder()

	tests := []struct {
		name     string
		input    string
		expected int // minimum number of tokens
	}{
		{
			name:     "simple sentence",
			input:    "Create a blog post about AI tools",
			expected: 3, // "create", "blog", "post", "ai", "tools" minus stop words
		},
		{
			name:     "with punctuation",
			input:    "Best AI tools for 2024! Check them out.",
			expected: 2,
		},
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := embedder.Tokenize(tt.input)
			if len(tokens) < tt.expected {
				t.Errorf("Tokenize() got %d tokens, want at least %d", len(tokens), tt.expected)
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
		delta    float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{1, 0, 0},
			expected: 1.0,
			delta:    0.001,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{0, 1, 0},
			expected: 0.0,
			delta:    0.001,
		},
		{
			name:     "similar vectors",
			a:        []float64{1, 1, 0},
			b:        []float64{1, 0.5, 0},
			expected: 0.9,
			delta:    0.1,
		},
		{
			name:     "empty vectors",
			a:        []float64{},
			b:        []float64{},
			expected: 0.0,
			delta:    0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CosineSimilarity(tt.a, tt.b)
			if result < tt.expected-tt.delta || result > tt.expected+tt.delta {
				t.Errorf("CosineSimilarity() = %v, want %v (±%v)", result, tt.expected, tt.delta)
			}
		})
	}
}

func TestOpportunityMatcher_FindDuplicate(t *testing.T) {
	matcher := NewOpportunityMatcher(0.5) // Lower threshold for more sensitive duplicate detection

	// Add some opportunities
	matcher.AddOpportunity("1", "content_gap", "Create blog post about AI tools for marketers", "Write a comprehensive blog post covering AI tools and solutions for marketing professionals")
	matcher.AddOpportunity("2", "reddit_participation", "Engage in r/marketing subreddit discussions", "Participate in Reddit r/marketing community discussions about marketing strategies")
	matcher.AddOpportunity("3", "case_study", "Add customer success story for healthcare industry", "Create a detailed case study showcasing healthcare industry customer success")

	tests := []struct {
		name            string
		oppType         string
		title           string
		description     string
		expectDuplicate bool
	}{
		{
			name:            "exact duplicate",
			oppType:         "content_gap",
			title:           "Create blog post about AI tools for marketers",
			description:     "Write a comprehensive blog post covering AI tools and solutions for marketing professionals",
			expectDuplicate: true,
		},
		{
			name:            "similar content gap same topic",
			oppType:         "content_gap",
			title:           "Write blog about AI tools for marketers",
			description:     "Create blog content about AI tools for marketing professionals",
			expectDuplicate: true,
		},
		{
			name:            "different platform LinkedIn vs Reddit",
			oppType:         "linkedin_presence",
			title:           "Share insights on LinkedIn",
			description:     "Post thought leadership content on LinkedIn about marketing",
			expectDuplicate: false,
		},
		{
			name:            "similar Reddit engagement same subreddit",
			oppType:         "reddit_participation",
			title:           "Participate in r/marketing discussions",
			description:     "Join and engage in r/marketing subreddit conversations",
			expectDuplicate: true,
		},
		{
			name:            "different case study different industry",
			oppType:         "case_study",
			title:           "Add customer success story for fintech industry",
			description:     "Create a detailed case study showcasing fintech industry customer success",
			expectDuplicate: false, // Different industry (fintech vs healthcare)
		},
		{
			name:            "completely different opportunity",
			oppType:         "review_sites",
			title:           "Get listed on G2 review platform",
			description:     "Claim and optimize G2 listing for better visibility",
			expectDuplicate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDup, matchID, score := matcher.FindDuplicate(tt.oppType, tt.title, tt.description)
			if isDup != tt.expectDuplicate {
				t.Errorf("FindDuplicate() = %v (score=%.3f, matchID=%s), want %v", isDup, score, matchID, tt.expectDuplicate)
			}
		})
	}
}

func TestJaccardSimilarityNGrams(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		n        int
		minSim   float64
	}{
		{
			name:   "identical strings",
			a:      "hello world",
			b:      "hello world",
			n:      3,
			minSim: 0.99,
		},
		{
			name:   "similar strings",
			a:      "create blog post about AI",
			b:      "write blog post about AI",
			n:      3,
			minSim: 0.5,
		},
		{
			name:   "different strings",
			a:      "hello world",
			b:      "goodbye universe",
			n:      3,
			minSim: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := JaccardSimilarityNGrams(tt.a, tt.b, tt.n)
			if sim < tt.minSim {
				t.Errorf("JaccardSimilarityNGrams() = %v, want at least %v", sim, tt.minSim)
			}
		})
	}
}

func TestOpportunityMatcher_Size(t *testing.T) {
	matcher := NewOpportunityMatcher(0.75)

	if matcher.Size() != 0 {
		t.Errorf("Initial size should be 0, got %d", matcher.Size())
	}

	matcher.AddOpportunity("1", "content_gap", "Test", "Description")
	if matcher.Size() != 1 {
		t.Errorf("Size after adding 1 should be 1, got %d", matcher.Size())
	}

	matcher.AddOpportunity("2", "content_gap", "Test 2", "Description 2")
	if matcher.Size() != 2 {
		t.Errorf("Size after adding 2 should be 2, got %d", matcher.Size())
	}

	matcher.Clear()
	if matcher.Size() != 0 {
		t.Errorf("Size after clear should be 0, got %d", matcher.Size())
	}
}
