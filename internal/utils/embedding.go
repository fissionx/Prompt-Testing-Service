package utils

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"
)

// TextEmbedder provides in-memory text embedding using TF-IDF vectors
// for semantic similarity comparison without external LLM calls
type TextEmbedder struct {
	mu          sync.RWMutex
	vocabulary  map[string]int    // word -> index mapping
	idfWeights  map[string]float64 // word -> IDF weight
	documents   []string          // stored documents for IDF calculation
	stopWords   map[string]bool   // common words to ignore
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

// buildStopWords returns a set of common English stop words
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
	// Convert to lowercase
	text = strings.ToLower(text)

	// Replace non-alphanumeric with spaces
	reg := regexp.MustCompile(`[^a-z0-9\s]`)
	text = reg.ReplaceAllString(text, " ")

	// Split into words
	words := strings.Fields(text)

	// Filter stop words and short words
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
	e.initialized = false // Mark for recalculation
}

// AddDocuments adds multiple documents to the corpus
func (e *TextEmbedder) AddDocuments(texts []string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.documents = append(e.documents, texts...)
	e.initialized = false
}

// buildVocabulary builds the vocabulary and IDF weights from documents
func (e *TextEmbedder) buildVocabulary() {
	if e.initialized {
		return
	}

	e.vocabulary = make(map[string]int)
	e.idfWeights = make(map[string]float64)

	// Count document frequency for each term
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

	// Build vocabulary with sorted keys for consistency
	var terms []string
	for term := range allTerms {
		terms = append(terms, term)
	}
	sort.Strings(terms)

	for i, term := range terms {
		e.vocabulary[term] = i
	}

	// Calculate IDF weights
	numDocs := float64(len(e.documents))
	if numDocs == 0 {
		numDocs = 1
	}

	for term, df := range docFreq {
		// IDF = log(N / df) + 1 (smoothed)
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

	// Calculate term frequencies
	tf := make(map[string]float64)
	for _, token := range tokens {
		tf[token]++
	}

	// Normalize TF by document length
	docLen := float64(len(tokens))
	for term := range tf {
		tf[term] /= docLen
	}

	// Create TF-IDF vector
	vocabSize := len(e.vocabulary)
	if vocabSize == 0 {
		// No vocabulary yet, create a simple hash-based vector
		return e.hashEmbed(text)
	}

	vector := make([]float64, vocabSize)
	for term, termFreq := range tf {
		if idx, exists := e.vocabulary[term]; exists {
			idf := e.idfWeights[term]
			if idf == 0 {
				idf = 1.0 // Default IDF for unknown terms
			}
			vector[idx] = termFreq * idf
		}
	}

	// Normalize the vector
	return normalizeVector(vector)
}

// hashEmbed creates a simple hash-based embedding when vocabulary is empty
func (e *TextEmbedder) hashEmbed(text string) []float64 {
	tokens := e.Tokenize(text)
	if len(tokens) == 0 {
		return nil
	}

	// Use a fixed dimension hash space
	dim := 256
	vector := make([]float64, dim)

	for _, token := range tokens {
		// Simple hash function
		hash := 0
		for _, c := range token {
			hash = (hash*31 + int(c)) % dim
		}
		vector[hash] += 1.0
	}

	return normalizeVector(vector)
}

// normalizeVector normalizes a vector to unit length
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

	// Handle different vector sizes by using the smaller dimension
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

	// Include remaining elements in norms
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

// SimilarityResult represents the result of a similarity search
type SimilarityResult struct {
	ID         string
	Similarity float64
}

// OpportunityEmbedding stores an opportunity's embedding
type OpportunityEmbedding struct {
	ID          string
	Type        string
	Title       string
	Description string
	Embedding   []float64
}

// OpportunityMatcher provides in-memory semantic matching for opportunities
type OpportunityMatcher struct {
	embedder    *TextEmbedder
	embeddings  []OpportunityEmbedding
	mu          sync.RWMutex
	threshold   float64 // Similarity threshold for duplicate detection
}

// NewOpportunityMatcher creates a new opportunity matcher
func NewOpportunityMatcher(threshold float64) *OpportunityMatcher {
	if threshold <= 0 || threshold > 1 {
		threshold = 0.75 // Default threshold
	}
	return &OpportunityMatcher{
		embedder:   NewTextEmbedder(),
		embeddings: []OpportunityEmbedding{},
		threshold:  threshold,
	}
}

// AddOpportunity adds an opportunity to the matcher
func (m *OpportunityMatcher) AddOpportunity(id, oppType, title, description string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create combined text for embedding
	combinedText := oppType + " " + title + " " + description
	m.embedder.AddDocument(combinedText)

	// Generate embedding
	embedding := m.embedder.Embed(combinedText)

	m.embeddings = append(m.embeddings, OpportunityEmbedding{
		ID:          id,
		Type:        oppType,
		Title:       title,
		Description: description,
		Embedding:   embedding,
	})
}

// FindDuplicate checks if an opportunity is semantically similar to existing ones
// Returns (isDuplicate, similarID, similarityScore)
func (m *OpportunityMatcher) FindDuplicate(oppType, title, description string) (bool, string, float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.embeddings) == 0 {
		return false, "", 0
	}

	// Extract key entities from new opportunity
	newEntities := extractKeyEntities(title + " " + description)
	
	// Find most similar opportunity using multiple similarity measures
	var bestMatch SimilarityResult
	bestMatch.Similarity = -1

	for _, existing := range m.embeddings {
		// Extract key entities from existing opportunity
		existingEntities := extractKeyEntities(existing.Title + " " + existing.Description)
		
		// Check for entity mismatch (different platforms, industries, etc.)
		entityMismatch := hasEntityMismatch(newEntities, existingEntities)
		
		// Check for exact entity matches (same subreddit, same platform, etc.)
		sameSubreddit := len(newEntities.Subreddits) > 0 && hasOverlap(newEntities.Subreddits, existingEntities.Subreddits)
		samePlatform := len(newEntities.Platforms) > 0 && hasOverlap(newEntities.Platforms, existingEntities.Platforms)
		sameIndustry := len(newEntities.Industries) > 0 && hasOverlap(newEntities.Industries, existingEntities.Industries)
		
		// Calculate multiple similarity scores
		
		// 1. Title similarity using n-grams (fuzzy matching)
		titleSim := JaccardSimilarityNGrams(title, existing.Title, 3)
		
		// 2. Description similarity using n-grams
		descSim := JaccardSimilarityNGrams(description, existing.Description, 3)
		
		// 3. Word overlap (Jaccard on words)
		titleWordSim := wordJaccardSimilarity(title, existing.Title)
		descWordSim := wordJaccardSimilarity(description, existing.Description)
		
		// Combine scores with weights
		var combinedSim float64
		
		if existing.Type == oppType {
			// Same type: heavily weight title similarity
			combinedSim = titleSim*0.4 + titleWordSim*0.3 + descSim*0.15 + descWordSim*0.15
			
			// Boost for same type + similar title
			if titleSim > 0.5 || titleWordSim > 0.5 {
				combinedSim += 0.15
			}
			
			// BOOST for matching key entities (same subreddit, platform, industry)
			if sameSubreddit {
				combinedSim += 0.25 // Strong boost for same subreddit
			}
			if samePlatform && !sameSubreddit {
				combinedSim += 0.2 // Boost for same platform
			}
			if sameIndustry {
				combinedSim += 0.1 // Smaller boost for same industry
			}
			
			// PENALTY for different key entities (different industry, platform, etc.)
			if entityMismatch {
				combinedSim *= 0.4 // Heavy penalty - 60% reduction
			}
		} else {
			// Different type: only consider duplicate if both title AND description are very similar
			combinedSim = (titleSim*0.3 + titleWordSim*0.3 + descSim*0.2 + descWordSim*0.2) * 0.7
		}
		
		// Cap at 1.0
		if combinedSim > 1.0 {
			combinedSim = 1.0
		}

		if combinedSim > bestMatch.Similarity {
			bestMatch.ID = existing.ID
			bestMatch.Similarity = combinedSim
		}
	}

	isDuplicate := bestMatch.Similarity >= m.threshold
	return isDuplicate, bestMatch.ID, bestMatch.Similarity
}

// KeyEntities holds extracted entity categories
type KeyEntities struct {
	Platforms  []string
	Industries []string
	Subreddits []string
	Topics     []string
}

// extractKeyEntities extracts important entities from text
func extractKeyEntities(text string) KeyEntities {
	text = strings.ToLower(text)
	entities := KeyEntities{}
	
	// Platform keywords
	platforms := []string{"linkedin", "reddit", "twitter", "g2", "capterra", "trustradius", 
		"medium", "youtube", "facebook", "instagram", "tiktok", "quora", "stackoverflow",
		"producthunt", "hackernews", "slack", "discord"}
	for _, p := range platforms {
		if strings.Contains(text, p) {
			entities.Platforms = append(entities.Platforms, p)
		}
	}
	
	// Industry keywords
	industries := []string{"healthcare", "fintech", "finance", "ecommerce", "retail", 
		"saas", "enterprise", "startup", "education", "legal", "insurance", "manufacturing",
		"logistics", "hospitality", "realestate", "automotive", "telecom", "media", "gaming",
		"biotech", "pharma", "energy", "agriculture"}
	for _, ind := range industries {
		if strings.Contains(text, ind) {
			entities.Industries = append(entities.Industries, ind)
		}
	}
	
	// Subreddit patterns (r/something)
	if idx := strings.Index(text, "r/"); idx != -1 {
		// Extract subreddit name
		end := idx + 2
		for end < len(text) && (text[end] >= 'a' && text[end] <= 'z' || text[end] >= '0' && text[end] <= '9' || text[end] == '_') {
			end++
		}
		if end > idx+2 {
			entities.Subreddits = append(entities.Subreddits, text[idx:end])
		}
	}
	
	return entities
}

// hasEntityMismatch checks if two opportunities target different entities
func hasEntityMismatch(a, b KeyEntities) bool {
	// Check platform mismatch
	if len(a.Platforms) > 0 && len(b.Platforms) > 0 {
		if !hasOverlap(a.Platforms, b.Platforms) {
			return true
		}
	}
	
	// Check industry mismatch
	if len(a.Industries) > 0 && len(b.Industries) > 0 {
		if !hasOverlap(a.Industries, b.Industries) {
			return true
		}
	}
	
	// Check subreddit mismatch
	if len(a.Subreddits) > 0 && len(b.Subreddits) > 0 {
		if !hasOverlap(a.Subreddits, b.Subreddits) {
			return true
		}
	}
	
	return false
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
	
	// Common stop words to remove
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "was": true,
		"are": true, "were": true, "be": true, "have": true, "has": true, "had": true,
		"do": true, "does": true, "did": true, "will": true, "would": true, "could": true,
		"should": true, "this": true, "that": true, "these": true, "those": true,
		"about": true, "create": true, "write": true, "add": true, "make": true,
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

// FindSimilar returns all opportunities above the similarity threshold
func (m *OpportunityMatcher) FindSimilar(oppType, title, description string, topK int) []SimilarityResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.embeddings) == 0 {
		return nil
	}

	// Create embedding for the new opportunity
	combinedText := oppType + " " + title + " " + description
	newEmbedding := m.embedder.Embed(combinedText)

	if newEmbedding == nil {
		return nil
	}

	// Calculate similarities
	var results []SimilarityResult
	for _, existing := range m.embeddings {
		similarity := CosineSimilarity(newEmbedding, existing.Embedding)

		// Boost similarity if same type
		if existing.Type == oppType {
			similarity = similarity*0.7 + 0.3
		}

		results = append(results, SimilarityResult{
			ID:         existing.ID,
			Similarity: similarity,
		})
	}

	// Sort by similarity descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Return top K
	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	return results
}

// Clear removes all stored embeddings
func (m *OpportunityMatcher) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.embeddings = []OpportunityEmbedding{}
	m.embedder = NewTextEmbedder()
}

// Size returns the number of stored opportunities
func (m *OpportunityMatcher) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.embeddings)
}

// CharacterNGrams generates character n-grams for fuzzy matching
func CharacterNGrams(text string, n int) []string {
	text = strings.ToLower(text)
	
	// Remove non-alphanumeric
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
