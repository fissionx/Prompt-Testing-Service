# GEGO API Integration Guide

> **Complete guide to track and optimize your brand's visibility in AI-generated responses**

## Table of Contents

1. [Overview](#overview)
2. [Getting Started](#getting-started)
3. [Complete Integration Flow](#complete-integration-flow)
4. [Phase 1: Setup](#phase-1-setup)
5. [Phase 2: Brand & Competitors](#phase-2-brand--competitors)
6. [Phase 3: Prompts](#phase-3-prompts)
7. [Phase 4: Execution](#phase-4-execution)
8. [Phase 5: Analytics](#phase-5-analytics)
9. [API Reference](#api-reference)
10. [Error Handling](#error-handling)

---

## Overview

### What is GEGO?

GEGO (Generative Engine Optimization) helps brands track and improve their visibility in AI-powered search responses from ChatGPT, Claude, Gemini, and other LLMs.

### How It Works

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           GEGO Platform Flow                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌────────┐│
│   │  Setup   │───▶│  Brand & │───▶│ Prompts  │───▶│ Execute  │───▶│Analytics││
│   │  LLMs    │    │Competitors│   │          │    │ Campaign │    │        ││
│   └──────────┘    └──────────┘    └──────────┘    └──────────┘    └────────┘│
│                                                                              │
│   Configure      Define your      Create or      Run against     View your  │
│   AI models      brand and        generate       multiple        visibility │
│   to test        competitors      test prompts   AI models       insights   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Base URL

```
http://localhost:8080/api/v1
```

### Response Format

All APIs return a consistent JSON format:

```json
{
  "success": true,          // true for success, false for errors
  "data": { ... },          // Response payload
  "message": "Description", // Human-readable message
  "error": "..."           // Error details (only when success=false)
}
```

---

## Getting Started

### Quick Start (5 Minutes)

```bash
# 1. Check API is running
curl http://localhost:8080/api/v1/health

# 2. Add an LLM
curl -X POST http://localhost:8080/api/v1/llms \
  -H "Content-Type: application/json" \
  -d '{"name":"GPT-4","provider":"openai","model":"gpt-4","apiKey":"sk-...","enabled":true}'

# 3. Suggest competitors for your brand
curl "http://localhost:8080/api/v1/geo/competitors/suggest?brand=YourBrand&website=https://yourbrand.com"

# 4. Generate prompts
curl -X POST http://localhost:8080/api/v1/geo/prompts/generate \
  -H "Content-Type: application/json" \
  -d '{"brand":"YourBrand","website":"https://yourbrand.com","count":10}'

# 5. Execute campaign
curl -X POST http://localhost:8080/api/v1/geo/execute/bulk \
  -H "Content-Type: application/json" \
  -d '{"campaignName":"First Test","brand":"YourBrand","promptIds":["..."],"llmIds":["..."]}'

# 6. View results
curl -X POST http://localhost:8080/api/v1/geo/insights \
  -H "Content-Type: application/json" \
  -d '{"brand":"YourBrand"}'
```

---

## Complete Integration Flow

### Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                                                                                  │
│  PHASE 1: SETUP                                                                  │
│  ─────────────────                                                              │
│  [POST /api/v1/llms] ──▶ Add LLM configurations (OpenAI, Anthropic, Google)     │
│                                                                                  │
│                              ▼                                                   │
│                                                                                  │
│  PHASE 2: BRAND & COMPETITORS                                                    │
│  ────────────────────────────                                                   │
│  [GET /geo/competitors/suggest] ──▶ Get AI-suggested competitors                │
│  [POST /geo/competitors]        ──▶ Save your competitor list                   │
│                                                                                  │
│                              ▼                                                   │
│                                                                                  │
│  PHASE 3: PROMPTS                                                               │
│  ────────────────                                                               │
│  [POST /geo/prompts/generate] ──▶ Generate test prompts for your brand          │
│                                                                                  │
│                              ▼                                                   │
│                                                                                  │
│  PHASE 4: EXECUTION                                                             │
│  ──────────────────                                                             │
│  [POST /geo/execute/bulk] ──▶ Run prompts against all LLMs                      │
│                                                                                  │
│                              ▼                                                   │
│                                                                                  │
│  PHASE 5: ANALYTICS                                                             │
│  ──────────────────                                                             │
│  [POST /geo/insights]              ──▶ Dashboard overview                       │
│  [POST /geo/analytics/competitive] ──▶ Competitor comparison                    │
│  [POST /geo/analytics/sources]     ──▶ Citation source analysis                 │
│  [POST /geo/analytics/prompt-performance] ──▶ Prompt effectiveness              │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Setup

### 1.1 Configure LLMs

Before running any analysis, you need to configure at least one LLM provider.

#### Add LLM Configuration

```
POST /api/v1/llms
```

**Request:**
```json
{
  "name": "GPT-4o",                // Display name (required)
  "provider": "openai",            // Provider: openai, anthropic, google, perplexity, ollama (required)
  "model": "gpt-4o",              // Model ID (required)
  "apiKey": "sk-...",             // API key (required for cloud providers)
  "baseUrl": "",                   // Custom base URL (optional, for self-hosted)
  "enabled": true                  // Enable for testing (required)
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "llm_abc123",           // Use this ID for execution
    "name": "GPT-4o",
    "provider": "openai",
    "model": "gpt-4o",
    "enabled": true,
    "createdAt": "2024-01-01T00:00:00Z"
  }
}
```

**Supported Providers:**

| Provider | Models | Notes |
|----------|--------|-------|
| `openai` | gpt-4o, gpt-4-turbo, gpt-3.5-turbo | Requires API key |
| `anthropic` | claude-3-opus, claude-3-sonnet | Requires API key |
| `google` | gemini-pro, gemini-1.5-pro | Requires API key, best for grounded search |
| `perplexity` | pplx-7b, pplx-70b | Requires API key |
| `ollama` | llama2, mistral, codellama | Self-hosted, set baseUrl |

#### List Configured LLMs

```
GET /api/v1/llms
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "llm_abc123",
      "name": "GPT-4o",
      "provider": "openai",
      "model": "gpt-4o",
      "enabled": true
    },
    {
      "id": "llm_def456",
      "name": "Claude 3",
      "provider": "anthropic",
      "model": "claude-3-sonnet-20240229",
      "enabled": true
    }
  ]
}
```

---

## Phase 2: Brand & Competitors

### 2.1 Get Competitor Suggestions

Use AI to suggest competitors based on your brand and website.

```
GET /api/v1/geo/competitors/suggest
```

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `brand` | string | Yes | Your brand/company name |
| `website` | string | No | Your website URL (recommended for better suggestions) |
| `description` | string | No | Brief description of your product/service |
| `category` | string | No | Industry category |
| `forceRefresh` | boolean | No | Force regeneration (default: false) |

**Example:**
```bash
curl "http://localhost:8080/api/v1/geo/competitors/suggest?brand=Cursor&website=https://cursor.sh"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "competitors": [
      "VS Code",
      "GitHub Copilot",
      "Tabnine",
      "Codeium",
      "JetBrains IDEs"
    ],
    "source": "llm",              // "llm" = freshly generated, "cached" = from cache
    "message": "Found 5 competitors for Cursor"
  }
}
```

**Key Points:**
- ✅ First call uses LLM to generate suggestions
- ✅ Subsequent calls return cached results (no LLM cost)
- ✅ Use `forceRefresh=true` to regenerate
- ✅ Providing website improves suggestion quality

### 2.2 Save Your Competitors

After reviewing suggestions, save your selected competitors.

```
POST /api/v1/geo/competitors
```

**Request:**
```json
{
  "brand": "Cursor",                    // Your brand name (required)
  "competitors": [                       // Your selected competitors (required)
    "VS Code",
    "GitHub Copilot",
    "Tabnine"
  ],
  "source": "suggested"                 // "suggested", "custom", or "mixed" (optional)
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

**Key Points:**
- ✅ These competitors are used automatically in all analytics
- ✅ You can mix suggestions with custom competitors
- ✅ Update anytime by calling this endpoint again

### 2.3 Get Saved Competitors

```
GET /api/v1/geo/competitors?brand=Cursor
```

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "competitors": ["VS Code", "GitHub Copilot", "Tabnine"],
    "suggestedList": ["VS Code", "GitHub Copilot", "Tabnine", "Codeium", "JetBrains IDEs"],
    "source": "suggested",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

---

## Phase 3: Prompts

### 3.1 Generate Prompts

Generate test prompts that represent what users ask AI about your industry.

```
POST /api/v1/geo/prompts/generate
```

**Request:**
```json
{
  "brand": "Cursor",                    // Your brand name (required)
  "website": "https://cursor.sh",       // Website for context (recommended)
  "category": "Developer Tools",        // Industry category (optional)
  "domain": "technology",               // Business domain (optional)
  "description": "AI-powered code editor", // Product description (optional)
  "count": 20                           // Number of prompts to generate (optional, default: 20)
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "category": "Developer Tools",
    "domain": "technology",
    "existingPrompts": 5,               // Reused from library
    "generatedPrompts": 15,             // Newly generated
    "promptsByType": {
      "comparison": [
        {
          "id": "prompt_abc123",
          "template": "What are the best AI code editors?",
          "promptType": "comparison",
          "reused": false
        }
      ],
      "recommendation": [...],
      "informational": [...]
    },
    "typeCounts": {
      "comparison": 8,
      "recommendation": 7,
      "informational": 5
    }
  }
}
```

**Prompt Types Explained:**

| Type | Description | Example |
|------|-------------|---------|
| `comparison` | Comparing options | "What are the best AI code editors?" |
| `recommendation` | Asking for suggestions | "Which IDE should I use for Python?" |
| `informational` | Learning about a topic | "How does AI code completion work?" |
| `top_best` | Top/best lists | "Top 10 developer tools in 2024" |
| `how_to` | How-to questions | "How to set up an AI coding assistant?" |

**Key Points:**
- ✅ Prompts are cached and reused across similar brands
- ✅ Website scraping enriches prompt relevance
- ✅ Generated prompts simulate real user queries

---

## Phase 4: Execution

### 4.1 Execute Campaign

Run your prompts against multiple LLMs to test brand visibility.

```
POST /api/v1/geo/execute/bulk
```

**Request:**
```json
{
  "campaignName": "Q1 Visibility Test",  // Campaign name (required)
  "brand": "Cursor",                      // Brand to analyze (required)
  "promptIds": [                          // Prompt IDs from Phase 3 (required)
    "prompt_abc123",
    "prompt_def456"
  ],
  "llmIds": [                            // LLM IDs from Phase 1 (required)
    "llm_abc123",
    "llm_def456"
  ],
  "temperature": 0.7,                    // LLM temperature 0.0-2.0 (optional)
  "totalRuns": 1                         // Runs per prompt (optional, default: 1)
}
```

**Response (202 Accepted):**
```json
{
  "success": true,
  "data": {
    "campaignId": "camp_xyz789",
    "campaignName": "Q1 Visibility Test",
    "brand": "Cursor",
    "totalRuns": 40,                     // 20 prompts × 2 LLMs × 1 run
    "status": "running",
    "startedAt": "2024-01-01T00:00:00Z",
    "message": "Campaign started successfully. Execution running in background."
  }
}
```

**Key Points:**
- ✅ Execution runs asynchronously (non-blocking)
- ✅ `totalRuns` = prompts × LLMs × runs
- ✅ Results are analyzed and stored automatically
- ✅ Check analytics endpoints for results

---

## Phase 5: Analytics

### 5.1 Dashboard Overview

Get a comprehensive overview of your brand's AI visibility.

```
POST /api/v1/geo/insights
```

**Request:**
```json
{
  "brand": "Cursor",                     // Brand name (required)
  "startTime": "2024-01-01T00:00:00Z",  // Date filter (optional)
  "endTime": "2024-03-31T23:59:59Z"     // Date filter (optional)
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "logoUrl": "https://logo.clearbit.com/cursor.sh",
    "fallbackLogoUrl": "https://ui-avatars.com/api/?name=Cursor",
    
    // KEY METRICS
    "averageVisibility": 7.5,           // 0-10 scale, how visible is your brand
    "mentionRate": 65.5,                // % of responses mentioning your brand
    "groundingRate": 45.2,              // % of responses citing your brand in sources
    "totalResponses": 120,              // Total AI responses analyzed
    
    // SENTIMENT
    "sentimentBreakdown": {
      "positive": 75,                   // Positive mentions
      "neutral": 30,                    // Neutral mentions
      "negative": 5                     // Negative mentions
    },
    
    // TOP COMPETITORS (from responses)
    "topCompetitors": [
      {
        "name": "VS Code",
        "logoUrl": "https://...",
        "mentionCount": 45,
        "visibilityAvg": 8.2
      }
    ],
    
    // PERFORMANCE BY LLM
    "performanceByLlm": [
      {
        "llmName": "GPT-4o",
        "llmProvider": "openai",
        "visibility": 8.5,
        "mentionRate": 75.5,
        "responseCount": 40
      }
    ],
    
    // PERFORMANCE BY CATEGORY
    "performanceByCategory": [
      {
        "category": "Comparison",
        "visibility": 7.8,
        "mentionRate": 68.5,
        "responseCount": 25
      }
    ]
  }
}
```

**Metric Interpretation:**

| Metric | Range | Good | Average | Poor |
|--------|-------|------|---------|------|
| `averageVisibility` | 0-10 | 7+ | 4-7 | <4 |
| `mentionRate` | 0-100% | 60%+ | 30-60% | <30% |
| `groundingRate` | 0-100% | 40%+ | 20-40% | <20% |

### 5.2 Competitive Benchmark

Compare your brand against competitors.

```
POST /api/v1/geo/analytics/competitive
```

**Request:**
```json
{
  "mainBrand": "Cursor",                // Your brand (required)
  "competitors": [],                    // Empty = use saved competitors
  "startTime": "2024-01-01T00:00:00Z", // Optional
  "endTime": "2024-03-31T23:59:59Z"    // Optional
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    // YOUR BRAND
    "mainBrand": {
      "brand": "Cursor",
      "logoUrl": "https://...",
      "visibility": 7.5,              // Your visibility score
      "mentionRate": 65.5,            // Your mention rate
      "averagePosition": 2.8,         // Your avg ranking position
      "marketSharePct": 35.5,         // Your share of voice
      "responseCount": 120
    },
    
    // COMPETITORS
    "competitors": [
      {
        "brand": "VS Code",
        "visibility": 8.2,
        "mentionRate": 72.3,
        "averagePosition": 2.1,
        "marketSharePct": 42.5,
        "responseCount": 145
      }
    ],
    
    // RANKING
    "marketLeader": "VS Code",        // Brand with highest visibility
    "yourRank": 2,                    // Your position
    "totalBrands": 4,                 // Total brands compared
    
    // PROMPT-LEVEL BREAKDOWN
    "promptBreakdown": [
      {
        "promptId": "prompt_abc123",
        "promptText": "What are the best AI code editors?",
        "promptType": "comparison",
        "mainBrandResult": {
          "mentioned": true,
          "visibilityScore": 8,
          "position": 2,
          "sentiment": "positive"
        },
        "competitorsMentioned": [
          { "brand": "VS Code", "mentioned": true },
          { "brand": "GitHub Copilot", "mentioned": true }
        ],
        "winner": "VS Code",
        "totalBrandsMentioned": 3
      }
    ],
    
    // RECOMMENDATIONS
    "recommendations": [
      {
        "type": "visibility_gap",
        "priority": "high",
        "title": "Close the gap with VS Code",
        "description": "VS Code has 6.8% higher mention rate",
        "action": "Focus on comparison-type prompts where they excel"
      }
    ]
  }
}
```

### 5.3 Source Analytics

See which sources AI models cite when mentioning your brand.

```
POST /api/v1/geo/analytics/sources
```

**Request:**
```json
{
  "brand": "Cursor",
  "topN": 20                           // Number of sources to return
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "totalSources": 45,
    "totalCitations": 230,
    
    "topSources": [
      {
        "domain": "cursor.sh",
        "citationCount": 45,
        "mentionRate": 37.5,
        "llmBreakdown": {
          "GPT-4o": 20,
          "Claude": 15,
          "Gemini": 10
        }
      }
    ],
    
    "recommendations": [
      {
        "type": "content_partnership",
        "priority": "high",
        "title": "Strengthen presence on dev.to",
        "description": "High citation domain, low brand mention",
        "action": "Publish content or get featured"
      }
    ]
  }
}
```

### 5.4 Prompt Performance

Analyze which prompts generate the best visibility.

```
POST /api/v1/geo/analytics/prompt-performance
```

**Request:**
```json
{
  "brand": "Cursor",
  "minResponses": 3                    // Minimum responses per prompt
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "totalPromptsAnalyzed": 25,
    "avgEffectiveness": 72.5,
    
    "prompts": [
      {
        "promptId": "prompt_abc123",
        "promptText": "What are the best AI code editors?",
        "promptType": "comparison",
        
        // Performance Metrics
        "avgVisibility": 8.5,
        "mentionRate": 85.5,
        "avgPosition": 2.3,
        "totalResponses": 20,
        
        // Effectiveness
        "effectivenessScore": 85.2,
        "effectivenessGrade": "A",      // A, B, C, D, F
        "status": "high_performing",
        "recommendation": "Keep this prompt. Excellent visibility."
      }
    ],
    
    "topPerformers": ["prompt_abc123", "prompt_def456"],
    "lowPerformers": ["prompt_xyz789"]
  }
}
```

**Effectiveness Grades:**

| Grade | Score | Status |
|-------|-------|--------|
| A | 85-100 | high_performing |
| B | 70-84 | performing |
| C | 50-69 | average |
| D | 30-49 | under_performing |
| F | 0-29 | failing |

---

## API Reference

### Quick Reference Table

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/llms` | GET | List configured LLMs |
| `/api/v1/llms` | POST | Add new LLM |
| `/api/v1/geo/competitors/suggest` | GET | Get AI-suggested competitors |
| `/api/v1/geo/competitors` | POST | Save competitor list |
| `/api/v1/geo/competitors` | GET | Get saved competitors |
| `/api/v1/geo/prompts/generate` | POST | Generate test prompts |
| `/api/v1/geo/execute/bulk` | POST | Execute campaign |
| `/api/v1/geo/insights` | POST | Dashboard overview |
| `/api/v1/geo/analytics/competitive` | POST | Competitor comparison |
| `/api/v1/geo/analytics/sources` | POST | Citation sources |
| `/api/v1/geo/analytics/prompt-performance` | POST | Prompt effectiveness |

---

## Error Handling

### Error Response Format

```json
{
  "success": false,
  "error": "Detailed error message"
}
```

### Common HTTP Status Codes

| Code | Meaning | Action |
|------|---------|--------|
| 200 | Success | Response contains data |
| 202 | Accepted | Async operation started |
| 400 | Bad Request | Check request parameters |
| 404 | Not Found | Resource doesn't exist |
| 500 | Server Error | Retry or contact support |

### Common Errors

**Missing brand parameter:**
```json
{
  "success": false,
  "error": "Brand parameter is required"
}
```

**No LLMs configured:**
```json
{
  "success": false,
  "error": "No LLM providers available"
}
```

**No responses found:**
```json
{
  "success": false,
  "error": "No responses found for brand Cursor"
}
```

---

## Best Practices

### 1. Setup Checklist

- [ ] Configure at least 2-3 different LLMs for diverse results
- [ ] Use your actual brand website for better context
- [ ] Save your competitor list before running campaigns

### 2. Prompt Generation

- [ ] Generate 15-25 prompts for comprehensive coverage
- [ ] Include different prompt types (comparison, recommendation, informational)
- [ ] Review and curate generated prompts

### 3. Campaign Execution

- [ ] Run campaigns regularly (weekly/monthly) to track trends
- [ ] Test across multiple LLMs to understand platform differences
- [ ] Use consistent prompt sets for meaningful comparisons

### 4. Analysis

- [ ] Compare performance across LLMs to identify strengths
- [ ] Focus on high-performing prompts for content strategy
- [ ] Act on recommendations to improve visibility

---

## Support

- **GitHub:** https://github.com/fissionx/gego
- **Documentation:** See `/docs` folder for detailed guides

---

**Version:** 2.0  
**Last Updated:** December 2024  
**API Version:** v1

