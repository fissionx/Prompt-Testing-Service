#!/bin/bash

# Gego Deployment Script for Fly.io
# This script automates the deployment process to Fly.io with MongoDB Atlas

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔══════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   Gego Fly.io Deployment Script         ║${NC}"
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

# Check if app exists
APP_NAME="gego"
if flyctl apps list | grep -q "^${APP_NAME}"; then
    echo -e "${GREEN}✓ App '${APP_NAME}' exists${NC}"
    DEPLOY_ONLY=true
else
    echo -e "${YELLOW}⚠ App '${APP_NAME}' does not exist${NC}"
    DEPLOY_ONLY=false
fi

# Database configuration (PostgreSQL or MongoDB)
if [ "$DEPLOY_ONLY" = false ]; then
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  Database Configuration${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    echo "Select database type:"
    echo "1) PostgreSQL (Recommended)"
    echo "2) MongoDB"
    echo ""
    read -p "Choice [1]: " DB_CHOICE
    DB_CHOICE=${DB_CHOICE:-1}
    
    if [ "$DB_CHOICE" = "1" ]; then
        # PostgreSQL configuration
        echo ""
        echo -e "${BLUE}PostgreSQL Configuration${NC}"
        echo ""
        echo "Please enter your PostgreSQL connection URI"
        echo "Example: postgresql://user:password@host:port/database?sslmode=require"
        echo ""
        read -p "PostgreSQL URI: " POSTGRESQL_URI
        
        if [ -z "$POSTGRESQL_URI" ]; then
            echo -e "${RED}❌ PostgreSQL URI is required${NC}"
            exit 1
        fi
        
        DB_TYPE="postgresql"
        DB_URI="$POSTGRESQL_URI"
    else
        # MongoDB configuration
        echo ""
        echo -e "${BLUE}MongoDB Configuration${NC}"
        echo ""
        DEFAULT_MONGODB_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/"
        
        echo "Please enter your MongoDB Atlas connection URI"
        echo -e "${YELLOW}(Press Enter to use default from docs)${NC}"
        echo "Default: ${DEFAULT_MONGODB_URI}"
        echo ""
        read -p "MongoDB URI: " MONGODB_URI
        
        if [ -z "$MONGODB_URI" ]; then
            MONGODB_URI="$DEFAULT_MONGODB_URI"
            echo -e "${GREEN}Using default MongoDB URI${NC}"
        fi
        
        DB_TYPE="mongodb"
        DB_URI="$MONGODB_URI"
    fi
    
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  Creating Fly.io App${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    # Try to create the app
    if flyctl launch --no-deploy --copy-config --name "${APP_NAME}"; then
        echo -e "${GREEN}✓ App created successfully${NC}"
    else
        echo -e "${RED}❌ Failed to create app${NC}"
        echo ""
        echo -e "${YELLOW}This might be due to:${NC}"
        echo "1. Account verification required: https://fly.io/high-risk-unlock"
        echo "2. App name already taken"
        echo "3. Network issues"
        echo ""
        echo "Please resolve the issue and run this script again."
        exit 1
    fi
    
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  Setting Secrets${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    # Set database secrets based on type
    if [ "$DB_TYPE" = "postgresql" ]; then
        if flyctl secrets set POSTGRESQL_URI="${DB_URI}" NOSQL_DATABASE_PROVIDER="postgresql" -a "${APP_NAME}"; then
            echo -e "${GREEN}✓ PostgreSQL URI secret set${NC}"
            echo -e "${GREEN}✓ Database provider set to postgresql${NC}"
        else
            echo -e "${RED}❌ Failed to set PostgreSQL secrets${NC}"
            exit 1
        fi
    else
        if flyctl secrets set MONGODB_URI="${DB_URI}" -a "${APP_NAME}"; then
            echo -e "${GREEN}✓ MongoDB URI secret set${NC}"
        else
            echo -e "${RED}❌ Failed to set MongoDB URI secret${NC}"
            exit 1
        fi
    fi
fi

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Deploying Application${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Deploy the app (remote builder uses Fly's cache for faster builds)
if flyctl deploy --remote-only; then
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  ✓ Deployment Successful!${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${BLUE}Application URLs:${NC}"
    echo "  • API Base:    https://${APP_NAME}.fly.dev/api/v1"
    echo "  • Health:      https://${APP_NAME}.fly.dev/api/v1/health"
    echo "  • LLMs:        https://${APP_NAME}.fly.dev/api/v1/llms"
    echo "  • Prompts:     https://${APP_NAME}.fly.dev/api/v1/prompts"
    echo ""
    echo -e "${BLUE}Useful Commands:${NC}"
    echo "  • View logs:   flyctl logs"
    echo "  • App status:  flyctl status"
    echo "  • SSH access:  flyctl ssh console"
    echo "  • Dashboard:   flyctl dashboard"
    echo ""
    echo -e "${YELLOW}Testing the deployment...${NC}"
    sleep 5
    
    # Test health endpoint
    HEALTH_URL="https://${APP_NAME}.fly.dev/api/v1/health"
    echo "Testing: ${HEALTH_URL}"
    
    if curl -s -f "${HEALTH_URL}" > /dev/null; then
        echo -e "${GREEN}✓ Health check passed!${NC}"
    else
        echo -e "${YELLOW}⚠ Health check failed (app might still be starting)${NC}"
        echo "Check logs with: flyctl logs"
    fi
    
else
    echo -e "${RED}❌ Deployment failed${NC}"
    echo ""
    echo "Check the error messages above and:"
    echo "1. Review the logs: flyctl logs"
    echo "2. Check app status: flyctl status"
    echo "3. Verify database connection string"
    echo "4. For PostgreSQL: Ensure SSL mode matches your database"
    echo "5. For MongoDB: Ensure MongoDB Atlas IP whitelist includes 0.0.0.0/0"
    exit 1
fi

echo ""
echo -e "${GREEN}🚀 Deployment complete!${NC}"
echo ""



