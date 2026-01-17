# Deploying Gego with PostgreSQL to Fly.io

This guide explains how to deploy Gego with PostgreSQL to Fly.io.

## Prerequisites

1. **Fly.io CLI installed**: [Install flyctl](https://fly.io/docs/hands-on/install-flyctl/)
2. **Fly.io account**: Sign up at [fly.io](https://fly.io)
3. **PostgreSQL database**: You need a PostgreSQL connection string (e.g., from Neon, Supabase, or your own PostgreSQL instance)

## Quick Deployment

### Option 1: Automated Script (Recommended)

```bash
./deploy-to-fly.sh
```

When prompted:
1. Select option **1** for PostgreSQL
2. Enter your PostgreSQL connection string

Example PostgreSQL URI:
```
postgresql://neondb_owner:password@ep-bold-sky-a11lqfs2-pooler.ap-southeast-1.aws.neon.tech/gego?sslmode=require&channel_binding=require
```

### Option 2: Manual Deployment

#### Step 1: Set PostgreSQL Secrets

```bash
flyctl secrets set POSTGRESQL_URI="postgresql://user:password@host:port/database?sslmode=require" -a gego
flyctl secrets set NOSQL_DATABASE_PROVIDER="postgresql" -a gego
```

#### Step 2: Deploy

```bash
flyctl deploy
```

#### Step 3: Verify Deployment

```bash
# Check status
flyctl status

# View logs
flyctl logs

# Test health endpoint
curl https://gego.fly.dev/api/v1/health
```

## PostgreSQL Connection String Format

Your PostgreSQL connection string should follow this format:

```
postgresql://[user]:[password]@[host]:[port]/[database]?[parameters]
```

### Example for Neon (Cloud PostgreSQL)

```
postgresql://neondb_owner:password@ep-bold-sky-a11lqfs2-pooler.ap-southeast-1.aws.neon.tech/gego?sslmode=require&channel_binding=require
```

### Example for Supabase

```
postgresql://postgres:password@db.xxxxx.supabase.co:5432/postgres?sslmode=require
```

### Example for Local/Standard PostgreSQL

```
postgresql://postgres:password@hostname:5432/gego?sslmode=require
```

## Environment Variables

The following environment variables are set in `fly.toml`:

- `POSTGRESQL_DATABASE=gego` - Database name
- `GEGO_ENV=dev` - Environment mode
- `CORS_ORIGIN=*` - CORS settings

Secrets (set via `flyctl secrets set`):
- `POSTGRESQL_URI` - Full PostgreSQL connection string
- `NOSQL_DATABASE_PROVIDER=postgresql` - Database provider

## Updating PostgreSQL Connection

If you need to update your PostgreSQL connection string:

```bash
flyctl secrets set POSTGRESQL_URI="new-connection-string" -a gego
flyctl apps restart gego
```

## Verifying PostgreSQL Connection

After deployment, check the logs to verify PostgreSQL connection:

```bash
flyctl logs
```

You should see:
```
✅ Database connection successful!
📊 Database Configuration:
  NoSQL Database: postgresql
  ✅ Using PostgreSQL for NoSQL operations
```

## Monitoring

### View Logs

```bash
# Real-time logs
flyctl logs -a gego

# Follow logs
flyctl logs -a gego --follow
```

### Check Request Latency

The logs will show API and database operation times:

```
[REQUEST] method=GET path=/api/v1/brands/xxx status=200 api_time_ms=45.23 db_time_ms=12.45
```

Where:
- `api_time_ms` = Total API request time
- `db_time_ms` = Total PostgreSQL operation time

### Check App Status

```bash
flyctl status -a gego
```

### SSH into Container

```bash
flyctl ssh console -a gego
```

Inside the container, you can check:
```bash
# Check environment variables
env | grep POSTGRESQL

# Check database connection
# (if you have psql installed)
```

## Troubleshooting

### Connection Failed

1. **Verify connection string**: Check your PostgreSQL URI is correct
2. **Check SSL settings**: Ensure `sslmode=require` matches your database configuration
3. **Verify network access**: Ensure your PostgreSQL database allows connections from Fly.io IPs
4. **Check credentials**: Verify username and password are correct

### Database Not Found

Make sure the database specified in your connection string exists:

```sql
CREATE DATABASE gego;
```

### SSL Mode Issues

If you encounter SSL errors, try different SSL modes:

- `sslmode=require` - Requires SSL (default for cloud providers)
- `sslmode=prefer` - Prefers SSL but allows non-SSL
- `sslmode=disable` - Disables SSL (only for local/dev)

Update the secret:
```bash
flyctl secrets set POSTGRESQL_URI="postgresql://...?sslmode=prefer" -a gego
```

### View Detailed Logs

```bash
# View last 100 lines
flyctl logs -a gego --limit 100

# View logs with timestamps
flyctl logs -a gego --timestamps
```

### Restart Application

If the app is having issues:

```bash
flyctl apps restart gego
```

### Check Database Schema

The application automatically creates the required schema on first connection. If you need to verify:

1. Connect to your PostgreSQL database
2. Check if tables exist:
   ```sql
   \dt
   ```

You should see tables like:
- `prompts`
- `responses`
- `prompt_library`
- `brand_profiles`
- `cached_geo_insights`
- etc.

## Scaling

### Scale Up Resources

```bash
# Increase memory
flyctl scale memory 1024 -a gego

# Increase CPU
flyctl scale vm shared-cpu-2x -a gego
```

### Scale Instances

```bash
# Run multiple instances
flyctl scale count 2 -a gego
```

## Useful Commands

```bash
# View all secrets (names only)
flyctl secrets list -a gego

# Remove a secret
flyctl secrets unset SECRET_NAME -a gego

# Open app in browser
flyctl open -a gego

# View dashboard
flyctl dashboard -a gego

# View app info
flyctl info -a gego
```

## Migration from MongoDB

If you're migrating from MongoDB to PostgreSQL:

1. **Set PostgreSQL secrets** (as shown above)
2. **Deploy the application**
3. **Data migration**: If you have existing MongoDB data, you'll need to export and import it separately

## Production Considerations

1. **Use connection pooling**: Your PostgreSQL provider (Neon, Supabase, etc.) should handle this
2. **Enable SSL**: Always use `sslmode=require` in production
3. **Monitor performance**: Use the latency logs to identify slow queries
4. **Backup strategy**: Set up regular backups with your PostgreSQL provider
5. **Resource limits**: Monitor Fly.io resource usage and scale as needed

## Support

If you encounter issues:

1. Check the logs: `flyctl logs -a gego`
2. Verify secrets: `flyctl secrets list -a gego`
3. Check app status: `flyctl status -a gego`
4. Review Fly.io status: https://status.fly.io

