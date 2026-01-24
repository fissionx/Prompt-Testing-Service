package utils

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// ============================================================================
// ENHANCED SEMANTIC MATCHING FOR OPPORTUNITY DEDUPLICATION
// Uses semantic normalization, synonym expansion, stemming, and fuzzy matching
// NO EXTERNAL LLM CALLS - Pure in-memory semantic understanding
// ============================================================================

// ============================================================================
// COMPREHENSIVE SYNONYM LEXICON - Maps words to canonical forms
// ============================================================================

// synonymLexicon maps words to their canonical/root form for semantic matching
var synonymLexicon = map[string]string{
	// === UPDATE/REFRESH synonyms ===
	"update": "update", "updates": "update", "updating": "update", "updated": "update",
	"refresh": "update", "refreshes": "update", "refreshing": "update", "refreshed": "update",
	"renew": "update", "renews": "update", "renewing": "update", "renewed": "update",
	"revise": "update", "revises": "update", "revising": "update", "revised": "update",
	"modify": "update", "modifies": "update", "modifying": "update", "modified": "update",
	"improve": "update", "improves": "update", "improving": "update", "improved": "update",
	"enhance": "update", "enhances": "update", "enhancing": "update", "enhanced": "update",
	"upgrade": "update", "upgrades": "update", "upgrading": "update", "upgraded": "update",
	"modernize": "update", "modernizes": "update", "modernizing": "update", "modernized": "update",

	// === CREATE/BUILD synonyms ===
	"create": "create", "creates": "create", "creating": "create", "created": "create",
	"build": "create", "builds": "create", "building": "create", "built": "create",
	"develop": "create", "develops": "create", "developing": "create", "developed": "create",
	"write": "create", "writes": "create", "writing": "create", "written": "create", "wrote": "create",
	"publish": "create", "publishes": "create", "publishing": "create", "published": "create",
	"produce": "create", "produces": "create", "producing": "create", "produced": "create",
	"generate": "create", "generates": "create", "generating": "create", "generated": "create",
	"craft": "create", "crafts": "create", "crafting": "create", "crafted": "create",
	"establish": "create", "establishes": "create", "establishing": "create", "established": "create",
	"launch": "create", "launches": "create", "launching": "create", "launched": "create",

	// === ENGAGE/PARTICIPATE synonyms ===
	"engage": "engage", "engages": "engage", "engaging": "engage", "engaged": "engage",
	"participate": "engage", "participates": "engage", "participating": "engage", "participated": "engage",
	"interact": "engage", "interacts": "engage", "interacting": "engage", "interacted": "engage",
	"respond": "engage", "responds": "engage", "responding": "engage", "responded": "engage",
	"comment": "engage", "comments": "engage", "commenting": "engage", "commented": "engage",
	"reply": "engage", "replies": "engage", "replying": "engage", "replied": "engage",
	"join": "engage", "joins": "engage", "joining": "engage", "joined": "engage",
	"contribute": "engage", "contributes": "engage", "contributing": "engage", "contributed": "engage",
	"discuss": "engage", "discusses": "engage", "discussing": "engage", "discussed": "engage",
	"converse": "engage", "converses": "engage", "conversing": "engage", "conversed": "engage",

	// === ADD/INCLUDE synonyms ===
	"add": "add", "adds": "add", "adding": "add", "added": "add",
	"include": "add", "includes": "add", "including": "add", "included": "add",
	"insert": "add", "inserts": "add", "inserting": "add", "inserted": "add",
	"incorporate": "add", "incorporates": "add", "incorporating": "add", "incorporated": "add",
	"implement": "add", "implements": "add", "implementing": "add", "implemented": "add",
	"integrate": "add", "integrates": "add", "integrating": "add", "integrated": "add",

	// === EXPAND/GROW synonyms ===
	"expand": "expand", "expands": "expand", "expanding": "expand", "expanded": "expand",
	"extend": "expand", "extends": "expand", "extending": "expand", "extended": "expand",
	"grow": "expand", "grows": "expand", "growing": "expand", "grew": "expand", "grown": "expand",
	"increase": "expand", "increases": "expand", "increasing": "expand", "increased": "expand",
	"broaden": "expand", "broadens": "expand", "broadening": "expand", "broadened": "expand",
	"scale": "expand", "scales": "expand", "scaling": "expand", "scaled": "expand",
	"amplify": "expand", "amplifies": "expand", "amplifying": "expand", "amplified": "expand",

	// === CONTENT types ===
	"content": "content", "contents": "content",
	"article": "content", "articles": "content",
	"blog": "content", "blogs": "content", "blogpost": "content", "blogposts": "content",
	"post": "content", "posts": "content",
	"page": "content", "pages": "content", "webpage": "content", "webpages": "content",
	"guide": "content", "guides": "content", "guidebook": "content",
	"documentation": "content", "docs": "content", "doc": "content",
	"resource": "content", "resources": "content",
	"material": "content", "materials": "content",

	// === ADMISSION/ENROLLMENT ===
	"admission": "admission", "admissions": "admission",
	"enrollment": "admission", "enrollments": "admission", "enroll": "admission", "enrolls": "admission",
	"apply": "admission", "applies": "admission", "applying": "admission", "applied": "admission",
	"application": "admission", "applications": "admission",
	"applicant": "admission", "applicants": "admission",
	"candidate": "admission", "candidates": "admission",
	"registration": "admission", "registrations": "admission", "register": "admission",

	// === STUDENT/LEARNER ===
	"student": "student", "students": "student",
	"learner": "student", "learners": "student",
	"pupil": "student", "pupils": "student",
	"scholar": "student", "scholars": "student",
	"graduate": "student", "graduates": "student", "graduating": "student",
	"alumni": "student", "alumnus": "student", "alumna": "student",
	"undergrad": "student", "undergraduate": "student", "undergraduates": "student",
	"postgrad": "student", "postgraduate": "student", "postgraduates": "student",

	// === REDDIT/FORUM ===
	"reddit": "reddit", "subreddit": "reddit", "subreddits": "reddit",
	"thread": "reddit", "threads": "reddit",
	"forum": "reddit", "forums": "reddit",
	"community": "reddit", "communities": "reddit",
	"discussion": "reddit", "discussions": "reddit",

	// === FAQ/QUESTION ===
	"faq": "faq", "faqs": "faq",
	"question": "faq", "questions": "faq",
	"answer": "faq", "answers": "faq",
	"howto": "faq", "how-to": "faq",
	"tutorial": "faq", "tutorials": "faq",
	"help": "faq", "helps": "faq",

	// === CASE STUDY/SUCCESS ===
	"case": "casestudy", "cases": "casestudy",
	"casestudy": "casestudy", "case-study": "casestudy",
	"success": "casestudy", "successes": "casestudy", "successful": "casestudy",
	"story": "casestudy", "stories": "casestudy",
	"testimonial": "casestudy", "testimonials": "casestudy",
	"example": "casestudy", "examples": "casestudy",
	"showcase": "casestudy", "showcases": "casestudy",

	// === COMPARISON/VERSUS ===
	"comparison": "comparison", "comparisons": "comparison",
	"compare": "comparison", "compares": "comparison", "comparing": "comparison", "compared": "comparison",
	"versus": "comparison", "vs": "comparison", "v/s": "comparison",
	"alternative": "comparison", "alternatives": "comparison",
	"difference": "comparison", "differences": "comparison", "different": "comparison",
	"contrast": "comparison", "contrasts": "comparison", "contrasting": "comparison",

	// === REVIEW/RATING ===
	"review": "review", "reviews": "review", "reviewing": "review", "reviewed": "review",
	"rating": "review", "ratings": "review", "rate": "review", "rated": "review",
	"feedback": "review", "feedbacks": "review",
	"opinion": "review", "opinions": "review",
	"assessment": "review", "assessments": "review",
	"evaluation": "review", "evaluations": "review",
	"g2": "review", "capterra": "review", "trustradius": "review",

	// === SEO/SEARCH ===
	"seo":    "seo",
	"search": "seo", "searches": "seo", "searching": "seo",
	"optimization": "seo", "optimize": "seo", "optimizes": "seo", "optimizing": "seo", "optimized": "seo",
	"ranking": "seo", "rank": "seo", "ranks": "seo", "ranked": "seo",
	"visibility": "seo", "visible": "seo",
	"schema": "seo", "schemas": "seo",
	"markup": "seo", "markups": "seo",
	"structured": "seo", "structure": "seo",

	// === FRESHNESS/CURRENT ===
	"fresh": "fresh", "freshness": "fresh",
	"latest": "fresh", "late": "fresh",
	"current": "fresh", "currently": "fresh",
	"recent": "fresh", "recently": "fresh",
	"new": "fresh", "newer": "fresh", "newest": "fresh",
	"today": "fresh", "now": "fresh",
	"modern": "fresh", "contemporary": "fresh",
	"date": "fresh", "dates": "fresh", "dated": "fresh",
	"stat": "fresh", "stats": "fresh", "statistics": "fresh", "statistic": "fresh",
	"info": "fresh", "information": "fresh",
	"year": "fresh", "yearly": "fresh", "annual": "fresh",
	"2024": "fresh", "2025": "fresh", "2026": "fresh",

	// === LINKEDIN/PROFESSIONAL ===
	"linkedin":     "linkedin",
	"professional": "linkedin", "professionals": "linkedin",
	"network": "linkedin", "networks": "linkedin", "networking": "linkedin",
	"connection": "linkedin", "connections": "linkedin",
	"career": "linkedin", "careers": "linkedin",

	// === PR/PRESS ===
	"pr":    "pr",
	"press": "pr", "presses": "pr",
	"media": "pr", "medias": "pr",
	"news": "pr", "newsletter": "pr", "newsletters": "pr",
	"coverage": "pr", "coverages": "pr",
	"mention": "pr", "mentions": "pr", "mentioned": "pr",
	"publication": "pr", "publications": "pr",
	"feature": "pr", "features": "pr", "featured": "pr",
	"journalist": "pr", "journalists": "pr",
}

// semanticGroups defines groups of semantically related concepts
var semanticGroups = map[string][]string{
	"content_update":  {"update", "content", "fresh"},
	"community":       {"engage", "reddit", "linkedin"},
	"social_proof":    {"casestudy", "review", "student"},
	"discoverability": {"seo", "comparison", "faq"},
	"outreach":        {"pr", "engage", "linkedin"},
}

// ConceptMapping maps related terms to canonical concepts (expanded)
var conceptMappings = map[string][]string{
	// Content freshness/updates
	"freshness": {"update", "latest", "current", "recent", "new", "fresh", "dates", "stats", "statistics", "year", "2024", "2025", "2026", "refresh", "renew", "modern", "today", "now", "info", "information", "annual", "yearly"},
	// Admissions/enrollment related
	"admissions": {"admission", "admissions", "enrollment", "enroll", "apply", "application", "applicant", "applicants", "student", "students", "candidate", "candidates", "registration", "register", "undergraduate", "postgraduate"},
	// Content types
	"content": {"content", "blog", "article", "post", "page", "pages", "guide", "guides", "landing", "website", "site", "documentation", "docs", "resource", "material", "webpage"},
	// Case studies/success stories
	"case_study": {"case", "study", "studies", "success", "story", "stories", "testimonial", "testimonials", "alumni", "graduate", "graduates", "showcase", "example"},
	// Reddit engagement
	"reddit": {"reddit", "subreddit", "subreddits", "thread", "threads", "r/", "forum", "community", "discussion"},
	// FAQ/Questions
	"faq": {"faq", "faqs", "question", "questions", "answer", "answers", "howto", "how-to", "tutorial", "help"},
	// Comparison content
	"comparison": {"comparison", "compare", "versus", "vs", "alternative", "alternatives", "competitor", "competitors", "difference", "contrast"},
	// Engagement/participation
	"engagement": {"engage", "engagement", "participate", "participation", "interact", "interaction", "comment", "comments", "reply", "respond", "authentic", "authentically", "discuss", "contribute", "join"},
	// LinkedIn
	"linkedin": {"linkedin", "professional", "network", "networking", "connection", "career"},
	// Reviews
	"reviews": {"review", "reviews", "rating", "ratings", "g2", "capterra", "trustradius", "feedback", "assessment", "evaluation"},
	// SEO
	"seo": {"seo", "search", "schema", "structured", "markup", "optimization", "optimize", "rank", "ranking", "visibility"},
	// PR/Press
	"pr": {"pr", "press", "media", "news", "coverage", "mention", "mentions", "publication", "publications", "feature", "journalist", "newsletter"},
}

// ActionVerbs categorizes action types (expanded)
var actionVerbCategories = map[string][]string{
	"update":  {"update", "refresh", "renew", "revise", "modify", "improve", "enhance", "upgrade", "modernize"},
	"create":  {"create", "build", "develop", "write", "publish", "add", "make", "establish", "launch", "produce", "generate", "craft"},
	"engage":  {"engage", "participate", "interact", "respond", "comment", "reply", "join", "contribute", "discuss", "converse"},
	"expand":  {"expand", "extend", "grow", "increase", "broaden", "scale", "amplify"},
	"add":     {"add", "include", "insert", "incorporate", "implement", "integrate"},
	"analyze": {"analyze", "research", "study", "investigate", "assess", "evaluate", "review"},
}

// ============================================================================
// SEMANTIC NORMALIZATION FUNCTIONS
// ============================================================================

// normalizeWord converts a word to its canonical form using the synonym lexicon
func normalizeWord(word string) string {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return ""
	}

	// Direct lookup in synonym lexicon
	if canonical, exists := synonymLexicon[word]; exists {
		return canonical
	}

	// Try with common suffixes removed (simple stemming)
	stems := []string{
		strings.TrimSuffix(word, "ing"),
		strings.TrimSuffix(word, "ed"),
		strings.TrimSuffix(word, "s"),
		strings.TrimSuffix(word, "es"),
		strings.TrimSuffix(word, "tion"),
		strings.TrimSuffix(word, "ment"),
		strings.TrimSuffix(word, "ly"),
		strings.TrimSuffix(word, "ness"),
	}

	for _, stem := range stems {
		if stem != word && len(stem) > 2 {
			if canonical, exists := synonymLexicon[stem]; exists {
				return canonical
			}
		}
	}

	// Fuzzy match for typos (only for longer words)
	if len(word) >= 5 {
		if match := findClosestSynonym(word, 2); match != "" {
			return match
		}
	}

	return word // Return original if no match found
}

// findClosestSynonym finds the closest match in the synonym lexicon using edit distance
func findClosestSynonym(word string, maxDistance int) string {
	bestMatch := ""
	bestDistance := maxDistance + 1

	for key, canonical := range synonymLexicon {
		if abs(len(key)-len(word)) > maxDistance {
			continue // Skip if length difference is too big
		}

		dist := levenshteinDistance(word, key)
		if dist <= maxDistance && dist < bestDistance {
			bestDistance = dist
			bestMatch = canonical
		}
	}

	return bestMatch
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Use two rows instead of full matrix for memory efficiency
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = min(
				prev[j]+1,      // deletion
				curr[j-1]+1,    // insertion
				prev[j-1]+cost, // substitution
			)
		}
		prev, curr = curr, prev
	}

	return prev[len(b)]
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(values ...int) int {
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// semanticTokenize tokenizes text and normalizes each word semantically
func semanticTokenize(text string) []string {
	text = strings.ToLower(text)

	// Split on non-alphanumeric
	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	// Stop words to filter
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
		"are": true, "were": true, "be": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "this": true, "that": true, "these": true, "those": true,
		"it": true, "its": true, "their": true, "your": true, "our": true, "my": true,
		"about": true, "into": true, "through": true, "during": true, "before": true,
		"after": true, "above": true, "below": true, "between": true, "under": true,
	}

	var normalized []string
	seen := make(map[string]bool)

	for _, word := range words {
		if len(word) < 2 || stopWords[word] {
			continue
		}

		// Normalize the word semantically
		canonical := normalizeWord(word)
		if canonical != "" && !seen[canonical] {
			normalized = append(normalized, canonical)
			seen[canonical] = true
		}
	}

	return normalized
}

// semanticSimilarity calculates similarity between two sets of normalized tokens
func semanticSimilarity(tokensA, tokensB []string) float64 {
	if len(tokensA) == 0 && len(tokensB) == 0 {
		return 1.0
	}
	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0.0
	}

	setA := make(map[string]bool)
	for _, t := range tokensA {
		setA[t] = true
	}

	setB := make(map[string]bool)
	for _, t := range tokensB {
		setB[t] = true
	}

	// Count matches (including semantic group matches)
	matches := 0
	for tokenA := range setA {
		if setB[tokenA] {
			matches++
			continue
		}
		// Check if tokens are in the same semantic group
		if areSemanticallySimilar(tokenA, setB) {
			matches++
		}
	}

	// Jaccard-like similarity with semantic awareness
	union := len(setA) + len(setB) - matches
	if union == 0 {
		return 0
	}

	return float64(matches) / float64(union)
}

// areSemanticallySimilar checks if a token is semantically similar to any token in a set
func areSemanticallySimilar(token string, tokenSet map[string]bool) bool {
	// Check semantic groups
	for _, group := range semanticGroups {
		tokenInGroup := false
		setHasGroupMember := false

		for _, member := range group {
			if member == token {
				tokenInGroup = true
			}
			if tokenSet[member] {
				setHasGroupMember = true
			}
		}

		if tokenInGroup && setHasGroupMember {
			return true
		}
	}

	return false
}

// SemanticOpportunity represents a semantically analyzed opportunity
type SemanticOpportunity struct {
	ID              string
	Type            string
	Title           string
	Description     string
	NormalizedText  string
	KeyConcepts     []string
	ActionVerbs     []string
	TargetEntities  []string
	SemanticTokens  []string // NEW: Semantically normalized tokens
	EmbeddingVector []float64
}

// OpportunityMatcher provides semantic matching for opportunities
type OpportunityMatcher struct {
	embedder      *TextEmbedder
	opportunities []SemanticOpportunity
	mu            sync.RWMutex
	threshold     float64
}

// NewOpportunityMatcher creates a new opportunity matcher with 0.80 threshold
func NewOpportunityMatcher(threshold float64) *OpportunityMatcher {
	if threshold <= 0 || threshold > 1 {
		threshold = 0.80 // Default to 80% as requested
	}
	return &OpportunityMatcher{
		embedder:      NewTextEmbedder(),
		opportunities: []SemanticOpportunity{},
		threshold:     threshold,
	}
}

// AddOpportunity adds an opportunity with full semantic analysis
func (m *OpportunityMatcher) AddOpportunity(id, oppType, title, description string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Perform semantic analysis
	semanticOpp := m.analyzeOpportunity(id, oppType, title, description)

	// Add to embedder for TF-IDF
	m.embedder.AddDocument(semanticOpp.NormalizedText)
	semanticOpp.EmbeddingVector = m.embedder.Embed(semanticOpp.NormalizedText)

	m.opportunities = append(m.opportunities, semanticOpp)
}

// AddOpportunityWithEmbedding adds an opportunity with pre-computed embedding from DB
func (m *OpportunityMatcher) AddOpportunityWithEmbedding(opp SemanticOpportunity) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.embedder.AddDocument(opp.NormalizedText)
	m.opportunities = append(m.opportunities, opp)
}

// analyzeOpportunity performs deep semantic analysis with synonym normalization
func (m *OpportunityMatcher) analyzeOpportunity(id, oppType, title, description string) SemanticOpportunity {
	combinedText := strings.ToLower(title + " " + description)

	// NEW: Semantically tokenize and normalize the text
	semanticTokens := semanticTokenize(title + " " + description)

	// Extract key concepts
	keyConcepts := extractConcepts(combinedText)

	// Extract action verbs
	actionVerbs := extractActionVerbs(combinedText)

	// Extract target entities (platforms, industries, etc.)
	targetEntities := extractTargetEntities(combinedText)

	// Create normalized text for embedding (now includes semantic tokens)
	normalizedText := createNormalizedText(oppType, keyConcepts, actionVerbs, targetEntities)

	return SemanticOpportunity{
		ID:             id,
		Type:           oppType,
		Title:          title,
		Description:    description,
		NormalizedText: normalizedText,
		KeyConcepts:    keyConcepts,
		ActionVerbs:    actionVerbs,
		TargetEntities: targetEntities,
		SemanticTokens: semanticTokens,
	}
}

// extractConcepts maps text to canonical concepts
func extractConcepts(text string) []string {
	conceptSet := make(map[string]bool)
	text = strings.ToLower(text)

	for concept, keywords := range conceptMappings {
		for _, kw := range keywords {
			if strings.Contains(text, kw) {
				conceptSet[concept] = true
				break
			}
		}
	}

	var concepts []string
	for c := range conceptSet {
		concepts = append(concepts, c)
	}
	sort.Strings(concepts)
	return concepts
}

// extractActionVerbs identifies the primary action type
func extractActionVerbs(text string) []string {
	actionSet := make(map[string]bool)
	text = strings.ToLower(text)
	words := strings.Fields(text)

	for _, word := range words {
		word = strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r)
		})
		for actionType, verbs := range actionVerbCategories {
			for _, v := range verbs {
				if word == v {
					actionSet[actionType] = true
				}
			}
		}
	}

	var actions []string
	for a := range actionSet {
		actions = append(actions, a)
	}
	sort.Strings(actions)
	return actions
}

// extractTargetEntities extracts platforms, industries, and specific targets
func extractTargetEntities(text string) []string {
	entities := make(map[string]bool)
	text = strings.ToLower(text)

	// Platform keywords
	platforms := []string{"linkedin", "reddit", "twitter", "g2", "capterra", "trustradius",
		"medium", "youtube", "facebook", "instagram", "quora", "stackoverflow",
		"producthunt", "hackernews", "google", "bing", "perplexity"}
	for _, p := range platforms {
		if strings.Contains(text, p) {
			entities["platform:"+p] = true
		}
	}

	// Industry keywords
	industries := []string{"healthcare", "fintech", "finance", "ecommerce", "retail",
		"saas", "enterprise", "startup", "education", "legal", "insurance", "manufacturing",
		"logistics", "hospitality", "realestate", "automotive", "telecom", "media", "gaming",
		"biotech", "pharma", "energy", "agriculture", "admissions", "engineering"}
	for _, ind := range industries {
		if strings.Contains(text, ind) {
			entities["industry:"+ind] = true
		}
	}

	// Extract subreddits (r/something)
	subredditRe := regexp.MustCompile(`r/([a-zA-Z0-9_]+)`)
	matches := subredditRe.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		if len(m) > 1 {
			entities["subreddit:"+strings.ToLower(m[1])] = true
		}
	}

	// Content types
	contentTypes := []string{"blog", "faq", "case study", "comparison", "landing page", "guide", "schema"}
	for _, ct := range contentTypes {
		if strings.Contains(text, ct) {
			entities["content:"+strings.ReplaceAll(ct, " ", "_")] = true
		}
	}

	var result []string
	for e := range entities {
		result = append(result, e)
	}
	sort.Strings(result)
	return result
}

// createNormalizedText creates a normalized representation for embedding
func createNormalizedText(oppType string, concepts, actions, entities []string) string {
	parts := []string{oppType}
	parts = append(parts, concepts...)
	parts = append(parts, actions...)
	parts = append(parts, entities...)
	return strings.Join(parts, " ")
}

// FindDuplicate checks if an opportunity is semantically similar to existing ones
// Returns (isDuplicate, similarID, similarityScore)
func (m *OpportunityMatcher) FindDuplicate(oppType, title, description string) (bool, string, float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.opportunities) == 0 {
		return false, "", 0
	}

	// Analyze the new opportunity
	newOpp := m.analyzeOpportunity("", oppType, title, description)

	var bestMatch struct {
		ID         string
		Similarity float64
	}
	bestMatch.Similarity = -1

	for _, existing := range m.opportunities {
		similarity := m.calculateSemanticSimilarity(newOpp, existing)

		if similarity > bestMatch.Similarity {
			bestMatch.ID = existing.ID
			bestMatch.Similarity = similarity
		}
	}

	isDuplicate := bestMatch.Similarity >= m.threshold
	return isDuplicate, bestMatch.ID, bestMatch.Similarity
}

// calculateSemanticSimilarity computes similarity using multiple signals including semantic understanding
func (m *OpportunityMatcher) calculateSemanticSimilarity(a, b SemanticOpportunity) float64 {
	// 1. Concept overlap (30%)
	conceptSim := jaccardSimilarity(a.KeyConcepts, b.KeyConcepts)

	// 2. Action verb match (20%)
	actionSim := jaccardSimilarity(a.ActionVerbs, b.ActionVerbs)

	// 3. Target entity overlap (15%)
	entitySim := jaccardSimilarity(a.TargetEntities, b.TargetEntities)

	// 4. NEW: Semantic token similarity with synonym awareness (25%)
	// This uses the normalized tokens where synonyms map to the same canonical form
	semanticSim := semanticSimilarity(a.SemanticTokens, b.SemanticTokens)

	// 5. Type match bonus (10%)
	typeSim := 0.0
	if a.Type == b.Type {
		typeSim = 1.0
	}

	// Weighted combination with semantic understanding
	baseSimilarity := conceptSim*0.30 + actionSim*0.20 + entitySim*0.15 + semanticSim*0.25 + typeSim*0.10

	// Apply boosting rules for strong semantic matches

	// BOOST: Same core concept + same action = likely duplicate
	if len(a.KeyConcepts) > 0 && len(b.KeyConcepts) > 0 {
		// Check for overlap in primary concepts
		hasConceptOverlap := false
		for _, ac := range a.KeyConcepts {
			for _, bc := range b.KeyConcepts {
				if ac == bc {
					hasConceptOverlap = true
					break
				}
			}
			if hasConceptOverlap {
				break
			}
		}

		// Check for same action
		hasSameAction := false
		for _, aa := range a.ActionVerbs {
			for _, ba := range b.ActionVerbs {
				if aa == ba {
					hasSameAction = true
					break
				}
			}
			if hasSameAction {
				break
			}
		}

		// Strong boost if both concepts and actions match
		if hasConceptOverlap && hasSameAction {
			baseSimilarity += 0.15
		} else if hasConceptOverlap {
			baseSimilarity += 0.10
		}
	}

	// BOOST: High semantic token similarity = strong semantic match
	if semanticSim > 0.6 {
		baseSimilarity += 0.10
	}

	// BOOST: Same opportunity type with similar content
	if a.Type == b.Type && baseSimilarity > 0.5 {
		baseSimilarity += 0.10
	}

	// PENALTY: Different specific entities (e.g., different subreddits)
	if hasSpecificEntityMismatch(a.TargetEntities, b.TargetEntities) {
		baseSimilarity *= 0.6
	}

	// Cap at 1.0
	if baseSimilarity > 1.0 {
		baseSimilarity = 1.0
	}

	return baseSimilarity
}

// hasSpecificEntityMismatch checks if two opportunities target different specific entities
func hasSpecificEntityMismatch(a, b []string) bool {
	// Group entities by type
	aByType := groupEntitiesByType(a)
	bByType := groupEntitiesByType(b)

	// Check subreddit mismatch
	if len(aByType["subreddit"]) > 0 && len(bByType["subreddit"]) > 0 {
		if !hasOverlap(aByType["subreddit"], bByType["subreddit"]) {
			return true
		}
	}

	// Check platform mismatch (only if both specify platforms)
	if len(aByType["platform"]) > 0 && len(bByType["platform"]) > 0 {
		if !hasOverlap(aByType["platform"], bByType["platform"]) {
			return true
		}
	}

	return false
}

// groupEntitiesByType groups entities like "platform:linkedin" by their type
func groupEntitiesByType(entities []string) map[string][]string {
	result := make(map[string][]string)
	for _, e := range entities {
		parts := strings.SplitN(e, ":", 2)
		if len(parts) == 2 {
			result[parts[0]] = append(result[parts[0]], parts[1])
		}
	}
	return result
}

// jaccardSimilarity calculates Jaccard similarity between two string slices
func jaccardSimilarity(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0 // Both empty = same
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	setA := make(map[string]bool)
	for _, s := range a {
		setA[s] = true
	}

	setB := make(map[string]bool)
	for _, s := range b {
		setB[s] = true
	}

	intersection := 0
	for s := range setA {
		if setB[s] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// GetSemanticOpportunity returns the semantic analysis for an opportunity with embedding vector
func (m *OpportunityMatcher) GetSemanticOpportunity(id, oppType, title, description string) SemanticOpportunity {
	semanticOpp := m.analyzeOpportunity(id, oppType, title, description)

	// Generate embedding vector
	m.embedder.AddDocument(semanticOpp.NormalizedText)
	semanticOpp.EmbeddingVector = m.embedder.Embed(semanticOpp.NormalizedText)

	return semanticOpp
}

// Size returns the number of stored opportunities
func (m *OpportunityMatcher) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.opportunities)
}

// Clear removes all stored opportunities
func (m *OpportunityMatcher) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.opportunities = []SemanticOpportunity{}
	m.embedder = NewTextEmbedder()
}

// FindSimilar returns all opportunities above the similarity threshold
func (m *OpportunityMatcher) FindSimilar(oppType, title, description string, topK int) []SimilarityResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.opportunities) == 0 {
		return nil
	}

	newOpp := m.analyzeOpportunity("", oppType, title, description)

	var results []SimilarityResult
	for _, existing := range m.opportunities {
		similarity := m.calculateSemanticSimilarity(newOpp, existing)
		results = append(results, SimilarityResult{
			ID:         existing.ID,
			Similarity: similarity,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	return results
}

// ============================================================================
// HELPER TYPES AND FUNCTIONS
// ============================================================================

// SimilarityResult represents the result of a similarity search
type SimilarityResult struct {
	ID         string
	Similarity float64
}

// OpportunityEmbedding stores an opportunity's embedding (for backward compat)
type OpportunityEmbedding struct {
	ID          string
	Type        string
	Title       string
	Description string
	Embedding   []float64
}

// hasOverlap checks if two string slices have any common elements
func hasOverlap(a, b []string) bool {
	setA := make(map[string]bool)
	for _, s := range a {
		setA[s] = true
	}
	for _, s := range b {
		if setA[s] {
			return true
		}
	}
	return false
}

// wordJaccardSimilarity calculates Jaccard similarity on word sets
func wordJaccardSimilarity(a, b string) float64 {
	wordsA := tokenizeForSimilarity(a)
	wordsB := tokenizeForSimilarity(b)

	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}

	setA := make(map[string]bool)
	for _, w := range wordsA {
		setA[w] = true
	}

	setB := make(map[string]bool)
	for _, w := range wordsB {
		setB[w] = true
	}

	intersection := 0
	for w := range setA {
		if setB[w] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// tokenizeForSimilarity tokenizes text for similarity comparison
func tokenizeForSimilarity(text string) []string {
	text = strings.ToLower(text)

	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
		"are": true, "were": true, "be": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "this": true, "that": true, "these": true, "those": true,
		"about": true,
	}

	words := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var result []string
	for _, w := range words {
		if len(w) > 2 && !stopWords[w] {
			result = append(result, w)
		}
	}
	return result
}

// ============================================================================
// TF-IDF EMBEDDER (Used as additional signal)
// ============================================================================

// TextEmbedder provides in-memory text embedding using TF-IDF vectors
type TextEmbedder struct {
	mu          sync.RWMutex
	vocabulary  map[string]int
	idfWeights  map[string]float64
	documents   []string
	stopWords   map[string]bool
	initialized bool
}

// NewTextEmbedder creates a new in-memory text embedder
func NewTextEmbedder() *TextEmbedder {
	return &TextEmbedder{
		vocabulary: make(map[string]int),
		idfWeights: make(map[string]float64),
		documents:  []string{},
		stopWords:  buildStopWords(),
	}
}

func buildStopWords() map[string]bool {
	words := []string{
		"a", "an", "the", "and", "or", "but", "in", "on", "at", "to", "for",
		"of", "with", "by", "from", "as", "is", "was", "are", "were", "been",
		"be", "have", "has", "had", "do", "does", "did", "will", "would",
		"could", "should", "may", "might", "must", "shall", "can", "need",
		"this", "that", "these", "those", "it", "its", "they", "them", "their",
		"we", "our", "you", "your", "i", "my", "me", "he", "she", "his", "her",
		"what", "which", "who", "whom", "when", "where", "why", "how",
		"all", "each", "every", "both", "few", "more", "most", "other", "some",
		"such", "no", "not", "only", "same", "so", "than", "too", "very",
		"just", "also", "now", "here", "there", "then", "once", "about",
	}
	stopWords := make(map[string]bool)
	for _, w := range words {
		stopWords[w] = true
	}
	return stopWords
}

// Tokenize converts text to normalized tokens
func (e *TextEmbedder) Tokenize(text string) []string {
	text = strings.ToLower(text)
	reg := regexp.MustCompile(`[^a-z0-9\s]`)
	text = reg.ReplaceAllString(text, " ")
	words := strings.Fields(text)

	var tokens []string
	for _, word := range words {
		word = strings.TrimSpace(word)
		if len(word) < 2 {
			continue
		}
		if e.stopWords[word] {
			continue
		}
		tokens = append(tokens, word)
	}
	return tokens
}

// AddDocument adds a document to the corpus for IDF calculation
func (e *TextEmbedder) AddDocument(text string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.documents = append(e.documents, text)
	e.initialized = false
}

// AddDocuments adds multiple documents to the corpus
func (e *TextEmbedder) AddDocuments(texts []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.documents = append(e.documents, texts...)
	e.initialized = false
}

func (e *TextEmbedder) buildVocabulary() {
	if e.initialized {
		return
	}

	e.vocabulary = make(map[string]int)
	e.idfWeights = make(map[string]float64)

	docFreq := make(map[string]int)
	allTerms := make(map[string]bool)

	for _, doc := range e.documents {
		tokens := e.Tokenize(doc)
		seen := make(map[string]bool)
		for _, token := range tokens {
			allTerms[token] = true
			if !seen[token] {
				docFreq[token]++
				seen[token] = true
			}
		}
	}

	var terms []string
	for term := range allTerms {
		terms = append(terms, term)
	}
	sort.Strings(terms)

	for i, term := range terms {
		e.vocabulary[term] = i
	}

	numDocs := float64(len(e.documents))
	if numDocs == 0 {
		numDocs = 1
	}

	for term, df := range docFreq {
		e.idfWeights[term] = math.Log(numDocs/float64(df)) + 1
	}

	e.initialized = true
}

// Embed creates a TF-IDF vector for the given text
func (e *TextEmbedder) Embed(text string) []float64 {
	e.mu.Lock()
	e.buildVocabulary()
	e.mu.Unlock()

	e.mu.RLock()
	defer e.mu.RUnlock()

	tokens := e.Tokenize(text)
	if len(tokens) == 0 {
		return nil
	}

	tf := make(map[string]float64)
	for _, token := range tokens {
		tf[token]++
	}

	docLen := float64(len(tokens))
	for term := range tf {
		tf[term] /= docLen
	}

	vocabSize := len(e.vocabulary)
	if vocabSize == 0 {
		return e.hashEmbed(text)
	}

	vector := make([]float64, vocabSize)
	for term, termFreq := range tf {
		if idx, exists := e.vocabulary[term]; exists {
			idf := e.idfWeights[term]
			if idf == 0 {
				idf = 1.0
			}
			vector[idx] = termFreq * idf
		}
	}

	return normalizeVector(vector)
}

func (e *TextEmbedder) hashEmbed(text string) []float64 {
	tokens := e.Tokenize(text)
	if len(tokens) == 0 {
		return nil
	}

	dim := 256
	vector := make([]float64, dim)

	for _, token := range tokens {
		hash := 0
		for _, c := range token {
			hash = (hash*31 + int(c)) % dim
		}
		vector[hash] += 1.0
	}

	return normalizeVector(vector)
}

func normalizeVector(v []float64) []float64 {
	var norm float64
	for _, val := range v {
		norm += val * val
	}
	norm = math.Sqrt(norm)

	if norm == 0 {
		return v
	}

	result := make([]float64, len(v))
	for i, val := range v {
		result[i] = val / norm
	}
	return result
}

// CosineSimilarity calculates the cosine similarity between two vectors
func CosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	var dotProduct float64
	var normA, normB float64

	for i := 0; i < minLen; i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	for i := minLen; i < len(a); i++ {
		normA += a[i] * a[i]
	}
	for i := minLen; i < len(b); i++ {
		normB += b[i] * b[i]
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (normA * normB)
}

// ============================================================================
// N-GRAM FUNCTIONS (For backward compatibility)
// ============================================================================

// CharacterNGrams generates character n-grams for fuzzy matching
func CharacterNGrams(text string, n int) []string {
	text = strings.ToLower(text)

	var cleaned []rune
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			cleaned = append(cleaned, r)
		}
	}
	text = string(cleaned)

	if len(text) < n {
		return []string{text}
	}

	var ngrams []string
	runes := []rune(text)
	for i := 0; i <= len(runes)-n; i++ {
		ngrams = append(ngrams, string(runes[i:i+n]))
	}
	return ngrams
}

// JaccardSimilarityNGrams calculates Jaccard similarity using character n-grams
func JaccardSimilarityNGrams(a, b string, n int) float64 {
	ngramsA := CharacterNGrams(a, n)
	ngramsB := CharacterNGrams(b, n)

	setA := make(map[string]bool)
	for _, ng := range ngramsA {
		setA[ng] = true
	}

	setB := make(map[string]bool)
	for _, ng := range ngramsB {
		setB[ng] = true
	}

	intersection := 0
	for ng := range setA {
		if setB[ng] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}
