CREATE TABLE teams (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 200),
    points INTEGER NOT NULL DEFAULT 0 CHECK (points >= 0),
    interests JSONB NOT NULL DEFAULT '[]',
    skills JSONB NOT NULL DEFAULT '[]',
    technologies JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE offers (
    id UUID PRIMARY KEY,
    task_id UUID NOT NULL REFERENCES tasks(id),
    team_id UUID NOT NULL REFERENCES teams(id),
    solution_idea TEXT NOT NULL CHECK (length(btrim(solution_idea)) BETWEEN 1 AND 10000),
    plan TEXT NOT NULL CHECK (length(btrim(plan)) BETWEEN 1 AND 10000),
    timeline TEXT NOT NULL CHECK (length(btrim(timeline)) BETWEEN 1 AND 500),
    prototype_link TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    decided_at TIMESTAMPTZ,
    UNIQUE (id, team_id)
);
CREATE INDEX offers_task_created_idx ON offers(task_id, created_at DESC, id);
CREATE INDEX offers_team_idx ON offers(team_id);

CREATE TABLE point_awards (
    offer_id UUID PRIMARY KEY,
    team_id UUID NOT NULL REFERENCES teams(id),
    points INTEGER NOT NULL CHECK (points = 10),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (offer_id, team_id) REFERENCES offers(id, team_id)
);

INSERT INTO teams (id, name, interests, skills, technologies) VALUES
('00000000-0000-4000-8000-000000000001', 'Team Alpha', '["retail"]', '["backend"]', '["Go","PostgreSQL"]'),
('00000000-0000-4000-8000-000000000002', 'Team Beta', '["education"]', '["frontend"]', '["Vue"]'),
('00000000-0000-4000-8000-000000000003', 'Team Gamma', '["logistics"]', '["analytics"]', '["Python"]'),
('00000000-0000-4000-8000-000000000004', 'Team Delta', '["retail"]', '["design"]', '["Figma"]'),
('00000000-0000-4000-8000-000000000005', 'Team Epsilon', '["education"]', '["backend"]', '["Go"]');
