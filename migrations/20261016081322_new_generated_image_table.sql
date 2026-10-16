-- +goose Up
CREATE TABLE IF NOT EXISTS generated_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    generation_id text NOT NULL UNIQUE,
    storage_key   text NOT NULL,
    content_type  text NOT NULL,
    size_bytes    bigint NOT NULL,
    source_url    text,
    status        text NOT NULL DEFAULT 'pending_review',
    version       int NOT NULL DEFAULT 1,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS generated_images;
