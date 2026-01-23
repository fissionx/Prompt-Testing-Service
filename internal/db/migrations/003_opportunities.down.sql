-- Migration: 003_opportunities.down.sql
-- Description: Rollback opportunities and actions tables
-- Author: Gego

-- Drop triggers first
DROP TRIGGER IF EXISTS trigger_opportunities_updated_at;
DROP TRIGGER IF EXISTS trigger_actions_updated_at;

-- Drop indexes
DROP INDEX IF EXISTS idx_opportunities_brand_id;
DROP INDEX IF EXISTS idx_opportunities_prompt_id;
DROP INDEX IF EXISTS idx_opportunities_status;
DROP INDEX IF EXISTS idx_opportunities_type;
DROP INDEX IF EXISTS idx_opportunities_impact_score;
DROP INDEX IF EXISTS idx_opportunities_brand_status;
DROP INDEX IF EXISTS idx_opportunities_prompt_status;
DROP INDEX IF EXISTS idx_opportunities_created_at;
DROP INDEX IF EXISTS idx_opportunities_prompt_content_hash;

DROP INDEX IF EXISTS idx_actions_opportunity_id;
DROP INDEX IF EXISTS idx_actions_brand_id;
DROP INDEX IF EXISTS idx_actions_status;
DROP INDEX IF EXISTS idx_actions_created_at;

-- Drop tables (actions first due to foreign key)
DROP TABLE IF EXISTS actions;
DROP TABLE IF EXISTS opportunities;
