# GEGO API Integration Guide for Frontend Developers

This document provides complete API specifications for integrating GEGO's GEO (Generative Engine Optimization) APIs with your frontend application.

## Base URL

```
http://localhost:8989/api/v1
```

## Authentication

Currently, no authentication is required. CORS is enabled for all origins by default.

---

## API Endpoints

## 1. Generate Prompts

**Endpoint:** `POST /api/v1/geo/prompts/generate`

**Purpose:** Generate or retrieve GEO-optimized prompts for a brand. This API intelligently reuses existing prompts from a library or generates new ones using LLM.

### Request Body

```json
{
  "brand": "string",          // Required: Brand name
  "website": "string",         // Optional: Brand website URL for scraping content
  "category": "string",        // Optional: Business category (e.g., "Education", "E-commerce")
  "domain": "string",          // Optional: Domain/industry (e.g., "Higher Education")
  "description": "string",     // Optional: Brand description
  "count": 20                  // Optional: Number of prompts to generate (default: 20)
}
```

### Response (200 OK)

```json
{
  "success": true,
  "message": "Prompts generated successfully",
  "data": {
    "brand": "string",
    "category": "string",
    "domain": "string",
    "existingPrompts": 15,                    // Number of reused prompts from library
    "generatedPrompts": 5,                    // Number of newly generated prompts
    "prompts": [
      {
        "id": "uuid",
        "template": "What are the best options for...",
        "promptType": "comparison",            // comparison, recommendation, informational, etc.
        "category": "string",
        "reused": true                         // true if reused from library, false if newly generated
      }
    ],
    "promptsByType": {
      "comparison": [...],                     // Prompts grouped by type
      "recommendation": [...],
      "informational": [...]
    },
    "typeCounts": {
      "comparison": 8,                         // Count per type
      "recommendation": 7,
      "informational": 5
    }
  }
}
```

### UI Integration Notes

**Essential Fields for UI:**
- ✅ `brand` - Display brand name
- ✅ `prompts[].id` - Use as unique key for list items
- ✅ `prompts[].template` - Display the prompt text
- ✅ `prompts[].promptType` - Display prompt category badge
- ✅ `prompts[].reused` - Show "Reused" or "New" indicator
- ✅ `existingPrompts` + `generatedPrompts` - Show summary statistics
- ✅ `promptsByType` - For grouped/tabbed view of prompts by category

**Optional Fields:**
- `category`, `domain` - Can be displayed in header/metadata
- `typeCounts` - Useful for showing type distribution charts

**UI Recommendations:**
1. Display prompts in a selectable list with checkboxes
2. Show a badge indicating if prompt is "Reused" (green) or "New" (blue)
3. Group prompts by type using tabs or accordion
4. Show summary: "Generated 5 new prompts, Reused 15 existing"
5. Allow users to select prompts for bulk execution

---

## 2. Bulk Execute Campaign

**Endpoint:** `POST /api/v1/geo/execute/bulk`

**Purpose:** Execute multiple prompts across multiple LLMs to test brand visibility. Runs asynchronously in the background.

### Request Body

```json
{
  "campaignName": "string",    // Required: Name for this campaign
  "brand": "string",           // Required: Brand name to analyze
  "promptIds": ["uuid"],       // Required: Array of prompt IDs to execute
  "llmIds": ["uuid"],          // Required: Array of LLM IDs to use
  "temperature": 0.7           // Optional: LLM temperature (0.0-2.0), default: 0.7
}
```

### Response (202 Accepted)

```json
{
  "success": true,
  "message": "Campaign execution started",
  "data": {
    "campaignId": "uuid",
    "campaignName": "string",
    "brand": "string",
    "totalRuns": 60,              // prompts.length × llms.length
    "status": "running",          // running, completed, failed
    "startedAt": "2024-01-01T00:00:00Z",
    "message": "Campaign started successfully. Execution running in background."
  }
}
```

### UI Integration Notes

**Essential Fields for UI:**
- ✅ `campaignId` - Store for tracking campaign progress
- ✅ `campaignName` - Display campaign name
- ✅ `brand` - Display brand being analyzed
- ✅ `totalRuns` - Show total execution count
- ✅ `status` - Display status indicator (running/completed/failed)
- ✅ `startedAt` - Show timestamp

**Not Needed for UI:**
- `message` - Internal status message (can be shown in toast notification)

**UI Recommendations:**
1. Show a loading indicator with text: "Executing 60 prompts across 3 LLMs..."
2. Display a progress notification or status card
3. Store `campaignId` to poll for results or redirect to insights page
4. Show estimated completion time based on `totalRuns`
5. After execution completes (poll status), redirect to insights/analytics

**Important:** This API returns immediately (202 Accepted) while execution happens in the background. You should:
- Show immediate feedback that execution started
- Navigate to insights page after 1-2 seconds
- Poll insights API to check for new results

---

## 3. GEO Insights

**Endpoint:** `POST /api/v1/geo/insights`

**Purpose:** Get comprehensive GEO analytics and performance insights for a brand.

### Request Body

```json
{
  "brand": "string",           // Optional: Brand name (if omitted, returns all brands)
  "campaignId": "string",      // Optional: Specific campaign ID
  "startTime": "2024-01-01T00:00:00Z",  // Optional: Filter by date range
  "endTime": "2024-12-31T23:59:59Z"     // Optional: Filter by date range
}
```

### Response (200 OK)

```json
{
  "success": true,
  "message": "GEO insights retrieved successfully",
  "data": {
    "brand": "string",
    "logoUrl": "https://logo.clearbit.com/brand.com",       // Brand logo URL
    "fallbackLogoUrl": "https://ui-avatars.com/...",        // Fallback if logo not found
    "averageVisibility": 7.5,           // 0-10 score indicating how visible the brand is
    "mentionRate": 65.5,                // Percentage (0-100) of responses mentioning the brand
    "groundingRate": 45.2,              // Percentage of responses citing brand in sources
    "totalResponses": 120,              // Total number of LLM responses analyzed
    
    "sentimentBreakdown": {
      "positive": 75,                    // Count of positive mentions
      "neutral": 30,
      "negative": 5
    },
    
    "topCompetitors": [
      {
        "name": "Competitor A",
        "logoUrl": "https://...",
        "fallbackLogoUrl": "https://...",
        "mentionCount": 45,
        "visibilityAvg": 8.2
      }
    ],
    
    "performanceByLlm": [
      {
        "llmName": "GPT-4",
        "llmProvider": "openai",
        "visibility": 8.5,               // Average visibility score for this LLM
        "mentionRate": 75.5,             // Mention rate for this LLM
        "responseCount": 40
      }
    ],
    
    "performanceByCategory": [
      {
        "category": "Comparison",
        "visibility": 7.8,
        "mentionRate": 68.5,
        "responseCount": 25
      }
    ],
    
    "trends": [                          // Time-series data (optional)
      {
        "date": "2024-01-01",
        "visibility": 7.5,
        "mentions": 12
      }
    ]
  }
}
```

### UI Integration Notes

**Essential Fields for Dashboard:**
- ✅ `brand` - Display brand name
- ✅ `logoUrl` / `fallbackLogoUrl` - Display brand logo (use fallback if primary fails)
- ✅ `averageVisibility` - **KEY METRIC** - Display as large number/gauge (0-10 scale)
- ✅ `mentionRate` - **KEY METRIC** - Display as percentage with progress bar
- ✅ `groundingRate` - **KEY METRIC** - Display as percentage
- ✅ `totalResponses` - Show as context: "Based on 120 AI responses"
- ✅ `sentimentBreakdown` - Display as pie/donut chart (positive/neutral/negative)
- ✅ `topCompetitors` - Display as ranked list with logos and scores
- ✅ `performanceByLlm` - Display as table or bar chart comparing LLM performance
- ✅ `performanceByCategory` - Display as table showing performance by prompt type

**Optional Fields:**
- `trends` - Display time-series line chart if available

**UI Recommendations:**

**Dashboard Layout:**
```
┌─────────────────────────────────────────────────┐
│ [Brand Logo] Brand Name                         │
│                                                  │
│ ┌─────────────┐ ┌──────────────┐ ┌────────────┐│
│ │ Visibility  │ │ Mention Rate │ │ Grounding  ││
│ │    7.5/10   │ │    65.5%     │ │   45.2%    ││
│ └─────────────┘ └──────────────┘ └────────────┘│
│                                                  │
│ Sentiment Breakdown          Top Competitors    │
│ [Pie Chart]                  [List with logos]  │
│                                                  │
│ Performance by LLM           By Prompt Type     │
│ [Bar Chart]                  [Table]            │
└─────────────────────────────────────────────────┘
```

**Metric Color Coding:**
- Visibility: 8-10 = Green, 5-7.9 = Yellow, 0-4.9 = Red
- Mention Rate: >60% = Green, 30-60% = Yellow, <30% = Red
- Grounding Rate: >40% = Green, 20-40% = Yellow, <20% = Red

---

## 4. Source Analytics

**Endpoint:** `POST /api/v1/geo/analytics/sources`

**Purpose:** Analyze which sources/domains AI models cite when mentioning your brand.

### Request Body

```json
{
  "brand": "string",           // Required: Brand name
  "startTime": "2024-01-01T00:00:00Z",  // Optional: Filter by date
  "endTime": "2024-12-31T23:59:59Z",    // Optional: Filter by date
  "topN": 20                   // Optional: Number of top sources to return (default: 20)
}
```

### Response (200 OK)

```json
{
  "success": true,
  "message": "Source analytics retrieved successfully",
  "data": {
    "brand": "string",
    "logoUrl": "https://...",
    "fallbackLogoUrl": "https://...",
    "period": "Last 30 days",
    "totalSources": 45,                  // Total unique domains cited
    "totalCitations": 230,               // Total number of citations
    
    "topSources": [
      {
        "domain": "example.com",
        "citationCount": 45,             // Number of times this domain was cited
        "mentionRate": 37.5,             // Percentage of responses citing this domain
        "llmBreakdown": {
          "GPT-4": 20,                   // Citation count per LLM
          "Claude": 15,
          "Gemini": 10
        },
        "categories": ["comparison", "informational"]  // Prompt types where cited
      }
    ],
    
    "recommendations": [
      {
        "type": "content_partnership",
        "priority": "high",              // high, medium, low
        "title": "Strengthen presence on example.com",
        "description": "This domain is cited 45 times. Consider contributing content.",
        "action": "Reach out for guest posting opportunities",
        "impact": "Could improve citation rate by 15%"
      }
    ]
  }
}
```

### UI Integration Notes

**Essential Fields for UI:**
- ✅ `brand` + logos - Display at top
- ✅ `totalSources` - Show as summary stat
- ✅ `totalCitations` - Show as summary stat
- ✅ `topSources[].domain` - Display as clickable links
- ✅ `topSources[].citationCount` - Display with bar chart
- ✅ `topSources[].mentionRate` - Show percentage
- ✅ `topSources[].llmBreakdown` - Display in expandable row or tooltip
- ✅ `recommendations` - Display as actionable cards with priority badges

**Optional:**
- `topSources[].categories` - Can show as tags
- `period` - Display as context

**UI Recommendations:**

**Layout:**
```
┌──────────────────────────────────────────────────┐
│ Sources Citing [Brand]                           │
│ 45 unique sources · 230 total citations          │
│                                                   │
│ Top Citing Domains:                              │
│ ┌────────────────────────────────────────────┐  │
│ │ 1. example.com          45 citations (38%) │  │
│ │    [Bar ████████████░░░░░░░░]              │  │
│ │    GPT-4: 20 | Claude: 15 | Gemini: 10    │  │
│ └────────────────────────────────────────────┘  │
│                                                   │
│ 💡 Recommendations [HIGH PRIORITY]               │
│ [Action Cards]                                   │
└──────────────────────────────────────────────────┘
```

**Priority Badge Colors:**
- HIGH = Red/Orange
- MEDIUM = Yellow
- LOW = Blue

---

## 5. Prompt Performance Analytics

**Endpoint:** `POST /api/v1/geo/analytics/prompt-performance`

**Purpose:** Analyze which prompts generate the best brand visibility and recommendations for optimization.

### Request Body

```json
{
  "brand": "string",           // Required: Brand name
  "startTime": "2024-01-01T00:00:00Z",  // Optional: Filter by date
  "endTime": "2024-12-31T23:59:59Z",    // Optional: Filter by date
  "minResponses": 3            // Optional: Minimum responses needed per prompt (default: 3)
}
```

### Response (200 OK)

```json
{
  "success": true,
  "message": "Prompt performance retrieved successfully",
  "data": {
    "brand": "string",
    "logoUrl": "https://...",
    "fallbackLogoUrl": "https://...",
    "period": "Last 30 days",
    "totalPromptsAnalyzed": 25,
    "avgEffectiveness": 72.5,          // Overall average effectiveness score
    
    "topPerformers": ["prompt-id-1", "prompt-id-2", "prompt-id-3"],  // Best prompt IDs
    "lowPerformers": ["prompt-id-4", "prompt-id-5"],                  // Worst prompt IDs
    
    "prompts": [
      {
        "promptId": "uuid",
        "promptText": "What are the best universities for engineering?",
        "promptType": "comparison",
        "category": "Education",
        
        // Performance Metrics
        "avgVisibility": 8.5,          // Average visibility score (0-10)
        "avgPosition": 2.3,            // Average ranking position (lower is better)
        "mentionRate": 85.5,           // Percentage brand is mentioned
        "topPositionRate": 67.5,       // Percentage of top 3 rankings
        "avgSentiment": 0.8,           // Sentiment score (-1 to +1)
        
        // Volume Metrics
        "totalResponses": 20,          // Total LLM responses for this prompt
        "brandMentions": 17,           // Times brand was mentioned
        
        // Effectiveness
        "effectivenessScore": 85.2,    // Composite score (0-100)
        "effectivenessGrade": "A",     // A, B, C, D, F
        "status": "high_performing",   // high_performing, performing, under_performing
        "recommendation": "Keep this prompt. It drives excellent visibility."
      }
    ]
  }
}
```

### UI Integration Notes

**Essential Fields for UI:**
- ✅ `totalPromptsAnalyzed` - Show as summary
- ✅ `avgEffectiveness` - Display as overall score
- ✅ `topPerformers` / `lowPerformers` - Use to highlight/filter prompts
- ✅ `prompts[].promptText` - Display prompt content
- ✅ `prompts[].promptType` - Show as badge
- ✅ `prompts[].avgVisibility` - **KEY METRIC** - Display prominently
- ✅ `prompts[].mentionRate` - **KEY METRIC** - Display as percentage
- ✅ `prompts[].avgPosition` - **KEY METRIC** - Show ranking
- ✅ `prompts[].effectivenessGrade` - Display as colored badge (A-F)
- ✅ `prompts[].status` - Use for color coding rows
- ✅ `prompts[].recommendation` - Display in tooltip or detail view
- ✅ `prompts[].totalResponses` - Show as context

**Optional:**
- `topPositionRate`, `avgSentiment`, `brandMentions` - Show in expanded view

**UI Recommendations:**

**Table View:**
```
┌─────────────────────────────────────────────────────────────────────┐
│ Prompt Performance: 25 prompts analyzed · Avg Effectiveness: 72.5  │
├─────────────────┬────────┬────────┬──────────┬────────────┬────────┤
│ Prompt          │ Type   │ Grade  │ Mention  │ Position   │ Status │
│                 │        │        │ Rate     │            │        │
├─────────────────┼────────┼────────┼──────────┼────────────┼────────┤
│ What are the... │ Comp.  │   A    │  85.5%   │    2.3     │   🟢   │
│ [Expand ▼]      │        │        │          │            │        │
├─────────────────┼────────┼────────┼──────────┼────────────┼────────┤
│ Which platform..│ Recom. │   B    │  72.3%   │    3.8     │   🟡   │
└─────────────────┴────────┴────────┴──────────┴────────────┴────────┘

💡 Recommendation: Keep this prompt. It drives excellent visibility.
```

**Grade Color Coding:**
- A (85-100) = Green/Excellent
- B (70-84) = Light Green/Good
- C (50-69) = Yellow/Average
- D (30-49) = Orange/Poor
- F (0-29) = Red/Failing

**Status Icons:**
- 🟢 high_performing
- 🟡 performing
- 🔴 under_performing

---

## 6. Competitive Benchmark

**Endpoint:** `POST /api/v1/geo/analytics/competitive`

**Purpose:** Compare your brand's AI visibility against competitors across the same prompts.

### Request Body

```json
{
  "mainBrand": "string",       // Required: Your brand name
  "competitors": ["string"],   // Required: Array of competitor brand names
  "promptIds": ["uuid"],       // Optional: Filter by specific prompts
  "llmIds": ["uuid"],          // Optional: Filter by specific LLMs
  "startTime": "2024-01-01T00:00:00Z",  // Optional: Date range
  "endTime": "2024-12-31T23:59:59Z",    // Optional: Date range
  "region": "string"           // Optional: Filter by region (e.g., "US", "EU")
}
```

### Response (200 OK)

```json
{
  "success": true,
  "message": "Competitive benchmark retrieved successfully",
  "data": {
    "mainBrand": {
      "brand": "Your Brand",
      "logoUrl": "https://...",
      "fallbackLogoUrl": "https://...",
      "visibility": 7.5,             // Average visibility (0-10)
      "mentionRate": 65.5,           // Percentage mentioned (0-100)
      "groundingRate": 45.2,         // Percentage in sources (0-100)
      "averagePosition": 2.8,        // Average ranking (lower is better)
      "topPositionRate": 55.5,       // Percentage in top 3
      "sentimentScore": 0.75,        // Sentiment (-1 to +1)
      "responseCount": 120,          // Total responses analyzed
      "marketSharePct": 35.5         // Share of voice (0-100)
    },
    
    "competitors": [
      {
        "brand": "Competitor A",
        "logoUrl": "https://...",
        "fallbackLogoUrl": "https://...",
        "visibility": 8.2,
        "mentionRate": 72.3,
        "groundingRate": 50.8,
        "averagePosition": 2.1,
        "topPositionRate": 68.5,
        "sentimentScore": 0.68,
        "responseCount": 145,
        "marketSharePct": 42.5
      }
    ],
    
    "marketLeader": "Competitor A",    // Brand with highest mention rate
    "yourRank": 2,                     // Your ranking among all brands
    "totalBrands": 4,                  // Total brands analyzed
    
    "promptBreakdown": [
      {
        "promptId": "uuid",
        "promptText": "What are the best...",
        "promptType": "comparison",
        "executedAt": "2024-01-01T00:00:00Z",
        
        "mainBrandResult": {
          "mentioned": true,
          "visibilityScore": 8,
          "position": 2,               // Ranking position
          "sentiment": "positive",
          "inSources": true            // Cited in grounding sources
        },
        
        "competitorsMentioned": [
          {
            "brand": "Competitor A",
            "mentioned": true
          },
          {
            "brand": "Competitor B",
            "mentioned": false
          }
        ],
        
        "winner": "Your Brand",        // Best performing brand for this prompt
        "totalBrandsMentioned": 3      // How many brands mentioned
      }
    ],
    
    "recommendations": [
      {
        "type": "visibility_gap",
        "priority": "high",
        "title": "Close the gap with Competitor A",
        "description": "Competitor A has 6.8% higher mention rate",
        "action": "Focus on comparison-type prompts where they excel",
        "impact": "Could improve market share by 8%"
      }
    ],
    
    "analyzedAt": "2024-01-01T00:00:00Z"
  }
}
```

### UI Integration Notes

**Essential Fields for Competitive Dashboard:**

**Overview Section:**
- ✅ `mainBrand.brand` + `logoUrl` - Your brand header
- ✅ `mainBrand.visibility` - **PRIMARY METRIC**
- ✅ `mainBrand.mentionRate` - **PRIMARY METRIC**
- ✅ `mainBrand.marketSharePct` - **PRIMARY METRIC**
- ✅ `yourRank` + `totalBrands` - Display: "Ranked #2 of 4"
- ✅ `marketLeader` - Highlight who's winning

**Competitive Table:**
- ✅ All metrics from `mainBrand` and `competitors[]`
- ✅ Display as sortable comparison table
- ✅ Use color coding: You (blue), Market Leader (gold), Others (gray)

**Detailed Analysis:**
- ✅ `promptBreakdown[]` - Show prompt-by-prompt competitive results
- ✅ `promptBreakdown[].mainBrandResult` - Your performance
- ✅ `promptBreakdown[].competitorsMentioned` - Who else appeared
- ✅ `promptBreakdown[].winner` - Highlight winner with trophy icon
- ✅ `recommendations[]` - Display as actionable cards

**Optional:**
- `averagePosition`, `topPositionRate`, `sentimentScore` - Show in detailed view
- `groundingRate` - Advanced metric

**UI Recommendations:**

**Dashboard Layout:**
```
┌──────────────────────────────────────────────────────────┐
│ Competitive Analysis                                      │
│ You're ranked #2 of 4 brands · Market Leader: CompetitorA│
│                                                            │
│ ┌────────────────────────────────────────────────────┐   │
│ │ Brand          │ Visibility│ Mention │ Market Share│   │
│ ├────────────────┼───────────┼─────────┼────────────┤   │
│ │ 🥇 Competitor A│    8.2    │  72.3%  │   42.5%    │   │
│ │ 🔷 Your Brand  │    7.5    │  65.5%  │   35.5%    │   │
│ │    Competitor B│    6.8    │  58.2%  │   22.0%    │   │
│ └────────────────────────────────────────────────────┘   │
│                                                            │
│ Prompt-by-Prompt Breakdown:                               │
│ ┌────────────────────────────────────────────────────┐   │
│ │ "What are the best..." [comparison]                │   │
│ │ Winner: 🏆 Your Brand (Position #2)                │   │
│ │ • Your Brand ✓ mentioned (pos. 2)                  │   │
│ │ • Competitor A ✓ mentioned                         │   │
│ │ • Competitor B ✗ not mentioned                     │   │
│ └────────────────────────────────────────────────────┘   │
│                                                            │
│ 💡 Recommendations [HIGH PRIORITY]                        │
│ [Action Cards]                                            │
└──────────────────────────────────────────────────────────┘
```

**Visual Indicators:**
- Use different colors for your brand vs competitors
- Market leader gets gold/star icon
- Show trend arrows (up/down) if comparing periods
- Winner badges for prompt-level analysis

---

## Error Responses

All APIs follow a consistent error format:

```json
{
  "success": false,
  "error": "Error message description"
}
```

**Common HTTP Status Codes:**
- `400 Bad Request` - Invalid input (missing required fields, invalid values)
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server-side error

---

## Field Summary for UI Development

### Priority 1 (Must Have):
These fields are critical for core UI functionality:

- **Metrics:** `visibility`, `mentionRate`, `averagePosition`
- **Identifiers:** `id`, `brand`, `campaignId`
- **Content:** `promptText`, `responseText`
- **Status:** `status`, `grade`, `winner`
- **Counts:** `totalResponses`, `totalRuns`, `totalBrands`

### Priority 2 (Should Have):
Important for enhanced user experience:

- **Branding:** `logoUrl`, `fallbackLogoUrl`
- **Breakdowns:** `sentimentBreakdown`, `llmBreakdown`, `promptBreakdown`
- **Insights:** `recommendations[]`, `topCompetitors[]`
- **Metadata:** `createdAt`, `analyzedAt`, `promptType`

### Priority 3 (Nice to Have):
Advanced features and analytics:

- **Advanced Metrics:** `groundingRate`, `sentimentScore`, `marketSharePct`
- **Trends:** `trends[]`, time-series data
- **Details:** `categories[]`, `llmProvider`, `description`

---

## Integration Checklist

### Phase 1: Basic Integration
- [ ] Set up API client with base URL
- [ ] Implement prompt generation UI
- [ ] Implement bulk execution trigger
- [ ] Display basic insights dashboard

### Phase 2: Analytics
- [ ] Add competitive benchmark comparison
- [ ] Add prompt performance analysis
- [ ] Add source analytics

### Phase 3: Polish
- [ ] Implement logo fallbacks
- [ ] Add loading states
- [ ] Add error handling
- [ ] Implement data refresh/polling
- [ ] Add export functionality

---

## 7. Competitor Management

### 7.1 Suggest Competitors

**Endpoint:** `GET /api/v1/geo/competitors/suggest`

**Purpose:** Get AI-powered competitor suggestions based on brand and website. First call uses LLM, subsequent calls return cached suggestions.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `brand` | string | Yes | Your brand name |
| `website` | string | No | Your website URL (recommended for better suggestions) |
| `description` | string | No | Brand description |
| `category` | string | No | Industry category |
| `forceRefresh` | boolean | No | Force LLM regeneration (default: false) |

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "competitors": ["VS Code", "GitHub Copilot", "Tabnine", "Codeium"],
    "source": "llm",
    "message": "Found 4 competitors for Cursor"
  }
}
```

**UI Integration Notes:**
- ✅ `competitors` - Display as selectable list with checkboxes
- ✅ `source` - Show "AI Generated" badge if "llm", "Cached" if "cached"
- ✅ Allow users to select competitors for saving
- ✅ Allow users to add custom competitors

### 7.2 Save Competitors

**Endpoint:** `POST /api/v1/geo/competitors`

**Purpose:** Save user-selected competitors for analytics.

**Request Body:**
```json
{
  "brand": "Cursor",
  "competitors": ["VS Code", "GitHub Copilot", "Tabnine"],
  "source": "suggested"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "competitors": ["VS Code", "GitHub Copilot", "Tabnine"],
    "source": "suggested",
    "savedAt": "2024-01-01T00:00:00Z",
    "message": "Successfully saved 3 competitors for Cursor"
  }
}
```

### 7.3 Get Saved Competitors

**Endpoint:** `GET /api/v1/geo/competitors?brand=Cursor`

**Purpose:** Retrieve saved competitors for a brand.

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "competitors": ["VS Code", "GitHub Copilot", "Tabnine"],
    "suggestedList": ["VS Code", "GitHub Copilot", "Tabnine", "Codeium", "JetBrains"],
    "source": "suggested",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

---

## Example Integration Flow

```javascript
// 1. Setup: Suggest and save competitors first
const suggestResponse = await fetch('/api/v1/geo/competitors/suggest?brand=MIT&website=https://mit.edu');
const { data: { competitors: suggestedCompetitors } } = await suggestResponse.json();

// 2. Save selected competitors
await fetch('/api/v1/geo/competitors', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    brand: 'MIT',
    competitors: ['Stanford', 'Harvard', 'Caltech'],
    source: 'suggested'
  })
});

// 3. Generate prompts for a brand
const generateResponse = await fetch('/api/v1/geo/prompts/generate', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    brand: 'MIT',
    website: 'https://mit.edu',
    category: 'Education',
    count: 20
  })
});
const { data: { promptsByType } } = await generateResponse.json();
const allPrompts = Object.values(promptsByType).flat();

// 4. Let user select prompts and LLMs, then execute campaign
const executeResponse = await fetch('/api/v1/geo/execute/bulk', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    campaignName: 'MIT Visibility Test',
    brand: 'MIT',
    promptIds: allPrompts.map(p => p.id),
    llmIds: selectedLlmIds,
    temperature: 0.7
  })
});
const { data: { campaignId } } = await executeResponse.json();

// 5. Wait a bit, then fetch insights
setTimeout(async () => {
  const insightsResponse = await fetch('/api/v1/geo/insights', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      brand: 'MIT'
    })
  });
  const { data: insights } = await insightsResponse.json();
  
  // Display dashboard with:
  // - Visibility: insights.averageVisibility
  // - Mention Rate: insights.mentionRate
  // - Sentiment: insights.sentimentBreakdown
  // - Competitors: insights.topCompetitors
}, 5000);

// 6. Get competitive analysis (automatically uses saved competitors!)
const competitiveResponse = await fetch('/api/v1/geo/analytics/competitive', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    mainBrand: 'MIT'
    // No need to specify competitors - uses saved list automatically
  })
});
const { data: benchmark } = await competitiveResponse.json();

// Display competitive table with mainBrand vs competitors
```

---

## Support

For questions or issues:
- GitHub: https://github.com/fissionx/gego
- Documentation: See README.md and ADVANCED_ANALYTICS.md

---

**Document Version:** 1.0  
**Last Updated:** 2025-11-30 
**API Version:** v1

