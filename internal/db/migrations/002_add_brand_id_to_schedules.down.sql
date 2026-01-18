-- Migration: 002_add_brand_id_to_schedules.down.sql
-- Description: Rollback adding brand_id column to schedules table
-- Author: AI2HU

-- Restore the original view without brand_id
DROP VIEW IF EXISTS v_enabled_schedules;
CREATE VIEW IF NOT EXISTS v_enabled_schedules AS
SELECT 
    id,
    name,
    prompt_ids,
    llm_ids,
    cron_expr,
    temperature,
    last_run,
    next_run,
    created_at,
    updated_at
FROM schedules 
WHERE enabled = 1
ORDER BY next_run ASC;

-- Drop index
DROP INDEX IF EXISTS idx_schedules_brand_id;

-- Remove brand_id column (SQLite doesn't support DROP COLUMN directly, 
-- so this would require recreating the table in a real scenario)
-- For SQLite, this migration down is informational only
-- ALTER TABLE schedules DROP COLUMN brand_id;

