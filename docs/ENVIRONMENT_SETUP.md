# Environment Configuration Guide

This guide explains how to configure Gego to work with different MongoDB environments (local vs cloud).

## Overview

Gego supports environment-based configuration through environment variables, allowing you to easily switch between:
- **Local MongoDB** (localhost)
- **Development MongoDB Atlas** (cloud)
- **Production MongoDB Atlas** (cloud)

## Environment Variables

### Primary Configuration

| Variable | Description | Values | Default |
|----------|-------------|--------|---------|
| `GEGO_ENV` | Environment selector | `local`, `dev`, `prod` | - |
| `MONGODB_URI` | Direct MongoDB URI override | Any valid MongoDB URI | - |
| `MONGODB_CLOUD_URI` | Cloud MongoDB URI for dev | MongoDB Atlas URI | - |
| `MONGODB_PROD_URI` | Cloud MongoDB URI for prod | MongoDB Atlas URI | - |
| `MONGODB_DATABASE` | Database name | Any string | `gego` |

### Additional Configuration

| Variable | Description |
|----------|-------------|
| `SQL_DATABASE_URI` | SQLite database path |
| `CORS_ORIGIN` | CORS origin for API server |

## Setup Instructions

### Option 1: Using Environment Files (Recommended)

Create environment-specific files in your home directory or project root:

#### For Local Development (`.env.local`)

```bash
GEGO_ENV=local
MONGODB_DATABASE=gego
```

#### For Cloud Development (`.env.dev`)

```bash
GEGO_ENV=dev
MONGODB_CLOUD_URI=mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/
MONGODB_DATABASE=gego
```

### Option 2: Export Environment Variables

#### Run with Local MongoDB

```bash
export GEGO_ENV=local
gego api start
```

#### Run with Cloud MongoDB (Dev)

```bash
export GEGO_ENV=dev
export MONGODB_CLOUD_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/"
gego api start
```

#### Run with Direct URI Override

```bash
export MONGODB_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/"
gego api start
```

### Option 3: One-Line Commands

Run with local MongoDB:
```bash
GEGO_ENV=local gego api start
```

Run with cloud MongoDB:
```bash
GEGO_ENV=dev MONGODB_CLOUD_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/" gego api start
```

## MongoDB Atlas Setup

Your MongoDB Atlas connection string:
```
mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/
```

### Important Notes

1. **IP Whitelist**: Add `106.222.202.9` to MongoDB Atlas Network Access
   - Go to MongoDB Atlas Dashboard
   - Navigate to "Network Access"
   - Click "Add IP Address"
   - Enter your IP: `106.222.202.9`
   - Or use `0.0.0.0/0` for development (allow from anywhere)

2. **Database Name**: The connection string above doesn't include a database name. 
   - The database name will be taken from your config or `MONGODB_DATABASE` environment variable
   - Default is `gego`

## Configuration Priority

The system applies configuration in this order (highest priority first):

1. **Direct Override**: `MONGODB_URI` environment variable
2. **Environment-based**: `GEGO_ENV` with corresponding cloud URIs
3. **Config File**: Values from `~/.gego/config.yaml`
4. **Defaults**: Built-in default values

## Examples

### Development Workflow

```bash
# Start with local MongoDB for development
export GEGO_ENV=local
gego init
gego llm add openai --api-key sk-xxx --model gpt-4
gego prompt create --template "What are the best {category} brands?"
gego api start

# Switch to cloud for testing
export GEGO_ENV=dev
export MONGODB_CLOUD_URI="mongodb+srv://user:pass@cluster.mongodb.net/"
gego api start
```

### Using Shell Scripts

Create a script `run-local.sh`:
```bash
#!/bin/bash
export GEGO_ENV=local
gego api start
```

Create a script `run-dev.sh`:
```bash
#!/bin/bash
export GEGO_ENV=dev
export MONGODB_CLOUD_URI="mongodb+srv://fissionx_geo_db_use:ConsultNext12@fissionxgeo.mcwvkmk.mongodb.net/"
gego api start
```

Make them executable:
```bash
chmod +x run-local.sh run-dev.sh
```

### Docker Environment

If using Docker, add to your `docker-compose.yml`:

```yaml
services:
  gego:
    environment:
      - GEGO_ENV=dev
      - MONGODB_CLOUD_URI=mongodb+srv://user:pass@cluster.mongodb.net/
      - MONGODB_DATABASE=gego
```

## Verification

To verify your configuration is working:

1. Check which environment is active:
```bash
echo $GEGO_ENV
```

2. Test the connection:
```bash
# Start API and check logs
gego api start

# In another terminal, test the API
curl http://localhost:8080/health
```

3. View the active configuration:
```bash
# The application will log the database connection on startup
# Look for: "Connected to database: mongodb://..."
```

## Troubleshooting

### Connection Refused (Local)

If you get "connection refused" with `GEGO_ENV=local`:
- Ensure MongoDB is running: `sudo systemctl status mongod` (Linux) or `brew services list` (Mac)
- Start MongoDB: `sudo systemctl start mongod` (Linux) or `brew services start mongodb-community` (Mac)

### Authentication Failed (Cloud)

If you get authentication errors with Atlas:
- Verify your username/password in the connection string
- Check that your IP is whitelisted in MongoDB Atlas Network Access
- Ensure the database user has proper permissions

### Database Not Found

If you get "database not found":
- Set `MONGODB_DATABASE` environment variable
- Check your config file at `~/.gego/config.yaml`

## Best Practices

1. **Never commit credentials**: Keep your `.env` files out of version control
2. **Use different databases**: Use separate database names for local/dev/prod
3. **Rotate credentials**: Regularly update your MongoDB Atlas passwords
4. **Monitor Atlas**: Keep an eye on your free tier limits (512MB storage)
5. **Backup data**: Regularly export important data from cloud databases

## Migration Between Environments

To migrate data from local to cloud:

```bash
# Export from local
GEGO_ENV=local gego prompt export > prompts.json

# Import to cloud
GEGO_ENV=dev gego prompt import < prompts.json
```

## Additional Resources

- [MongoDB Atlas Documentation](https://docs.atlas.mongodb.com/)
- [MongoDB Connection Strings](https://docs.mongodb.com/manual/reference/connection-string/)
- [Gego Configuration Reference](./API_QUICK_REFERENCE.md)

