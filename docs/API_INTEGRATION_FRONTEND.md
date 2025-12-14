# GEGO API Integration Guide for Frontend Developers

This document provides complete API specifications for integrating GEGO's GEO (Generative Engine Optimization) APIs with your frontend application.

## Base URL

**Local Development:**
```
http://localhost:8989/api/v1
```

**Production:**
```
{{HOST}}/api/v1
```

## Authentication

Currently, no authentication is required. CORS is enabled for all origins by default.

## API Response Format

All APIs follow a consistent response format:

**Success Response:**
```json
{
  "success": true,
  "message": "Operation completed successfully",
  "data": { /* response data */ }
}
```

**Error Response:**
```json
{
  "success": false,
  "error": "Error message description"
}
```

---

## Table of Contents

1. [System APIs](#1-system-apis)
   - [Health Check](#11-health-check)
2. [LLM Configuration APIs](#2-llm-configuration-apis)
   - [Get All LLMs](#21-get-all-llms)
   - [Add New LLM](#22-add-new-llm)
3. [Competitor APIs](#3-competitor-apis)
   - [Suggest Competitors](#31-suggest-competitors)
   - [Save Competitors](#32-save-competitors)
   - [Get Competitors](#33-get-competitors)
   - [Delete Competitors](#34-delete-competitors)
4. [Prompt APIs](#4-prompt-apis)
   - [Generate Prompts](#41-generate-prompts)
   - [Get Prompts by Brand](#42-get-prompts-by-brand)
5. [Execution APIs](#5-execution-apis)
   - [Execute Bulk Campaign](#51-execute-bulk-campaign)
   - [Delete All Campaigns](#52-delete-all-campaigns)
6. [Analytics APIs](#6-analytics-apis)
   - [Dashboard Overview](#61-dashboard-overview)
   - [Source Analytics](#62-source-analytics)
   - [Prompt Performance](#63-prompt-performance)
   - [Prompt Time Series](#64-prompt-time-series)
   - [Position Analytics](#65-position-analytics)
   - [Model Analytics](#66-model-analytics)
   - [Competitive Benchmark](#67-competitive-benchmark)
   - [Trend Comparison](#68-trend-comparison)

---

## API Endpoints

---

# 1. System APIs

## 1.1 Health Check

**Endpoint:** `GET /api/v1/health`

**Purpose:** Check if the API server is running and responsive.

**Request:** No parameters required

**Response (200 OK):**
```json
{
  "status": "ok",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

**UI Use Case:**
- Display server status indicator (green dot when healthy)
- Show in settings/status page
- Use for connection testing

---

# 2. LLM Configuration APIs

## 2.1 Get All LLMs

**Endpoint:** `GET /api/v1/llms`

**Purpose:** Retrieve all configured LLM providers and their details for prompt execution.

**Request:** No parameters required

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "llms": [
      {
        "id": "uuid",
        "name": "GPT-4",
        "provider": "openai",
        "model": "gpt-4",
        "enabled": true,
        "apiKeyConfigured": true
      },
      {
        "id": "uuid",
        "name": "Claude 3.5 Sonnet",
        "provider": "anthropic",
        "model": "claude-3-5-sonnet-20241022",
        "enabled": true,
        "apiKeyConfigured": true
      },
      {
        "id": "uuid",
        "name": "Gemini Flash 2.5",
        "provider": "google",
        "model": "models/gemini-2.5-flash",
        "enabled": true,
        "apiKeyConfigured": false
      }
    ],
    "total": 3,
    "enabled": 2
  }
}
```

**UI Integration:**
- ✅ `id` - Use for selecting LLMs in execution requests
- ✅ `name` - Display name in LLM selection UI
- ✅ `provider` - Show provider icon/logo (OpenAI, Anthropic, Google, etc.)
- ✅ `model` - Display model identifier
- ✅ `enabled` - Only show enabled LLMs in selection
- ✅ `apiKeyConfigured` - Show warning if false

**UI Recommendations:**
```
┌─────────────────────────────────────────────┐
│ Select LLMs for Testing                     │
├─────────────────────────────────────────────┤
│ ☑ GPT-4 (OpenAI) ✓ Ready                   │
│ ☑ Claude 3.5 Sonnet (Anthropic) ✓ Ready    │
│ ☐ Gemini Flash 2.5 (Google) ⚠️ No API Key  │
└─────────────────────────────────────────────┘
```

---

## 2.2 Add New LLM

**Endpoint:** `POST /api/v1/llms`

**Purpose:** Add a new LLM configuration to the system.

**Request Body:**
```json
{
  "name": "Gemini Flash 2.5",
  "provider": "google",
  "model": "models/gemini-2.5-flash",
  "apiKey": "your-api-key-here",
  "enabled": true
}
```

**Field Descriptions:**
- `name` (string, required) - Display name for the LLM
- `provider` (string, required) - Provider identifier: `openai`, `anthropic`, `google`, `perplexity`, `ollama`
- `model` (string, required) - Model identifier from the provider
- `apiKey` (string, optional) - API key for the provider (not needed for Ollama)
- `enabled` (boolean, optional) - Whether to enable this LLM (default: true)

**Response (201 Created):**
```json
{
  "success": true,
  "message": "LLM configuration added successfully",
  "data": {
    "id": "uuid",
    "name": "Gemini Flash 2.5",
    "provider": "google",
    "model": "models/gemini-2.5-flash",
    "enabled": true,
    "createdAt": "2024-01-01T00:00:00Z"
  }
}
```

**UI Use Case:**
- Settings page for adding new LLM providers
- Form with provider dropdown, model input, and API key field
- Validation for required fields

---

# 3. Competitor APIs

## 3.1 Suggest Competitors

**Endpoint:** `GET /api/v1/geo/competitors/suggest`

**Purpose:** Get AI-powered competitor suggestions based on your brand and website. Uses LLM to analyze your brand and suggest relevant competitors.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `brand` | string | Yes | Your brand name |
| `website` | string | No | Your website URL (recommended for better suggestions) |
| `description` | string | No | Brand description |
| `category` | string | No | Industry category |
| `forceRefresh` | boolean | No | Force new LLM suggestions (default: false, uses cache) |

**Example Request:**
```
GET /api/v1/geo/competitors/suggest?brand=Cursor&website=https://cursor.com
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "competitors": [
      "GitHub Copilot",
      "Codeium",
      "Tabnine",
      "Amazon CodeWhisperer",
      "Replit"
    ],
    "source": "llm",
    "message": "Found 5 competitors for Cursor",
    "generatedAt": "2024-01-01T00:00:00Z"
  }
}
```

**Field Descriptions:**
- `competitors` (array) - List of suggested competitor names
- `source` (string) - `llm` (newly generated) or `cached` (from previous request)
- `message` (string) - Human-readable summary

**UI Integration:**
- ✅ Display as selectable checkboxes
- ✅ Show "AI Generated" badge if source is "llm"
- ✅ Show "Cached" badge if source is "cached"
- ✅ Allow users to select competitors to save
- ✅ Provide option to add custom competitors manually

**UI Example:**
```
┌────────────────────────────────────────┐
│ AI Suggested Competitors 🤖            │
│ (Based on your website analysis)       │
├────────────────────────────────────────┤
│ ☑ GitHub Copilot                       │
│ ☑ Codeium                              │
│ ☑ Tabnine                              │
│ ☐ Amazon CodeWhisperer                 │
│ ☐ Replit                               │
├────────────────────────────────────────┤
│ [+ Add Custom Competitor]              │
│                                         │
│ [Save Selected] [Refresh Suggestions]  │
└────────────────────────────────────────┘
```

---

## 3.2 Save Competitors

**Endpoint:** `POST /api/v1/geo/competitors`

**Purpose:** Save user-selected competitors for future analytics and benchmarking.

**Request Body:**
```json
{
  "brand": "Cursor",
  "competitors": [
    "GitHub Copilot",
    "Codeium"
  ],
  "source": "suggested"
}
```

**Field Descriptions:**
- `brand` (string, required) - Your brand name
- `competitors` (array, required) - List of competitor names to save
- `source` (string, required) - Source of competitors: `suggested`, `manual`, or `mixed`

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "competitors": [
      "GitHub Copilot",
      "Codeium"
    ],
    "source": "suggested",
    "savedAt": "2024-01-01T00:00:00Z",
    "message": "Successfully saved 2 competitors for Cursor"
  }
}
```

**UI Use Case:**
- Save button after competitor selection
- Show success toast: "2 competitors saved successfully"
- Automatically used in competitive analytics

---

## 3.3 Get Competitors

**Endpoint:** `GET /api/v1/geo/competitors`

**Purpose:** Retrieve saved competitors for a brand.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `brand` | string | Yes | Brand name to get competitors for |

**Example Request:**
```
GET /api/v1/geo/competitors?brand=Cursor
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "competitors": [
      "GitHub Copilot",
      "Codeium"
    ],
    "suggestedList": [
      "GitHub Copilot",
      "Codeium",
      "Tabnine",
      "Amazon CodeWhisperer",
      "Replit"
    ],
    "source": "suggested",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

**Field Descriptions:**
- `competitors` (array) - Currently saved competitors
- `suggestedList` (array) - Full list of AI suggestions (if available)
- `source` (string) - How competitors were added
- `updatedAt` (timestamp) - Last update time

**UI Integration:**
- ✅ Display saved competitors with edit option
- ✅ Show suggested but not saved competitors as "Add more"
- ✅ Allow removing competitors

---

## 3.4 Delete Competitors

**Endpoint:** `DELETE /api/v1/geo/competitors`

**Purpose:** Delete all saved competitors for a brand.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `brand` | string | Yes | Brand name to delete competitors for |

**Example Request:**
```
DELETE /api/v1/geo/competitors?brand=Cursor
```

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Successfully deleted all competitors for Cursor"
}
```

**UI Use Case:**
- "Clear all competitors" button with confirmation dialog
- Show in settings or competitor management page

---

# 4. Prompt APIs

## 4.1 Generate Prompts

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

## 4.2 Get Prompts by Brand

**Endpoint:** `GET /api/v1/geo/prompts`

**Purpose:** Retrieve all generated prompts for a specific brand.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `brand` | string | Yes | Brand name to get prompts for |

**Example Request:**
```
GET /api/v1/geo/prompts?brand=Cursor
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "prompts": [
      {
        "id": "28bf8648-cb51-45df-a8fa-fadc8140c95a",
        "template": "What are the best AI coding assistants for developers?",
        "promptType": "comparison",
        "category": "AI coding",
        "reused": true,
        "createdAt": "2024-01-01T00:00:00Z"
      },
      {
        "id": "uuid-2",
        "template": "Which code completion tool should I choose?",
        "promptType": "recommendation",
        "category": "AI coding",
        "reused": false,
        "createdAt": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 25,
    "promptsByType": {
      "comparison": [...],
      "recommendation": [...],
      "informational": [...]
    },
    "typeCounts": {
      "comparison": 10,
      "recommendation": 8,
      "informational": 7
    }
  }
}
```

**UI Integration:**
- ✅ Display as searchable/filterable list
- ✅ Group by `promptType` using tabs or accordion
- ✅ Show `reused` badge (green for reused, blue for new)
- ✅ Use `id` for selection in bulk execution
- ✅ Display `typeCounts` in summary cards

**UI Example:**
```
┌────────────────────────────────────────────┐
│ Prompts for Cursor (25 total)             │
│ [Comparison: 10] [Recommendation: 8]       │
├────────────────────────────────────────────┤
│ ☑ What are the best AI coding...  [Reused]│
│ ☐ Which code completion tool...   [New]   │
│ ☐ How does Cursor compare to...   [Reused]│
└────────────────────────────────────────────┘
```

---

# 5. Execution APIs

## 5.1 Execute Bulk Campaign

**Endpoint:** `POST /api/v1/geo/prompts/execute/bulk`

**Purpose:** Execute multiple prompts across multiple LLMs to test brand visibility. Supports one-time execution and optional scheduled recurring runs. Runs asynchronously in the background.

**Request Body:**
```json
{
  "campaignName": "cursor brand check",
  "brand": "Cursor",
  "promptIds": [
    "28bf8648-cb51-45df-a8fa-fadc8140c95a"
  ],
  "customPrompts": [
    {
      "template": "compare cursor with copilot and windsurf",
      "promptType": "compare"
    }
  ],
  "llmIds": [
    "f03b45b7-bf22-4449-a32d-8a90540432dd"
  ],
  "temperature": 0.7,
  "scheduleCron": "*/2 * * * *",
  "totalRuns": 1
}
```

**Field Descriptions:**

**Required Fields:**
- `campaignName` (string) - Name for this campaign
- `brand` (string) - Brand name to analyze
- `promptIds` (array) - Array of prompt IDs from generated prompts
- `llmIds` (array) - Array of LLM IDs to execute prompts on

**Optional Fields:**
- `customPrompts` (array) - Custom prompts to execute alongside saved prompts
  - `template` (string) - The prompt text
  - `promptType` (string) - Type: `comparison`, `recommendation`, `informational`, etc.
- `temperature` (float) - LLM temperature (0.0-2.0), default: 0.7
  - Lower (0.0-0.3): More focused and deterministic
  - Medium (0.4-0.7): Balanced creativity
  - Higher (0.8-2.0): More creative and varied
- `scheduleCron` (string) - Cron expression for recurring execution (e.g., `*/2 * * * *` = every 2 minutes)
  - If omitted: One-time execution
  - If provided: Recurring scheduled execution
- `totalRuns` (number) - Number of times to run the campaign (for scheduled campaigns)

**Response (202 Accepted):**
```json
{
  "success": true,
  "message": "Campaign execution started",
  "data": {
    "campaignId": "uuid",
    "campaignName": "cursor brand check",
    "brand": "Cursor",
    "totalRuns": 2,
    "status": "running",
    "startedAt": "2024-01-01T00:00:00Z",
    "scheduleCron": "*/2 * * * *",
    "scheduled": true,
    "message": "Campaign started successfully. Execution running in background."
  }
}
```

**Field Descriptions (Response):**
- `campaignId` (string) - Unique campaign identifier
- `totalRuns` (number) - Total executions (promptIds + customPrompts) × llmIds
- `status` (string) - `running`, `scheduled`, `completed`, `failed`
- `scheduled` (boolean) - true if recurring, false if one-time
- `scheduleCron` (string) - Cron schedule if recurring

**UI Integration:**

**Essential Fields:**
- ✅ `campaignId` - Store for tracking
- ✅ `campaignName` - Display in progress indicator
- ✅ `totalRuns` - Show execution count
- ✅ `status` - Display status badge
- ✅ `scheduled` - Show "One-time" vs "Recurring" indicator

**UI Recommendations:**

**For One-time Execution:**
```
┌────────────────────────────────────────┐
│ ⏳ Executing Campaign                  │
│ "cursor brand check"                   │
│                                         │
│ Status: Running                        │
│ Progress: 2 prompt executions          │
│ Started: 2 seconds ago                 │
│                                         │
│ [View Live Results]                    │
└────────────────────────────────────────┘
```

**For Scheduled Execution:**
```
┌────────────────────────────────────────┐
│ 📅 Scheduled Campaign Active           │
│ "cursor brand check"                   │
│                                         │
│ Schedule: Every 2 minutes              │
│ Next Run: In 1m 30s                    │
│ Runs Completed: 5 / 10                 │
│                                         │
│ [View Results] [Stop Campaign]         │
└────────────────────────────────────────┘
```

**Important Notes:**
- API returns immediately (202 Accepted) while execution happens in background
- For one-time: Redirect to analytics after 2-3 seconds
- For scheduled: Show campaign management page
- Poll analytics APIs to see new results appear

---

## 5.2 Delete All Campaigns

**Endpoint:** `DELETE /api/v1/geo/campaigns/all`

**Purpose:** Delete all background scheduled campaigns (stops all recurring executions).

**Request:** No parameters required

**Response (200 OK):**
```json
{
  "success": true,
  "message": "All campaigns deleted successfully",
  "data": {
    "deletedCount": 5
  }
}
```

**UI Use Case:**
- "Stop all campaigns" button in settings
- Show confirmation dialog: "This will stop 5 running campaigns"
- Use when cleaning up test campaigns

---

# 6. Analytics APIs

## 6.1 Dashboard Overview

**Endpoint:** `GET /api/v1/geo/dashboard/overview`

**Purpose:** Get a comprehensive overview of brand performance with key metrics at a glance.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `brand` | string | Yes | Brand name to analyze |
| `startTime` | ISO 8601 | No | Filter results from this date |
| `endTime` | ISO 8601 | No | Filter results until this date |

**Example Request:**
```
GET /api/v1/geo/dashboard/overview?brand=Cursor
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "logoUrl": "https://logo.clearbit.com/cursor.com",
    "fallbackLogoUrl": "https://ui-avatars.com/api/?name=Cursor",
    
    "summary": {
      "totalResponses": 150,
      "totalPrompts": 25,
      "totalLLMs": 3,
      "dateRange": {
        "start": "2024-01-01T00:00:00Z",
        "end": "2024-01-31T23:59:59Z"
      }
    },
    
    "keyMetrics": {
      "averageVisibility": 7.8,
      "mentionRate": 72.5,
      "averagePosition": 2.3,
      "groundingRate": 48.5
    },
    
    "sentiment": {
      "positive": 95,
      "neutral": 40,
      "negative": 15,
      "averageScore": 0.72
    },
    
    "topCompetitors": [
      {
        "name": "GitHub Copilot",
        "logoUrl": "https://...",
        "mentionCount": 85,
        "averageVisibility": 8.2
      },
      {
        "name": "Codeium",
        "logoUrl": "https://...",
        "mentionCount": 65,
        "averageVisibility": 7.1
      }
    ],
    
    "performanceByLLM": [
      {
        "llmName": "GPT-4",
        "provider": "openai",
        "visibility": 8.5,
        "mentionRate": 80.0,
        "responseCount": 50
      }
    ],
    
    "recentActivity": [
      {
        "type": "campaign_completed",
        "campaignName": "cursor brand check",
        "timestamp": "2024-01-01T00:00:00Z",
        "totalRuns": 25
      }
    ]
  }
}
```

**UI Integration:**

**Essential for Dashboard:**
- ✅ `keyMetrics.averageVisibility` - **PRIMARY METRIC** (0-10 scale)
- ✅ `keyMetrics.mentionRate` - **PRIMARY METRIC** (percentage)
- ✅ `keyMetrics.averagePosition` - Ranking position (lower is better)
- ✅ `keyMetrics.groundingRate` - Citation rate (percentage)
- ✅ `sentiment` - Pie chart breakdown
- ✅ `topCompetitors` - Competitive landscape
- ✅ `performanceByLLM` - LLM comparison chart
- ✅ `summary` - Context stats

**Dashboard Layout:**
```
┌──────────────────────────────────────────────┐
│ [Logo] Cursor Dashboard                      │
│ Based on 150 responses · 25 prompts · 3 LLMs│
├──────────────────────────────────────────────┤
│ ┌────────────┐ ┌────────────┐ ┌───────────┐ │
│ │ Visibility │ │ Mention    │ │ Position  │ │
│ │   7.8/10   │ │   72.5%    │ │    #2.3   │ │
│ │     🟢     │ │     🟢     │ │     🟢    │ │
│ └────────────┘ └────────────┘ └───────────┘ │
│                                               │
│ Sentiment Analysis      Top Competitors      │
│ [Pie: 95/40/15]         1. GitHub Copilot    │
│                         2. Codeium           │
│                                               │
│ Performance by LLM                           │
│ GPT-4:  ████████ 8.5 (80% mention)          │
│ Claude: ██████   7.2 (65% mention)          │
└──────────────────────────────────────────────┘
```

**Metric Color Coding:**
- Visibility: 8-10=🟢 5-7.9=🟡 0-4.9=🔴
- Mention Rate: >70%=🟢 40-70%=🟡 <40%=🔴
- Position: 1-3=🟢 4-5=🟡 6+=🔴

---

## 6.2 Source Analytics

**Endpoint:** `POST /api/v1/geo/analytics/sources`

**Purpose:** Analyze which sources/domains AI models cite when mentioning your brand. Critical for understanding your digital footprint in AI training data.

**Request Body:**
```json
{
  "brand": "Cursor",
  "startTime": "2025-01-01T00:00:00Z",
  "endTime": "2025-12-31T23:59:59Z",
  "topN": 2
}
```

**Field Descriptions:**
- `brand` (string, required) - Brand name to analyze
- `startTime` (ISO 8601, optional) - Filter citations from this date
- `endTime` (ISO 8601, optional) - Filter citations until this date
- `topN` (number, optional) - Number of top sources to return (default: 20)

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Source analytics retrieved successfully",
  "data": {
    "brand": "Cursor",
    "logoUrl": "https://...",
    "fallbackLogoUrl": "https://...",
    "period": "Last 30 days",
    "totalSources": 45,
    "totalCitations": 230,
    
    "topSources": [
      {
        "domain": "techcrunch.com",
        "citationCount": 45,
        "mentionRate": 37.5,
        "llmBreakdown": {
          "GPT-4": 20,
          "Claude": 15,
          "Gemini": 10
        },
        "categories": ["comparison", "informational"]
      },
      {
        "domain": "ycombinator.com",
        "citationCount": 38,
        "mentionRate": 31.7,
        "llmBreakdown": {
          "GPT-4": 18,
          "Claude": 12,
          "Gemini": 8
        },
        "categories": ["recommendation"]
      }
    ],
    
    "recommendations": [
      {
        "type": "content_partnership",
        "priority": "high",
        "title": "Strengthen presence on techcrunch.com",
        "description": "This domain is cited 45 times. Consider contributing content.",
        "action": "Reach out for guest posting opportunities",
        "impact": "Could improve citation rate by 15%"
      }
    ]
  }
}
```

**Field Descriptions (Response):**
- `totalSources` - Unique domains citing your brand
- `totalCitations` - Total citation count across all LLM responses
- `topSources[].domain` - The citing website
- `topSources[].citationCount` - How many times this domain was cited
- `topSources[].mentionRate` - Percentage of responses citing this domain
- `topSources[].llmBreakdown` - Which LLMs cite this source
- `recommendations` - Actionable insights for improving source presence

**UI Integration:**

**Essential Fields:**
- ✅ `totalSources` & `totalCitations` - Summary stats
- ✅ `topSources[].domain` - Display as clickable links
- ✅ `topSources[].citationCount` - Bar chart visualization
- ✅ `topSources[].mentionRate` - Percentage display
- ✅ `topSources[].llmBreakdown` - Expandable detail
- ✅ `recommendations` - Action cards with priority badges

**UI Example:**
```
┌─────────────────────────────────────────────────┐
│ Sources Citing Cursor                           │
│ 45 unique sources · 230 total citations         │
├─────────────────────────────────────────────────┤
│ Top Citing Domains:                             │
│                                                  │
│ 1. techcrunch.com          45 citations (37.5%) │
│    ████████████████░░░░░░░░                     │
│    GPT-4: 20 | Claude: 15 | Gemini: 10         │
│                                                  │
│ 2. ycombinator.com         38 citations (31.7%) │
│    ██████████████░░░░░░░░░░                     │
│    GPT-4: 18 | Claude: 12 | Gemini: 8          │
├─────────────────────────────────────────────────┤
│ 💡 Recommendations [HIGH PRIORITY]              │
│                                                  │
│ [🔴 HIGH] Strengthen presence on techcrunch.com │
│ Reach out for guest posting opportunities       │
│ Impact: Could improve citation rate by 15%     │
└─────────────────────────────────────────────────┘
```

**Priority Colors:**
- HIGH = 🔴 Red/Orange
- MEDIUM = 🟡 Yellow  
- LOW = 🔵 Blue

---

## 6.3 Prompt Performance

**Endpoint:** `POST /api/v1/geo/analytics/prompt-performance`

**Purpose:** Analyze which prompts generate the best brand visibility and identify optimization opportunities.

**Request Body:**
```json
{
  "brand": "Cursor",
  "startTime": "2025-01-01T00:00:00Z",
  "endTime": "2025-12-31T23:59:59Z",
  "minResponses": 3
}
```

**Field Descriptions:**
- `brand` (string, required) - Brand name to analyze
- `startTime` (ISO 8601, optional) - Filter from this date
- `endTime` (ISO 8601, optional) - Filter until this date
- `minResponses` (number, optional) - Minimum responses needed per prompt (default: 3)

**Response (200 OK):**
```json
{
  "success": true,
  "message": "Prompt performance retrieved successfully",
  "data": {
    "brand": "Cursor",
    "logoUrl": "https://...",
    "fallbackLogoUrl": "https://...",
    "period": "Last 30 days",
    "totalPromptsAnalyzed": 25,
    "avgEffectiveness": 72.5,
    
    "topPerformers": ["prompt-id-1", "prompt-id-2", "prompt-id-3"],
    "lowPerformers": ["prompt-id-4", "prompt-id-5"],
    
    "prompts": [
      {
        "promptId": "uuid",
        "promptText": "What are the best AI coding assistants?",
        "promptType": "comparison",
        "category": "AI coding",
        
        "avgVisibility": 8.5,
        "avgPosition": 2.3,
        "mentionRate": 85.5,
        "topPositionRate": 67.5,
        "avgSentiment": 0.8,
        
        "totalResponses": 20,
        "brandMentions": 17,
        
        "effectivenessScore": 85.2,
        "effectivenessGrade": "A",
        "status": "high_performing",
        "recommendation": "Keep this prompt. It drives excellent visibility."
      },
      {
        "promptId": "uuid-2",
        "promptText": "How to improve coding productivity?",
        "promptType": "informational",
        "category": "Productivity",
        
        "avgVisibility": 3.2,
        "avgPosition": 8.5,
        "mentionRate": 25.0,
        "topPositionRate": 10.0,
        "avgSentiment": 0.5,
        
        "totalResponses": 20,
        "brandMentions": 5,
        
        "effectivenessScore": 32.5,
        "effectivenessGrade": "D",
        "status": "under_performing",
        "recommendation": "Consider revising or removing this prompt."
      }
    ]
  }
}
```

**Field Descriptions (Response):**
- `avgEffectiveness` - Overall average effectiveness (0-100)
- `topPerformers` / `lowPerformers` - Best/worst prompt IDs
- `prompts[].avgVisibility` - Average visibility score (0-10)
- `prompts[].avgPosition` - Average ranking position (lower = better)
- `prompts[].mentionRate` - % of responses mentioning brand
- `prompts[].topPositionRate` - % of times brand is in top 3
- `prompts[].effectivenessGrade` - Letter grade (A-F)
- `prompts[].status` - `high_performing`, `performing`, `under_performing`
- `prompts[].recommendation` - Actionable insight

**UI Integration:**

**Essential Fields:**
- ✅ `totalPromptsAnalyzed` & `avgEffectiveness` - Summary
- ✅ `prompts[].promptText` - Display prompt
- ✅ `prompts[].avgVisibility` - **KEY METRIC**
- ✅ `prompts[].mentionRate` - **KEY METRIC**  
- ✅ `prompts[].avgPosition` - **KEY METRIC**
- ✅ `prompts[].effectivenessGrade` - Color-coded badge
- ✅ `prompts[].status` - Status icon
- ✅ `prompts[].recommendation` - Tooltip/expandable

**UI Example:**
```
┌─────────────────────────────────────────────────────────────┐
│ Prompt Performance: 25 prompts · Avg: 72.5                 │
├────────────────┬──────┬───────┬─────────┬─────────┬────────┤
│ Prompt         │ Type │ Grade │ Mention │ Position│ Status │
├────────────────┼──────┼───────┼─────────┼─────────┼────────┤
│ What are the...│ Comp │ A 🟢 │  85.5%  │   2.3   │   🟢   │
│ [Expand ▼]     │      │       │         │         │        │
│                                                              │
│ 💡 Keep this prompt. It drives excellent visibility.       │
├────────────────┼──────┼───────┼─────────┼─────────┼────────┤
│ How to improve.│ Info │ D 🔴 │  25.0%  │   8.5   │   🔴   │
│                                                              │
│ ⚠️ Consider revising or removing this prompt.               │
└─────────────────────────────────────────────────────────────┘
```

**Grade Colors:**
- A (85-100): 🟢 Green
- B (70-84): 🟡 Light Green
- C (50-69): 🟡 Yellow
- D (30-49): 🟠 Orange
- F (0-29): 🔴 Red

---

## 6.4 Prompt Time Series

**Endpoint:** `GET /api/v1/geo/analytics/prompt-timeseries`

**Purpose:** Track how a specific prompt's performance changes over time.

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `promptId` | string | Yes | Prompt ID to analyze |
| `startTime` | ISO 8601 | No | Filter from this date |
| `endTime` | ISO 8601 | No | Filter until this date |

**Example Request:**
```
GET /api/v1/geo/analytics/prompt-timeseries?promptId=63307b70-8d00-4aad-ae0e-60b7bd9d55a4
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "promptId": "63307b70-8d00-4aad-ae0e-60b7bd9d55a4",
    "promptText": "What are the best AI coding assistants?",
    "brand": "Cursor",
    
    "timeseries": [
      {
        "date": "2024-01-01",
        "visibility": 7.5,
        "position": 2.3,
        "mentionRate": 75.0,
        "sentiment": 0.8,
        "responseCount": 10
      },
      {
        "date": "2024-01-02",
        "visibility": 8.2,
        "position": 1.8,
        "mentionRate": 82.5,
        "sentiment": 0.85,
        "responseCount": 12
      }
    ],
    
    "summary": {
      "totalDataPoints": 30,
      "trend": "improving",
      "visibilityChange": "+12.5%",
      "positionChange": "-0.8"
    }
  }
}
```

**Field Descriptions:**
- `timeseries[]` - Array of daily performance snapshots
- `summary.trend` - `improving`, `stable`, `declining`
- `summary.visibilityChange` - Change over period (percentage)
- `summary.positionChange` - Position improvement (negative is better)

**UI Integration:**
- ✅ Display as line chart with multiple metrics
- ✅ Show trend indicator (↑ improving, → stable, ↓ declining)
- ✅ Allow toggling between metrics (visibility, position, mention rate)
- ✅ Display summary stats

**UI Example:**
```
┌────────────────────────────────────────────────┐
│ Performance Trend: "What are the best..."     │
│ Status: ↑ Improving (+12.5% visibility)       │
├────────────────────────────────────────────────┤
│ [Line Chart]                                   │
│ 10│                              ●             │
│  8│                    ●     ●                 │
│  6│         ●     ●                            │
│  4│    ●                                       │
│  2│                                            │
│   └────────────────────────────────────────   │
│    Jan 1  Jan 8  Jan 15 Jan 22 Jan 29        │
│                                                │
│ Metrics: [● Visibility] [ Position] [ Mention]│
└────────────────────────────────────────────────┘
```

---

## 6.5 Position Analytics

**Endpoint:** `POST /api/v1/geo/analytics/position`

**Purpose:** Analyze where your brand ranks in AI responses (position 1, 2, 3, etc.).

**Request Body:**
```json
{
  "brand": "Cursor",
  "startTime": "2025-01-01T00:00:00Z",
  "endTime": "2025-12-31T23:59:59Z"
}
```

**Field Descriptions:**
- `brand` (string, required) - Brand name
- `startTime` (ISO 8601, optional) - Filter from date
- `endTime` (ISO 8601, optional) - Filter until date

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "logoUrl": "https://...",
    
    "positionDistribution": {
      "1": 45,
      "2": 38,
      "3": 22,
      "4": 15,
      "5": 8,
      "6+": 5
    },
    
    "metrics": {
      "averagePosition": 2.3,
      "topPositionRate": 68.5,
      "topThreeRate": 79.2,
      "totalMentions": 133
    },
    
    "byLLM": [
      {
        "llmName": "GPT-4",
        "averagePosition": 1.8,
        "topPositionRate": 75.0
      },
      {
        "llmName": "Claude",
        "averagePosition": 2.5,
        "topPositionRate": 60.0
      }
    ],
    
    "byPromptType": [
      {
        "promptType": "comparison",
        "averagePosition": 2.1,
        "topPositionRate": 72.0
      },
      {
        "promptType": "recommendation",
        "averagePosition": 2.8,
        "topPositionRate": 55.0
      }
    ]
  }
}
```

**Field Descriptions:**
- `positionDistribution` - Count of appearances at each ranking position
- `averagePosition` - Mean ranking (lower is better)
- `topPositionRate` - % of times ranked #1
- `topThreeRate` - % of times in top 3
- `byLLM` - Position breakdown by LLM
- `byPromptType` - Position breakdown by prompt category

**UI Integration:**
- ✅ Display position distribution as bar chart
- ✅ Show key metrics prominently
- ✅ Compare performance across LLMs
- ✅ Compare performance across prompt types

**UI Example:**
```
┌────────────────────────────────────────────────┐
│ Position Analysis for Cursor                  │
│ Avg Position: #2.3 · Top 3 Rate: 79.2%       │
├────────────────────────────────────────────────┤
│ Position Distribution:                         │
│ #1: ██████████████████████ 45 (33.8%)        │
│ #2: ████████████████░░░░░░ 38 (28.6%)        │
│ #3: ██████████░░░░░░░░░░░░ 22 (16.5%)        │
│ #4: ██████░░░░░░░░░░░░░░░░ 15 (11.3%)        │
│ #5: ███░░░░░░░░░░░░░░░░░░░  8 (6.0%)         │
│ 6+: ██░░░░░░░░░░░░░░░░░░░░  5 (3.8%)         │
├────────────────────────────────────────────────┤
│ Best Performance:                              │
│ • GPT-4: Avg #1.8 (75% top position)          │
│ • Comparison prompts: Avg #2.1               │
└────────────────────────────────────────────────┘
```

---

## 6.6 Model Analytics

**Endpoint:** `POST /api/v1/geo/analytics/models`

**Purpose:** Compare your brand's performance across different LLM models.

**Request Body:**
```json
{
  "brand": "Cursor",
  "startTime": "2025-01-01T00:00:00Z",
  "endTime": "2025-12-31T23:59:59Z"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    
    "modelPerformance": [
      {
        "llmId": "uuid",
        "llmName": "GPT-4",
        "provider": "openai",
        "model": "gpt-4",
        
        "metrics": {
          "visibility": 8.5,
          "mentionRate": 85.0,
          "averagePosition": 1.8,
          "groundingRate": 52.3,
          "sentiment": 0.82
        },
        
        "stats": {
          "totalResponses": 50,
          "brandMentions": 42,
          "topPositionCount": 28
        },
        
        "grade": "A",
        "status": "excellent"
      },
      {
        "llmId": "uuid-2",
        "llmName": "Claude 3.5 Sonnet",
        "provider": "anthropic",
        "model": "claude-3-5-sonnet-20241022",
        
        "metrics": {
          "visibility": 7.2,
          "mentionRate": 68.5,
          "averagePosition": 2.8,
          "groundingRate": 45.2,
          "sentiment": 0.75
        },
        
        "stats": {
          "totalResponses": 45,
          "brandMentions": 31,
          "topPositionCount": 18
        },
        
        "grade": "B",
        "status": "good"
      }
    ],
    
    "bestPerforming": {
      "byVisibility": "GPT-4",
      "byMentionRate": "GPT-4",
      "byPosition": "GPT-4"
    },
    
    "recommendations": [
      {
        "type": "model_priority",
        "priority": "high",
        "title": "Prioritize GPT-4 optimization",
        "description": "GPT-4 shows 18% better visibility than Claude",
        "action": "Focus GEO efforts on GPT-4 training data sources"
      }
    ]
  }
}
```

**Field Descriptions:**
- `modelPerformance[]` - Performance breakdown per LLM
- `metrics` - Key performance indicators per model
- `stats` - Volume and count metrics
- `grade` - Performance grade (A-F)
- `bestPerforming` - Which model performs best for each metric
- `recommendations` - Strategic insights

**UI Integration:**
- ✅ Display as comparison table
- ✅ Show provider logos
- ✅ Color-code grades
- ✅ Highlight best performing model
- ✅ Display recommendations

**UI Example:**
```
┌──────────────────────────────────────────────────────────┐
│ Model Performance Comparison                              │
├──────────────┬────────────┬─────────┬──────────┬─────────┤
│ Model        │ Visibility │ Mention │ Position │ Grade   │
├──────────────┼────────────┼─────────┼──────────┼─────────┤
│ 🥇 GPT-4     │    8.5     │  85.0%  │   1.8    │  A 🟢  │
│ (OpenAI)     │            │         │          │         │
├──────────────┼────────────┼─────────┼──────────┼─────────┤
│ Claude 3.5   │    7.2     │  68.5%  │   2.8    │  B 🟡  │
│ (Anthropic)  │            │         │          │         │
├──────────────┼────────────┼─────────┼──────────┼─────────┤
│ Gemini Flash │    6.8     │  62.0%  │   3.2    │  C 🟡  │
│ (Google)     │            │         │          │         │
└──────────────┴────────────┴─────────┴──────────┴─────────┘

💡 [HIGH] Prioritize GPT-4 optimization
GPT-4 shows 18% better visibility than Claude
```

---

## 6.7 Competitive Benchmark

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

