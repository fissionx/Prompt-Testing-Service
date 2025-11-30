# Quick Deployment Commands

## Step 1: Verify Your Account
Visit: https://fly.io/high-risk-unlock

## Step 2: Run Automated Deployment Script

```bash
./deploy-to-fly.sh
```

This script will:
- Create the Fly.io app
- Set up MongoDB connection secrets
- Deploy the application
- Test the health endpoint

## OR: Manual Deployment (Step by Step)

### 2.1: Create the App (if running manually)

```bash
cd /Users/senyarav/workspace/opensource/gego
flyctl launch --no-deploy --copy-config --name gego
```

### 2.2: Set MongoDB Secret

```bash
flyctl secrets set MONGODB_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/" -a gego
```

### 2.3: Deploy

```bash
flyctl deploy
```

### 2.4: Check Status

```bash
flyctl status
flyctl logs
```

### 2.5: Test the API

```bash
# Health check
curl https://gego.fly.dev/api/v1/health

# List LLMs
curl https://gego.fly.dev/api/v1/llms

# Get stats
curl https://gego.fly.dev/api/v1/stats
```

## MongoDB Atlas Setup

**Important**: Ensure your MongoDB Atlas cluster allows connections from Fly.io:

1. Go to MongoDB Atlas Dashboard: https://cloud.mongodb.com/
2. Navigate to: Network Access → Add IP Address
3. Add: `0.0.0.0/0` (Allow from anywhere) for development
4. Click "Confirm"

## Environment Variables Already Configured

These are set in `fly.toml`:
- `GEGO_ENV=dev` - Development mode
- `MONGODB_DATABASE=gego` - Database name  
- `CORS_ORIGIN=*` - Allow all origins
- Data persistence via Fly.io volume

## After Deployment

Your application will be available at:
- **API**: https://gego.fly.dev/api/v1
- **Health**: https://gego.fly.dev/api/v1/health

## Troubleshooting

If deployment fails:

```bash
# View logs
flyctl logs

# Check app status
flyctl status

# Restart app
flyctl apps restart gego

# SSH into the container
flyctl ssh console

# Check MongoDB connection
flyctl ssh console -C "env | grep MONGODB"
```

## Useful Commands

```bash
# View real-time logs
flyctl logs -a gego

# Scale the app
flyctl scale count 1 -a gego

# View secrets (names only)
flyctl secrets list -a gego

# Update MongoDB URI
flyctl secrets set MONGODB_URI="new-uri" -a gego

# Open app in browser
flyctl open -a gego

# View dashboard
flyctl dashboard -a gego
```

