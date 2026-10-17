-- +goose Up
CREATE TABLE IF NOT EXISTS scenarios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    pain_point  text NOT NULL DEFAULT '',
    title       text NOT NULL,
    source      text NOT NULL,
    status      text NOT NULL DEFAULT 'pending_review',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_scenarios_project_status ON scenarios(project_id, status);

CREATE TABLE IF NOT EXISTS scenes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id uuid NOT NULL REFERENCES scenarios(id) ON DELETE CASCADE,
    position    int NOT NULL,
    description text NOT NULL,
    UNIQUE (scenario_id, position)
);

-- +goose Down
DROP TABLE IF EXISTS scenes;
DROP TABLE IF EXISTS scenarios;
