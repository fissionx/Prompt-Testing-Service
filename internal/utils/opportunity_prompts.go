package utils

import (
	"fmt"
	"strings"
)

// GEOAnalysisWithOpportunitiesPrompt generates a prompt for analyzing search responses
// and identifying improvement opportunities in a single LLM call
func GEOAnalysisWithOpportunitiesPrompt(brand, searchQuery, searchAnswer, sourcesInfo string, competitors []string) string {
	escapedSearchAnswer := escapeJSONString(searchAnswer)

	competitorsInfo := ""
	if len(competitors) > 0 {
		competitorsInfo = fmt.Sprintf("\n\nKNOWN COMPETITORS: %s", strings.Join(competitors, ", "))
	}

	return fmt.Sprintf(`Analyze the following search response for brand visibility, sentiment, competitors, and identify actionable improvement opportunities.

BRAND TO ANALYZE: %s

SEARCH QUERY: %s

SEARCH RESPONSE:
%s%s%s

---

CRITICAL ANALYSIS INSTRUCTIONS:
1. Check if "%s" is mentioned in the search response text
2. Check if the brand's domain appears in the GROUNDING SOURCES (cited URLs)  
3. Identify ALL competitor brands/products mentioned in the response
4. If brand is mentioned, analyze the sentiment (positive/neutral/negative)
5. Identify specific OPPORTUNITIES for the brand to improve visibility

OPPORTUNITY IDENTIFICATION - KEY FACTORS TO CONSIDER:

1. CONTENT FRESHNESS (High Priority):
   - LLMs favor recently updated content
   - Identify if brand's existing content might be outdated
   - Suggest updating older blog posts with fresh information, recent stats, current year references
   - Recommend adding "Last Updated: [Date]" to key pages

2. CONTENT TYPES TO CREATE/IMPROVE:
   - Case Studies: Real customer success stories with metrics
   - LinkedIn Posts: Thought leadership, industry insights, company updates
   - Reddit Threads: Authentic participation in relevant subreddits (r/[industry], r/[topic])
   - Industry Articles: Guest posts, contributed content on authoritative sites
   - Comparison Pages: "[Brand] vs [Competitor]" content that's balanced and informative
   - Integration Guides: How to use with popular tools/platforms
   - Alternatives Research: Compare the brand with competitors and identify the best options
   - Features and Benefits: List the features and benefits of the brand
   - Quality / Credibility Check: Check the quality and credibility of the brand
   - Pros & Cons Evaluation: List the pros and cons of the brand
   - Pricing & Value Assessment: Check the pricing and value of the brand
   - Support & Reliability: Check the support and reliability of the brand
   - Security & Compliance: Check the security and compliance of the brand

3. BRAND-PROBLEM ASSOCIATION:
   - Does the response clearly associate the brand with specific problems it solves?
   - Suggest content that directly ties brand to problem categories
   - Recommend FAQ pages, "What we solve" content
   - Identify missing use case documentation

4. ANSWER-OPTIMIZED CONTENT STRUCTURE:
   - Content should directly answer questions (LLMs extract this better)
   - Recommend structures: Concise summary at top, clear definitions, "When to use / When not to use", FAQ sections
   - Suggest adding structured data (FAQ schema, HowTo schema)
   - Content should be scannable with clear headings

OPPORTUNITY TYPES:
- content_gap: Missing blog posts, articles, landing pages on topics mentioned
- content_freshness: Existing content that needs updating with current information
- reddit_participation: Reddit threads or subreddits to engage with authentically
- review_sites: Missing presence on review platforms (G2, Capterra, TrustRadius, etc.)
- case_study: Need for customer success stories or case studies
- linkedin_presence: Professional content, thought leadership posts
- comparison_content: Head-to-head comparison pages with competitors
- citation_source: Authoritative sources (industry reports, research sites) to get mentioned in
- faq_content: FAQ pages or structured Q&A content
- competitor_response: Counter competitor advantages mentioned in the response
- integration_content: Documentation about integrations or partnerships
- other: Any other improvement opportunity

SCORING GUIDE:
- visibility_score 0: Not in text, not in sources
- visibility_score 1-3: In sources but not in text (low visibility)
- visibility_score 4-6: Mentioned in text with context
- visibility_score 7-10: Prominently featured in text AND sources

- impact_score 90-100: Critical (brand not mentioned but competitors are, or missing from key sources cited)
- impact_score 70-89: High priority (content gap on exact topic being discussed)
- impact_score 50-69: Medium priority (related content improvements)
- impact_score 30-49: Lower priority (nice-to-have improvements)
- impact_score 1-29: Minor optimization opportunities

You MUST respond with ONLY a valid JSON object (no markdown, no code blocks):

{"search_answer":"%s","geo_analysis":{"visibility_score":0,"brand_mentioned":false,"in_grounding_sources":false,"mention_status":"Where/how brand appeared or why absent","reason":"Why brand is/isn't cited, considering text and sources","sentiment":"positive|neutral|negative","competitors":["Competitor1","Competitor2"],"insights":["Insight 1","Insight 2","Insight 3"],"actions":["Action 1","Action 2","Action 3"],"competitor_info":"What competitors are doing to get cited"},"opportunities":[{"type":"content_gap","title":"Create comparison page: Brand vs CompetitorX","description":"Detailed description of what to create and how","gap_analysis":"WHY this gap exists - evidence from the AI response showing brand is missing","current_state":"What the brand currently has or lacks in this area","competitor_context":"CompetitorX is mentioned because they have detailed comparison pages and documentation","source_evidence":"The response cites competitorx.com/comparisons which directly addresses this query","recommended_action":"Create a detailed comparison page at brand.com/vs/competitorx covering features, pricing, and use cases","expected_outcome":"Brand will be cited in future responses when users ask about comparisons","target_platform":"website","target_audience":"Technical decision makers evaluating solutions","keywords":["brand vs competitor","comparison","alternatives"],"related_urls":["https://competitorx.com/comparisons"],"impact_score":85,"impact_reasoning":"High impact because competitors are directly cited for comparisons while brand is absent","urgency":"high","effort_estimate":"medium"}]}

OPPORTUNITY FIELD REQUIREMENTS:

Each opportunity MUST include these fields to give users clear understanding:

WHY (Evidence & Reasoning):
- gap_analysis: Explain WHY this gap exists based on the AI response. What's missing? (required)
- current_state: What does the brand currently have or lack in this area? (required)
- competitor_context: What are competitors doing better that gets them cited? (if applicable)
- source_evidence: Specific sources/URLs from the response that show the gap (if available)

WHAT (Specific Recommendation):
- title: Short, actionable title (max 100 chars, e.g., "Create comparison page: Brand vs CompetitorX")
- description: Detailed description of the opportunity (max 300 chars)
- recommended_action: SPECIFIC action statement - what exactly to do (required, max 500 chars)
- expected_outcome: What success looks like after implementing this (required)

WHERE (Target & Context):
- target_platform: Where to take action (website, reddit, linkedin, g2, capterra, medium, youtube, etc.)
- target_audience: Who is this content/action for?
- keywords: Array of relevant keywords to include in the content
- related_urls: Array of competitor URLs or source URLs that demonstrate the gap

PRIORITY:
- impact_score: 1-100 score
- impact_reasoning: WHY this score - explain the reasoning (required)
- urgency: "high" (competitors gaining ground), "medium" (important but not urgent), "low" (nice to have)
- effort_estimate: "low" (<2 hours), "medium" (2-8 hours), "high" (>8 hours)

Rules:
- visibility_score: integer 0-10
- brand_mentioned: true if in text OR sources
- in_grounding_sources: true if brand domain in cited URLs
- sentiment: "positive", "neutral", "negative", or empty string if not mentioned
- competitors: array of competitor names mentioned
- insights: 3-5 insights about visibility
- actions: 3-5 specific actionable recommendations (brief)
- opportunities: array of 1-5 specific improvement opportunities with ALL fields above

RESPOND WITH ONLY THE JSON OBJECT, NO OTHER TEXT.`, brand, searchQuery, searchAnswer, sourcesInfo, competitorsInfo, brand, escapedSearchAnswer)
}

// ActionGenerationPrompt generates a prompt for creating a detailed action plan from an opportunity
func ActionGenerationPrompt(brand, opportunityTitle, opportunityDescription, opportunityType string, metadata map[string]interface{}, additionalContext string) string {
	metadataStr := ""
	if len(metadata) > 0 {
		var parts []string
		for k, v := range metadata {
			parts = append(parts, fmt.Sprintf("- %s: %v", k, v))
		}
		metadataStr = fmt.Sprintf("\n\nRELATED CONTEXT:\n%s", strings.Join(parts, "\n"))
	}

	additionalStr := ""
	if additionalContext != "" {
		additionalStr = fmt.Sprintf("\n\nADDITIONAL CONTEXT FROM USER:\n%s", additionalContext)
	}

	// Add type-specific guidance
	typeGuidance := getTypeSpecificGuidance(opportunityType)

	return fmt.Sprintf(`Create a detailed, actionable plan to address the following opportunity for brand visibility improvement.

BRAND: %s

OPPORTUNITY TYPE: %s

OPPORTUNITY TITLE: %s

OPPORTUNITY DESCRIPTION:
%s%s%s

%s

---

TASK: Generate a comprehensive, SPECIFIC action plan with step-by-step instructions.

REQUIREMENTS:
1. Title: Create a clear, action-oriented title for this plan
2. Description: Provide an executive summary of the plan (2-3 sentences)
3. Steps: Create 4-7 detailed, actionable steps that are SPECIFIC to this brand and opportunity
4. Estimated Effort: Rate as "low" (< 2 hours), "medium" (2-8 hours), or "high" (> 8 hours)
5. Resources: List helpful tools, templates, or reference materials

STEP GUIDELINES:
- Each step should be SPECIFIC and immediately actionable (not generic advice)
- Include specific examples, templates, or formats to use
- For content creation: Include suggested headlines, structure, key points to cover
- For platform engagement: Include specific subreddits, groups, or communities
- Order steps logically (research → create → publish → promote → measure)
- Make steps achievable by a marketing professional

CONTENT STRUCTURE BEST PRACTICES (for content opportunities):
- Start with a concise summary answering the main question
- Include clear definitions of key terms
- Add "When to use" and "When not to use" sections
- Include an FAQ section with 3-5 common questions
- Use structured data markup (FAQ schema, HowTo schema) where applicable
- End with a clear call-to-action

You MUST respond with ONLY a valid JSON object (no markdown, no code blocks):

{"title":"Action Plan Title","description":"Brief executive summary of the plan","steps":[{"order":1,"title":"Step title","description":"Detailed description of what to do, how to do it, and expected outcome. Be SPECIFIC with examples."},{"order":2,"title":"Next step title","description":"Next step details with specific examples"}],"estimated_effort":"low|medium|high","resources":["Tool or template name","Reference URL or guide"]}

Rules:
- title: clear action-oriented title (max 100 chars)
- description: executive summary (max 300 chars)
- steps: array of 4-7 steps, each with:
  - order: integer starting from 1
  - title: short step title (max 80 chars)
  - description: detailed, SPECIFIC instructions with examples (max 500 chars)
- estimated_effort: "low", "medium", or "high"
- resources: array of helpful tools, templates, or references (2-5 items)

RESPOND WITH ONLY THE JSON OBJECT, NO OTHER TEXT.`, brand, opportunityType, opportunityTitle, opportunityDescription, metadataStr, additionalStr, typeGuidance)
}

// getTypeSpecificGuidance returns specific guidance based on opportunity type
func getTypeSpecificGuidance(opportunityType string) string {
	switch opportunityType {
	case "content_gap":
		return `TYPE-SPECIFIC GUIDANCE (Content Gap):
- Focus on creating content that directly answers the search query
- Structure content with clear headings, bullet points, and summaries
- Include relevant keywords naturally throughout
- Add internal links to related content
- Consider adding comparison tables or infographics`

	case "content_freshness":
		return `TYPE-SPECIFIC GUIDANCE (Content Freshness):
- Identify which existing content needs updating
- Add current year statistics and references
- Update examples with recent case studies
- Add "Last Updated" date prominently
- Refresh meta descriptions and titles with current year`

	case "reddit_participation":
		return `TYPE-SPECIFIC GUIDANCE (Reddit Participation):
- Identify specific subreddits where the target audience is active
- Focus on providing genuine value, not promotion
- Build karma before posting about the brand
- Answer questions helpfully, only mention brand when relevant
- Consider doing an AMA if appropriate`

	case "case_study":
		return `TYPE-SPECIFIC GUIDANCE (Case Study):
- Structure: Challenge → Solution → Results
- Include specific metrics and percentages
- Get customer quotes and testimonials
- Add visuals: charts, before/after comparisons
- Make it skimmable with clear headings`

	case "linkedin_presence":
		return `TYPE-SPECIFIC GUIDANCE (LinkedIn):
- Post thought leadership content 2-3x per week
- Engage authentically with industry discussions
- Share insights from customer interactions
- Use document posts and carousels for engagement
- Tag relevant people and companies`

	case "comparison_content":
		return `TYPE-SPECIFIC GUIDANCE (Comparison Content):
- Be fair and balanced (acknowledge competitor strengths)
- Use feature comparison tables
- Include pricing information if public
- Add "Best for" recommendations for each option
- Update regularly as products change`

	case "faq_content":
		return `TYPE-SPECIFIC GUIDANCE (FAQ Content):
- Group questions by category
- Start answers with a direct response
- Keep answers concise (2-4 sentences ideal)
- Use FAQ schema markup for rich snippets
- Link to detailed articles for complex topics`

	case "review_sites":
		return `TYPE-SPECIFIC GUIDANCE (Review Sites):
- Claim and complete profiles on G2, Capterra, TrustRadius
- Respond to all reviews (positive and negative)
- Encourage satisfied customers to leave reviews
- Keep product information current
- Add screenshots and demos`

	case "citation_source":
		return `TYPE-SPECIFIC GUIDANCE (Citation Sources):
- Identify authoritative sites in the industry
- Create data or research worth citing
- Reach out to journalists covering the space
- Contribute guest posts to industry publications
- Get listed in relevant directories and roundups`

	case "integration_content":
		return `TYPE-SPECIFIC GUIDANCE (Integration Content):
- Create dedicated landing pages for each integration
- Include step-by-step setup guides
- Add use case examples for the integration
- Get listed in partner directories
- Create co-marketing content with integration partners`

	default:
		return `TYPE-SPECIFIC GUIDANCE:
- Focus on creating content that directly addresses the identified gap
- Ensure content is structured for easy consumption by both humans and AI
- Include relevant keywords and internal links
- Measure impact through visibility tracking`
	}
}
