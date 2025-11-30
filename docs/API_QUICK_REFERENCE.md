# GEGO API Quick Reference for UI Developers

Quick reference for integrating GEGO GEO APIs into your frontend.

## Base URL
```
POST http://localhost:8989/api/v1
```

---

## 📊 At a Glance: API Endpoints

| Endpoint | Purpose | When to Use |
|----------|---------|-------------|
| `/geo/prompts/generate` | Generate GEO prompts | Start of workflow - get optimized prompts for a brand |
| `/geo/execute/bulk` | Run campaign | After prompt selection - test brand visibility across LLMs |
| `/geo/insights` | Get brand analytics | View overall performance metrics |
| `/geo/analytics/sources` | Analyze citations | See which websites cite your brand |
| `/geo/analytics/prompt-performance` | Analyze prompts | Identify best/worst performing prompts |
| `/geo/analytics/competitive` | Compare brands | See how you stack up against competitors |

---

## 🎯 Key Metrics Explained

### Primary Metrics (Show Prominently)

| Metric | Range | Description | Color Code |
|--------|-------|-------------|------------|
| **Visibility Score** | 0-10 | How visible/prominent your brand is in AI responses | 8-10=🟢, 5-7=🟡, 0-4=🔴 |
| **Mention Rate** | 0-100% | Percentage of responses that mention your brand | >60%=🟢, 30-60%=🟡, <30%=🔴 |
| **Average Position** | 1-N | Your ranking in AI responses (lower is better) | 1-3=🟢, 4-7=🟡, 8+=🔴 |
| **Grounding Rate** | 0-100% | Percentage cited in sources | >40%=🟢, 20-40%=🟡, <20%=🔴 |
| **Market Share** | 0-100% | Your share of mentions vs competitors | >35%=🟢, 20-35%=🟡, <20%=🔴 |

### Secondary Metrics (Show in Details)

| Metric | Description |
|--------|-------------|
| **Sentiment Score** | -1 to +1 (negative to positive) |
| **Top Position Rate** | Percentage of times ranked in top 3 |
| **Effectiveness Grade** | A-F score for prompt performance |

---

## 🔄 Typical User Flow

```
1. Generate Prompts
   POST /geo/prompts/generate
   → Get 20 optimized prompts
   
2. User Selects Prompts & LLMs
   → UI: Show checkboxes
   
3. Execute Campaign
   POST /geo/execute/bulk
   → Returns campaignId, runs in background
   → Show "Running..." indicator
   
4. View Insights (after 30-60 seconds)
   POST /geo/insights
   → Display dashboard
   
5. Drill Down (optional)
   - POST /geo/analytics/competitive
   - POST /geo/analytics/prompt-performance
   - POST /geo/analytics/sources
```

---

## 💡 UI Component Mapping

### Dashboard Overview
**Endpoint:** `POST /geo/insights`

```javascript
// Hero Stats (Large Cards)
{
  visibility: data.averageVisibility,        // Display: "7.5 / 10"
  mentionRate: data.mentionRate,            // Display: "65.5%"
  groundingRate: data.groundingRate,        // Display: "45.2%"
  totalResponses: data.totalResponses       // Display: "Based on 120 responses"
}

// Sentiment Chart (Pie/Donut)
data.sentimentBreakdown  // { positive: 75, neutral: 30, negative: 5 }

// Competitor List (Ranked)
data.topCompetitors.map(comp => ({
  name: comp.name,
  logo: comp.logoUrl || comp.fallbackLogoUrl,
  mentions: comp.mentionCount
}))

// LLM Performance Table
data.performanceByLlm.map(llm => ({
  name: llm.llmName,
  provider: llm.llmProvider,
  visibility: llm.visibility,
  mentionRate: llm.mentionRate
}))
```

### Prompt Generator
**Endpoint:** `POST /geo/prompts/generate`

```javascript
// Prompt List with Checkboxes
data.prompts.map(prompt => ({
  id: prompt.id,                    // For checkbox value
  text: prompt.template,            // Display text
  type: prompt.promptType,          // Badge: "comparison"
  isNew: !prompt.reused            // Badge: "New" or "Reused"
}))

// Summary Stats
{
  total: data.prompts.length,
  existing: data.existingPrompts,
  generated: data.generatedPrompts
}

// Grouped View (Tabs)
data.promptsByType  // { comparison: [...], recommendation: [...] }
```

### Competitive Benchmark
**Endpoint:** `POST /geo/analytics/competitive`

```javascript
// Comparison Table
const allBrands = [
  {
    brand: data.mainBrand.brand,
    logo: data.mainBrand.logoUrl,
    visibility: data.mainBrand.visibility,
    mentionRate: data.mainBrand.mentionRate,
    marketShare: data.mainBrand.marketSharePct,
    isYou: true  // Highlight row
  },
  ...data.competitors.map(comp => ({
    brand: comp.brand,
    logo: comp.logoUrl,
    visibility: comp.visibility,
    mentionRate: comp.mentionRate,
    marketShare: comp.marketSharePct,
    isLeader: comp.brand === data.marketLeader
  }))
]

// Ranking Banner
"You're ranked #{data.yourRank} of {data.totalBrands}"
"Market Leader: {data.marketLeader}"

// Prompt-by-Prompt Analysis
data.promptBreakdown.map(item => ({
  prompt: item.promptText,
  yourPosition: item.mainBrandResult.position,
  yourMentioned: item.mainBrandResult.mentioned,
  winner: item.winner,
  competitorsPresent: item.competitorsMentioned.filter(c => c.mentioned)
}))
```

### Prompt Performance
**Endpoint:** `POST /geo/analytics/prompt-performance`

```javascript
// Performance Table (Sortable)
data.prompts.map(prompt => ({
  text: prompt.promptText,
  type: prompt.promptType,
  grade: prompt.effectivenessGrade,      // Display: "A" with green badge
  visibility: prompt.avgVisibility,      // Display: "8.5"
  mentionRate: prompt.mentionRate,       // Display: "85.5%"
  position: prompt.avgPosition,          // Display: "2.3"
  status: prompt.status,                 // Color code row
  recommendation: prompt.recommendation  // Show in tooltip
}))

// Summary
{
  totalAnalyzed: data.totalPromptsAnalyzed,
  avgScore: data.avgEffectiveness,
  topPerformers: data.topPerformers.length,
  needsImprovement: data.lowPerformers.length
}

// Grade Colors
const gradeColors = {
  'A': 'green',    // 85-100
  'B': 'lightgreen', // 70-84
  'C': 'yellow',   // 50-69
  'D': 'orange',   // 30-49
  'F': 'red'       // 0-29
}
```

### Source Analytics
**Endpoint:** `POST /geo/analytics/sources`

```javascript
// Top Domains List
data.topSources.map((source, index) => ({
  rank: index + 1,
  domain: source.domain,              // Link to domain
  citations: source.citationCount,    // Display count
  mentionRate: source.mentionRate,    // Display percentage
  llmBreakdown: source.llmBreakdown  // Show in expandable row
}))

// Summary
{
  totalDomains: data.totalSources,
  totalCitations: data.totalCitations
}

// Recommendations
data.recommendations.map(rec => ({
  priority: rec.priority,      // Badge color
  title: rec.title,
  description: rec.description,
  action: rec.action,
  impact: rec.impact
}))
```

---

## 🎨 UI Design Guidelines

### Color Scheme for Metrics

```css
/* Visibility Score (0-10) */
.score-excellent { color: #10b981; }  /* 8-10 */
.score-good      { color: #fbbf24; }  /* 5-7.9 */
.score-poor      { color: #ef4444; }  /* 0-4.9 */

/* Mention Rate (%) */
.rate-high       { color: #10b981; }  /* >60% */
.rate-medium     { color: #fbbf24; }  /* 30-60% */
.rate-low        { color: #ef4444; }  /* <30% */

/* Status Indicators */
.status-running  { color: #3b82f6; }  /* Blue */
.status-completed{ color: #10b981; }  /* Green */
.status-failed   { color: #ef4444; }  /* Red */

/* Priority Badges */
.priority-high   { background: #fee2e2; color: #991b1b; }
.priority-medium { background: #fef3c7; color: #92400e; }
.priority-low    { background: #dbeafe; color: #1e40af; }
```

### Component Hierarchy

```
Dashboard
├── Hero Metrics (3 large cards)
│   ├── Visibility Score
│   ├── Mention Rate
│   └── Grounding Rate
│
├── Sentiment Chart (Pie/Donut)
│
├── Competitors Section
│   └── List with logos & counts
│
└── Performance Tables
    ├── By LLM
    └── By Category

Competitive Analysis
├── Ranking Banner
├── Comparison Table (You vs Competitors)
├── Prompt Breakdown (Expandable)
└── Recommendations

Prompt Performance
├── Summary Stats
├── Performance Table (Sortable)
│   ├── Grade badges
│   ├── Metrics
│   └── Recommendations
└── Filter/Sort Controls

Source Analytics
├── Summary (Total sources/citations)
├── Top Domains List
│   ├── Citation bars
│   ├── LLM breakdown (expandable)
│   └── Categories tags
└── Action Recommendations
```

---

## 🚦 Loading States

```javascript
// Campaign Execution (Async)
1. User clicks "Run Campaign"
2. Show: "Starting campaign..." (2s)
3. API returns 202 Accepted
4. Show: "Executing 60 prompts across 3 LLMs..."
5. Redirect to insights after 3-5 seconds
6. On insights page, show: "Analyzing results..."
7. Poll insights API every 5s until data appears
8. Show full dashboard

// Recommended Loading Messages
{
  generating: "Generating prompts for {brand}...",
  executing: "Running campaign: {completed}/{total} executions",
  analyzing: "Analyzing {count} responses...",
  loading: "Loading insights..."
}
```

---

## ⚠️ Error Handling

```javascript
// Standard Error Response
{
  success: false,
  error: "Error message"
}

// HTTP Status Codes
400 Bad Request    → Show validation error
404 Not Found      → Show "Resource not found"
500 Server Error   → Show "Something went wrong, please try again"

// User-Friendly Messages
const errorMessages = {
  'Brand is required': 'Please enter a brand name',
  'At least one competitor is required': 'Add at least one competitor',
  'no responses found': 'No data available yet. Run a campaign first.',
  'Failed to generate prompts': 'Unable to generate prompts. Please try again.'
}
```

---

## 🔌 API Response Times

| Endpoint | Expected Time | UI Action |
|----------|--------------|-----------|
| `/prompts/generate` | 5-15s | Show spinner with "Generating..." |
| `/execute/bulk` | Instant (202) | Show "Started", redirect after 2s |
| `/insights` | 1-3s | Show loading skeleton |
| `/analytics/*` | 2-5s | Show loading skeleton |

---

## 📱 Responsive Design Tips

### Mobile (< 768px)
- Stack metrics vertically
- Use carousel for prompt list
- Simplify competitive table (show only top metrics)
- Make charts touch-friendly

### Tablet (768px - 1024px)
- 2-column layout for metrics
- Full tables with horizontal scroll
- Collapsible sections

### Desktop (> 1024px)
- 3-column layout for metrics
- Side-by-side comparisons
- Full featured tables

---

## 🎯 Priority Fields for Minimum Viable Product (MVP)

### Must Have for MVP:
```javascript
{
  // Insights
  averageVisibility: number,
  mentionRate: number,
  totalResponses: number,
  sentimentBreakdown: object,
  
  // Prompts
  prompts[].id: string,
  prompts[].template: string,
  prompts[].promptType: string,
  
  // Competitive
  mainBrand.mentionRate: number,
  competitors[].mentionRate: number,
  yourRank: number,
  
  // Performance
  prompts[].effectivenessGrade: string,
  prompts[].avgVisibility: number,
  prompts[].mentionRate: number
}
```

### Add in Phase 2:
- Logo URLs
- Detailed breakdowns (by LLM, by category)
- Source analytics
- Recommendations
- Time-series trends

---

## 🔧 Development Checklist

### Phase 1: Core Functionality
- [ ] API client setup with error handling
- [ ] Prompt generation page
  - [ ] Form: brand, category, count
  - [ ] Loading state
  - [ ] Display prompts with checkboxes
- [ ] Campaign execution
  - [ ] LLM selection
  - [ ] Execute button
  - [ ] Loading indicator
- [ ] Basic insights dashboard
  - [ ] 3 key metrics
  - [ ] Sentiment chart
  - [ ] Competitor list

### Phase 2: Analytics
- [ ] Competitive benchmark page
  - [ ] Comparison table
  - [ ] Your rank display
- [ ] Prompt performance page
  - [ ] Sortable table
  - [ ] Grade badges
- [ ] Source analytics page
  - [ ] Top domains list

### Phase 3: Polish
- [ ] Logo integration with fallbacks
- [ ] Responsive design
- [ ] Empty states
- [ ] Error handling
- [ ] Loading skeletons
- [ ] Export/share functionality

---

## 📚 Related Documentation

- **Full API Spec:** See `API_INTEGRATION_FRONTEND.md`
- **Advanced Analytics:** See `ADVANCED_ANALYTICS.md`
- **Main Docs:** See `README.md`

---

**Quick Ref Version:** 1.0  
**Last Updated:** 2024-01-01

