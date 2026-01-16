# PostgreSQL Setup Guide

This guide explains how to configure Gego to use PostgreSQL as the NoSQL database backend.

## Quick Start

Set the `POSTGRESQL_URI` environment variable with your PostgreSQL connection string:

```bash
export POSTGRESQL_URI="postgresql://username:password@host:port/database?sslmode=require"
```

Then start the API server:

```bash
gego api
```

## Environment Variables

### Required
- `POSTGRESQL_URI` - Full PostgreSQL connection string

### Optional
- `POSTGRESQL_DATABASE` - Database name (if not specified in URI)
- `NOSQL_DATABASE_PROVIDER` - Explicitly set to "postgresql" (auto-set when POSTGRESQL_URI is provided)

## Connection String Format

The PostgreSQL connection string should follow this format:

```
postgresql://[user[:password]@][netloc][:port][/dbname][?param1=value1&...]
```

### Example for Neon (Cloud PostgreSQL)

```bash
export POSTGRESQL_URI="postgresql://neondb_owner:password@ep-bold-sky-a11lqfs2-pooler.ap-southeast-1.aws.neon.tech/gego?sslmode=require&channel_binding=require"
```

### Example for Local PostgreSQL

```bash
export POSTGRESQL_URI="postgresql://postgres:password@localhost:5432/gego?sslmode=disable"
```

## Verification

When you start the API server, you should see output like:

```
📊 Database Configuration:
  SQL Database: sqlite (gego.db)
  NoSQL Database: postgresql
  PostgreSQL URI: postgresql://neondb_owner:***@ep-bold-sky-a11lqfs2-pooler.ap-southeast-1.aws.neon.tech/gego?sslmode=require&channel_binding=require
  Database Name: gego
  ✅ Using PostgreSQL for NoSQL operations

✅ Database connection successful!
```

## Configuration File Alternative

You can also configure PostgreSQL in your config file (`~/.gego/config.yaml`):

```yaml
sql_database:
  provider: sqlite
  uri: gego.db
  database: gego

nosql_database:
  provider: postgresql
  uri: postgresql://neondb_owner:password@ep-bold-sky-a11lqfs2-pooler.ap-southeast-1.aws.neon.tech/gego?sslmode=require&channel_binding=require
  database: gego
```

**Note:** Environment variables take precedence over config file settings.

## Troubleshooting

### Connection Failed
- Verify your connection string is correct
- Check that the database server is accessible from your network
- Ensure SSL settings match your database configuration
- Verify credentials are correct

### SSL Mode Issues
If you encounter SSL-related errors, try:
- `sslmode=require` (default for cloud providers)
- `sslmode=prefer` (tries SSL, falls back if not available)
- `sslmode=disable` (only for local development)

### Database Not Found
Make sure the database specified in the connection string exists. You may need to create it:

```sql
CREATE DATABASE gego;
```

## What Uses PostgreSQL?

When PostgreSQL is configured, it handles:
- Prompts storage
- Responses storage
- Brand profiles
- Prompt libraries
- Analytics cache
- GEO insights
- And all other NoSQL operations

SQLite continues to be used for:
- LLM configurations
- Schedule configurations

