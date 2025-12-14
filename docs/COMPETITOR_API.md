# Competitor Management API

## Overview

The Competitor Management API provides a complete flow for managing competitors in your GEO analytics:

1. **Suggest competitors** - AI-powered suggestions using LLM based on brand and website (cached to reduce LLM calls)
2. **Save competitors** - Store user-defined competitor list
3. **Get competitors** - Retrieve saved competitor list
4. **Use in analytics** - All competitive analytics automatically use saved competitors

---

## API Endpoints

### 1. Suggest Competitors (GET)

Suggests competitors for a brand using LLM based on brand info and website. First call uses LLM to generate suggestions, subsequent calls return cached suggestions (reducing LLM costs).

**Endpoint:** `GET /api/v1/geo/competitors/suggest`

**Query Parameters:**
- `brand` (required): Brand name
- `website` (optional): Brand's website URL (for richer context)
- `description` (optional): Brand description
- `category` (optional): Industry/category
- `forceRefresh` (optional): Force re-generation even if cached (default: false)

**Example Request:**

```bash
# Basic - suggest competitors for "Cursor"
curl -X GET "http://localhost:8080/api/v1/geo/competitors/suggest?brand=Cursor" \
  -H "Content-Type: application/json"

# With website (recommended for better suggestions)
curl -X GET "http://localhost:8080/api/v1/geo/competitors/suggest?brand=Cursor&website=https://cursor.sh" \
  -H "Content-Type: application/json"

# With full context
curl -X GET "http://localhost:8080/api/v1/geo/competitors/suggest?brand=Cursor&website=https://cursor.sh&description=AI-powered%20code%20editor&category=Developer%20Tools" \
  -H "Content-Type: application/json"

# Force refresh (regenerate using LLM)
curl -X GET "http://localhost:8080/api/v1/geo/competitors/suggest?brand=Cursor&website=https://cursor.sh&forceRefresh=true" \
  -H "Content-Type: application/json"
```

**Example Response:**

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
      "Replit",
      "JetBrains IDEs",
      "Sourcegraph Cody",
      "Amazon CodeWhisperer"
    ],
    "source": "llm",
    "message": "Found 8 competitors for Cursor"
  }
}
```

**Response Fields:**
- `brand`: The brand name
- `competitors`: Array of suggested competitor names (5-15 direct competitors)
- `source`: 
  - `"cached"` - Returned from cache (no LLM call)
  - `"llm"` - Freshly generated using LLM
- `message`: Informational message

---

### 2. Save Competitors (POST)

Saves user-defined competitors for a brand. Can be from suggestions or custom additions.

**Endpoint:** `POST /api/v1/geo/competitors`

**Request Body:**
- `brand` (required): Brand name
- `competitors` (required): Array of competitor names
- `source` (optional): "suggested", "custom", or "mixed" (default: "custom")

**Example Request:**

```bash
# Save competitors from suggestions
curl -X POST "http://localhost:8080/api/v1/geo/competitors" \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "Cursor",
    "competitors": [
      "VS Code",
      "GitHub Copilot",
      "Tabnine"
    ],
    "source": "suggested"
  }'

# Save custom competitors
curl -X POST "http://localhost:8080/api/v1/geo/competitors" \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "Cursor",
    "competitors": [
      "VS Code",
      "JetBrains IDEs",
      "Sublime Text",
      "Atom"
    ],
    "source": "custom"
  }'

# Save mixed competitors (some from suggestions, some custom)
curl -X POST "http://localhost:8080/api/v1/geo/competitors" \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "Cursor",
    "competitors": [
      "VS Code",
      "GitHub Copilot",
      "My Custom Tool"
    ],
    "source": "mixed"
  }'
```

**Example Response:**

```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "competitors": [
      "VS Code",
      "GitHub Copilot",
      "Tabnine"
    ],
    "source": "suggested",
    "savedAt": "2024-12-14T10:30:00Z",
    "message": "Successfully saved 3 competitors for Cursor"
  }
}
```

---

### 3. Get Saved Competitors (GET)

Retrieves the saved competitor list for a brand.

**Endpoint:** `GET /api/v1/geo/competitors`

**Query Parameters:**
- `brand` (required): Brand name

**Example Request:**

```bash
# Get saved competitors
curl -X GET "http://localhost:8080/api/v1/geo/competitors?brand=Cursor" \
  -H "Content-Type: application/json"
```

**Example Response:**

```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "competitors": [
      "VS Code",
      "GitHub Copilot",
      "Tabnine"
    ],
    "suggestedList": [
      "VS Code",
      "GitHub Copilot",
      "Tabnine",
      "Replit",
      "Codeium"
    ],
    "source": "suggested",
    "updatedAt": "2024-12-14T10:30:00Z"
  },
  "message": "Competitors retrieved successfully"
}
```

**Response Fields:**
- `competitors`: Current saved competitor list (what user selected)
- `suggestedList`: Original AI suggestions (for reference)
- `source`: How competitors were selected
- `updatedAt`: Last update time

**No Competitors Saved Response:**

```json
{
  "success": true,
  "data": {
    "brand": "Cursor",
    "competitors": [],
    "source": "none",
    "updatedAt": "2024-12-14T10:30:00Z"
  },
  "message": "No competitors saved for this brand"
}
```

---

### 4. Delete Competitors (DELETE)

Deletes the saved competitor list for a brand.

**Endpoint:** `DELETE /api/v1/geo/competitors`

**Query Parameters:**
- `brand` (required): Brand name

**Example Request:**

```bash
# Delete saved competitors
curl -X DELETE "http://localhost:8080/api/v1/geo/competitors?brand=Cursor" \
  -H "Content-Type: application/json"
```

**Example Response:**

```json
{
  "success": true,
  "message": "Competitors deleted successfully"
}
```

---

## Integration with Competitive Analytics

Once you save competitors, all competitive analytics endpoints automatically use them:

### Competitive Benchmark

```bash
# Without saved competitors - auto-detects from responses
curl -X POST "http://localhost:8080/api/v1/geo/analytics/competitive" \
  -H "Content-Type: application/json" \
  -d '{
    "mainBrand": "Cursor"
  }'

# With saved competitors - automatically uses saved list
# (Same request, but uses your saved competitors!)
curl -X POST "http://localhost:8080/api/v1/geo/analytics/competitive" \
  -H "Content-Type: application/json" \
  -d '{
    "mainBrand": "Cursor"
  }'

# Override with specific competitors (ignores saved list)
curl -X POST "http://localhost:8080/api/v1/geo/analytics/competitive" \
  -H "Content-Type: application/json" \
  -d '{
    "mainBrand": "Cursor",
    "competitors": ["VS Code", "Sublime Text"]
  }'
```

### Competitor Matrix

```bash
# Uses saved competitors automatically
curl -X POST "http://localhost:8080/api/v1/geo/analytics/competitor-matrix" \
  -H "Content-Type: application/json" \
  -d '{
    "mainBrand": "Cursor"
  }'
```

### Trend Comparison

```bash
# Uses saved competitors automatically
curl -X POST "http://localhost:8080/api/v1/geo/analytics/trend-comparison" \
  -H "Content-Type: application/json" \
  -d '{
    "mainBrand": "Cursor",
    "metric": "visibility"
  }'
```

---

## Complete Workflow Example

Here's a complete flow from suggesting to using competitors:

```bash
# Step 1: Get competitor suggestions using LLM (first time - calls LLM)
curl -X GET "http://localhost:8080/api/v1/geo/competitors/suggest?brand=Cursor&website=https://cursor.sh"

# Response shows LLM-suggested competitors based on brand and website

# Step 2: Save selected competitors (from suggestions or custom)
curl -X POST "http://localhost:8080/api/v1/geo/competitors" \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "Cursor",
    "competitors": ["VS Code", "GitHub Copilot", "Tabnine"],
    "source": "suggested"
  }'

# Step 3: Verify saved competitors
curl -X GET "http://localhost:8080/api/v1/geo/competitors?brand=Cursor"

# Step 4: Run competitive analysis (automatically uses saved competitors)
curl -X POST "http://localhost:8080/api/v1/geo/analytics/competitive" \
  -H "Content-Type: application/json" \
  -d '{
    "mainBrand": "Cursor"
  }'

# Step 5: Get suggestions again (returns cached, no LLM call!)
curl -X GET "http://localhost:8080/api/v1/geo/competitors/suggest?brand=Cursor"

# Step 6: Update competitors
curl -X POST "http://localhost:8080/api/v1/geo/competitors" \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "Cursor",
    "competitors": ["VS Code", "Sublime Text", "Atom", "Nova"],
    "source": "mixed"
  }'

# Step 7: Delete competitors (if needed)
curl -X DELETE "http://localhost:8080/api/v1/geo/competitors?brand=Cursor"
```

---

## Error Handling

### Missing Brand Parameter

```bash
curl -X GET "http://localhost:8080/api/v1/geo/competitors/suggest"
```

**Response:**
```json
{
  "success": false,
  "error": "Brand parameter is required"
}
```

### Empty Competitors List

```bash
curl -X POST "http://localhost:8080/api/v1/geo/competitors" \
  -H "Content-Type: application/json" \
  -d '{
    "brand": "Cursor",
    "competitors": []
  }'
```

**Response:**
```json
{
  "success": false,
  "error": "At least one competitor is required"
}
```

### No Response Data Available

```bash
curl -X GET "http://localhost:8080/api/v1/geo/competitors/suggest?brand=NewBrand"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "brand": "NewBrand",
    "competitors": [],
    "source": "computed",
    "message": "No competitors found in responses. Execute some prompts first."
  }
}
```

---

## Key Features

### 1. **LLM-Powered Suggestions with Caching**
- First call to `/suggest` uses LLM to generate competitor suggestions
- Subsequent calls return cached suggestions instantly (no LLM cost!)
- Use `forceRefresh=true` to regenerate if needed
- Website scraping enriches context for better suggestions

### 2. **Flexible Competitor Selection**
- Use LLM-suggested competitors
- Add custom competitors
- Mix both approaches

### 3. **Automatic Integration**
- Save once, use everywhere
- All competitive analytics respect saved competitors
- Can override with specific competitors per request

### 4. **Fallback Behavior**
- No saved competitors? Auto-detects from existing responses
- Gradual migration - works with or without saved data
- Backwards compatible with existing analytics

---

## Testing Checklist

- [ ] Suggest competitors for a brand with website
- [ ] Verify LLM suggestions are relevant
- [ ] Verify suggestions are cached (second call is instant, no LLM call)
- [ ] Save competitors from suggestions
- [ ] Get saved competitors
- [ ] Run competitive benchmark (uses saved competitors)
- [ ] Update competitor list
- [ ] Run analytics again (uses updated list)
- [ ] Override with specific competitors in analytics request
- [ ] Delete competitors
- [ ] Verify analytics falls back to auto-detection
- [ ] Test force refresh regenerates suggestions

---

## Notes

- **Storage**: Competitors are stored in MongoDB (NoSQL database)
- **One per brand**: Each brand can have one competitor list
- **Updates**: Saving new competitors replaces the old list
- **Suggestions preserved**: Original LLM suggestions are kept for reference even after saving custom list
- **Website Context**: Providing website URL significantly improves suggestion quality
- **5-15 Competitors**: LLM returns 5-15 direct competitors sorted by relevance
- **LLM Provider**: Uses Google (preferred) or any available LLM provider

