-- Migration: 003_opportunities.sql
-- Description: Create opportunities and actions tables for GEO optimization tracking
-- Author: Gego

-- Opportunities table - stores identified gaps and improvement areas
CREATE TABLE IF NOT EXISTS opportunities (
    id TEXT PRIMARY KEY,
    org_id TEXT,
    brand_id TEXT NOT NULL,
    prompt_id TEXT NOT NULL,
    response_id TEXT,
    type TEXT NOT NULL CHECK (type IN ('content_gap', 'content_freshness', 'reddit_participation', 'review_sites', 'case_study', 'seo_improvement', 'pr_opportunity', 'linkedin_presence', 'comparison_content', 'citation_source', 'faq_content', 'competitor_response', 'integration_content', 'other')),
    status TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'in_progress', 'completed', 'archived')),
    
    -- Core opportunity details
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    
    -- WHY - Evidence and reasoning for this opportunity
    gap_analysis TEXT,           -- Why this gap exists (evidence from AI response)
    current_state TEXT,          -- What the brand currently has/lacks
    competitor_context TEXT,     -- What competitors are doing better
    source_evidence TEXT,        -- Specific sources/citations that show the gap
    
    -- WHAT - Specific recommendation
    recommended_action TEXT,     -- Clear action statement
    expected_outcome TEXT,       -- What success looks like
    
    -- WHERE - Target and context
    target_platform TEXT,        -- Where to take action (website, reddit, linkedin, g2, etc.)
    target_audience TEXT,        -- Who this content/action is for
    keywords TEXT DEFAULT '[]',  -- JSON array of relevant keywords
    related_urls TEXT DEFAULT '[]', -- JSON array of competitor/source URLs
    
    -- Priority and scoring
    impact_score INTEGER NOT NULL CHECK (impact_score >= 1 AND impact_score <= 100),
    impact_reasoning TEXT,       -- Why this score was assigned
    urgency TEXT CHECK (urgency IN ('high', 'medium', 'low')),
    effort_estimate TEXT CHECK (effort_estimate IN ('low', 'medium', 'high')),
    
    -- Internal fields
    content_hash TEXT NOT NULL,
    action_id TEXT,
    metadata TEXT DEFAULT '{}',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Actions table - stores detailed action plans for opportunities
CREATE TABLE IF NOT EXISTS actions (
    id TEXT PRIMARY KEY,
    org_id TEXT,
    opportunity_id TEXT NOT NULL REFERENCES opportunities(id) ON DELETE CASCADE,
    brand_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    steps TEXT DEFAULT '[]',
    estimated_effort TEXT CHECK (estimated_effort IN ('low', 'medium', 'high')),
    resources TEXT DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for opportunities table
CREATE INDEX IF NOT EXISTS idx_opportunities_org_id ON opportunities(org_id);
CREATE INDEX IF NOT EXISTS idx_opportunities_brand_id ON opportunities(brand_id);
CREATE INDEX IF NOT EXISTS idx_opportunities_prompt_id ON opportunities(prompt_id);
CREATE INDEX IF NOT EXISTS idx_opportunities_status ON opportunities(status);
CREATE INDEX IF NOT EXISTS idx_opportunities_type ON opportunities(type);
CREATE INDEX IF NOT EXISTS idx_opportunities_impact_score ON opportunities(impact_score DESC);
CREATE INDEX IF NOT EXISTS idx_opportunities_urgency ON opportunities(urgency);
CREATE INDEX IF NOT EXISTS idx_opportunities_brand_status ON opportunities(brand_id, status);
CREATE INDEX IF NOT EXISTS idx_opportunities_prompt_status ON opportunities(prompt_id, status);
CREATE INDEX IF NOT EXISTS idx_opportunities_created_at ON opportunities(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_opportunities_target_platform ON opportunities(target_platform);

-- Unique constraint to prevent duplicate opportunities per prompt
CREATE UNIQUE INDEX IF NOT EXISTS idx_opportunities_prompt_content_hash ON opportunities(prompt_id, content_hash);

-- Indexes for actions table
CREATE INDEX IF NOT EXISTS idx_actions_org_id ON actions(org_id);
CREATE INDEX IF NOT EXISTS idx_actions_opportunity_id ON actions(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_actions_brand_id ON actions(brand_id);
CREATE INDEX IF NOT EXISTS idx_actions_status ON actions(status);
CREATE INDEX IF NOT EXISTS idx_actions_created_at ON actions(created_at DESC);

-- Create triggers to automatically update the updated_at timestamp
CREATE TRIGGER IF NOT EXISTS trigger_opportunities_updated_at 
    AFTER UPDATE ON opportunities
    FOR EACH ROW
    BEGIN
        UPDATE opportunities SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;

CREATE TRIGGER IF NOT EXISTS trigger_actions_updated_at 
    AFTER UPDATE ON actions
    FOR EACH ROW
    BEGIN
        UPDATE actions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
    END;
