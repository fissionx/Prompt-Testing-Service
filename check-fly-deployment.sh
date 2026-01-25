#!/bin/bash

# Script to check and fix Fly.io deployment issues for Gego

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

APP_NAME="gego"

echo -e "${BLUE}╔══════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   Gego Fly.io Deployment Checker       ║${NC}"
echo -e "${BLUE}╔══════════════════════════════════════════╗${NC}"
echo ""

# Check if flyctl is installed
if ! command -v flyctl &> /dev/null; then
    echo -e "${RED}❌ flyctl is not installed${NC}"
    echo "Install it from: https://fly.io/docs/hands-on/install-flyctl/"
    exit 1
fi

# Check if authenticated
if ! flyctl auth whoami &> /dev/null; then
    echo -e "${RED}❌ Not authenticated with Fly.io${NC}"
    echo "Run: flyctl auth login"
    exit 1
fi

echo -e "${GREEN}✓ Authenticated with Fly.io${NC}"
echo ""

# Check app status
echo -e "${BLUE}Checking app status...${NC}"
if flyctl status --app "${APP_NAME}" &> /dev/null; then
    echo -e "${GREEN}✓ App '${APP_NAME}' exists${NC}"
    flyctl status --app "${APP_NAME}"
else
    echo -e "${RED}❌ App '${APP_NAME}' does not exist${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}Checking secrets...${NC}"
SECRETS=$(flyctl secrets list --app "${APP_NAME}" 2>/dev/null || echo "")

if echo "$SECRETS" | grep -q "POSTGRESQL_URI"; then
    echo -e "${GREEN}✓ POSTGRESQL_URI secret is set${NC}"
    # Mask the URI for display
    MASKED_URI=$(echo "$SECRETS" | grep "POSTGRESQL_URI" | sed 's/.*POSTGRESQL_URI=\(.*\)/\1/' | sed 's/:[^:]*@/:***@/')
    echo "  URI: ${MASKED_URI}"
else
    echo -e "${RED}❌ POSTGRESQL_URI secret is NOT set${NC}"
    echo ""
    echo -e "${YELLOW}This is likely the cause of your 503 errors!${NC}"
    echo ""
    echo "To fix this, run:"
    echo "  flyctl secrets set POSTGRESQL_URI=\"your-postgresql-connection-string\" --app ${APP_NAME}"
    echo ""
    echo "Example:"
    echo "  flyctl secrets set POSTGRESQL_URI=\"postgresql://user:password@host:port/database?sslmode=require\" --app ${APP_NAME}"
    echo ""
    exit 1
fi

if echo "$SECRETS" | grep -q "NOSQL_DATABASE_PROVIDER"; then
    echo -e "${GREEN}✓ NOSQL_DATABASE_PROVIDER secret is set${NC}"
else
    echo -e "${YELLOW}⚠ NOSQL_DATABASE_PROVIDER secret is not set (optional)${NC}"
    echo "  This will default to 'postgresql' if POSTGRESQL_URI is set"
fi

echo ""
echo -e "${BLUE}Checking recent logs...${NC}"
echo ""
flyctl logs --app "${APP_NAME}" | tail -30

echo ""
echo -e "${BLUE}Checking health endpoint...${NC}"
HEALTH_URL="https://${APP_NAME}.fly.dev/api/v1/health"
echo "Testing: ${HEALTH_URL}"

if curl -s -f "${HEALTH_URL}" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Health check passed!${NC}"
    curl -s "${HEALTH_URL}" | jq '.' 2>/dev/null || curl -s "${HEALTH_URL}"
else
    echo -e "${RED}❌ Health check failed${NC}"
    echo ""
    echo "Common issues:"
    echo "1. POSTGRESQL_URI secret not set or incorrect"
    echo "2. Database connection failing"
    echo "3. App not starting properly"
    echo ""
    echo "Check logs with: flyctl logs --app ${APP_NAME}"
    echo "Check status with: flyctl status --app ${APP_NAME}"
fi

echo ""
echo -e "${GREEN}Check complete!${NC}"
