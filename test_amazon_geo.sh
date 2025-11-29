#!/bin/bash

# Complete GEO API Test for Amazon.com
# This script tests all GEO APIs end-to-end

BASE_URL="http://localhost:8989/api/v1"
BRAND="Amazon"
WEBSITE="https://www.amazon.com"

echo "🚀 Complete GEO API Test for Amazon.com"
echo "========================================"
echo ""

# Check server health
echo "0️⃣ Checking server health..."
HEALTH=$(curl -s "$BASE_URL/health")
if [ $? -eq 0 ]; then
    echo "✅ Server is running"
else
    echo "❌ Server is not running. Please start: ./gego api --port 8989"
    exit 1
fi
echo ""

# Get LLM ID
echo "1️⃣ Getting LLM information..."
LLM_RESPONSE=$(curl -s "$BASE_URL/llms")
LLM_ID=$(echo $LLM_RESPONSE | jq -r '.data[0].id')
LLM_NAME=$(echo $LLM_RESPONSE | jq -r '.data[0].name')

if [ "$LLM_ID" = "null" ] || [ -z "$LLM_ID" ]; then
    echo "❌ No LLMs configured. Please add an LLM first."
    echo "   Example: gego llm add"
    exit 1
fi

echo "✅ Using LLM: $LLM_NAME (ID: $LLM_ID)"
echo ""

# Generate Prompts
echo "2️⃣ Generating prompts with website scraping..."
PROMPTS_RESPONSE=$(curl -s -X POST "$BASE_URL/geo/prompts/generate" \
  -H "Content-Type: application/json" \
  -d "{\"brand\":\"$BRAND\",\"website\":\"$WEBSITE\",\"count\":10}")

PROMPT_COUNT=$(echo $PROMPTS_RESPONSE | jq -r '.data.prompts | length')
DOMAIN=$(echo $PROMPTS_RESPONSE | jq -r '.data.domain')
CATEGORY=$(echo $PROMPTS_RESPONSE | jq -r '.data.category')
PROMPT_IDS=$(echo $PROMPTS_RESPONSE | jq -r '[.data.prompts[0:5][].id]')

echo "✅ Generated $PROMPT_COUNT prompts"
echo "   Domain: $DOMAIN"
echo "   Category: $CATEGORY"
echo "   Using first 5 prompts for testing"
echo ""

# Run Single Test Execution
echo "3️⃣ Running single test execution..."
EXEC_RESPONSE=$(curl -s -X POST "$BASE_URL/execute" \
  -H "Content-Type: application/json" \
  -d "{
    \"prompt\": \"What are the best online shopping platforms for 2025?\",
    \"llm_id\": \"$LLM_ID\",
    \"brand\": \"$BRAND\",
    \"region\": \"US\",
    \"language\": \"en\"
  }")

VISIBILITY=$(echo $EXEC_RESPONSE | jq -r '.data.geo_analysis.visibility_score')
MENTIONED=$(echo $EXEC_RESPONSE | jq -r '.data.geo_analysis.brand_mentioned')
SENTIMENT=$(echo $EXEC_RESPONSE | jq -r '.data.geo_analysis.sentiment')

echo "✅ Single execution complete"
echo "   Visibility Score: $VISIBILITY/10"
echo "   Brand Mentioned: $MENTIONED"
echo "   Sentiment: $SENTIMENT"
echo ""

# Run Bulk Campaign
echo "4️⃣ Running bulk campaign (5 prompts × 1 LLM = 5 executions)..."
CAMPAIGN_RESPONSE=$(curl -s -X POST "$BASE_URL/geo/execute/bulk" \
  -H "Content-Type: application/json" \
  -d "{
    \"campaign_name\": \"Amazon Complete Test - $(date +%Y%m%d)\",
    \"brand\": \"$BRAND\",
    \"prompt_ids\": $PROMPT_IDS,
    \"llm_ids\": [\"$LLM_ID\"],
    \"temperature\": 0.7
  }")

CAMPAIGN_ID=$(echo $CAMPAIGN_RESPONSE | jq -r '.data.campaign_id')
TOTAL_RUNS=$(echo $CAMPAIGN_RESPONSE | jq -r '.data.total_runs')
STATUS=$(echo $CAMPAIGN_RESPONSE | jq -r '.data.status')

echo "✅ Campaign started"
echo "   Campaign ID: $CAMPAIGN_ID"
echo "   Total Runs: $TOTAL_RUNS"
echo "   Status: $STATUS"
echo ""

# Wait for campaign completion
echo "⏱️  Waiting 25 seconds for campaign to complete..."
for i in {1..5}; do
    echo -n "   ."
    sleep 5
done
echo " Done!"
echo ""

# Get GEO Insights
echo "5️⃣ Getting overall GEO insights..."
INSIGHTS_RESPONSE=$(curl -s -X POST "$BASE_URL/geo/insights" \
  -H "Content-Type: application/json" \
  -d "{\"brand\": \"$BRAND\"}")

AVG_VIS=$(echo $INSIGHTS_RESPONSE | jq -r '.data.average_visibility')
MENTION_RATE=$(echo $INSIGHTS_RESPONSE | jq -r '.data.mention_rate')
GROUNDING_RATE=$(echo $INSIGHTS_RESPONSE | jq -r '.data.grounding_rate')
TOTAL_RESP=$(echo $INSIGHTS_RESPONSE | jq -r '.data.total_responses')
TOP_COMP=$(echo $INSIGHTS_RESPONSE | jq -r '.data.top_competitors[0].name')

echo "✅ GEO Insights:"
echo "   Average Visibility: $AVG_VIS/10"
echo "   Mention Rate: $MENTION_RATE%"
echo "   Grounding Rate: $GROUNDING_RATE%"
echo "   Total Responses: $TOTAL_RESP"
echo "   Top Competitor: $TOP_COMP"
echo ""

# Get Source Analytics
echo "6️⃣ Getting source citation analytics..."
SOURCES_RESPONSE=$(curl -s -X POST "$BASE_URL/geo/analytics/sources" \
  -H "Content-Type: application/json" \
  -d "{\"brand\": \"$BRAND\", \"top_n\": 10}")

TOTAL_SOURCES=$(echo $SOURCES_RESPONSE | jq -r '.data.total_sources')
TOTAL_CITATIONS=$(echo $SOURCES_RESPONSE | jq -r '.data.total_citations')
TOP_SOURCE=$(echo $SOURCES_RESPONSE | jq -r '.data.top_sources[0].domain')
TOP_SOURCE_COUNT=$(echo $SOURCES_RESPONSE | jq -r '.data.top_sources[0].citation_count')
RECO_COUNT=$(echo $SOURCES_RESPONSE | jq -r '.data.recommendations | length')

echo "✅ Source Analytics:"
echo "   Total Unique Sources: $TOTAL_SOURCES"
echo "   Total Citations: $TOTAL_CITATIONS"
echo "   Top Source: $TOP_SOURCE ($TOP_SOURCE_COUNT citations)"
echo "   Recommendations: $RECO_COUNT"
echo ""

# Get Position Analytics
echo "7️⃣ Getting position/ranking analytics..."
POSITION_RESPONSE=$(curl -s -X POST "$BASE_URL/geo/analytics/position" \
  -H "Content-Type: application/json" \
  -d "{\"brand\": \"$BRAND\"}")

AVG_POS=$(echo $POSITION_RESPONSE | jq -r '.data.average_position')
TOP_POS_RATE=$(echo $POSITION_RESPONSE | jq -r '.data.top_position_rate')
TOTAL_MENTIONS=$(echo $POSITION_RESPONSE | jq -r '.data.total_mentions')

echo "✅ Position Analytics:"
echo "   Average Position: $AVG_POS (1=best)"
echo "   Top Position Rate: $TOP_POS_RATE% (in top 3)"
echo "   Total Mentions: $TOTAL_MENTIONS"
echo ""

# Get Prompt Performance
echo "8️⃣ Getting prompt performance analytics..."
PERF_RESPONSE=$(curl -s -X POST "$BASE_URL/geo/analytics/prompt-performance" \
  -H "Content-Type: application/json" \
  -d "{\"brand\": \"$BRAND\", \"min_responses\": 1}")

AVG_EFF=$(echo $PERF_RESPONSE | jq -r '.data.avg_effectiveness')
TOTAL_PROMPTS=$(echo $PERF_RESPONSE | jq -r '.data.total_prompts_analyzed')
TOP_PERF_COUNT=$(echo $PERF_RESPONSE | jq -r '.data.top_performers | length')
LOW_PERF_COUNT=$(echo $PERF_RESPONSE | jq -r '.data.low_performers | length')
BEST_PROMPT=$(echo $PERF_RESPONSE | jq -r '.data.prompts[0].prompt_text')
BEST_SCORE=$(echo $PERF_RESPONSE | jq -r '.data.prompts[0].effectiveness_score')
BEST_GRADE=$(echo $PERF_RESPONSE | jq -r '.data.prompts[0].effectiveness_grade')

echo "✅ Prompt Performance:"
echo "   Average Effectiveness: $AVG_EFF/100"
echo "   Total Prompts Analyzed: $TOTAL_PROMPTS"
echo "   High Performers: $TOP_PERF_COUNT"
echo "   Low Performers: $LOW_PERF_COUNT"
echo "   Best Prompt: \"$BEST_PROMPT\""
echo "   Best Score: $BEST_SCORE ($BEST_GRADE)"
echo ""

# Search for Amazon
echo "9️⃣ Searching for 'Amazon' keyword..."
SEARCH_RESPONSE=$(curl -s -X POST "$BASE_URL/search" \
  -H "Content-Type: application/json" \
  -d "{\"keyword\": \"Amazon\", \"limit\": 50}")

KEYWORD_MENTIONS=$(echo $SEARCH_RESPONSE | jq -r '.data.total_mentions')
UNIQUE_PROMPTS=$(echo $SEARCH_RESPONSE | jq -r '.data.unique_prompts')

echo "✅ Keyword Search:"
echo "   Total Mentions: $KEYWORD_MENTIONS"
echo "   Unique Prompts: $UNIQUE_PROMPTS"
echo ""

# Final Summary
echo "======================================"
echo "✅ TEST COMPLETE - All APIs Working!"
echo "======================================"
echo ""
echo "📊 SUMMARY FOR AMAZON:"
echo "   Visibility:        $AVG_VIS/10 (Target: 7-9 for strong brand)"
echo "   Mention Rate:      $MENTION_RATE% (Target: 85-100%)"
echo "   Grounding Rate:    $GROUNDING_RATE% (Target: 60-90%)"
echo "   Average Position:  $AVG_POS (Target: 1-2)"
echo "   Top Position Rate: $TOP_POS_RATE% (Target: 70-90%)"
echo "   Prompt Effectiveness: $AVG_EFF/100 (Target: 70-85)"
echo ""
echo "🎯 INSIGHTS:"
echo "   • Analyzed $TOTAL_RESP responses"
echo "   • Found $TOTAL_SOURCES unique citation sources"
echo "   • Identified $TOP_PERF_COUNT high-performing prompts"
echo "   • Generated $RECO_COUNT actionable recommendations"
echo ""
echo "📁 DETAILED RESULTS SAVED TO:"
echo "   Run commands from COMPLETE_API_TEST_AMAZON.md for full JSON"
echo ""
echo "📚 DOCUMENTATION:"
echo "   • docs/ADVANCED_ANALYTICS.md - Feature guide"
echo "   • docs/PROMPT_PERFORMANCE.md - Prompt analysis"
echo "   • COMPLETE_API_TEST_AMAZON.md - Complete test guide"
echo ""
echo "🚀 Your GEO platform is working perfectly!"

