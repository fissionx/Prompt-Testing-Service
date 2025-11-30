# Fly.io Deployment Guide for Gego

This guide provides step-by-step instructions for deploying Gego to Fly.io with MongoDB Atlas integration.

## Prerequisites

1. ✅ Fly.io account (already authenticated as senthil2017tce@gmail.com)
2. ✅ MongoDB Atlas cluster and connection string
3. ⚠️ **REQUIRED**: Verify your Fly.io account at https://fly.io/high-risk-unlock

## Step 1: Account Verification

Before you can deploy, you need to verify your Fly.io account:

1. Visit: https://fly.io/high-risk-unlock
2. Complete the verification process (usually requires ID or credit card)
3. Wait for verification approval (usually instant)

## Step 2: Set MongoDB Connection String

After your account is verified, set up the MongoDB Atlas connection as a secret:

```bash
# Set your MongoDB Atlas connection string
flyctl secrets set MONGODB_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/" -a gego
```

**Important**: Replace the connection string with your actual MongoDB Atlas URI if different.

## Step 3: Deploy the Application

Once secrets are set, deploy the app:

```bash
# Deploy to Fly.io
flyctl deploy
```

This will:
- Build the Docker image
- Push it to Fly.io registry
- Create a persistent volume for SQLite data
- Start the app with MongoDB Atlas connection
- Set up health checks

## Step 4: Verify Deployment

After deployment completes:

```bash
# Check app status
flyctl status

# Check app logs
flyctl logs

# Open the app in your browser
flyctl open

# Test the API health endpoint
curl https://gego.fly.dev/api/v1/health
```

## Application URLs

- **API Base URL**: `https://gego.fly.dev/api/v1`
- **Health Check**: `https://gego.fly.dev/api/v1/health`
- **List LLMs**: `https://gego.fly.dev/api/v1/llms`
- **List Prompts**: `https://gego.fly.dev/api/v1/prompts`

## Environment Configuration

The following environment variables are already configured in `fly.toml`:

- `GEGO_ENV=dev` - Use development/cloud mode
- `MONGODB_DATABASE=gego` - Database name
- `CORS_ORIGIN=*` - Allow all origins
- `GEGO_CONFIG_PATH=/app/config/config.yaml` - Config file path
- `GEGO_DATA_PATH=/app/data` - Data directory (persisted volume)
- `GEGO_LOG_PATH=/app/logs` - Logs directory

## MongoDB Atlas Configuration

Ensure your MongoDB Atlas cluster is properly configured:

1. **Network Access**: Add Fly.io's IP ranges or use `0.0.0.0/0` (allow from anywhere)
   - Go to MongoDB Atlas → Network Access → Add IP Address
   - Add `0.0.0.0/0` or specific Fly.io IPs

2. **Database User**: Ensure the user has read/write permissions
   - Go to MongoDB Atlas → Database Access
   - Verify user permissions

3. **Connection String Format**:
   ```
   mongodb+srv://<username>:<password>@<cluster>.mongodb.net/
   ```

## Persistent Storage

The app uses a Fly.io volume for SQLite data storage:
- **Mount Point**: `/app/data`
- **Size**: 1GB
- **Volume Name**: `gego_data`

This ensures your SQLite database (LLM configs, schedules) persists across deployments.

## Scaling

To scale your application:

```bash
# Scale to 2 instances
flyctl scale count 2

# Scale VM resources
flyctl scale vm shared-cpu-2x --memory 1024
```

## Monitoring

Monitor your application:

```bash
# View real-time logs
flyctl logs

# View metrics
flyctl dashboard

# SSH into the machine
flyctl ssh console
```

## Troubleshooting

### Connection Issues

If you get connection errors:

```bash
# Check if app is running
flyctl status

# View logs for errors
flyctl logs --tail

# Restart the app
flyctl apps restart gego
```

### MongoDB Connection Errors

If MongoDB connection fails:

1. **Verify IP Whitelist**: Ensure `0.0.0.0/0` is added in MongoDB Atlas Network Access
2. **Check Credentials**: Verify the connection string is correct
3. **Update Secret**: Update the MongoDB URI secret if needed
   ```bash
   flyctl secrets set MONGODB_URI="your-new-uri" -a gego
   ```

### Volume Issues

If SQLite data isn't persisting:

```bash
# List volumes
flyctl volumes list

# Check volume status
flyctl volumes show gego_data
```

## Updating the Application

To deploy updates:

```bash
# Make your code changes
git add .
git commit -m "Your changes"

# Deploy
flyctl deploy

# Monitor the deployment
flyctl logs
```

## Cost Optimization

Fly.io provides generous free tier:
- 3 shared-cpu-1x machines with 256MB RAM (free)
- 160GB outbound data transfer (free)
- Additional resources are billed

Current configuration uses:
- 1 machine (shared-cpu-1x, 512MB RAM)
- This is within free tier for hobby projects

## Security Best Practices

1. **Rotate Secrets**: Regularly rotate your MongoDB credentials
2. **CORS Configuration**: Update `CORS_ORIGIN` in `fly.toml` to restrict origins in production
3. **Network Access**: Restrict MongoDB Atlas network access to specific IPs when possible
4. **Monitoring**: Set up alerts for unusual activity

## Quick Reference Commands

```bash
# Deploy
flyctl deploy

# View logs
flyctl logs

# Check status
flyctl status

# Set secrets
flyctl secrets set KEY=VALUE

# List secrets
flyctl secrets list

# Scale
flyctl scale count N

# SSH access
flyctl ssh console

# Open in browser
flyctl open

# Destroy app (careful!)
flyctl apps destroy gego
```

## API Testing

Once deployed, test the API:

```bash
# Health check
curl https://gego.fly.dev/api/v1/health

# List LLMs
curl https://gego.fly.dev/api/v1/llms

# Create an LLM (example)
curl -X POST https://gego.fly.dev/api/v1/llms \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GPT-4",
    "provider": "openai",
    "model": "gpt-4",
    "api_key": "sk-...",
    "enabled": true
  }'

# Get stats
curl https://gego.fly.dev/api/v1/stats
```

## Support

If you encounter issues:

1. Check the logs: `flyctl logs`
2. Review the Fly.io documentation: https://fly.io/docs/
3. Check MongoDB Atlas status: https://status.mongodb.com/
4. Contact Fly.io support: https://community.fly.io/

## Next Steps

After successful deployment:

1. Initialize the database with your LLM providers
2. Create prompts for tracking
3. Set up schedules for automated execution
4. Monitor the analytics dashboard
5. Consider setting up a frontend application to visualize the data

---

**Made with ❤️ for the GEO tracking community**

