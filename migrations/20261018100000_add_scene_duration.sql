-- +goose Up
ALTER TABLE scenes
    ADD COLUMN IF NOT EXISTS duration_sec int NOT NULL DEFAULT 5
        CONSTRAINT scenes_duration_sec_check CHECK (duration_sec BETWEEN 3 AND 10);

-- +goose Down
ALTER TABLE scenes DROP COLUMN IF EXISTS duration_sec;
