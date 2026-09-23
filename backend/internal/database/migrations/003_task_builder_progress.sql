ALTER TABLE tasks
    ADD COLUMN title TEXT NOT NULL DEFAULT '',
    ADD COLUMN industry TEXT NOT NULL DEFAULT '',
    ADD COLUMN draft TEXT NOT NULL DEFAULT '',
    ADD COLUMN answers JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN working_fields JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN confirmed_snapshot JSONB,
    ADD COLUMN revision INTEGER NOT NULL DEFAULT 1 CHECK (revision > 0),
    ADD COLUMN confirmed_revision INTEGER,
    ADD COLUMN score INTEGER NOT NULL DEFAULT 0 CHECK (score BETWEEN 0 AND 100),
    ADD COLUMN readiness TEXT NOT NULL DEFAULT 'draft'
        CHECK (readiness IN ('draft', 'working', 'ready', 'priority'));
CREATE INDEX tasks_catalog_idx ON tasks(score DESC, published_at, id) WHERE published_at IS NOT NULL;
CREATE INDEX tasks_owner_idx ON tasks(owner_id, created_at DESC, id);

-- Preserve the original offer columns and routes while adding the agreed API shape.
ALTER TABLE offers
    ADD COLUMN plan_steps JSONB,
    ADD COLUMN duration_days INTEGER CHECK (duration_days > 0);

CREATE TABLE milestones (
    id UUID PRIMARY KEY,
    proposal_id UUID NOT NULL UNIQUE REFERENCES offers(id),
    team_id UUID NOT NULL,
    result_text TEXT NOT NULL CHECK (length(btrim(result_text)) BETWEEN 1 AND 10000),
    evidence_url TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'submitted' CHECK (status IN ('submitted', 'confirmed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    confirmed_at TIMESTAMPTZ,
    FOREIGN KEY (proposal_id, team_id) REFERENCES offers(id, team_id),
    UNIQUE (id, team_id),
    CHECK ((status = 'confirmed') = (confirmed_at IS NOT NULL))
);
CREATE TABLE progress_awards (
    milestone_id UUID PRIMARY KEY,
    team_id UUID NOT NULL REFERENCES teams(id),
    points INTEGER NOT NULL CHECK (points = 10),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (milestone_id, team_id) REFERENCES milestones(id, team_id)
);
CREATE INDEX progress_awards_team_idx ON progress_awards(team_id);
-- point_awards is retained as historical data, but only progress_awards now
-- contributes to displayed team points. No existing records are deleted.
