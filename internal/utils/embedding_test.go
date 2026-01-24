package utils

import (
	"testing"
)

func TestConceptExtraction(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "Freshness concepts",
			text:     "Update Admissions Content with Latest Dates and Stats",
			expected: []string{"admissions", "content", "freshness"},
		},
		{
			name:     "Reddit engagement",
			text:     "Engage in r/IndianEducation subreddits",
			expected: []string{"engagement", "reddit"},
		},
		{
			name:     "Case study",
			text:     "Publish Student Success Stories and Alumni Case Studies",
			expected: []string{"case_study"},
		},
		{
			name:     "FAQ content",
			text:     "Add Structured FAQ and HowTo Schema to Admissions Pages",
			expected: []string{"admissions", "content", "faq", "seo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			concepts := extractConcepts(tt.text)
			for _, exp := range tt.expected {
				found := false
				for _, c := range concepts {
					if c == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected concept %q not found in %v", exp, concepts)
				}
			}
		})
	}
}

func TestActionVerbExtraction(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "Update action",
			text:     "Update Admissions Content with Latest Dates",
			expected: []string{"update"},
		},
		{
			name:     "Engage action",
			text:     "Engage Authentically in Relevant Reddit Threads",
			expected: []string{"engage"},
		},
		{
			name:     "Create action",
			text:     "Create 'VIT vs IIITs' Comparison Page",
			expected: []string{"create"},
		},
		{
			name:     "Multiple actions",
			text:     "Create and publish student success stories",
			expected: []string{"create"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actions := extractActionVerbs(tt.text)
			for _, exp := range tt.expected {
				found := false
				for _, a := range actions {
					if a == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected action %q not found in %v", exp, actions)
				}
			}
		})
	}
}

func TestSemanticDuplicateDetection(t *testing.T) {
	matcher := NewOpportunityMatcher(0.80) // 80% threshold

	// Add some baseline opportunities
	matcher.AddOpportunity("opp1", "content_gap", 
		"Update Admissions Content with Latest Dates and Stats",
		"Refresh the admissions pages with current year information")
	
	matcher.AddOpportunity("opp2", "reddit_participation",
		"Engage Authentically in Relevant Reddit Threads",
		"Participate in r/IndianEducation discussions about admissions")

	tests := []struct {
		name        string
		oppType     string
		title       string
		description string
		expectDup   bool
	}{
		{
			name:        "Same content different wording - should be duplicate",
			oppType:     "content_gap",
			title:       "Update Admissions Content with Latest Dates and 'Last Updated' Tag",
			description: "Update the admissions pages with fresh dates",
			expectDup:   true,
		},
		{
			name:        "Similar update content - should be duplicate",
			oppType:     "content_gap",
			title:       "Update admission guides and landing pages with current year info",
			description: "Refresh admission content with latest information",
			expectDup:   true,
		},
		{
			name:        "Reddit engagement duplicate",
			oppType:     "reddit_participation",
			title:       "Engage in r/IndianEducation and r/EngineeringStudents subreddits",
			description: "Participate in education-related Reddit discussions",
			expectDup:   true,
		},
		{
			name:        "Different content - should NOT be duplicate",
			oppType:     "case_study",
			title:       "Publish Student Success Stories and Alumni Case Studies",
			description: "Create compelling case studies featuring successful alumni",
			expectDup:   false,
		},
		{
			name:        "Comparison page - should NOT be duplicate",
			oppType:     "comparison_content",
			title:       "Create 'VIT vs IIITs' Comparison Page",
			description: "Create a detailed comparison between VIT and IIITs",
			expectDup:   false,
		},
		{
			name:        "FAQ content - should NOT be duplicate",
			oppType:     "faq_content",
			title:       "Add Structured FAQ and HowTo Schema to Admissions Pages",
			description: "Add schema markup and FAQ sections to improve SEO",
			expectDup:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDup, similarID, similarity := matcher.FindDuplicate(tt.oppType, tt.title, tt.description)
			
			if isDup != tt.expectDup {
				t.Errorf("Expected duplicate=%v, got=%v (similarity=%.2f, similarTo=%s)\n  Title: %s", 
					tt.expectDup, isDup, similarity, similarID, tt.title)
			}
			
			t.Logf("Similarity: %.2f for '%s'", similarity, tt.title)
		})
	}
}

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected float64
	}{
		{
			name:     "Identical sets",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			expected: 1.0,
		},
		{
			name:     "No overlap",
			a:        []string{"a", "b"},
			b:        []string{"c", "d"},
			expected: 0.0,
		},
		{
			name:     "Partial overlap",
			a:        []string{"a", "b", "c"},
			b:        []string{"b", "c", "d"},
			expected: 0.5, // 2 common out of 4 unique
		},
		{
			name:     "Both empty",
			a:        []string{},
			b:        []string{},
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jaccardSimilarity(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Expected %.2f, got %.2f", tt.expected, result)
			}
		})
	}
}

func TestTargetEntityExtraction(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "Reddit subreddit",
			text:     "Engage in r/IndianEducation subreddit",
			expected: []string{"platform:reddit", "subreddit:indianeducation", "industry:education"},
		},
		{
			name:     "LinkedIn platform",
			text:     "Share content on LinkedIn for professional networking",
			expected: []string{"platform:linkedin"},
		},
		{
			name:     "Multiple platforms",
			text:     "Cross-post content on Reddit and LinkedIn",
			expected: []string{"platform:reddit", "platform:linkedin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entities := extractTargetEntities(tt.text)
			for _, exp := range tt.expected {
				found := false
				for _, e := range entities {
					if e == exp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected entity %q not found in %v", exp, entities)
				}
			}
		})
	}
}

func TestOpportunityMatcherSize(t *testing.T) {
	matcher := NewOpportunityMatcher(0.80)
	
	if matcher.Size() != 0 {
		t.Errorf("Expected size 0, got %d", matcher.Size())
	}
	
	matcher.AddOpportunity("1", "content_gap", "Title 1", "Description 1")
	matcher.AddOpportunity("2", "reddit_participation", "Title 2", "Description 2")
	
	if matcher.Size() != 2 {
		t.Errorf("Expected size 2, got %d", matcher.Size())
	}
	
	matcher.Clear()
	
	if matcher.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", matcher.Size())
	}
}

func TestSimilarAdmissionsContent(t *testing.T) {
	// Specific test for the user's examples
	matcher := NewOpportunityMatcher(0.80)
	
	// Add first opportunity
	matcher.AddOpportunity("opp1", "content_gap",
		"Update Admissions Content with Latest Dates and Stats",
		"Ensure admissions pages show current dates and statistics")
	
	// These should all be detected as duplicates
	duplicates := []struct {
		title string
		desc  string
	}{
		{
			"Update Admissions Content with Latest Dates and 'Last Updated' Tag",
			"Add last updated tag to admissions content",
		},
		{
			"Update admission guides and landing pages with current year info",
			"Refresh admission guides with 2024 information",
		},
	}
	
	for _, dup := range duplicates {
		isDup, _, sim := matcher.FindDuplicate("content_gap", dup.title, dup.desc)
		if !isDup {
			t.Errorf("Expected '%s' to be detected as duplicate (similarity: %.2f)", dup.title, sim)
		} else {
			t.Logf("✓ Correctly detected duplicate (%.0f%%): %s", sim*100, dup.title)
		}
	}
}

func TestSimilarRedditEngagement(t *testing.T) {
	matcher := NewOpportunityMatcher(0.80)
	
	// Add first Reddit opportunity
	matcher.AddOpportunity("opp1", "reddit_participation",
		"Engage Authentically in Relevant Reddit Threads",
		"Participate in discussions on Reddit about education")
	
	// This should be detected as duplicate
	isDup, _, sim := matcher.FindDuplicate("reddit_participation",
		"Engage in r/IndianEducation and r/EngineeringStudents subreddits",
		"Join Reddit communities and engage in discussions")
	
	if !isDup {
		t.Errorf("Expected Reddit engagement to be detected as duplicate (similarity: %.2f)", sim)
	} else {
		t.Logf("✓ Correctly detected Reddit duplicate (%.0f%%)", sim*100)
	}
}

func TestWordJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		min  float64 // minimum expected similarity
	}{
		{
			name: "Similar admission content",
			a:    "Update Admissions Content with Latest Dates",
			b:    "Update admission guides with current dates",
			min:  0.2, // Should have reasonable word overlap
		},
		{
			name: "Different topics",
			a:    "Create case studies",
			b:    "Engage on Reddit",
			min:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := wordJaccardSimilarity(tt.a, tt.b)
			if sim < tt.min {
				t.Errorf("Expected similarity >= %.2f, got %.2f", tt.min, sim)
			}
			t.Logf("Word Jaccard similarity: %.2f", sim)
		})
	}
}
