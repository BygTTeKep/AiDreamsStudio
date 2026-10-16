-- +goose Up
CREATE TABLE IF NOT EXISTS product_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    storage_key   text NOT NULL,
    content_type  text NOT NULL,
    size_bytes    bigint NOT NULL,
    original_name TEXT NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS product_images;
