# MongoDB Environment Configuration - Setup Summary

## ✅ What Was Done

I've configured your Gego application to easily switch between local MongoDB and MongoDB Atlas (cloud) using environment variables.

### 1. Code Changes

**Modified: `internal/config/config.go`**
- Added environment variable support
- Implemented `applyEnvironmentOverrides()` function
- Priority: ENV variables > Config file > Defaults

**Modified: `.gitignore`**
- Added `.env` and `.env.*` files to prevent committing sensitive data
- Added `!.env.example` to allow example file

**Modified: `README.md`**
- Added MongoDB Configuration section
- Documented quick setup methods

### 2. Documentation Created

- **`MONGODB_SETUP.md`** - Quick reference guide (at project root)
- **`docs/ENVIRONMENT_SETUP.md`** - Complete configuration guide
- **`docs/env.template`** - Environment variable template

### 3. Helper Scripts Created

- **`scripts/run-local.sh`** - Run with local MongoDB
- **`scripts/run-dev.sh`** - Run with MongoDB Atlas cloud

## 🚀 How to Use

### MongoDB Atlas Information

**Your Connection String:**
```
mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/
```

**Your IP Address (for Atlas Whitelist):**
```
106.222.202.9
```

### Method 1: Using Helper Scripts (Easiest)

```bash
# Run with local MongoDB
./scripts/run-local.sh api start

# Run with cloud MongoDB Atlas
./scripts/run-dev.sh api start
```

### Method 2: Using Environment Variables

```bash
# Local
GEGO_ENV=local gego api start

# Cloud
GEGO_ENV=dev \
MONGODB_CLOUD_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/" \
gego api start
```

### Method 3: Export Variables

```bash
# For local development session
export GEGO_ENV=local
gego init
gego llm add openai --api-key sk-xxx --model gpt-4
gego api start

# For cloud development session
export GEGO_ENV=dev
export MONGODB_CLOUD_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/"
gego api start
```

## 📋 Environment Variables Reference

| Variable | Purpose | Example |
|----------|---------|---------|
| `GEGO_ENV` | Environment selector | `local`, `dev`, `prod` |
| `MONGODB_URI` | Direct MongoDB URI (overrides all) | `mongodb://localhost:27017` |
| `MONGODB_CLOUD_URI` | Cloud URI for dev environment | `mongodb+srv://...` |
| `MONGODB_PROD_URI` | Cloud URI for prod environment | `mongodb+srv://...` |
| `MONGODB_DATABASE` | Database name | `gego` |

## ⚙️ MongoDB Atlas Setup Steps

1. **Go to MongoDB Atlas**: https://cloud.mongodb.com/
2. **Navigate to Network Access** (left sidebar)
3. **Click "Add IP Address"**
4. **Enter your IP**: `106.222.202.9`
   - OR click "Allow Access from Anywhere" for dev (less secure)
5. **Click "Confirm"**
6. **Test connection**:
   ```bash
   ./scripts/run-dev.sh api start
   ```

## 🎯 Quick Start Examples

### Example 1: Initialize with Local MongoDB

```bash
# Make sure MongoDB is running locally
# Linux: sudo systemctl start mongod
# macOS: brew services start mongodb-community

GEGO_ENV=local gego init
```

### Example 2: Run API with Cloud MongoDB

```bash
./scripts/run-dev.sh api start
```

### Example 3: Create and Execute a Schedule

```bash
# Setup
export GEGO_ENV=dev
export MONGODB_CLOUD_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/"

# Add LLM
gego llm add openai --api-key sk-xxx --model gpt-4

# Create prompt
gego prompt create --template "What are the best {category} brands?"

# Create schedule
gego schedule add

# Run schedule
gego scheduler run --schedule-id <id>
```

## 🔍 Verification

To verify everything is working:

```bash
# Check build
go build -o gego cmd/gego/main.go

# Test local connection
GEGO_ENV=local ./gego api start

# Test cloud connection (in another terminal)
GEGO_ENV=dev \
MONGODB_CLOUD_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/" \
./gego api start
```

## 📚 Documentation Files

- **MONGODB_SETUP.md** - Quick reference (READ THIS FIRST)
- **docs/ENVIRONMENT_SETUP.md** - Detailed configuration guide
- **docs/env.template** - Environment variable template
- **README.md** - Updated with MongoDB configuration section

## 🛠️ Troubleshooting

### "Connection refused" (Local)
```bash
# Check if MongoDB is running
# Linux:
sudo systemctl status mongod
sudo systemctl start mongod

# macOS:
brew services list
brew services start mongodb-community
```

### "Authentication failed" (Cloud)
- Verify username/password in connection string
- Check IP whitelist in MongoDB Atlas Network Access
- Ensure database user has proper permissions

### "IP not whitelisted" (Cloud)
- Add your IP `106.222.202.9` to MongoDB Atlas
- Or use `0.0.0.0/0` for development (allows from anywhere)

## 🔐 Security Notes

1. **Never commit `.env` files** - They're now in `.gitignore`
2. **Rotate credentials regularly** - Update MongoDB Atlas passwords
3. **Use IP whitelisting** - Don't use `0.0.0.0/0` in production
4. **Different databases per environment** - Use separate DBs for local/dev/prod
5. **Monitor usage** - Watch your MongoDB Atlas free tier limits (512MB)

## 🎉 Summary

You can now easily:
- ✅ Switch between local and cloud MongoDB
- ✅ Use environment variables for configuration
- ✅ Run with helper scripts for convenience
- ✅ Deploy to different environments (local, dev, prod)
- ✅ Keep sensitive data out of version control

## 📞 Next Steps

1. Whitelist your IP in MongoDB Atlas: `106.222.202.9`
2. Test local connection: `./scripts/run-local.sh api start`
3. Test cloud connection: `./scripts/run-dev.sh api start`
4. Start building your GEO tracking system!

For questions or issues, refer to the detailed documentation in `docs/ENVIRONMENT_SETUP.md`.

