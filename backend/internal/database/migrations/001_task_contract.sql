-- Minimal shared boundary for block 2. Block 1 extends this table with its
-- draft/card/rating fields in subsequent migrations and implements task routes.
CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'confirmed', 'published')),
    published_at TIMESTAMPTZ,
    work_status TEXT NOT NULL DEFAULT 'open' CHECK (work_status IN ('open', 'in_progress')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK ((status = 'published') = (published_at IS NOT NULL))
);
