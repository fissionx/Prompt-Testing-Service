# 🚀 Quick Start: Advanced Analytics Features

## What's New?

Your gego platform now has **Peec AI-level capabilities**! 🎉

### Key Features Added
1. ✅ **Position/Ranking** - See where your brand ranks (1st, 2nd, 3rd...)
2. ✅ **Source Analytics** - Which sites (G2, Reddit, etc.) cite your brand
3. ✅ **Recommendations** - AI tells you exactly what to do to improve
4. ✅ **Competitive Benchmarking** - Compare with competitors + market share
5. ✅ **Multi-Region** - Track US, UK, DE, FR separately
6. ✅ **Time-Series** - Track trends over weeks/months/quarters

---

## 🎯 Try It Now (5 Minutes)

### Step 1: Start Your Server
```bash
cd /Users/senyarav/workspace/opensource/gego
./gego api --port 8989
```

### Step 2: Get Source Analytics
See which websites AI models cite most:

```bash
curl -X POST http://localhost:8989/api/v1/geo/analytics/sources \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "YourBrandName",
    "top_n": 10
  }'
```

**You'll get:**
- Top 10 most cited sources (domains)
- How often each source is cited
- **Specific recommendations** like:
  - "Optimize your G2 profile"
  - "Engage in Reddit discussions"
  - "Target news outlets like TechCrunch"

### Step 3: Compare with Competitors
See how you stack up:

```bash
curl -X POST http://localhost:8989/api/v1/geo/analytics/competitive \
  -H "Content-Type: application/json" \
  -d '{
    "main_brand": "YourBrandName",
    "competitors": ["Competitor1", "Competitor2"]
  }'
```

**You'll get:**
- Visibility scores for each brand
- Average position (1st, 2nd, 3rd...)
- **Market share %**
- Sentiment comparison
- **Recommendations** like:
  - "Close 2.5 point gap with market leader"
  - "Improve position from 4th to top 3"
  - "Address sentiment issues"

### Step 4: Check Your Positioning
See where you rank in lists:

```bash
curl -X POST http://localhost:8989/api/v1/geo/analytics/position \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "YourBrandName"
  }'
```

**You'll get:**
- Average position (e.g., 3.2 = usually 3rd or 4th)
- % of times you're in top 3
- Position breakdown by prompt type
- Position breakdown by LLM

---

## 📊 Complete Workflow

### Full GEO Campaign with Advanced Analytics

```bash
# 1. Generate smart prompts
curl -X POST http://localhost:8989/api/v1/geo/prompts/generate \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "MyBrand",
    "website": "https://mybrand.com",
    "count": 30
  }'

# Save the prompt IDs from response

# 2. Get your LLM IDs
curl http://localhost:8989/api/v1/llms

# 3. Run bulk campaign
curl -X POST http://localhost:8989/api/v1/geo/execute/bulk \
  -H "Content-Type: application/json" \
  -d '{
    "campaign_name": "December 2025 Analysis",
    "brand": "MyBrand",
    "prompt_ids": ["id1", "id2", "id3"],
    "llm_ids": ["llm-id1", "llm-id2"],
    "temperature": 0.7
  }'

# Wait for campaign to complete (check logs)

# 4. Get ALL insights

# Source Analytics
curl -X POST http://localhost:8989/api/v1/geo/analytics/sources \
  -H "Content-Type: application/json" \
  -d '{"brand": "MyBrand", "top_n": 20}'

# Competitive Benchmark
curl -X POST http://localhost:8989/api/v1/geo/analytics/competitive \
  -H "Content-Type: application/json" \
  -d '{
    "main_brand": "MyBrand",
    "competitors": ["Comp1", "Comp2", "Comp3"]
  }'

# Position Analytics
curl -X POST http://localhost:8989/api/v1/geo/analytics/position \
  -H "Content-Type: application/json" \
  -d '{"brand": "MyBrand"}'

# Standard GEO Insights
curl -X POST http://localhost:8989/api/v1/geo/insights \
  -H "Content-Type: application/json" \
  -d '{"brand": "MyBrand"}'
```

---

## 🎓 Understanding Your Results

### Source Analytics Response
```json
{
  "top_sources": [
    {
      "domain": "g2.com",
      "citation_count": 45,
      "mention_rate": 75.0
    }
  ],
  "recommendations": [
    {
      "priority": "high",
      "title": "Optimize presence on g2.com",
      "action": "Create G2 profile. Get customer reviews."
    }
  ]
}
```

**What to do:**
1. Focus on the top 3-5 sources
2. Follow the recommendations exactly
3. Track changes over time

### Competitive Benchmark Response
```json
{
  "main_brand": {
    "brand": "YourBrand",
    "visibility": 4.2,
    "market_share_pct": 18.5,
    "average_position": 3.5
  },
  "market_leader": "Competitor A",
  "your_rank": 3
}
```

**What it means:**
- **Visibility 4.2** = Moderate visibility (0-10 scale)
- **Market Share 18.5%** = You have 18.5% of total visibility
- **Average Position 3.5** = Usually ranked 3rd or 4th
- **Your Rank 3** = 3rd out of all analyzed brands

---

## 🌍 Multi-Region Tracking

Track performance in different countries:

```bash
# US Market
curl -X POST http://localhost:8989/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Best AI SEO tools 2025",
    "llm_id": "your-llm-id",
    "brand": "MyBrand",
    "region": "US",
    "language": "en"
  }'

# UK Market
curl -X POST http://localhost:8989/api/v1/execute \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Best AI SEO tools 2025",
    "llm_id": "your-llm-id",
    "brand": "MyBrand",
    "region": "UK",
    "language": "en"
  }'

# Then compare by region
curl -X POST http://localhost:8989/api/v1/geo/analytics/competitive \
  -H "Content-Type: application/json" \
  -d '{
    "main_brand": "MyBrand",
    "competitors": ["Comp1"],
    "region": "US"
  }'
```

---

## 💡 Pro Tips

### 1. Use Google Gemini for Best Results
Google Gemini provides citation sources (grounding). Other LLMs don't.

```bash
# Add Gemini if you haven't
gego llm add
# Choose: google, model: models/gemini-1.5-flash
```

### 2. Run Campaigns Weekly
Track improvements over time:
- Week 1: Baseline
- Week 2-4: Implement recommendations
- Week 4: See improvements

### 3. Focus on Top 3 Positions
Being #1-3 matters most. Focus efforts on:
- "Best", "Top", "Comparison" prompts
- Review sites (G2, Capterra)
- Getting testimonials

### 4. Act on Recommendations
The AI tells you exactly what to do. Just do it:
- ✅ G2 cited often? → Create G2 profile
- ✅ Reddit appears? → Join subreddit discussions
- ✅ Position 5th? → Improve review scores

---

## 📈 Tracking Improvements

### Week 1 (Baseline)
```bash
curl -X POST http://localhost:8989/api/v1/geo/analytics/sources \
  -H "Content-Type: application/json" \
  -d '{"brand": "MyBrand"}' > week1.json
```

### Week 4 (After Actions)
```bash
curl -X POST http://localhost:8989/api/v1/geo/analytics/sources \
  -H "Content-Type: application/json" \
  -d '{"brand": "MyBrand"}' > week4.json
```

Compare:
- Citation count increased?
- Average position improved?
- Market share growing?
- New sources appearing?

---

## 🚨 Common Issues

### "No position data"
- Position extraction works on list-based responses
- Use prompts like "What are the best..." or "Compare..."
- Avoid yes/no questions

### "No source data"  
- Source citations require Google Gemini
- Make sure you're using Gemini with grounding enabled
- Other LLMs don't provide source URLs

### "No recommendations"
- Need at least 20-30 responses for good recommendations
- Run more prompts across more LLMs
- Wait for bulk campaigns to complete

---

## 📚 Full Documentation

- **Feature Details:** `docs/ADVANCED_ANALYTICS.md`
- **API Reference:** `docs/GEO_API.md`
- **Implementation:** `IMPLEMENTATION_SUMMARY.md`

---

## 🎉 You're Ready!

Your gego platform now has:
- ✅ Same features as Peec AI
- ✅ PLUS open-source benefits
- ✅ PLUS multi-LLM support
- ✅ PLUS self-hosted control

**Start tracking, get insights, take action, dominate AI search!** 🚀

---

**Questions?**
- GitHub: https://github.com/AI2HU/gego/issues
- Email: jonathan@blocs.fr

