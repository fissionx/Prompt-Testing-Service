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
- Generates 10-50 unique questions using AI
- Reuses existing prompts from the same category/domain
- Stores all prompts for future reuse

**Endpoint:** `POST /api/v1/geo/prompts/generate`

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

