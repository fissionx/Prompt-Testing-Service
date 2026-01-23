package utils

import (
	"fmt"
	"strings"
)

// BrandMetadataDerivationPrompt generates a prompt for deriving brand domain and category
func BrandMetadataDerivationPrompt(brandContext string) string {
	return fmt.Sprintf(`Analyze this brand based on the provided information and determine its industry domain and BROAD category.

IMPORTANT: Use BROAD, GENERIC categories that apply to many similar organizations. DO NOT be too specific.

%s

Respond in EXACTLY this format (one line for domain, one for category):
Domain: <industry domain like "technology", "healthcare", "finance", "retail", "education", etc>
Category: <BROAD category that would apply to similar organizations>

Examples of GOOD (broad) categories:
- Domain: technology, Category: ai tools
- Domain: technology, Category: crm software
- Domain: education, Category: engineering college
- Domain: education, Category: business school
- Domain: healthcare, Category: hospital
- Domain: finance, Category: payment platform

Examples of BAD (too specific) categories:
- "premier higher education institution" ❌ (too specific, use "engineering college")
- "technical university in south asia" ❌ (too specific, use "engineering college")
- "AI-powered content optimization platform" ❌ (too specific, use "ai tools")

Choose the BROADEST category that accurately describes what this organization does.`, brandContext)
}

// PromptGenerationPrompt generates a prompt for creating new prompts for a brand
func PromptGenerationPrompt(brandInfo, existingText, brand string, count int, distribution map[string]int) string {
	return fmt.Sprintf(`Generate %d unique, natural questions that people would ask when searching for products/services in this space.

%s%s

CRITICAL REQUIREMENTS:
1. Questions MUST BE GENERIC - DO NOT mention the specific brand name "%s"
2. Questions should apply to ANY brand in this category
3. Questions MUST be balanced across these 5 types:

TYPE 1: WHAT QUESTIONS (%d questions) [PREFIX: WHAT|]
- Definitional, explanatory questions
- Examples: "What is GEO?", "What are the benefits of..."
- Format: WHAT|What is GEO and why is it important?

TYPE 2: HOW QUESTIONS (%d questions) [PREFIX: HOW|]
- Instructional, process-oriented questions
- Examples: "How to appear in AI search?", "How does X work?"
- Format: HOW|How to optimize content for AI search?

TYPE 3: COMPARISON QUESTIONS (%d questions) [PREFIX: COMPARE|]
- Competitive analysis, versus questions
- Examples: "X vs Y", "Which is better for...", "How does X compare to Y?"
- Format: COMPARE|How does brand A compare to brand B for SEO?

TYPE 4: TOP/BEST QUESTIONS (%d questions) [PREFIX: TOPBEST|]
- List-based, recommendation questions
- Examples: "Best AI SEO tools", "Top platforms for...", "Most popular..."
- Format: TOPBEST|What are the best AI SEO tools for small businesses?

TYPE 5: BRAND-SPECIFIC PATTERN (%d questions) [PREFIX: BRAND|]
- Questions that follow "what does [BRAND] do" pattern (but keep generic)
- Examples: "What features do [category] platforms offer?", "How do [category] tools help?"
- Format: BRAND|What features do AI SEO platforms typically offer?

IMPORTANT:
- Use EXACT prefixes (WHAT|, HOW|, COMPARE|, TOPBEST|, BRAND|)
- Generate the EXACT count for each type as specified above
- Questions must be generic (no specific brand names)
- One question per line
- NO numbers, bullets, or extra formatting

Generate exactly %d questions in this format:`, count, brandInfo, existingText, brand,
		distribution["what"], distribution["how"], distribution["comparison"], distribution["top_best"], distribution["brand"], count)
}

// GEOPromptTemplate generates a prompt template for GEO statistics
func GEOPromptTemplate(userInput string, existingPrompts []string, language string, count int) string {
	existingPromptsText := ""
	if len(existingPrompts) > 0 {
		existingPromptsText = fmt.Sprintf(`

IMPORTANT: Avoid repetition! The following prompts already exist in the system:
%s

Please generate NEW prompts that are different from the existing ones above.`, strings.Join(existingPrompts, "\n"))
	}

	languageInstruction := ""
	if language != "EN" {
		languageInstruction = fmt.Sprintf(`

IMPORTANT: All prompts must be written in %s language. Do not use English unless specifically requested.`, language)
	}

	return fmt.Sprintf(`You are a prompt generation assistant. The user wants to create prompts that people would naturally ask LLMs when looking for information.

User's request: %s%s%s

Please generate exactly %d different prompts that users would typically ask when searching for information. Each prompt should:
1. Be a natural, conversational question or request that people would ask an AI assistant
2. Be likely to generate responses that mention specific brands, products, services, or companies
3. Be varied in style and approach (questions, requests, scenarios, comparisons, recommendations)
4. Be suitable for different LLM models
5. Sound like something a real person would ask when looking for information
6. Cover different aspects and angles of the topic%s

Format your response as a simple list with each prompt on a new line. Do not include numbers, bullet points, dashes (-), or any additional text or explanations.`, userInput, existingPromptsText, languageInstruction, count, languageInstruction)
}

// PromptSuggestionPrompt generates a prompt for suggesting SEO-optimized prompts for a brand
func PromptSuggestionPrompt(brandContext string, count int) string {
	return fmt.Sprintf(`You are a prompt generation expert. Based on the following brand information, generate %d SEO-optimized prompts/questions that would help this brand appear in AI search results.

%s

---

TASK: Generate %d diverse prompts/questions that:
1. Are relevant to this brand/company
2. Would help the brand appear in AI search results (like ChatGPT, Gemini, etc.)
3. Cover different question types (what, how, comparison, top/best lists, brand-specific)
4. Are natural, conversational, and search-friendly

RULES:
1. Each prompt should be a complete question or search query
2. Make prompts specific and actionable
3. Include brand name naturally where appropriate
4. Vary the question types (what, how, which, best, top, etc.)
5. Focus on topics relevant to the brand's domain/category

RESPOND WITH ONLY A JSON ARRAY of prompt strings. No explanations, no markdown, just the JSON array.

Example response format:
["What is Brand X?", "How does Brand X work?", "Best alternatives to Brand X", "Brand X vs Competitor Y"]

RESPOND NOW:`, count, brandContext, count)
}

// CompetitorSuggestionPrompt generates a prompt for identifying competitors
func CompetitorSuggestionPrompt(brandContext string) string {
	return fmt.Sprintf(`You are a competitive intelligence analyst. Based on the following brand information, identify the main competitors in the market.

%s

---

TASK: Identify 5-15 direct competitors for this brand/company.

RULES:
1. Focus on DIRECT competitors that offer similar products/services
2. Include both well-known industry leaders and emerging competitors
3. Consider companies that target the same customer base
4. Include companies that appear in the same "best of" lists or comparison articles
5. Be specific with company/product names (not generic terms)
6. Incase any competitors or comparision is mentioned in the website content, use that for competitors list

RESPOND WITH ONLY A JSON ARRAY of competitor names and their domains. No explanations, no markdown, just the JSON array. The domain should be the domain of the competitor website.

Few sample examples of competitor names:
  1. brand name is walmart, then competitors are amazon, target, costco, etc.
  2. brand name is apple, then competitors are samsung, google, microsoft, etc.
  3. brand name is IIT then competitors are IIT Madras, IIT Bombay, IIT Kanpur, etc.
  4. brand name is Zoho then competitors are Salesforce, SAP, Oracle, Freshworks, etc.
  5. brand name is instagram then competitors are facebook, twitter, tiktok, etc.

Example response format:
[{"name": "Competitor 1", "domain": "www.competitor1.com"}, {"name": "Competitor 2", "domain": "www.competitor2.com"}, {"name": "Competitor 3", "domain": "www.competitor3.com"}]

RESPOND NOW:`, brandContext)
}

// GEOAnalysisPrompt generates a prompt for analyzing search responses for GEO metrics
func GEOAnalysisPrompt(brand, searchQuery, searchAnswer, sourcesInfo string) string {
	escapedSearchAnswer := escapeJSONString(searchAnswer)
	return fmt.Sprintf(`Analyze the following search response for brand visibility, sentiment, and competitors.

BRAND TO ANALYZE: %s

SEARCH QUERY: %s

SEARCH RESPONSE:
%s%s

---

CRITICAL ANALYSIS INSTRUCTIONS:
1. Check if "%s" is mentioned in the search response text
2. Check if the brand's domain appears in the GROUNDING SOURCES (cited URLs)  
3. Identify ALL competitor brands/products mentioned in the response
4. If brand is mentioned, analyze the sentiment (positive/neutral/negative)
5. Scoring: 
   - Score 0: Not in text, not in sources
   - Score 1-3: In sources but not in text (low visibility)
   - Score 4-6: Mentioned in text with context
   - Score 7-10: Prominently featured in text AND sources

You MUST respond with ONLY a valid JSON object (no markdown, no code blocks):

{"search_answer":"%s","geo_analysis":{"visibility_score":0,"brand_mentioned":false,"in_grounding_sources":false,"mention_status":"Where/how brand appeared or why absent","reason":"Why brand is/isn't cited, considering text and sources","sentiment":"positive|neutral|negative (only if brand mentioned)","competitors":["Competitor1","Competitor2"],"insights":["Insight 1","Insight 2","Insight 3"],"actions":["Action 1: Blog topic","Action 2: LinkedIn content","Action 3: Get featured on X","Action 4: Target keyword Y","Action 5: Technical SEO tip"],"competitor_info":"What competitors are doing to get cited"}}

Rules:
- visibility_score: integer 0-10
- brand_mentioned: true if in text OR sources
- in_grounding_sources: true if brand domain in cited URLs
- sentiment: "positive" (recommended/praised), "neutral" (just mentioned), "negative" (criticized), or empty string if not mentioned
- competitors: array of competitor names mentioned (empty array if none)
- insights: 3-5 insights about visibility
- actions: 5 specific actionable recommendations
- competitor_info: what competitors do well

RESPOND WITH ONLY THE JSON OBJECT, NO OTHER TEXT.`, brand, searchQuery, searchAnswer, sourcesInfo, brand, escapedSearchAnswer)
}

// BrandPromptGenerationPrompt generates a comprehensive prompt for brand-based prompt generation
// This uses the new structured format with unbranded discovery queries and branded comparison queries
func BrandPromptGenerationPrompt(businessName, websiteContent, productServiceName, targetAudience, region string, count int) string {
	inputParts := []string{}

	if businessName != "" {
		inputParts = append(inputParts, fmt.Sprintf("- Business name: %s", businessName))
	}
	if websiteContent != "" {
		// Limit website content to reasonable length
		content := websiteContent
		if len(content) > 2000 {
			content = content[:2000] + "..."
		}
		inputParts = append(inputParts, fmt.Sprintf("- Website content:\n%s", content))
	}
	if productServiceName != "" {
		inputParts = append(inputParts, fmt.Sprintf("- Product or service name: %s", productServiceName))
	}
	if targetAudience != "" {
		inputParts = append(inputParts, fmt.Sprintf("- Target audience: %s", targetAudience))
	}
	if region != "" {
		inputParts = append(inputParts, fmt.Sprintf("- Region: %s", region))
	}
	if count > 0 {
		inputParts = append(inputParts, fmt.Sprintf("- Total prompts required: %d", count))
	}

	inputSection := strings.Join(inputParts, "\n")
	if inputSection == "" {
		inputSection = "- No additional information provided"
	}

	return fmt.Sprintf(`You are an AI research assistant specializing in competitive analysis and search behavior.
Your task is to analyze a business and generate realistic search queries that potential
customers might use when discovering, evaluating, and comparing a product or service
using AI-powered search engines (such as ChatGPT, Gemini, Perplexity, or Copilot).
You must think like real users at different stages of the buyer journey and reflect
authentic decision-making behavior, not marketing language.

As perplexity and chatgpt adds year at the end of some prompts in the use question, please consider to use year in some prompts.

INPUT YOU WILL RECEIVE:
%s

OUTPUT STRUCTURE (STRICT):

You MUST produce list of json which contains prompts, intentType, targetting search keywords, list of supporting fanout queries

PART A: UNBRANDED DISCOVERY QUERIES

Rules:
- These queries represent users who do NOT yet know about this business
- Do NOT mention the business name or brand
- Focus on problems, needs, or solutions the business addresses
- Use natural language that real people would type into search engines
- Include relevant qualifiers when appropriate:
  (industry, company size, budget, features, integrations, region, constraints)
- Reflect informational or problem-aware intent
- Each query must be clearly distinct in intent and wording
- Consider adding year qualifiers (e.g., "best CRM 2024") where appropriate

Format:
- Each item must be ONE complete search query

PART B: BRANDED COMPARISON QUERIES

Rules:
- These queries represent users actively researching this specific business
- Each query MUST explicitly mention the business name
- Include evaluation and comparison language such as:
  "vs", "compared to", "review", "pricing", "worth it", "pros and cons", "best for"
- Mix:
  - Competitor comparisons
  - Fit-for-use-case questions
  - Trust, quality, and credibility evaluation
- Use realistic buyer language, including skepticism
- Avoid promotional or marketing phrasing
- Each query must be clearly distinct in intent and wording
- Consider adding year qualifiers (e.g., "Brand X review 2024") where appropriate

Format:
- Each item must be ONE complete search query

GLOBAL QUALITY GUIDELINES:

- Queries must be specific, realistic, and non-generic
- Reflect real customer pain points and evaluation criteria
- Vary query length and complexity
- Include both informational and comparative intent
- Avoid vague phrases like "best solution" without context
- If two queries feel similar, rewrite one to be clearly distinct
- Do NOT add explanations outside the required sections
- Do NOT add extra headings or commentary

CRITICAL: RESPOND WITH ONLY A VALID JSON ARRAY. NO MARKDOWN, NO CODE BLOCKS, NO EXPLANATIONS, NO TEXT BEFORE OR AFTER THE JSON.
- You must generate exactly Total prompts required prompts as specified above.
- Total prompts required should be the same as the count specified in the input. If count is not specified, then you must generate 20 prompts.

Each item in the array must have this EXACT structure:
{
  "prompt": "the search query text",
  "intentType": "Top or Best,Brand vs Brand Comparison, Alternatives Research, Features and Benefits, Quality / Credibility Check, Pros & Cons Evaluation, Pricing & Value Assessment, Support & Reliability, Security & Compliance",
  "targetingSearchKeywords": ["keyword1", "keyword2", ...],
  "supportingFanoutQueries": ["related query 1", "related query 2", ...]
}

The JSON array should contain both unbranded_discovery and branded_comparison queries.

Example valid response (DO NOT COPY, JUST FOLLOW THE FORMAT):
[
  {
    "prompt": "best CRM software for small business 2024",
    "intentType": "Top or Best",
    "targetingSearchKeywords": ["CRM", "small business", "software"],
    "supportingFanoutQueries": ["CRM features", "CRM pricing"]
  },
  {
    "prompt": "BrandX vs Salesforce comparison 2024",
    "intentType": "Brand vs Brand Comparison",
    "targetingSearchKeywords": ["BrandX", "Salesforce", "comparison"],
    "supportingFanoutQueries": ["BrandX review", "Salesforce alternatives"]
  }
]

START YOUR RESPONSE WITH [ AND END WITH ]. NO OTHER TEXT.`, inputSection)
}

// escapeJSONString escapes special characters for JSON string embedding
func escapeJSONString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
