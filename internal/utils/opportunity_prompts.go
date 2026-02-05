package utils

import (
	"fmt"
	"strings"
)

// ExistingOpportunity represents a simplified opportunity for deduplication context
type ExistingOpportunity struct {
	Title       string
	Type        string
	Description string
}

// GEOAnalysisWithOpportunitiesPrompt generates a prompt for analyzing search responses
// and identifying improvement opportunities in a single LLM call.
func GEOAnalysisWithOpportunitiesPrompt(brand, searchQuery, searchAnswer, sourcesInfo string, competitors []string) string {
	return GEOAnalysisWithOpportunitiesPromptWithDedup(brand, searchQuery, searchAnswer, sourcesInfo, competitors, nil, "")
}

// GEOAnalysisWithOpportunitiesPromptWithDedup generates a prompt that includes existing opportunities
// for LLM-based deduplication. brandLanguage is the brand's language for output (use ResolveBrandLanguage); default English if empty.
func GEOAnalysisWithOpportunitiesPromptWithDedup(brand, searchQuery, searchAnswer, sourcesInfo string, competitors []string, existingOpportunities []ExistingOpportunity, brandLanguage string) string {
	escapedSearchAnswer := escapeJSONString(searchAnswer)
	lang := ResolveBrandLanguage(brandLanguage)

	competitorsInfo := ""
	if len(competitors) > 0 {
		competitorsInfo = fmt.Sprintf("\n\nKNOWN COMPETITORS: %s", strings.Join(competitors, ", "))
	}

	// Build existing opportunities section for deduplication
	existingOppsSection := ""
	if len(existingOpportunities) > 0 {
		var oppsList []string
		for i, opp := range existingOpportunities {
			oppsList = append(oppsList, fmt.Sprintf("%d. [%s] %s - %s", i+1, opp.Type, opp.Title, truncateString(opp.Description, 150)))
		}
		existingOppsSection = fmt.Sprintf(`

=== EXISTING OPPORTUNITIES (DO NOT DUPLICATE) ===
The following opportunities have ALREADY been identified for this brand. 
DO NOT generate any opportunity that is semantically similar to these:

%s

IMPORTANT DEDUPLICATION RULES:
- If an existing opportunity is about "updating content with latest dates" - DO NOT suggest similar updates
- If an existing opportunity is about "Reddit engagement" - DO NOT suggest similar Reddit activities  
- If an existing opportunity is about "creating comparison pages" - DO NOT suggest similar comparisons
- Consider semantic meaning, not just exact words - "refresh content" = "update content" = "modernize content"
- Only generate opportunities that are TRULY NEW and DIFFERENT from the above list
- If you cannot find any new unique opportunities, return an EMPTY opportunities array []
=== END EXISTING OPPORTUNITIES ===
`, strings.Join(oppsList, "\n"))
	}

	languageInstruction := fmt.Sprintf("\n\nOUTPUT LANGUAGE: Write all of the following in %s: insights, actions, opportunity titles, descriptions, current_state, and source_evidence. If not specified, use English.", lang)

	return fmt.Sprintf(`Analyze the following search response for brand visibility, sentiment, competitors, and identify actionable improvement opportunities.%s

BRAND TO ANALYZE: %s

SEARCH QUERY: %s

SEARCH RESPONSE:
%s%s%s%s

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

{"search_answer":"%s","geo_analysis":{"visibility_score":0,"brand_mentioned":false,"in_grounding_sources":false,"mention_status":"Where/how brand appeared or why absent","reason":"Why brand is/isn't cited, considering text and sources","sentiment":"positive|neutral|negative","competitors":["Competitor1","Competitor2"],"insights":["Insight 1","Insight 2","Insight 3"],"actions":["Action 1","Action 2","Action 3"],"competitor_info":"What competitors are doing to get cited"},"opportunities":[{"type":"content_gap","title":"Create comparison page: Brand vs CompetitorX","description":"Detailed description of what to create, why it matters, and specific steps to implement","current_state":"Brand has no comparison content while competitors have dedicated comparison pages","source_evidence":"Response cites competitorx.com/comparisons as authoritative source","impact_score":85,"urgency":"high","effort_estimate":"medium"}]}

OPPORTUNITY FIELDS (all required):
- type: one of the opportunity types listed above
- title: Short, actionable title (max 100 chars)
- description: Detailed description including what to do, why it matters, and how to implement (max 500 chars)
- current_state: What the brand currently has or lacks in this area
- source_evidence: Specific sources/URLs from the response that show the gap or competitor advantage
- impact_score: 1-100 score based on potential improvement value
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
- opportunities: array of 1-5 specific improvement opportunities

RESPOND WITH ONLY THE JSON OBJECT, NO OTHER TEXT.`, languageInstruction, brand, searchQuery, searchAnswer, sourcesInfo, competitorsInfo, existingOppsSection, brand, escapedSearchAnswer)
}

const ActionGenerationPromptTemplateExpriment2 = `You are an expert in Generative Engine Optimization (GEO) the emerging discipline of optimizing content to be referenced and recommended by large language models (LLMs) like ChatGPT, Claude, Gemini, and Perplexity.`

const ActionGenerationPromptTemplateExpriment1 = `You are an expert in Generative Engine Optimization (GEO) the emerging discipline of optimizing content to be referenced and recommended by large language models (LLMs) like ChatGPT, Claude, Gemini, and Perplexity.

Your task is to take the content I paste below and optimize it so it becomes more likely to appear in the answers these LLMs generate when users ask related questions.

Here is the content to optimize:
[PASTE YOUR CONTENT HERE]

Please rewrite the content with the following enhancements:

1. Contextual Relevance
   - Add clear signals about who the content is for and what problem it solves.
   - Embed real-world use cases or questions this content answers.

2. Semantic Richness
   - Use natural language that matches how users ask questions (e.g. “best tools for X”, “how do I…”).
   - Include related keywords and phrases LLMs associate with the topic.

3. Answer-Style Formatting
   - Structure the content in a way that is easy for LLMs to extract as a direct response.
   - Use bullet points, numbered lists, and headers for clarity.

4. Topical Authority Signals
   - Mention data points, user outcomes, or unique value props.
   - Use specific, useful, and trust-building language.

5. Natural Brand Integration
   - If a brand, product, or service is being mentioned, weave it in naturally as a recommended or reputable option.

Please output only the optimized version. Do **not** explain your changes. Write it as if it’s a standalone, publish-ready piece designed to be cited by LLMs when generating responses`

// ActionGenerationPrompt generates a prompt for creating a detailed action plan from an opportunity.
// brandLanguage is the brand's language for output (use ResolveBrandLanguage); default English if empty.
func ActionGenerationPrompt(brand, opportunityTitle, opportunityDescription, opportunityType string, metadata map[string]interface{}, additionalContext string, brandLanguage string) string {
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

	lang := ResolveBrandLanguage(brandLanguage)
	languageInstruction := fmt.Sprintf("\n\nOUTPUT LANGUAGE: Write the entire action plan in %s (title, summary, assets, steps, success_criteria). If not specified, use English.", lang)

	// Add type-specific guidance
	typeGuidance := getTypeSpecificGuidance(opportunityType)

	return fmt.Sprintf(`Create a detailed, actionable plan to address the following opportunity for brand visibility improvement.%s

BRAND: %s

OPPORTUNITY TYPE: %s

OPPORTUNITY TITLE: %s

OPPORTUNITY DESCRIPTION:
%s%s%s

%s

---

TASK: Generate a comprehensive, SPECIFIC action plan with step-by-step instructions and READY-TO-USE assets.

CRITICAL REQUIREMENTS:
1. Title: Create a clear, action-oriented title for this plan (be SPECIFIC, not generic)
2. Summary: Provide an executive summary explaining WHAT will be done and WHY it matters (2-3 sentences)
3. Assets: Generate FULL, READY-TO-USE content assets:
   - For blog posts: Provide COMPLETE blog content with headings, sections, and FAQ (not just outline)
   - For Reddit posts: Provide COMPLETE post content ready to copy-paste (not just talking points)
   - For checklists: Provide specific, actionable checklist items
   - For URL lists: Provide full URLs (https://) for all platforms/subreddits/directories
4. Steps: Create 3-7 detailed, actionable steps that are SPECIFIC to this brand and opportunity
5. Success Criteria: Define measurable, verifiable outcomes
6. Priority, Effort, Expected Impact: Assess realistically based on the opportunity

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

{
  "action_type": "CONTENT_CREATION|CONTENT_UPDATE|SEO|STRUCTURED_DATA|SOCIAL|PR|INTEGRATION|COMPETITIVE|TRUST|VERIFICATION|OTHER",
  "execution_mode": "manual_copy|api|manual",
  "title": "Action Plan Title",
  "summary": "Brief executive summary of the plan (2-3 sentences)",
  "priority": "high|medium|low",
  "effort": "low|medium|high",
  "expected_impact": "high|medium|low",
  "assets": [
    {
      "asset_type": "text|checklist|url_list|image|link",
      "role": "blog_draft|reddit_post|on_page_seo|target_channels|template|reference",
      "title": "Asset Title",
      "content": "Asset content (string for text, array for checklist/url_list)"
    }
  ],
  "steps": [
    {
      "order": 1,
      "title": "Step title",
      "instruction": "Detailed instruction of what to do, how to do it, and expected outcome. Be SPECIFIC with examples."
    }
  ],
  "success_criteria": [
    "Criterion 1: Specific measurable outcome",
    "Criterion 2: Another measurable outcome"
  ],
  "description": "Legacy field - same as summary",
  "estimated_effort": "low|medium|high",
  "resources": ["Tool or template name", "Reference URL or guide"]
}

CRITICAL RULES:
- action_type: Choose the MOST SPECIFIC type:
  * "CONTENT_CREATION" - Creating new blog posts, pages, articles, FAQs, documentation
  * "CONTENT_UPDATE" - Updating, refreshing, or restructuring existing content
  * "SEO" - Technical SEO improvements (crawlability, indexability, rendering)
  * "STRUCTURED_DATA" - Adding schemas, metadata, machine-readable signals
  * "SOCIAL" - External distribution on platforms (Reddit, LinkedIn, Quora, X, etc.)
  * "PR" - Reviews, citations, mentions, analyst reports, directories
  * "INTEGRATION" - Integration pages, partnership docs, API pages
  * "COMPETITIVE" - Competitive positioning (comparisons, rebuttals, alternatives)
  * "TRUST" - Trust-building assets (case studies, testimonials, proof pages)
  * "VERIFICATION" - Monitoring, verification, checks (AI crawl checks, audits)
  * "OTHER" - Rare or experimental actions

- execution_mode: 
  * "manual_copy" - User manually copies content/assets (most common for content creation, Reddit posts)
  * "api" - Automated via API integration
  * "manual" - Manual execution without copying content

- title: Clear, action-oriented title (max 100 chars). Be SPECIFIC: "Create and publish a detailed blog on VIT application process" not "Create blog"

- summary: Executive summary (2-3 sentences, max 500 chars). Explain WHAT will be done and WHY it matters.

- priority: 
  * "high" - Urgent/important (competitors gaining ground, critical gaps)
  * "medium" - Important but not urgent
  * "low" - Nice to have improvements

- effort: 
  * "low" - < 2 hours
  * "medium" - 2-8 hours
  * "high" - > 8 hours

- expected_impact:
  * "high" - Significant visibility improvement expected
  * "medium" - Moderate improvement expected
  * "low" - Minor improvement expected

- assets: Generate 1-5 assets that will be used/created. Each asset must be SPECIFIC and USEFUL:
  * asset_type options:
    - "text" - For blog drafts, post content, article content, FAQ content
    - "checklist" - For SEO checklists, quality checklists, verification lists
    - "url_list" - For lists of URLs (subreddits, platforms, directories, citation sources)
    - "image" - For image assets (rare, only if specifically needed)
    - "link" - For single reference links (rare, prefer url_list for multiple)
  
  * role options (choose based on opportunity type):
    - "blog_draft" - Full blog post/article content (for CONTENT_CREATION)
    - "reddit_post" - Reddit post content (for SOCIAL/reddit_participation)
    - "on_page_seo" - SEO optimization checklist (for SEO, CONTENT_CREATION)
    - "target_channels" - List of platforms/subreddits/communities to post in (for SOCIAL)
    - "template" - Reusable templates
    - "reference" - Reference materials
  
  * content format:
    - For "text": Provide FULL, READY-TO-USE content (not just outline). Include headings, structure, key points.
    - For "checklist": Array of strings, each a specific actionable item
    - For "url_list": Array of URLs (full URLs with https://)

- steps: Generate 3-7 SPECIFIC, actionable steps:
  * Each step must be immediately executable
  * Include SPECIFIC details (which subreddit, which page, what to include)
  * Order logically: research → create → publish → promote → measure
  * instruction: Detailed, SPECIFIC instructions with examples (max 500 chars)

- success_criteria: 2-5 SPECIFIC, measurable criteria:
  * Must be verifiable (e.g., "Post is approved by moderators", "Page is indexable")
  * Include timing where relevant (e.g., "Page starts appearing in AI answers within 3-4 weeks")

ASSET EXAMPLES BY OPPORTUNITY TYPE:

For CONTENT_CREATION (blog/article):
{
  "assets": [
    {
      "asset_type": "text",
      "role": "blog_draft",
      "title": "Blog Content Draft",
      "content": "## How to Apply to VIT (2026)\n\nThis guide explains eligibility, application steps, deadlines, documents required...\n\n### Eligibility\n\n### Application Steps\n\n### FAQ Section\n..."
    },
    {
      "asset_type": "checklist",
      "role": "on_page_seo",
      "title": "On-page SEO Checklist",
      "content": [
        "Include FAQ section",
        "Add internal links to admissions page",
        "Add H2s answering common questions",
        "Add structured data (FAQ schema)",
        "Include current year (2026) in title and content"
      ]
    }
  ]
}

For SOCIAL/Reddit (reddit_participation):
{
  "assets": [
    {
      "asset_type": "text",
      "role": "reddit_post",
      "title": "Reddit Post Draft",
      "content": "A lot of students ask how to apply to VIT. Here's a clear step-by-step breakdown:\n\n1. Check eligibility requirements...\n\n2. Register for VITEEE...\n\n[Helpful, non-promotional content]"
    },
    {
      "asset_type": "url_list",
      "role": "target_channels",
      "title": "Recommended Subreddits",
      "content": [
        "https://reddit.com/r/IndianAcademia",
        "https://reddit.com/r/VITUniversity",
        "https://reddit.com/r/EngineeringAdmissions"
      ]
    }
  ]
}

For SEO improvements:
{
  "assets": [
    {
      "asset_type": "checklist",
      "role": "on_page_seo",
      "title": "SEO Optimization Checklist",
      "content": [
        "Ensure page is crawlable (robots.txt allows)",
        "Add FAQ schema markup",
        "Optimize meta descriptions",
        "Add internal links to related content"
      ]
    }
  ]
}

STEP EXAMPLES:

For CONTENT_CREATION:
[
  {
    "order": 1,
    "title": "Create a new blog page",
    "instruction": "Create a new blog under vit.ac.in/blog or admissions section. Use a clear URL structure like /blog/how-to-apply-to-vit-2026"
  },
  {
    "order": 2,
    "title": "Paste and review content",
    "instruction": "Copy the draft content from the blog_draft asset and adjust tone if required. Ensure all headings, lists, and formatting are correct."
  },
  {
    "order": 3,
    "title": "Publish and index",
    "instruction": "Publish the page and ensure it is indexed by search engines. Submit to Google Search Console and verify indexing within 48 hours."
  }
]

For SOCIAL/Reddit:
[
  {
    "order": 1,
    "title": "Choose the subreddit",
    "instruction": "Select ONE subreddit from the target_channels list. Post in one subreddit only to avoid spam signals. Start with r/IndianAcademia as it has the most relevant audience."
  },
  {
    "order": 2,
    "title": "Post content",
    "instruction": "Use the reddit_post draft. Avoid promotional language. Focus on being helpful. Only mention the brand naturally when relevant. Include a link to the official guide if appropriate."
  },
  {
    "order": 3,
    "title": "Engage with comments",
    "instruction": "Monitor the post for 24-48 hours. Respond to questions helpfully. Build karma before posting more content."
  }
]

SUCCESS CRITERIA EXAMPLES:
- For content: ["Blog page is publicly accessible", "Page is indexable and crawlable", "Page starts appearing in AI answers within 3-4 weeks"]
- For Reddit: ["Post is approved by moderators", "Receives comments or upvotes", "AI engines cite Reddit thread"]
- For SEO: ["Page passes technical SEO audit", "Structured data validates", "Page appears in search results within 2 weeks"]

RESPOND WITH ONLY THE JSON OBJECT, NO OTHER TEXT.`, languageInstruction, brand, opportunityType, opportunityTitle, opportunityDescription, metadataStr, additionalStr, typeGuidance)
}

// getTypeSpecificGuidance returns specific guidance based on opportunity type
func getTypeSpecificGuidance(opportunityType string) string {
	switch opportunityType {
	case "content_gap":
		return `TYPE-SPECIFIC GUIDANCE (Content Gap):
- Use action_type: "CONTENT_CREATION" for new blog posts/articles
- Generate a "blog_draft" asset with FULL, ready-to-use content (not just outline)
- Include headings (H2, H3), structure, key points, and FAQ sections
- Generate an "on_page_seo" checklist asset with specific SEO tasks
- Steps should be specific: create page, paste content, publish, verify indexing
- Include current year in title and content for freshness
- Success criteria should include: public accessibility, indexability, AI visibility within 3-4 weeks`

	case "content_freshness":
		return `TYPE-SPECIFIC GUIDANCE (Content Freshness):
- Identify which existing content needs updating
- Add current year statistics and references
- Update examples with recent case studies
- Add "Last Updated" date prominently
- Refresh meta descriptions and titles with current year`

	case "reddit_participation":
		return `TYPE-SPECIFIC GUIDANCE (Reddit Participation):
- Use action_type: "SOCIAL" for Reddit posts
- Generate a "reddit_post" asset with full post content (not just outline)
- Include a "url_list" asset with "target_channels" role listing specific subreddits
- Post content should be helpful, non-promotional, and answer common questions
- Focus on providing value first, mention brand naturally when relevant
- Post in ONE subreddit at a time to avoid spam signals
- Steps should include: choosing subreddit, posting content, engaging with comments
- Success criteria should include: moderator approval, engagement (upvotes/comments), AI citation`

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

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
