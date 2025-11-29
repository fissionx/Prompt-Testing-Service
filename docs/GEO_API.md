# GEO (Generative Engine Optimization) API Documentation

Complete API guide for using the GEO platform to analyze and improve your brand's visibility in AI-generated responses.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Workflow](#workflow)
3. [API Endpoints](#api-endpoints)
4. [Examples](#examples)

---

## Quick Start

### Prerequisites

1. Start the API server:
```bash
gego api --port 8080
```

2. Add Google Gemini LLM (required for web search):
```bash
curl -X POST http://localhost:8080/api/v1/llms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Gemini Flash",
    "provider": "google",
    "model": "models/gemini-1.5-flash",
    "api_key": "YOUR_GOOGLE_API_KEY",
    "enabled": true
  }'
```

---

## Workflow

### Complete GEO Campaign Flow

```
1. Generate Prompts → 2. Bulk Execute → 3. Get Insights
```

### 1. Generate AI-Powered Prompts

Generate natural search queries tailored to your brand and industry. The system:
- **Scrapes your website** for real content understanding
- Generates 10-50 unique questions using AI
- Reuses existing prompts from the same category/domain
- Stores all prompts for future reuse

**Endpoint:** `POST /api/v1/geo/prompts/generate`

#### Option A: With Website Scraping (Recommended for Best Results)

```bash
curl -X POST http://localhost:8080/api/v1/geo/prompts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "FissionX.ai",
    "website": "https://fissionx.ai",
    "count": 30
  }'
```

**Benefits:**
- 🎯 System scrapes your website to understand what you actually do
- 🤖 AI uses real content to generate hyper-realistic prompts
- 📊 Auto-derives domain/category from your website
- ⚡ No need to manually write descriptions

#### Option B: Manual Context (Traditional Way)

```bash
curl -X POST http://localhost:8080/api/v1/geo/prompts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "FissionX.ai",
    "category": "AI SEO Tools",
    "domain": "technology",
    "description": "AI-powered SEO content optimization platform",
    "count": 30
  }'
```

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "FissionX.ai",
    "category": "AI SEO Tools",
    "domain": "technology",
    "prompts": [
      {
        "id": "uuid-1",
        "template": "What are the best AI SEO tools for content optimization?",
        "category": "AI SEO Tools",
        "reused": false
      },
      {
        "id": "uuid-2",
        "template": "How do AI SEO platforms compare for technical optimization?",
        "category": "AI SEO Tools",
        "reused": true
      }
    ],
    "existing_prompts": 10,
    "generated_prompts": 20
  }
}
```

### 2. Bulk Execute Campaign

Run all generated prompts across multiple LLMs to analyze brand visibility.

**Endpoint:** `POST /api/v1/geo/execute/bulk`

```bash
curl -X POST http://localhost:8080/api/v1/geo/execute/bulk \
  -H "Content-Type: application/json" \
  -d '{
    "campaign_name": "FissionX Q4 2025 Campaign",
    "brand": "FissionX.ai",
    "prompt_ids": ["uuid-1", "uuid-2", "uuid-3", ...],
    "llm_ids": ["llm-uuid-1", "llm-uuid-2"],
    "temperature": 0.7
  }'
```

**Response:**
```json
{
  "success": true,
  "data": {
    "campaign_id": "campaign-uuid",
    "campaign_name": "FissionX Q4 2025 Campaign",
    "brand": "FissionX.ai",
    "total_runs": 60,
    "status": "running",
    "started_at": "2025-11-28T10:00:00Z",
    "message": "Campaign started successfully. Execution running in background."
  }
}
```

### 3. Get GEO Insights

Analyze campaign results with comprehensive metrics.

**Endpoint:** `POST /api/v1/geo/insights`

```bash
curl -X POST http://localhost:8080/api/v1/geo/insights \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "FissionX.ai",
    "start_time": "2025-11-01T00:00:00Z",
    "end_time": "2025-11-30T23:59:59Z"
  }'
```

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "FissionX.ai",
    "average_visibility": 3.2,
    "mention_rate": 15.5,
    "grounding_rate": 8.3,
    "sentiment_breakdown": {
      "positive": 12,
      "neutral": 5,
      "negative": 1
    },
    "top_competitors": [
      {
        "name": "Surfer SEO",
        "mention_count": 45,
        "visibility_avg": 7.8
      },
      {
        "name": "Clearscope",
        "mention_count": 38,
        "visibility_avg": 7.2
      }
    ],
    "performance_by_llm": [
      {
        "llm_name": "Gemini Flash",
        "llm_provider": "google",
        "visibility": 4.1,
        "mention_rate": 22.3,
        "response_count": 30
      }
    ],
    "performance_by_category": [
      {
        "category": "AI SEO Tools",
        "visibility": 3.8,
        "mention_rate": 18.5,
        "response_count": 25
      }
    ],
    "total_responses": 60
  }
}
```

---

## API Endpoints

### Single Execution (Real-time)

For testing or single prompt execution with immediate GEO analysis.

**Endpoint:** `POST /api/v1/execute`

```bash
curl -X POST http://localhost:8080/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "What are the best AI tools for content optimization?",
    "llm_id": "YOUR_LLM_ID",
    "brand": "FissionX.ai",
    "temperature": 0.7
  }'
```

**Response includes:**
- Search answer from LLM
- GEO analysis with visibility score
- Brand mention detection
- Grounding source analysis
- Sentiment analysis
- Competitor mentions
- Actionable insights
- Recommended actions

---

## Examples

### Complete Campaign Example

```bash
# 1. Generate 50 prompts for AI SEO tool category
PROMPTS_RESPONSE=$(curl -X POST http://localhost:8080/api/v1/geo/prompts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "FissionX.ai",
    "category": "AI SEO Tools",
    "domain": "technology",
    "count": 50
  }')

# Extract prompt IDs (using jq)
PROMPT_IDS=$(echo $PROMPTS_RESPONSE | jq -r '.data.prompts[].id' | jq -R -s -c 'split("\n")[:-1]')

# 2. Get LLM IDs
LLMS=$(curl http://localhost:8080/api/v1/llms?enabled=true)
LLM_IDS=$(echo $LLMS | jq -r '.data[].id' | jq -R -s -c 'split("\n")[:-1]')

# 3. Execute bulk campaign
curl -X POST http://localhost:8080/api/v1/geo/execute/bulk \
  -H "Content-Type: application/json" \
  -d "{
    \"campaign_name\": \"FissionX Full Analysis\",
    \"brand\": \"FissionX.ai\",
    \"prompt_ids\": $PROMPT_IDS,
    \"llm_ids\": $LLM_IDS,
    \"temperature\": 0.7
  }"

# 4. Wait for campaign to complete (check logs or poll status)
# Then get insights:

curl -X POST http://localhost:8080/api/v1/geo/insights \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "FissionX.ai"
  }'
```

---

## Key Metrics Explained

### Visibility Score (0-10)
- **0**: Brand not mentioned anywhere
- **1-3**: Brand in grounding sources but not in text (low visibility)
- **4-6**: Brand mentioned in text with context
- **7-10**: Brand prominently featured in text and sources

### Mention Rate
Percentage of responses where your brand is mentioned in text or grounding sources.

### Grounding Rate  
Percentage of responses where your brand's website appears in cited sources.

### Sentiment
When your brand is mentioned, the sentiment can be:
- **Positive**: Recommended, praised, or highlighted positively
- **Neutral**: Just mentioned factually
- **Negative**: Criticized or mentioned negatively

---

## Best Practices

1. **Start with Prompt Generation**: Let AI generate natural queries users actually ask
2. **Run Across Multiple LLMs**: Different LLMs have different visibility patterns
3. **Monitor Trends**: Run campaigns monthly to track visibility improvements
4. **Act on Insights**: Follow the recommended actions from GEO analysis
5. **Track Competitors**: Monitor who appears more often and why
6. **Improve Grounding**: Focus on getting cited in sources, not just text mentions

---

## Rate Limiting & Performance

- Bulk execution runs asynchronously with 3 concurrent requests max
- Large campaigns (100+ prompts × 5 LLMs = 500 executions) may take 10-30 minutes
- Monitor server logs for progress updates

---

## Troubleshooting

**Q: Prompts aren't being generated**
- Ensure Google Gemini LLM is configured
- Check API key is valid
- Review server logs for errors

**Q: Low visibility scores**
- This is normal for new brands - use the recommended actions
- Focus on content creation and authoritative backlinks
- Track improvements over time

**Q: Campaign stuck at "running"**
- Check server logs for errors
- Verify all LLM configurations are valid
- Ensure sufficient API credits/quotas

---

For more information, see the main [README.md](../README.md) and [EXAMPLES.md](./EXAMPLES.md).


---

## Library Management API

### List All Prompt Libraries

Get all prompt libraries organized by brand, domain, and category.

**Endpoint:** `GET /api/v1/geo/libraries`

```bash
curl -X GET http://localhost:8080/api/v1/geo/libraries
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "lib-1",
      "brand": "FissionX.ai",
      "domain": "technology",
      "category": "AI SEO Tools",
      "prompt_ids": ["prompt-1", "prompt-2", "prompt-3"],
      "usage_count": 5,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-02T00:00:00Z"
    }
  ]
}
```

### List All Brand Profiles

Get all brand profiles with their derived metadata.

**Endpoint:** `GET /api/v1/geo/profiles`

```bash
curl -X GET http://localhost:8080/api/v1/geo/profiles
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "profile-1",
      "brand_name": "FissionX.ai",
      "domain": "technology",
      "category": "AI SEO Tools",
      "website": "https://fissionx.ai",
      "description": "AI-powered SEO platform",
      "competitors": ["Competitor A", "Competitor B"],
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### Get Specific Brand Profile

Get detailed profile for a specific brand.

**Endpoint:** `GET /api/v1/geo/profiles/:brand`

```bash
curl -X GET http://localhost:8080/api/v1/geo/profiles/FissionX.ai
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "profile-1",
    "brand_name": "FissionX.ai",
    "domain": "technology",
    "category": "AI SEO Tools",
    "website": "https://fissionx.ai",
    "description": "AI-powered SEO platform",
    "competitors": ["Competitor A", "Competitor B"],
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

---

## Prompt Library System

### Important: Generic Prompts Only

**Critical Requirement:** All prompts in the library must be **GENERIC** and applicable to any brand in the category.

❌ **Bad (Brand-Specific):**
- "What are the scholarship opportunities at Thiagarajar College of Engineering?"
- "How is the placement record at MIT?"
- "What courses does Stanford offer?"

✅ **Good (Generic):**
- "What are the scholarship opportunities for engineering students?"
- "How to evaluate placement records when choosing an engineering college?"
- "What are the most popular courses in engineering colleges?"

**Why?** Generic prompts can be safely reused across all brands in the same category.

### How It Works

The prompt library system automatically organizes and reuses prompts to optimize costs and maintain consistency:

1. **First brand in a category**: When you generate prompts for a new domain/category combination, the system:
   - Derives domain and category using AI (if not provided)
   - Creates a brand profile
   - Generates new prompts using AI
   - Stores prompts in a library indexed by **domain + category** (NOT brand-specific)

2. **Similar brands (Same domain/category)**: When you request prompts for another brand with the same domain/category:
   - System checks if a library exists for that domain/category combination
   - Returns existing prompts instantly (no AI call needed) ⚡
   - Increments library usage count
   - **Example**: All engineering colleges share the same prompts!

3. **Smart categorization**: 
   - Domain: Industry (e.g., "education", "technology", "healthcare")
   - Category: Specific niche (e.g., "Engineering College", "AI SEO Tools", "CRM Software")
   - **All brands with same domain/category share the same prompt library**

### Benefits

- **Cost Savings**: Reuse prompts instead of regenerating
- **Consistency**: Same prompts across time for fair comparison
- **Speed**: Instant prompt retrieval for existing categories
- **Organization**: Clear structure by domain and category

### Example Workflow

#### Example 1: Engineering Colleges (Auto-Categorization)

```bash
# First college - VIT
curl -X POST http://localhost:8080/api/v1/geo/prompts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "VIT",
    "website": "https://vit.ac.in/",
    "count": 10
  }'
# System derives: domain="education", category="engineering college"
# Response: generated_prompts: 10, existing_prompts: 0
# Logs: "📚 Creating new prompt library: domain=education, category=engineering college"

# Second college - TCE (Same domain/category!)
curl -X POST http://localhost:8080/api/v1/geo/prompts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "Thiagarajar College of Engineering",
    "website": "https://tce.edu/",
    "count": 10
  }'
# System derives: domain="education", category="engineering college" (SAME!)
# Response: generated_prompts: 0, existing_prompts: 10 ⚡ INSTANT REUSE!
# Logs: "♻️  Reusing existing prompt library for domain=education, category=engineering college (created for: VIT)"
```

#### Example 2: CRM Software (Manual Category)

```bash
# First CRM - Brand A
curl -X POST http://localhost:8080/api/v1/geo/prompts/generate \
  -d '{"brand": "Salesforce", "category": "CRM Software", "domain": "technology", "count": 30}'
# Response: generated_prompts: 30, existing_prompts: 0

# Second CRM - Brand B (Same category!)
curl -X POST http://localhost:8080/api/v1/geo/prompts/generate \
  -d '{"brand": "HubSpot", "category": "CRM Software", "domain": "technology", "count": 30}'
# Response: generated_prompts: 0, existing_prompts: 30 ⚡ INSTANT REUSE!
```

#### Verify Libraries

```bash
# Check what libraries exist
curl -X GET http://localhost:8080/api/v1/geo/libraries

# Check brand profiles  
curl -X GET http://localhost:8080/api/v1/geo/profiles
```


---

## Website Scraping for Context-Aware Prompts

### How It Works

When you provide a website URL, the system:
1. **Scrapes the homepage** to extract:
   - Page title
   - Meta description
   - Keywords
   - Main content (headings, paragraphs)
2. **Analyzes the content** using AI to understand:
   - What the brand actually does
   - Industry domain
   - Specific category
   - Key features and benefits
3. **Generates prompts** that are:
   - Hyper-realistic (based on real content)
   - Contextually relevant
   - More likely to trigger brand mentions

### What Gets Scraped

```
✅ Homepage title
✅ Meta description & keywords
✅ H1, H2, H3 headings
✅ Paragraph content
✅ First ~2000 characters of main content

❌ Images, videos
❌ Scripts, styles
❌ Navigation, footer content
```

### Example: Compare Results

#### Without Website Scraping (Generic)
```bash
curl -X POST http://localhost:8080/api/v1/geo/prompts/generate \
  -d '{"brand": "Acme Software", "description": "Project management tool", "count": 5}'
```

Generated prompts might be:
- "What are the best project management tools?"
- "How to choose a project management software?"
- "Top project management platforms for teams"

#### With Website Scraping (Specific)
```bash
curl -X POST http://localhost:8080/api/v1/geo/prompts/generate \
  -d '{"brand": "Acme Software", "website": "https://acme.com", "count": 5}'
```

If website mentions "Agile teams", "sprint planning", "Kanban boards":
- "What are the best Agile project management tools?"
- "How to manage sprint planning for remote teams?"
- "Which project management software has the best Kanban boards?"
- "Agile workflow tools for software development teams"
- "Project management with built-in sprint tracking"

**See the difference?** Scraped prompts are hyper-specific to what you actually offer! 🎯

### Privacy & Ethics

- ✅ Only scrapes **public content** from your provided URL
- ✅ Respects HTTP status codes (404, 403, etc.)
- ✅ 15-second timeout to avoid hanging
- ✅ Follows up to 5 redirects
- ✅ Identifies as "GeoBot" in User-Agent
- ❌ Does NOT bypass paywalls or authentication
- ❌ Does NOT crawl multiple pages
- ❌ Does NOT store scraped HTML (only extracted text)

### Error Handling

If website scraping fails:
- System logs warning but continues
- Falls back to manual description (if provided)
- Falls back to brand name only (if no description)
- You still get prompts generated!

### Best Practices

| Scenario | Website URL | Description | Result |
|----------|-------------|-------------|--------|
| **Best** | ✅ Provided | Optional | Hyper-specific prompts from real content |
| **Good** | ✅ Provided | ✅ Provided | Combines both sources |
| **OK** | ❌ Not provided | ✅ Provided | Generic prompts from description |
| **Minimal** | ❌ Not provided | ❌ Not provided | Basic prompts from brand name only |

**Recommendation:** Always provide website URL for best results! 🚀


---

## Category Consistency & Normalization

### The Challenge

When using AI to derive categories, slight variations can prevent prompt reuse:

**Problem Example:**
- IITM → AI derives "higher education institution"
- IITB → AI derives "technical university"
- ❌ Different categories = separate libraries (no reuse!)

### The Solution

The system now uses **two-layer consistency**:

#### 1. Better AI Prompting
The AI is now instructed to use **BROAD, GENERIC** categories:
- ✅ "engineering college" (GOOD - broad)
- ❌ "premier technical university in south asia" (BAD - too specific)

#### 2. Automatic Normalization
After AI derivation, categories are normalized to standard forms:

```
Input Category              → Normalized Category
--------------------------- → ---------------------
"technical university"      → "engineering college"
"institute of technology"   → "engineering college"
"higher education institution" → "higher education"
"ai tool"                   → "ai tools"
"ai platform"               → "ai tools"
"seo tool"                  → "seo tools"
"payment gateway"           → "payment platform"
"crm"                       → "crm software"
"medical center"            → "hospital"
```

### Best Practice: Provide Category Manually

For maximum consistency, **provide the category manually**:

```bash
# Option 1: Let AI derive (may vary)
curl POST /api/v1/geo/prompts/generate \
  -d '{
    "brand": "IITM",
    "website": "https://iitm.ac.in/",
    "count": 10
  }'
# AI might derive: "higher education institution" or "technical university"

# Option 2: Specify category (guaranteed consistency) ✅
curl POST /api/v1/geo/prompts/generate \
  -d '{
    "brand": "IITM",
    "website": "https://iitm.ac.in/",
    "domain": "education",
    "category": "engineering college",
    "count": 10
  }'
# Category is exactly "engineering college" - guaranteed reuse!
```

### Common Standardized Categories

| Domain | Standardized Categories |
|--------|------------------------|
| **Education** | `engineering college`, `business school`, `higher education` |
| **Technology** | `ai tools`, `seo tools`, `crm software`, `cloud storage` |
| **Healthcare** | `hospital`, `clinic`, `telemedicine` |
| **Finance** | `payment platform`, `banking`, `insurance` |
| **Retail** | `ecommerce`, `marketplace` |

### Ensuring Reuse for Similar Brands

#### Method 1: Manual Category (Recommended)
```bash
# First IIT
curl POST /api/v1/geo/prompts/generate \
  -d '{"brand": "IITM", "category": "engineering college", "domain": "education", "count": 10}'
# Creates library: domain=education, category=engineering college

# Second IIT
curl POST /api/v1/geo/prompts/generate \
  -d '{"brand": "IITB", "category": "engineering college", "domain": "education", "count": 10}'
# Reuses! Same domain + category ✅
```

#### Method 2: Let AI Derive + Trust Normalization
```bash
# First IIT
curl POST /api/v1/geo/prompts/generate \
  -d '{"brand": "IITM", "website": "https://iitm.ac.in/", "count": 10}'
# AI derives: "technical university" → normalized to "engineering college"

# Second IIT
curl POST /api/v1/geo/prompts/generate \
  -d '{"brand": "IITB", "website": "https://iitb.ac.in/", "count": 10}'
# AI derives: "higher education institution" → normalized to "engineering college"
# Reuses! Both normalized to same category ✅
```

### Logs to Watch For

```
🤖 AI derived metadata for 'IITM': domain=education, category=engineering college
📚 Creating new prompt library: domain=education, category=engineering college

(Later...)

🤖 AI derived metadata for 'IITB': domain=education, category=engineering college
♻️  Reusing existing prompt library for domain=education, category=engineering college (created for: IITM)
```

If categories don't match in logs, provide category manually!


---

## Prompt Validation & Quality Control

### The Problem: Brand-Specific Prompts

If prompts contain specific brand names, they cannot be reused:

**Example of the Bug:**
```
1. TCE generates prompts → "What scholarships does Thiagarajar College offer?"
2. SRM tries to reuse → Gets TCE's prompt ❌ (mentions wrong brand!)
```

This is a **critical bug** that we've fixed.

### The Solution: Multi-Layer Validation

#### 1. Generation-Time Prevention

When generating prompts, AI is explicitly instructed:
- ✅ Generate GENERIC questions only
- ❌ DO NOT mention the specific brand name
- ✅ Make questions applicable to entire category

**AI Prompt Example:**
```
Generate questions for "engineering college" category.
CRITICAL: Do NOT mention "Thiagarajar College of Engineering"
Make questions generic like:
- "What are the best engineering colleges?" ✅
- "What does Thiagarajar College offer?" ❌
```

#### 2. Reuse-Time Validation

Before reusing prompts from library, system validates each prompt:
```go
1. Check if prompt contains original brand name
2. Check if prompt contains current brand name  
3. Filter out any brand-specific prompts
4. Only reuse truly generic prompts
```

**Example Validation:**
```
Library created by: "Thiagarajar College of Engineering"
Current request from: "SRM University"

Prompt 1: "What are scholarship opportunities for engineering students?"
→ ✅ Generic (no brand names) → REUSE

Prompt 2: "What scholarships does Thiagarajar College offer?"
→ ❌ Contains "Thiagarajar" → SKIP

Prompt 3: "How is campus life at TCE?"
→ ❌ Contains original brand → SKIP
```

#### 3. Smart Gap-Filling

System intelligently handles partial libraries:
- Has **all** prompts needed → Reuse all ✅
- Has **some** generic prompts → Reuse + generate missing ones ✅
- Has **no** generic prompts → Generate all new ✅

**Example Scenarios:**

**Scenario A: Complete Library**
```
Request: 10 prompts
Library: 12 generic prompts
→ ✅ Reuse 10 prompts (pick from 12)
```

**Scenario B: Partial Library (Smart!)**
```
Request: 10 prompts
Library: 6 generic + 4 brand-specific
→ ✅ Reuse 6 generic prompts
→ 🆕 Generate 4 NEW prompts to fill gap
→ 💾 Add 4 new prompts to library
→ 🎉 Return all 10 prompts (6 existing + 4 new)
```

**Scenario C: Bad Library**
```
Request: 10 prompts
Library: 0 generic + 10 brand-specific
→ 🆕 Generate all 10 new prompts
→ 💾 Replace library with new prompts
```

**Logs:**
```
♻️  Found 6 generic prompts, generating 4 more to reach 10 total
🤖 Generating 4 new prompts...
✅ Using 6 existing + 4 newly generated = 10 total prompts
💾 Updated library: now has 10 generic prompts
```

### Common Words Ignored

System ignores common words when checking for brand mentions:
- "college", "university", "institute", "engineering"
- "the", "and", "for", "best", "top", "good"

**Why?** These appear in generic questions too:
- "What are the best engineering colleges?" ✅ (contains "engineering" but it's generic)
- "Which college has best placement?" ✅ (contains "college" but it's generic)

### Manual Verification

You can verify prompt quality by listing the library:

```bash
curl -X GET http://localhost:8080/api/v1/geo/libraries
```

Check the `prompt_ids` and verify they're generic.

### Smart Library Management

System automatically handles imperfect libraries:

**Step 1: Validation**
- Check each prompt for brand-specific content
- Filter out any prompts mentioning brand names

**Step 2: Gap Analysis**
- Count how many generic prompts remain
- Calculate how many more needed

**Step 3: Smart Action**
- If **enough** prompts → Use them ✅
- If **some** prompts → Use + generate missing ones 🔄
- If **no** good prompts → Generate all new 🆕

**Step 4: Library Update**
- Add newly generated prompts to library
- Library grows over time with more generic prompts
- Future requests benefit from larger library

**Example:**
```
Request 1 (TCE): Generates 10 prompts, 8 generic + 2 brand-specific
Request 2 (SRM): Uses 8 generic, generates 2 new → Library now has 10 generic!
Request 3 (VIT): Uses all 10 generic → No generation needed! ⚡
```

**No manual intervention needed!** 🎉

### Best Practice

When generating prompts, verify they're generic:
```bash
# Generate prompts
curl POST /api/v1/geo/prompts/generate -d '{"brand": "XYZ College", ...}'

# Check response
{
  "prompts": [
    {"template": "What are best engineering colleges?"}, ✅
    {"template": "How to choose engineering college?"}, ✅
    {"template": "What courses does XYZ offer?"} ❌ BAD!
  ]
}
```

If you see brand names in prompts → Report as bug!

