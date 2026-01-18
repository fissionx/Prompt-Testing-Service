-- Migration: 002_add_brand_id_to_schedules.up.sql
-- Description: Add brand_id column to schedules table
-- Author: AI2HU

-- Add brand_id column to schedules table
-- For existing records, we'll set a default empty string (migration scenario)
-- In production, this should be handled by setting a proper brand_id for existing schedules
ALTER TABLE schedules ADD COLUMN brand_id TEXT NOT NULL DEFAULT '';

-- Create index on brand_id for efficient filtering
CREATE INDEX IF NOT EXISTS idx_schedules_brand_id ON schedules(brand_id);

-- Update the view to include brand_id
DROP VIEW IF EXISTS v_enabled_schedules;
CREATE VIEW IF NOT EXISTS v_enabled_schedules AS
SELECT 
    id,
    brand_id,
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

