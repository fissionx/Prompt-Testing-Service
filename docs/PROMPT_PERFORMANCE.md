# 🎯 Prompt Performance Analytics

## Overview

Prompt Performance Analytics helps you identify which prompts/questions drive the most brand visibility and mentions. This feature answers the critical question: **"Which prompts should I focus on for maximum ROI?"**

---

## 🚀 Quick Start

### Basic Usage

```bash
curl -X POST http://localhost:8989/api/v1/geo/analytics/prompt-performance \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "YourBrand"
  }'
```

### With Date Range

```bash
curl -X POST http://localhost:8989/api/v1/geo/analytics/prompt-performance \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "YourBrand",
    "start_time": "2025-11-01T00:00:00Z",
    "end_time": "2025-11-30T23:59:59Z",
    "min_responses": 5
  }'
```

---

## 📊 Understanding the Response

### Example Response

```json
{
  "success": true,
  "data": {
    "brand": "YourBrand",
    "period": "2025-11-01 to 2025-11-30",
    "prompts": [
      {
        "prompt_id": "prompt-1",
        "prompt_text": "What are the best AI SEO tools?",
        "prompt_type": "top_best",
        "category": "AI SEO Tools",
        
        // Performance Metrics
        "avg_visibility": 8.2,
        "avg_position": 1.8,
        "mention_rate": 85.0,
        "top_position_rate": 78.0,
        "avg_sentiment": 0.85,
        
        // Volume Metrics
        "total_responses": 30,
        "brand_mentions": 26,
        
        // Effectiveness
        "effectiveness_score": 95.2,
        "effectiveness_grade": "A+",
        "status": "high_performer",
        "recommendation": "Excellent performance! Keep using this prompt frequently. Consider creating similar prompts. Optimize content around this topic."
      },
      {
        "prompt_id": "prompt-5",
        "prompt_text": "What is GEO?",
        "prompt_type": "what",
        
        "avg_visibility": 2.1,
        "avg_position": 0,
        "mention_rate": 12.0,
        "top_position_rate": 0,
        "avg_sentiment": 0.0,
        
        "total_responses": 25,
        "brand_mentions": 3,
        
        "effectiveness_score": 23.5,
        "effectiveness_grade": "D",
        "status": "low_performer",
        "recommendation": "Very low performance. Consider removing this prompt or completely changing approach. May not be relevant to your brand positioning."
      }
    ],
    
    // Summary
    "top_performers": ["prompt-1", "prompt-3", "prompt-7"],
    "low_performers": ["prompt-5", "prompt-12", "prompt-18"],
    "avg_effectiveness": 68.5,
    "total_prompts_analyzed": 25
  }
}
```

---

## 📈 Key Metrics Explained

### Performance Metrics

#### 1. **Average Visibility** (0-10 scale)
How prominently your brand appears in responses.

- **8-10**: Excellent - Featured prominently
- **5-7**: Good - Mentioned with context
- **3-4**: Fair - Brief mention
- **0-2**: Poor - Not mentioned or barely visible

#### 2. **Average Position** (1 = best)
Your typical ranking when mentioned in lists.

- **1-2**: Excellent - Usually #1 or #2
- **3-4**: Good - Top half of lists
- **5-8**: Fair - Middle of pack
- **9+**: Poor - Bottom of lists

#### 3. **Mention Rate** (%)
Percentage of responses that mention your brand.

- **80-100%**: Excellent - Almost always mentioned
- **60-80%**: Good - Frequently mentioned
- **30-60%**: Fair - Sometimes mentioned
- **0-30%**: Poor - Rarely mentioned

#### 4. **Top Position Rate** (%)
Percentage of times you're in top 3 positions.

- **70-100%**: Excellent - Dominant position
- **40-70%**: Good - Competitive
- **20-40%**: Fair - Occasional top 3
- **0-20%**: Poor - Rarely in top 3

#### 5. **Average Sentiment** (-1 to +1)
Overall sentiment when mentioned.

- **0.6 to 1.0**: Very Positive
- **0.2 to 0.6**: Positive
- **-0.2 to 0.2**: Neutral
- **-0.6 to -0.2**: Negative
- **-1.0 to -0.6**: Very Negative

---

## 🎯 Effectiveness Score

### How It's Calculated

```
Effectiveness Score (0-100) = 
  (avg_visibility / 10) × 40% +          // Visibility weight
  (mention_rate / 100) × 30% +           // Mention weight
  (top_position_rate / 100) × 20% +      // Position weight
  (1 - avg_position / 10) × 10%          // Ranking bonus
```

### Grade Scale

| Score | Grade | Status | Meaning |
|-------|-------|--------|---------|
| 90-100 | A+ | high_performer | Exceptional - Keep using! |
| 85-89 | A | high_performer | Excellent - Very effective |
| 80-84 | A- | high_performer | Very Good - Strong performer |
| 75-79 | B+ | average_performer | Good - Above average |
| 70-74 | B | average_performer | Good - Solid performance |
| 65-69 | B- | average_performer | Decent - Room to improve |
| 60-64 | C+ | average_performer | Fair - Needs work |
| 55-59 | C | average_performer | Fair - Below expectations |
| 50-54 | C- | average_performer | Mediocre - Consider changes |
| 45-49 | D+ | low_performer | Poor - Needs attention |
| 40-44 | D | low_performer | Poor - Major issues |
| 0-39 | F | very_low_performer | Failing - Remove or revise |

---

## 💡 Using the Insights

### 1. Prioritize High Performers

**Action:** Focus 80% of your efforts on top 20% of prompts.

```bash
# Get performance data
RESPONSE=$(curl -X POST http://localhost:8989/api/v1/geo/analytics/prompt-performance \
  -H "Content-Type: application/json" \
  -d '{"brand": "YourBrand"}')

# Extract top performers
TOP_PROMPTS=$(echo $RESPONSE | jq -r '.data.top_performers[]')

# Use these prompts more frequently in campaigns
```

### 2. Optimize Content for Top Prompts

**Example:**
- Prompt: "What are the best AI SEO tools?" (Score: 95)
- Action: Create comprehensive blog post about "Best AI SEO Tools 2025"
- Include your brand prominently
- Get it cited by authoritative sources

### 3. Fix or Remove Low Performers

**Decision Tree:**

```
Low Performer Detected
├─ Visibility < 3.0?
│  └─ Create authoritative content on this topic
│     Build citations from high-quality sources
│
├─ Mention Rate < 30%?
│  └─ Content not relevant to this question
│     Either optimize content OR stop tracking
│
└─ Position > 5?
   └─ Mentioned but ranked low
      Improve competitive positioning
      Get more positive reviews
```

### 4. A/B Test Variations

**Example:**
- Original: "What is GEO?" (Score: 23, Grade: D)
- Variation: "What are the benefits of GEO?" (Score: ?, Grade: ?)
- Compare after 2 weeks

---

## 🎓 Real-World Examples

### Example 1: SaaS Company

**Before Analysis:**
- Running 50 prompts equally
- Total visibility: 4.2
- No clear strategy

**After Analysis:**
```json
{
  "top_performers": [
    "What are the best CRM tools for startups?" (Score: 92),
    "CRM software comparison 2025" (Score: 88),
    "How to choose a CRM system?" (Score: 85)
  ],
  "low_performers": [
    "What is CRM?" (Score: 25),
    "CRM definition" (Score: 18)
  ]
}
```

**Actions Taken:**
1. ✅ Focused on top 3 prompts (ran 5x more often)
2. ✅ Created detailed comparison content
3. ✅ Removed educational "what is" prompts
4. ✅ Improved G2 profile (cited in top prompts)

**Results After 1 Month:**
- Visibility increased: 4.2 → 6.8
- Average position improved: 4.5 → 2.1
- Mention rate up: 35% → 62%

---

### Example 2: E-commerce Brand

**Discovery:**
```json
{
  "prompts": [
    {
      "prompt_text": "Best sustainable fashion brands",
      "effectiveness_score": 78,
      "avg_position": 3.2,
      "recommendation": "Good performance. Focus on sustainability messaging."
    },
    {
      "prompt_text": "Affordable ethical clothing",
      "effectiveness_score": 45,
      "avg_position": 7.1,
      "recommendation": "Below average. Improve price competitiveness."
    }
  ]
}
```

**Insights:**
- Strong in "sustainability" positioning
- Weak in "affordable" positioning

**Strategy:**
- Double down on sustainability content
- Either improve pricing OR stop competing on price

---

## 📊 API Reference

### Request Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `brand` | string | Yes | - | Brand name to analyze |
| `start_time` | timestamp | No | null | Start of date range |
| `end_time` | timestamp | No | null | End of date range |
| `min_responses` | integer | No | 3 | Minimum responses per prompt |

### Response Fields

#### PromptPerformance Object

```typescript
{
  prompt_id: string,
  prompt_text: string,
  prompt_type: "what" | "how" | "comparison" | "top_best" | "brand",
  category: string,
  
  // Performance
  avg_visibility: number,        // 0-10
  avg_position: number,          // 1+ (1 = best)
  mention_rate: number,          // 0-100%
  top_position_rate: number,     // 0-100%
  avg_sentiment: number,         // -1 to +1
  
  // Volume
  total_responses: number,
  brand_mentions: number,
  
  // Effectiveness
  effectiveness_score: number,   // 0-100
  effectiveness_grade: string,   // "A+", "A", "B", etc.
  status: string,                // "high_performer", "average_performer", etc.
  recommendation: string         // Actionable advice
}
```

---

## 🔧 Advanced Usage

### Filter by Date Range

Track improvement over time:

```bash
# November performance
curl -X POST http://localhost:8989/api/v1/geo/analytics/prompt-performance \
  -d '{
    "brand": "YourBrand",
    "start_time": "2025-11-01T00:00:00Z",
    "end_time": "2025-11-30T23:59:59Z"
  }' > november.json

# December performance
curl -X POST http://localhost:8989/api/v1/geo/analytics/prompt-performance \
  -d '{
    "brand": "YourBrand",
    "start_time": "2025-12-01T00:00:00Z",
    "end_time": "2025-12-31T23:59:59Z"
  }' > december.json

# Compare improvements
```

### Set Minimum Response Threshold

Only analyze prompts with enough data:

```bash
curl -X POST http://localhost:8989/api/v1/geo/analytics/prompt-performance \
  -d '{
    "brand": "YourBrand",
    "min_responses": 10
  }'
```

---

## 📈 Best Practices

### 1. Run Analysis Monthly
Track improvement over time and adjust strategy.

### 2. Minimum 3-5 Responses Per Prompt
Need enough data for meaningful insights.

### 3. Focus on Score + Grade
Not just individual metrics.

### 4. Act on Recommendations
The system tells you exactly what to do.

### 5. Track Top Performers Over Time
Ensure they stay effective.

### 6. Test New Prompts
Add variations of high performers.

---

## 🚨 Common Issues

### "No prompts returned"
- ✅ Ensure you have responses for this brand
- ✅ Check `min_responses` setting (default: 3)
- ✅ Verify date range includes your data

### "All low scores"
- ⚠️ This indicates GEO issues
- 📝 Focus on recommendations
- 📝 Improve content and citations
- 📝 Run competitive benchmark to see how others perform

### "Inconsistent scores across time"
- ✅ Normal - AI responses vary
- ✅ Look at trends over weeks/months
- ✅ Focus on directional changes

---

## 💪 Competitive Advantage

This feature gives you:

1. ✅ **Data-Driven Decisions** - Know exactly which prompts work
2. ✅ **ROI Optimization** - Focus on high-impact prompts
3. ✅ **Continuous Improvement** - Track progress over time
4. ✅ **Strategic Insights** - Understand your positioning
5. ✅ **Resource Efficiency** - Stop wasting time on bad prompts

---

## 🎯 What's Next?

After analyzing prompt performance:

1. **Run Competitive Benchmark** - See how competitors perform on same prompts
2. **Check Source Analytics** - See which sources drive top prompts
3. **Get Recommendations** - Act on system suggestions
4. **Optimize Content** - Focus on high-performing topics
5. **Track Trends** - Monitor monthly improvements

---

**Made with ❤️ for data-driven GEO optimization**

