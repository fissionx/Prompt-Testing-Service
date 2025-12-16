# Geo Analytics API Documentation

## Table of Contents

1. [Prompt Performance Analytics](#prompt-performance-analytics)
2. [Dashboard Overview Metrics](#dashboard-overview-metrics)
3. [Trend Comparison](#trend-comparison)

---

## Prompt Performance Analytics

**Endpoint:** `API: geo/analytics/prompt-performance`

### Field Explanations

#### **promptType**
- **Description:** Category of the prompt question
- **Possible values:** `"what"`, `"how"`, `"comparison"`, `"top_best"`, `"brand"`, `"custom"`

#### **category**
- **Description:** Business category or industry classification for the prompt

#### **avgVisibility**
- **Description:** Average visibility score (0-10) across all responses for this prompt
- **Formula:** `Sum of all visibility scores / Total responses`

#### **avgPosition**
- **Description:** Average ranking position when your brand was mentioned (lower is better)
- **Formula:** `Sum of all position numbers / Number of times brand was mentioned`
- **Note:** Only calculated when brand was mentioned with a position

#### **mentionRate**
- **Description:** Percentage of responses where your brand was mentioned
- **Formula:** `(Number of responses with brand mention / Total responses) × 100`

#### **topPositionRate**
- **Description:** Percentage of mentions that appeared in top 3 positions
- **Formula:** `(Number of mentions in positions 1-3 / Total mentions) × 100`
- **Note:** Only calculated when brand was mentioned with a position

#### **avgSentiment**
- **Description:** Average sentiment score (-1 to +1) across all responses
- **Values:** Positive = +1, Neutral = 0, Negative = -1
- **Note:** Only calculated when sentiment data is available

#### **totalResponses**
- **Description:** Total number of AI responses analyzed for this prompt

#### **brandMentions**
- **Description:** Total number of times your brand was mentioned across all responses

#### **effectivenessScore**
- **Description:** Composite performance score (0-100) based on visibility, mention rate, top position rate, and average position
- **Formula:** Weighted combination
  - Visibility: 40%
  - Mention Rate: 30%
  - Top Position Rate: 20%
  - Position: 10%

#### **effectivenessGrade**
- **Description:** Letter grade based on effectiveness score
- **Possible values:** `"A+"`, `"A"`, `"A-"`, `"B+"`, `"B"`, `"B-"`, `"C+"`, `"C"`, `"C-"`, `"D+"`, `"D"`, `"F"`

**Grade Ranges:**

| Grade | Score Range |
|-------|-------------|
| A+    | 90-100      |
| A     | 85-89       |
| A-    | 80-84       |
| B+    | 75-79       |
| B     | 70-74       |
| B-    | 65-69       |
| C+    | 60-64       |
| C     | 55-59       |
| C-    | 50-54       |
| D+    | 45-49       |
| D     | 40-44       |
| F     | 0-39        |

#### **status**
- **Description:** Performance category based on effectiveness score
- **Possible values:** `"high_performer"`, `"average_performer"`, `"low_performer"`, `"very_low_performer"`

**Status Ranges:**

| Status | Score Range |
|--------|-------------|
| high_performer | ≥ 75 |
| average_performer | 50-74 |
| low_performer | 30-49 |
| very_low_performer | < 30 |

### Summary Example

```json
{
  "value": 3,
  "change": 0,
  "trend": "stable",
  "rank": 0,
  "totalBrands": 3
}
```

- **value:** `3` = Average visibility score of 3 out of 10
- **change:** `0` = Not calculated (placeholder)
- **trend:** `"stable"` = Not calculated (placeholder)
- **rank:** `0` = Not ranked (no competitors or insufficient data)
- **totalBrands:** `3` = 3 brands total (you + 2 competitors), but ranking wasn't computed

---

## Dashboard Overview Metrics

**Endpoint:** `/api/v1/geo/dashboard/overview?brand=Cursor`

### Metric Fields

- **value:** Current value
- **change:** Percentage change from previous period (0 means not applicable)
- **trend:** Possible values: `"up"`, `"down"`, `"stable"` from previous run
- **totalBrands:** Total brands in comparison
- **rank:** Current rank (e.g., 1/5). 0 means not applicable

### Calculation Details

#### 1. **value**

Calculated from all responses in the selected time period:

##### **For "visibility" metric:**
- **Formula:** `(Number of times brand was mentioned / Total responses) × 100`
- **Example:** If mentioned in 30 out of 100 responses, value = 30%
- **Meaning:** Percentage of responses where your brand appeared

##### **For "sentiment" metric:**
- **Formula:** `(Average sentiment score + 1) × 50`
- **Sentiment scores:** Positive = +1, Neutral = 0, Negative = -1
- **Conversion:** Converts from -1 to 1 scale to 0-100 scale
- **Example:** If average sentiment is 0.4, value = (0.4 + 1) × 50 = 70
- **Meaning:** Average sentiment score (0 = very negative, 50 = neutral, 100 = very positive)

##### **For "position" metric:**
- **Formula:** `Sum of all position numbers / Number of times brand was mentioned`
- **Example:** If mentioned 3 times at positions 1, 2, and 3, value = (1+2+3)/3 = 2.0
- **Meaning:** Average ranking position when mentioned (lower is better)

##### **For "groundingRate" metric:**
- **Formula:** `(Number of responses with grounding sources / Total responses) × 100`
- **Example:** If 25 out of 100 responses had grounding sources, value = 25%
- **Meaning:** Percentage of responses that included citation sources

#### 2. **change**
- **Current status:** Set to 0 (not calculated yet)
- **Intended meaning:** Percentage change from the previous period
- **Future calculation:** `((Current period value - Previous period value) / Previous period value) × 100`
- **Example:** If previous was 25% and current is 30%, change would be +20%

#### 3. **trend**
- **Current status:** Set to "stable" (not calculated yet)
- **Possible values:** `"up"`, `"down"`, or `"stable"`
- **Intended meaning:** Direction indicator based on change
  - `"up"` = improving
  - `"down"` = declining
  - `"stable"` = no significant change

#### 4. **rank**
- **Formula:** Your position among all brands (including competitors) for this metric
- **Calculation:**
  1. Get competitors (saved competitors or auto-detected from responses)
  2. Calculate the same metric for all brands
  3. Sort brands by metric value (higher is better for visibility/sentiment/grounding, lower is better for position)
  4. Find your position in the sorted list
- **Example:** If you have 30% visibility and competitors have 40% and 20%, you rank 2nd out of 3
- **Note:** Returns 0 if no competitors are found or if there's insufficient data for ranking

#### 5. **totalBrands**
- **Formula:** Total number of brands being compared (your brand + all competitors)
- **Example:** If you have 2 competitors, totalBrands = 3 (you + 2 competitors)
- **Note:** Returns 0 if no competitors are found

### Example Interpretation

For the example: `value: 3, change: 0, trend: "stable", rank: 0, totalBrands: 3`

- **value:** `3` = Your metric value (e.g., 3% visibility, or position 3, depending on the metric)
- **change:** `0` = No change calculated (feature not implemented yet)
- **trend:** `"stable"` = Default value (not calculated yet)
- **rank:** `0` = Not ranked (either no competitors found, or insufficient data for ranking)
- **totalBrands:** `3` = There are 3 brands total (you + 2 competitors), but ranking wasn't calculated

> **Note:** rank is 0 even though totalBrands is 3, which suggests competitors exist but ranking wasn't computed (possibly due to insufficient data for all brands).

---

## Trend Comparison

**Endpoint:** `api/v1/geo/analytics/trend-comparison`

### Request Parameters

```json
{
  "granularity": "monthly",
  "metric": "visibility"
}
```

- **granularity:** `"daily"`, `"weekly"`, `"monthly"`
- **metric:** `"visibility"`, `"sentiment"`, `"position"`

### Calculation Details

#### 1. **value** (in each values array item)

Each value is computed for one time period (day/week/month) based on the selected metric:

##### **For "visibility" metric:**
- **Formula:** `(Number of times brand was mentioned / Total responses in that period) × 100`
- **Example:** If your brand was mentioned 30 times out of 100 responses, visibility = 30%
- **Meaning:** Percentage of responses where the brand appeared

##### **For "sentiment" metric:**
- **Formula:** Average sentiment score converted to 0-100 scale
- **Sentiment scores:** Positive = +1, Neutral = 0, Negative = -1
- **Conversion:** `(average + 1) × 50` to get 0-100 (0 = very negative, 50 = neutral, 100 = very positive)
- **Example:** If average sentiment is 0.5, value = (0.5 + 1) × 50 = 75

##### **For "position" metric:**
- **Formula:** `Total position numbers / Number of times brand was mentioned`
- **Example:** If mentioned 3 times at positions 1, 2, and 3, value = (1+2+3)/3 = 2.0
- **Meaning:** Average ranking position (lower is better)

#### 2. **currentValue**
- **Definition:** The most recent value in the values array (last time period)
- **Example:** If values has `[30.5, 35.2, 28.1, 40.0]`, then currentValue = 40.0
- **Meaning:** The latest metric value

#### 3. **change**
- **Formula:** `((Last value - First value) / First value) × 100`
- **Example:** If first value is 30.0 and last value is 40.0, change = ((40-30)/30) × 100 = +33.33%
- **Meaning:** Percentage change from the first period to the last
  - Positive = improvement
  - Negative = decline
- **Note:** Only calculated if there are at least 2 values and the first value is greater than 0

### Example Response

If you have weekly data for December 2025:

```json
{
  "values": [
    {"value": 30.5, "date": "2025-12-01"},  // Week 1
    {"value": 35.2, "date": "2025-12-08"},  // Week 2
    {"value": 40.0, "date": "2025-12-15"}   // Week 3
  ],
  "currentValue": 40.0,  // Last value (Week 3)
  "change": 31.15        // ((40.0 - 30.5) / 30.5) × 100 = +31.15% improvement
}
```

This shows a **31.15% improvement** from the first week to the last week.

---

## Summary

The effectiveness score combines multiple metrics to give an overall performance rating, with grades and status categories to help identify top-performing prompts. All metrics are designed to provide actionable insights into brand performance across AI-generated responses.