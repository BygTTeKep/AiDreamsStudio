-- +goose Up
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    user_created INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (user_created) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_projects_user_created ON projects(user_created);

-- +goose Down
DROP TABLE IF EXISTS projects;
