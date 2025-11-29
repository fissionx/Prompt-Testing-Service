# 🧪 Complete End-to-End API Testing Guide
## Test Brand: Amazon.com

This guide provides a complete workflow to test all GEO APIs using Amazon.com as the test brand.

---

## 🚀 Prerequisites

```bash
# 1. Start the API server
cd /Users/senyarav/workspace/opensource/gego
./gego api --port 8989

# 2. Verify server is running
curl http://localhost:8989/api/v1/health
# Expected: {"status":"ok"}
```

---

## 📋 Complete Testing Workflow

### **Phase 1: Setup LLMs**

#### Step 1.1: List Available LLMs
```bash
curl http://localhost:8989/api/v1/llms
```

**Expected Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "llm-id-1",
      "name": "Gemini Flash",
      "provider": "google",
      "model": "models/gemini-1.5-flash",
      "enabled": true
    }
  ]
}
```

**If no LLMs exist, add one:**
```bash
# Add Google Gemini (required for grounding sources)
curl -X POST http://localhost:8989/api/v1/llms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Gemini Flash",
    "provider": "google",
    "model": "models/gemini-1.5-flash",
    "api_key": "YOUR_GOOGLE_API_KEY",
    "enabled": true
  }'
```

**Save the LLM ID for later use:**
```bash
LLM_ID="llm-id-from-response"
```

---

### **Phase 2: Generate Smart Prompts**

#### Step 2.1: Generate Prompts with Website Scraping
```bash
curl -X POST http://localhost:8989/api/v1/geo/prompts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "Amazon",
    "website": "https://www.amazon.com",
    "count": 20
  }' | jq '.'
```

**What This Does:**
- Scrapes amazon.com to understand what they do
- AI derives domain/category automatically
- Generates 20 relevant prompts
- Stores prompts in database

**Expected Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Amazon",
    "domain": "technology",
    "category": "ecommerce",
    "prompts": [
      {
        "id": "prompt-1",
        "template": "What are the best online shopping platforms?",
        "prompt_type": "top_best",
        "category": "ecommerce",
        "reused": false
      },
      {
        "id": "prompt-2",
        "template": "How does Amazon Prime compare to other services?",
        "prompt_type": "comparison",
        "category": "ecommerce",
        "reused": false
      }
      // ... 18 more prompts
    ],
    "prompts_by_type": {
      "what": [...],
      "how": [...],
      "comparison": [...],
      "top_best": [...],
      "brand": [...]
    },
    "existing_prompts": 0,
    "generated_prompts": 20,
    "type_counts": {
      "what": 4,
      "how": 4,
      "comparison": 4,
      "top_best": 4,
      "brand": 4
    }
  }
}
```

**Save Prompt IDs:**
```bash
# Extract all prompt IDs and save
PROMPT_IDS='["prompt-1", "prompt-2", "prompt-3", "prompt-4", "prompt-5"]'
```

**✅ What to Verify:**
- Domain should be "technology" or "retail" or "ecommerce"
- Category should be relevant to Amazon
- Should have 20 prompts
- Prompts should be diverse (5 types)
- Prompts should be generic (not mention "Amazon" in template)

---

### **Phase 3: Execute Single Prompt (Test)**

#### Step 3.1: Test Execute with GEO Analysis
```bash
curl -X POST http://localhost:8989/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "What are the best online shopping platforms for 2025?",
    "llm_id": "'"$LLM_ID"'",
    "brand": "Amazon",
    "temperature": 0.7,
    "region": "US",
    "language": "en"
  }' | jq '.'
```

**Expected Response:**
```json
{
  "success": true,
  "data": {
    "response_id": "response-1",
    "prompt": "What are the best online shopping platforms for 2025?",
    "brand": "Amazon",
    "response": "For online shopping in 2025, here are the top platforms...",
    "geo_analysis": {
      "visibility_score": 8,
      "brand_mentioned": true,
      "in_grounding_sources": true,
      "mention_status": "Featured prominently in top 3",
      "reason": "Amazon appears as #1 recommendation with comprehensive features",
      "sentiment": "positive",
      "competitors": ["eBay", "Walmart", "Target"],
      "insights": [
        "Amazon dominates e-commerce with Prime membership",
        "Strong presence in AI search results",
        "Frequently cited in authoritative sources"
      ],
      "actions": [
        "Maintain Prime benefits visibility",
        "Highlight fast shipping advantage",
        "Emphasize product selection breadth"
      ],
      "competitor_info": "eBay and Walmart also mentioned but ranked lower"
    },
    "llm_name": "Gemini Flash",
    "llm_provider": "google",
    "tokens_used": 450,
    "latency_ms": 2341,
    "created_at": "2025-11-29T..."
  }
}
```

**✅ What to Verify:**
- `visibility_score`: Should be 5-10 for Amazon (strong brand)
- `brand_mentioned`: Should be true
- `in_grounding_sources`: Likely true (Amazon.com cited)
- `sentiment`: Should be "positive" or "neutral"
- `competitors`: Should list eBay, Walmart, etc.
- Response includes position and domain data

---

### **Phase 4: Bulk Campaign Execution**

#### Step 4.1: Run Bulk Campaign
```bash
curl -X POST http://localhost:8989/api/v1/geo/execute/bulk \
  -H "Content-Type: application/json" \
  -d '{
    "campaign_name": "Amazon GEO Analysis - December 2025",
    "brand": "Amazon",
    "prompt_ids": '"$PROMPT_IDS"',
    "llm_ids": ["'"$LLM_ID"'"],
    "temperature": 0.7
  }' | jq '.'
```

**Expected Response:**
```json
{
  "success": true,
  "data": {
    "campaign_id": "campaign-123",
    "campaign_name": "Amazon GEO Analysis - December 2025",
    "brand": "Amazon",
    "total_runs": 5,
    "status": "running",
    "started_at": "2025-11-29T12:00:00Z",
    "message": "Campaign started successfully. Execution running in background."
  }
}
```

**⏱️ Wait Time:**
- 5 prompts × 1 LLM = 5 executions
- ~2-3 seconds per execution
- Total: ~15-20 seconds

**Monitor Progress:**
```bash
# Check server logs to see execution progress
# Look for messages like:
# "✅ Successfully executed prompt X with LLM Y"
```

---

### **Phase 5: Basic GEO Insights**

#### Step 5.1: Get Overall GEO Insights
```bash
curl -X POST http://localhost:8989/api/v1/geo/insights \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "Amazon"
  }' | jq '.'
```

**Expected Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Amazon",
    "average_visibility": 7.8,
    "mention_rate": 95.5,
    "grounding_rate": 78.2,
    "sentiment_breakdown": {
      "positive": 18,
      "neutral": 4,
      "negative": 1
    },
    "top_competitors": [
      {
        "name": "eBay",
        "mention_count": 15,
        "visibility_avg": 6.2
      },
      {
        "name": "Walmart",
        "mention_count": 12,
        "visibility_avg": 5.8
      }
    ],
    "performance_by_llm": [
      {
        "llm_name": "Gemini Flash",
        "llm_provider": "google",
        "visibility": 7.8,
        "mention_rate": 95.5,
        "response_count": 5
      }
    ],
    "performance_by_category": [
      {
        "category": "ecommerce",
        "visibility": 7.8,
        "mention_rate": 95.5,
        "response_count": 5
      }
    ],
    "total_responses": 5
  }
}
```

**✅ What to Verify:**
- `average_visibility`: Amazon should be 7-9 (strong brand)
- `mention_rate`: Should be 80-100% (mentioned almost always)
- `grounding_rate`: 50-90% (Amazon.com frequently cited)
- `sentiment_breakdown`: Mostly positive
- `top_competitors`: eBay, Walmart, Target, etc.

---

### **Phase 6: Source Analytics** 🆕

#### Step 6.1: Analyze Citation Sources
```bash
curl -X POST http://localhost:8989/api/v1/geo/analytics/sources \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "Amazon",
    "top_n": 15
  }' | jq '.'
```

**Expected Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Amazon",
    "period": "all-time",
    "top_sources": [
      {
        "domain": "amazon.com",
        "citation_count": 4,
        "mention_rate": 80.0,
        "llm_breakdown": {
          "Gemini Flash": 4
        },
        "categories": ["company_website"]
      },
      {
        "domain": "techcrunch.com",
        "citation_count": 2,
        "mention_rate": 40.0,
        "llm_breakdown": {
          "Gemini Flash": 2
        },
        "categories": ["news"]
      },
      {
        "domain": "forbes.com",
        "citation_count": 2,
        "mention_rate": 40.0,
        "categories": ["publication"]
      }
    ],
    "recommendations": [
      {
        "type": "source_opportunity",
        "priority": "high",
        "title": "Strong presence in cited sources",
        "description": "Amazon.com appears in 80% of responses with citations. This is excellent visibility.",
        "action": "Maintain authoritative content. Continue building citations from tech publications.",
        "impact": "high"
      }
    ],
    "total_sources": 8,
    "total_citations": 12
  }
}
```

**✅ What to Verify:**
- Amazon.com should be in top sources (high citation rate)
- Should see news sites (TechCrunch, Forbes, etc.)
- Recommendations should be actionable
- `total_sources`: Should have 5-15 unique domains

---

### **Phase 7: Competitive Benchmarking** 🆕

#### Step 7.1: Compare Amazon vs Competitors
```bash
curl -X POST http://localhost:8989/api/v1/geo/analytics/competitive \
  -H "Content-Type: application/json" \
  -d '{
    "main_brand": "Amazon",
    "competitors": ["eBay", "Walmart", "Target"],
    "region": "US"
  }' | jq '.'
```

**Note:** This requires existing data for competitors. For testing, you might only see Amazon data.

**Expected Response:**
```json
{
  "success": true,
  "data": {
    "main_brand": {
      "brand": "Amazon",
      "visibility": 7.8,
      "mention_rate": 95.5,
      "grounding_rate": 78.2,
      "average_position": 1.4,
      "top_position_rate": 85.0,
      "sentiment_score": 0.82,
      "response_count": 5,
      "market_share_pct": 58.3
    },
    "competitors": [
      {
        "brand": "eBay",
        "visibility": 6.2,
        "mention_rate": 65.0,
        "grounding_rate": 45.0,
        "average_position": 2.8,
        "top_position_rate": 45.0,
        "sentiment_score": 0.65,
        "response_count": 5,
        "market_share_pct": 23.1
      }
      // ... other competitors if data exists
    ],
    "market_leader": "Amazon",
    "your_rank": 1,
    "total_brands": 2,
    "recommendations": [
      {
        "type": "maintain_leadership",
        "priority": "medium",
        "title": "Maintain market leadership position",
        "description": "You're the market leader with 7.8 visibility. Focus on maintaining and extending this lead.",
        "action": "Continue your current strategy. Monitor emerging competitors. Expand to new prompt categories and regions.",
        "impact": "medium"
      }
    ],
    "analyzed_at": "2025-11-29T12:30:00Z"
  }
}
```

**✅ What to Verify:**
- Amazon should be market leader (highest visibility)
- `market_share_pct`: Amazon should have 40-60%
- `average_position`: Should be 1-2 (top ranked)
- `top_position_rate`: Should be 70-90%
- Recommendations reflect leadership position

---

### **Phase 8: Position Analytics** 🆕

#### Step 8.1: Analyze Ranking Positions
```bash
curl -X POST http://localhost:8989/api/v1/geo/analytics/position \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "Amazon"
  }' | jq '.'
```

**Expected Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Amazon",
    "average_position": 1.4,
    "top_position_rate": 85.0,
    "position_breakdown": {
      "position_1": 3,
      "position_2": 1,
      "position_3": 1
    },
    "by_prompt_type": {
      "top_best": 1.2,
      "comparison": 1.5,
      "how": 1.8,
      "what": 1.3,
      "brand": 1.0
    },
    "by_llm": {
      "Gemini Flash": 1.4
    },
    "total_mentions": 5
  }
}
```

**✅ What to Verify:**
- `average_position`: Amazon should be 1-2 (market leader)
- `top_position_rate`: 70-90% (usually in top 3)
- `position_breakdown`: Most mentions at position 1
- Position varies by prompt type (brand-specific = position 1)

---

### **Phase 9: Prompt Performance** 🆕

#### Step 9.1: Analyze Which Prompts Work Best
```bash
curl -X POST http://localhost:8989/api/v1/geo/analytics/prompt-performance \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "Amazon",
    "min_responses": 1
  }' | jq '.'
```

**Expected Response:**
```json
{
  "success": true,
  "data": {
    "brand": "Amazon",
    "period": "all-time",
    "prompts": [
      {
        "prompt_id": "prompt-1",
        "prompt_text": "What are the best online shopping platforms?",
        "prompt_type": "top_best",
        "category": "ecommerce",
        "avg_visibility": 9.0,
        "avg_position": 1.0,
        "mention_rate": 100.0,
        "top_position_rate": 100.0,
        "avg_sentiment": 0.9,
        "total_responses": 1,
        "brand_mentions": 1,
        "effectiveness_score": 95.5,
        "effectiveness_grade": "A+",
        "status": "high_performer",
        "recommendation": "Excellent performance! Keep using this prompt frequently. Consider creating similar prompts. Optimize content around this topic."
      },
      {
        "prompt_id": "prompt-2",
        "prompt_text": "How does Amazon Prime compare to other services?",
        "prompt_type": "comparison",
        "avg_visibility": 8.5,
        "avg_position": 1.0,
        "mention_rate": 100.0,
        "top_position_rate": 100.0,
        "effectiveness_score": 92.3,
        "effectiveness_grade": "A+",
        "status": "high_performer",
        "recommendation": "Excellent performance! Keep using this prompt frequently..."
      },
      {
        "prompt_id": "prompt-5",
        "prompt_text": "What is e-commerce?",
        "prompt_type": "what",
        "avg_visibility": 2.0,
        "avg_position": 0,
        "mention_rate": 15.0,
        "effectiveness_score": 28.5,
        "effectiveness_grade": "D",
        "status": "low_performer",
        "recommendation": "Very low performance. Consider removing this prompt or completely changing approach. May not be relevant to your brand positioning."
      }
    ],
    "top_performers": ["prompt-1", "prompt-2", "prompt-3"],
    "low_performers": ["prompt-5"],
    "avg_effectiveness": 72.8,
    "total_prompts_analyzed": 5
  }
}
```

**✅ What to Verify:**
- Amazon should have many A/A+ grade prompts
- "Top/Best" and "Comparison" prompts score highest
- "What is" educational prompts score lower
- `avg_effectiveness`: Should be 60-80 for Amazon
- Clear distinction between high and low performers

---

### **Phase 10: Search Responses**

#### Step 10.1: Search for Specific Keywords
```bash
curl -X POST http://localhost:8989/api/v1/search \
  -H "Content-Type: application/json" \
  -d '{
    "keyword": "Amazon",
    "limit": 50
  }' | jq '.'
```

**Expected Response:**
```json
{
  "success": true,
  "data": {
    "keyword": "Amazon",
    "total_mentions": 127,
    "unique_prompts": 5,
    "unique_llms": 1,
    "by_prompt": {
      "prompt-1": 28,
      "prompt-2": 25,
      "prompt-3": 24
    },
    "by_llm": {
      "llm-id-1": 127
    },
    "by_provider": {
      "google": 127
    },
    "first_seen": "2025-11-29T12:00:00Z",
    "last_seen": "2025-11-29T12:15:00Z",
    "responses": [
      // ... actual response objects
    ]
  }
}
```

---

## 📊 Complete Test Script

### Save this as `test_amazon_geo.sh`:

```bash
#!/bin/bash

# Complete GEO API Test for Amazon
# Run this after starting the API server

BASE_URL="http://localhost:8989/api/v1"
BRAND="Amazon"
WEBSITE="https://www.amazon.com"

echo "🚀 Starting Complete GEO API Test for Amazon"
echo "=============================================="

# Get LLM ID
echo -e "\n1️⃣ Getting LLM information..."
LLM_RESPONSE=$(curl -s "$BASE_URL/llms")
LLM_ID=$(echo $LLM_RESPONSE | jq -r '.data[0].id')
echo "✅ LLM ID: $LLM_ID"

# Generate Prompts
echo -e "\n2️⃣ Generating prompts with website scraping..."
PROMPTS_RESPONSE=$(curl -s -X POST "$BASE_URL/geo/prompts/generate" \
  -H "Content-Type: application/json" \
  -d "{\"brand\":\"$BRAND\",\"website\":\"$WEBSITE\",\"count\":10}")

PROMPT_IDS=$(echo $PROMPTS_RESPONSE | jq -r '[.data.prompts[0:5][].id]')
echo "✅ Generated prompts. Using first 5: $PROMPT_IDS"

# Run Single Test Execution
echo -e "\n3️⃣ Running single test execution..."
curl -s -X POST "$BASE_URL/execute" \
  -H "Content-Type: application/json" \
  -d "{
    \"prompt\": \"What are the best online shopping platforms?\",
    \"llm_id\": \"$LLM_ID\",
    \"brand\": \"$BRAND\",
    \"region\": \"US\"
  }" | jq '.data | {visibility_score: .geo_analysis.visibility_score, brand_mentioned: .geo_analysis.brand_mentioned, sentiment: .geo_analysis.sentiment}'

# Run Bulk Campaign
echo -e "\n4️⃣ Running bulk campaign..."
CAMPAIGN_RESPONSE=$(curl -s -X POST "$BASE_URL/geo/execute/bulk" \
  -H "Content-Type: application/json" \
  -d "{
    \"campaign_name\": \"Amazon Test Campaign\",
    \"brand\": \"$BRAND\",
    \"prompt_ids\": $PROMPT_IDS,
    \"llm_ids\": [\"$LLM_ID\"],
    \"temperature\": 0.7
  }")
echo $CAMPAIGN_RESPONSE | jq '.data | {campaign_id, total_runs, status}'

echo -e "\n⏱️  Waiting 20 seconds for campaign to complete..."
sleep 20

# Get GEO Insights
echo -e "\n5️⃣ Getting GEO insights..."
curl -s -X POST "$BASE_URL/geo/insights" \
  -H "Content-Type: application/json" \
  -d "{\"brand\": \"$BRAND\"}" \
  | jq '.data | {brand, average_visibility, mention_rate, grounding_rate, total_responses}'

# Get Source Analytics
echo -e "\n6️⃣ Getting source analytics..."
curl -s -X POST "$BASE_URL/geo/analytics/sources" \
  -H "Content-Type: application/json" \
  -d "{\"brand\": \"$BRAND\", \"top_n\": 10}" \
  | jq '.data | {brand, total_sources, total_citations, top_sources: .top_sources[0:3]}'

# Get Position Analytics
echo -e "\n7️⃣ Getting position analytics..."
curl -s -X POST "$BASE_URL/geo/analytics/position" \
  -H "Content-Type: application/json" \
  -d "{\"brand\": \"$BRAND\"}" \
  | jq '.data | {brand, average_position, top_position_rate, total_mentions}'

# Get Prompt Performance
echo -e "\n8️⃣ Getting prompt performance..."
curl -s -X POST "$BASE_URL/geo/analytics/prompt-performance" \
  -H "Content-Type: application/json" \
  -d "{\"brand\": \"$BRAND\", \"min_responses\": 1}" \
  | jq '.data | {brand, avg_effectiveness, total_prompts_analyzed, top_performers, low_performers}'

# Search for Amazon
echo -e "\n9️⃣ Searching for 'Amazon' keyword..."
curl -s -X POST "$BASE_URL/search" \
  -H "Content-Type: application/json" \
  -d "{\"keyword\": \"Amazon\", \"limit\": 10}" \
  | jq '.data | {keyword, total_mentions, unique_prompts, unique_llms}'

echo -e "\n✅ Test Complete!"
echo "=============================================="
echo "📊 Summary:"
echo "- Generated 10 prompts"
echo "- Ran 5 executions"
echo "- Analyzed visibility, sources, positions, and prompt performance"
echo ""
echo "📁 Next Steps:"
echo "- Review detailed responses above"
echo "- Check docs/ADVANCED_ANALYTICS.md for interpretation"
echo "- Run competitive analysis by adding competitor brands"
```

### Run the test:
```bash
chmod +x test_amazon_geo.sh
./test_amazon_geo.sh
```

---

## 🎯 Expected Results Summary

### For Amazon (Strong Brand):

| Metric | Expected Range | Why |
|--------|---------------|-----|
| **Visibility Score** | 7-9 | Dominant e-commerce brand |
| **Mention Rate** | 85-100% | Almost always mentioned |
| **Grounding Rate** | 60-90% | Amazon.com frequently cited |
| **Average Position** | 1-2 | Usually #1 or #2 |
| **Top Position Rate** | 70-90% | Dominant positioning |
| **Sentiment Score** | 0.7-0.9 | Very positive |
| **Prompt Effectiveness** | 70-85 avg | Strong across most prompts |
| **Market Share** | 40-60% | Market leader |

---

## ✅ Validation Checklist

- [ ] Health check passes
- [ ] LLM configured and responding
- [ ] Prompts generated (10-20)
- [ ] Single execution works with GEO analysis
- [ ] Bulk campaign completes successfully
- [ ] GEO insights show high visibility (7+)
- [ ] Source analytics shows amazon.com in top sources
- [ ] Position analytics shows top rankings (1-2)
- [ ] Prompt performance identifies best prompts
- [ ] Search finds Amazon mentions
- [ ] All responses have proper structure
- [ ] Recommendations are actionable

---

## 🔍 Troubleshooting

### Issue: Low Visibility for Amazon
**Cause:** Google Gemini might not have recent data  
**Solution:** Try different prompts focused on "online shopping" or "e-commerce platforms"

### Issue: No Grounding Sources
**Cause:** LLM not configured for grounding  
**Solution:** Ensure using Google Gemini with grounding enabled

### Issue: Campaign takes too long
**Cause:** Rate limiting or slow API  
**Solution:** Reduce number of prompts or add delay between requests

### Issue: "No prompts analyzed"
**Cause:** Insufficient responses per prompt  
**Solution:** Lower `min_responses` to 1

---

## 📚 Documentation References

- **API Reference:** `docs/GEO_API.md`
- **Advanced Analytics:** `docs/ADVANCED_ANALYTICS.md`
- **Prompt Performance:** `docs/PROMPT_PERFORMANCE.md`
- **Quick Start:** `QUICK_START_ADVANCED.md`

---

**Happy Testing! 🚀**

All APIs should work perfectly for Amazon. The results will demonstrate the full power of your GEO platform!

