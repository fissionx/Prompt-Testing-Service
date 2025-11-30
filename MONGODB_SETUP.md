# MongoDB Setup - Quick Reference

## Your MongoDB Atlas Connection

**Connection String:**
```
mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/
```

**Your IP Address (for Atlas Whitelist):**
```
106.222.202.9
```

## Quick Start

### Option 1: Using Helper Scripts (Easiest)

```bash
# Run with local MongoDB
./scripts/run-local.sh api start

# Run with cloud MongoDB Atlas
./scripts/run-dev.sh api start
```

### Option 2: Using Environment Variables

```bash
# Run with local MongoDB
GEGO_ENV=local gego api start

# Run with cloud MongoDB Atlas
GEGO_ENV=dev \
MONGODB_CLOUD_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/" \
gego api start
```

### Option 3: Export and Run

```bash
# For local development
export GEGO_ENV=local
gego api start

# For cloud development
export GEGO_ENV=dev
export MONGODB_CLOUD_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/"
gego api start
```

## MongoDB Atlas Setup Steps

1. **Whitelist Your IP Address:**
   - Go to [MongoDB Atlas Dashboard](https://cloud.mongodb.com/)
   - Click on "Network Access" in the left sidebar
   - Click "Add IP Address" button
   - Enter your IP: `106.222.202.9`
   - Or select "Allow Access from Anywhere" (less secure, for development only)
   - Click "Confirm"

2. **Verify Connection:**
   ```bash
   ./scripts/run-dev.sh api start
   ```

3. **Check the logs** to confirm connection:
   - Should see: "Connected to database: mongodb+srv://..."

## Environment Variables Reference

| Variable | Purpose | Example |
|----------|---------|---------|
| `GEGO_ENV` | Switch environments | `local`, `dev`, `prod` |
| `MONGODB_URI` | Direct URI override | `mongodb+srv://...` |
| `MONGODB_CLOUD_URI` | Cloud URI for dev | `mongodb+srv://...` |
| `MONGODB_DATABASE` | Database name | `gego` |

## Common Commands

```bash
# Initialize with local MongoDB
GEGO_ENV=local gego init

# Add an LLM (works with any environment)
gego llm add openai --api-key sk-xxx --model gpt-4

# Create a prompt
gego prompt create --template "What are the best {category} brands?"

# Start API server (local)
./scripts/run-local.sh api start

# Start API server (cloud)
./scripts/run-dev.sh api start

# Run a schedule
GEGO_ENV=dev gego scheduler run --schedule-id <id>
```

## Troubleshooting

### "Connection refused" error
- **Local:** Check if MongoDB is running: `sudo systemctl status mongod` or `brew services list`
- **Cloud:** Check your internet connection

### "Authentication failed" error
- Verify your MongoDB Atlas username/password
- Check that your IP is whitelisted in Network Access

### "IP not whitelisted" error
- Add your IP `106.222.202.9` to MongoDB Atlas Network Access
- Or temporarily use `0.0.0.0/0` for development

## Full Documentation

For more details, see:
- [docs/ENVIRONMENT_SETUP.md](docs/ENVIRONMENT_SETUP.md) - Complete environment configuration guide
- [docs/env.template](docs/env.template) - Environment variable template

## Need Help?

If you encounter issues:
1. Check MongoDB Atlas Network Access settings
2. Verify your connection string is correct
3. Ensure MongoDB service is running (for local)
4. Check firewall settings
5. Review application logs for detailed error messages

