ALTER TABLE opportunities ADD COLUMN llm_id TEXT;
CREATE INDEX idx_opportunities_llm_id ON opportunities(llm_id);
