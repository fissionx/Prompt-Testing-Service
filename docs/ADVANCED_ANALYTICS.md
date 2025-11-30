# Advanced Analytics Features

This document describes the newly implemented advanced analytics features that bring gego to feature parity with Peec AI.

## 🎯 Overview

The following features have been implemented to enhance gego's GEO analytics capabilities:

1. **Position/Ranking Tracking** - Track where your brand appears in list-based responses
2. **Source Citation Analytics** - Analyze which sources AI models cite most frequently
3. **Actionable Recommendations** - AI-generated insights on how to improve visibility
4. **Competitive Benchmarking** - Side-by-side comparison with competitors including market share
5. **Multi-Region Support** - Track performance across different regions and languages
6. **Time-Series Data** - Track trends over time (week, month, quarter)

---

## 📊 New Data Model Fields

### Response Model Enhancements

The `Response` model now includes:

```go
// Position/Ranking tracking
BrandPosition      int      // 1=first, 2=second, 0=not in list
TotalBrandsListed  int      // Total brands mentioned in response

// Enhanced source analytics
GroundingDomains   []string // Extracted domains (e.g., "g2.com", "reddit.com")

// Time-series support
Week               string   // "2025-W48"
Month              string   // "2025-11"
Quarter            string   // "2025-Q4"

// Regional/Language support
Region             string   // "US", "UK", "DE"
Language           string   // "en", "es", "de"
```

---

## 🚀 New API Endpoints

### 1. Source Analytics

**Endpoint:** `POST /api/v1/geo/analytics/sources`

Analyzes which citation sources (domains) AI models use most frequently and provides actionable recommendations.

**Request:**
```json
{
  "brand": "YourBrand",
  "start_time": "2025-11-01T00:00:00Z",
  "end_time": "2025-11-30T23:59:59Z",
  "top_n": 20
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "YourBrand",
    "period": "2025-11-01 to 2025-11-30",
    "top_sources": [
      {
        "domain": "g2.com",
        "citation_count": 45,
        "mention_rate": 75.0,
        "llm_breakdown": {
          "Gemini Flash": 25,
          "GPT-4": 20
        },
        "categories": ["review_site"]
      },
      {
        "domain": "reddit.com",
        "citation_count": 32,
        "mention_rate": 53.3,
        "llm_breakdown": {
          "Gemini Flash": 18,
          "GPT-4": 14
        },
        "categories": ["social_media"]
      }
    ],
    "recommendations": [
      {
        "type": "source_opportunity",
        "priority": "high",
        "title": "Optimize presence on g2.com",
        "description": "The review site g2.com is frequently cited (45 times, 75.0% of responses). This is a high-value source for AI visibility.",
        "action": "Create or optimize your profile on g2.com. Encourage customers to leave reviews. Ensure your listing is complete and up-to-date.",
        "impact": "high"
      }
    ],
    "total_sources": 12,
    "total_citations": 156
  }
}
```

**Key Insights:**
- Identifies which sources (domains) drive AI visibility
- Provides specific, actionable recommendations
- Categorizes sources (review sites, social media, news, etc.)

---

### 2. Competitive Benchmarking

**Endpoint:** `POST /api/v1/geo/analytics/competitive`

Performs side-by-side competitive analysis with market share calculations.

**Request:**
```json
{
  "main_brand": "YourBrand",
  "competitors": ["Competitor A", "Competitor B", "Competitor C"],
  "prompt_ids": ["prompt-1", "prompt-2"],
  "llm_ids": ["llm-1", "llm-2"],
  "start_time": "2025-11-01T00:00:00Z",
  "end_time": "2025-11-30T23:59:59Z",
  "region": "US"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "main_brand": {
      "brand": "YourBrand",
      "visibility": 4.2,
      "mention_rate": 35.5,
      "grounding_rate": 22.1,
      "average_position": 3.5,
      "top_position_rate": 28.3,
      "sentiment_score": 0.75,
      "response_count": 120,
      "market_share_pct": 18.5
    },
    "competitors": [
      {
        "brand": "Competitor A",
        "visibility": 6.8,
        "mention_rate": 58.2,
        "grounding_rate": 45.3,
        "average_position": 1.8,
        "top_position_rate": 62.5,
        "sentiment_score": 0.82,
        "response_count": 120,
        "market_share_pct": 32.1
      },
      {
        "brand": "Competitor B",
        "visibility": 5.1,
        "mention_rate": 42.8,
        "grounding_rate": 31.2,
        "average_position": 2.9,
        "top_position_rate": 38.7,
        "sentiment_score": 0.68,
        "response_count": 120,
        "market_share_pct": 24.3
      }
    ],
    "market_leader": "Competitor A",
    "your_rank": 3,
    "total_brands": 4,
    "recommendations": [
      {
        "type": "competitor_threat",
        "priority": "critical",
        "title": "Significant visibility gap with Competitor A",
        "description": "Your visibility (4.2) is 2.6 points behind market leader Competitor A (6.8). This represents a 38.2% gap.",
        "action": "Analyze what content and sources drive competitor visibility. Focus on getting mentioned in their key citation sources.",
        "impact": "critical"
      },
      {
        "type": "position_improvement",
        "priority": "high",
        "title": "Improve average position in lists",
        "description": "Your average position is 3.5 (where 1 is best). You're often mentioned but not at the top.",
        "action": "Focus on being the 'best' or 'top choice' in content. Improve review scores and ratings. Get more positive testimonials.",
        "impact": "high"
      }
    ],
    "analyzed_at": "2025-11-29T12:00:00Z"
  }
}
```

**Key Metrics:**
- **Market Share %** - Your share of total visibility in the market
- **Average Position** - Where you rank in list-based responses (1 = best)
- **Top Position Rate** - % of times you appear in positions 1-3
- **Sentiment Score** - -1 (negative) to +1 (positive)

---

### 3. Position Analytics

**Endpoint:** `POST /api/v1/geo/analytics/position`

Analyzes brand positioning and ranking performance.

**Request:**
```json
{
  "brand": "YourBrand",
  "start_time": "2025-11-01T00:00:00Z",
  "end_time": "2025-11-30T23:59:59Z"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "YourBrand",
    "average_position": 3.2,
    "top_position_rate": 28.5,
    "position_breakdown": {
      "position_1": 12,
      "position_2": 8,
      "position_3": 14,
      "position_4": 9,
      "position_5": 5
    },
    "by_prompt_type": {
      "what": 3.8,
      "how": 2.9,
      "comparison": 2.1,
      "top_best": 3.5,
      "brand": 1.5
    },
    "by_llm": {
      "Gemini Flash": 3.1,
      "GPT-4": 3.4,
      "Claude": 2.8
    },
    "total_mentions": 48
  }
}
```

**Key Insights:**
- See which prompt types give you the best positions
- Identify which LLMs rank you higher
- Track improvement in positions over time

---

## 🔧 Enhanced Execute Endpoint

The existing `/api/v1/execute` endpoint now supports:

**New Request Fields:**
```json
{
  "prompt": "What are the best AI SEO tools?",
  "llm_id": "your-llm-id",
  "brand": "YourBrand",
  "temperature": 0.7,
  "region": "US",
  "language": "en"
}
```

**Enhanced Response:**
- Automatically extracts brand position in lists
- Parses citation source domains
- Adds time-series fields (week, month, quarter)
- Stores region/language for filtering

---

## 🎓 Usage Examples

### Example 1: Complete GEO Campaign with Advanced Analytics

```bash
# 1. Generate prompts
curl -X POST http://localhost:8989/api/v1/geo/prompts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "MyStartup",
    "website": "https://mystartup.com",
    "count": 50
  }'

# 2. Run bulk campaign
curl -X POST http://localhost:8989/api/v1/geo/execute/bulk \
  -H "Content-Type: application/json" \
  -d '{
    "campaign_name": "Q4 2025 Analysis",
    "brand": "MyStartup",
    "prompt_ids": ["prompt-1", "prompt-2", ...],
    "llm_ids": ["llm-1", "llm-2"],
    "temperature": 0.7
  }'

# 3. Get source analytics
curl -X POST http://localhost:8989/api/v1/geo/analytics/sources \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "MyStartup",
    "top_n": 20
  }'

# 4. Get competitive benchmark
curl -X POST http://localhost:8989/api/v1/geo/analytics/competitive \
  -H "Content-Type: application/json" \
  -d '{
    "main_brand": "MyStartup",
    "competitors": ["Competitor1", "Competitor2"]
  }'

# 5. Get position analytics
curl -X POST http://localhost:8989/api/v1/geo/analytics/position \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "MyStartup"
  }'
```

### Example 2: Multi-Region Tracking

```bash
# Execute prompts for different regions
for REGION in US UK DE FR; do
  curl -X POST http://localhost:8989/api/v1/execute \
    -H "Content-Type: application/json" \
    -d "{
      \"prompt\": \"Best AI SEO tools for 2025\",
      \"llm_id\": \"your-llm-id\",
      \"brand\": \"MyStartup\",
      \"region\": \"$REGION\",
      \"language\": \"en\"
    }"
done

# Then filter analytics by region
curl -X POST http://localhost:8989/api/v1/geo/analytics/competitive \
  -H "Content-Type: application/json" \
  -d '{
    "main_brand": "MyStartup",
    "competitors": ["Competitor1"],
    "region": "US"
  }'
```

---

## 🧠 Recommendations Engine

The recommendations engine analyzes your GEO data and provides specific, actionable insights:

### Recommendation Types

1. **source_opportunity** - High-value citation sources you should target
2. **content_opportunity** - Content gaps you should fill
3. **pr_opportunity** - PR/media opportunities
4. **competitor_threat** - Competitive threats requiring attention
5. **position_improvement** - Ways to improve your ranking
6. **sentiment_warning** - Sentiment issues to address
7. **diversity_warning** - Over-reliance on single sources

### Priority Levels

- **critical** - Immediate action required
- **high** - Important, address soon
- **medium** - Valuable but not urgent
- **low** - Nice to have

### Example Recommendations

```json
{
  "type": "source_opportunity",
  "priority": "high",
  "title": "Optimize presence on g2.com",
  "description": "g2.com is cited 45 times (75% of responses)",
  "action": "Create or optimize your G2 profile. Get customer reviews.",
  "impact": "high"
}
```

---

## 📈 Time-Series Analysis

All responses now include time-series fields for trend analysis:

### Fields Added
- `week` - ISO week (e.g., "2025-W48")
- `month` - Year-month (e.g., "2025-11")
- `quarter` - Year-quarter (e.g., "2025-Q4")

### Future Enhancement
Build aggregation queries to track:
- Visibility trends over weeks/months
- Position improvements over time
- Market share changes quarter-over-quarter

---

## 🌍 Multi-Region Support

### Supported Fields
- `region` - Country/region code (e.g., "US", "UK", "DE", "FR", "JP")
- `language` - Language code (e.g., "en", "es", "de", "fr", "ja")

### Use Cases
- Track regional performance differences
- Optimize for specific markets
- Compare brand visibility across countries

---

## 🔍 Source Categories

Sources are automatically categorized:

- **review_site** - G2, Capterra, Trustpilot, Yelp, TripAdvisor
- **social_media** - Reddit, Twitter, LinkedIn, Facebook, YouTube
- **news** - NYTimes, WSJ, BBC, CNN, Reuters, Bloomberg
- **publication** - Forbes, Inc, Entrepreneur, Wired
- **company_website** - Direct company/brand websites

---

## 🎯 Key Differences from Peec AI

### What Gego Has (Advantages)
✅ **Open Source** - Full control, no vendor lock-in
✅ **Self-Hosted** - Data privacy, no usage limits
✅ **Multi-LLM Support** - Works with 5+ LLM providers
✅ **API-First** - Build your own dashboards
✅ **Pluggable Architecture** - Easy to extend

### What Peec AI Has (Future Enhancements)
- Web dashboard (can be built separately)
- CSV export (easy to add)
- Alerts/webhooks (can be added)
- Prompt search volume data (requires external data)

---

## 📚 Next Steps

### For Development
1. Build a web dashboard (React/Next.js)
2. Add CSV export functionality
3. Implement webhook notifications
4. Add trend visualization endpoints

### For Users
1. Start tracking your brand with advanced analytics
2. Monitor competitor performance
3. Act on recommendations
4. Track improvements over time

---

## 🐛 Troubleshooting

### No Position Data?
- Position extraction works best with list-based responses
- Ensure your prompts ask for lists/comparisons
- Use prompt types: `comparison`, `top_best`

### No Source Data?
- Source citations require Google Gemini with grounding enabled
- Other LLMs don't provide citation sources
- Consider using Gemini for citation tracking

### Low Recommendations?
- Recommendations are generated based on data volume
- Run more campaigns to get better insights
- Ensure at least 20-30 responses per brand

---

## 📞 Support

For issues or questions:
- GitHub Issues: https://github.com/fissionx/gego/issues
- Email: jonathan@blocs.fr

---

**Made with ❤️ for the open-source community**

