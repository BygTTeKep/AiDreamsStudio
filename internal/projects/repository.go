package projects

import (
	"context"
	"errors"
	"log/slog"

	"github.com/BygTTeKep/AiDreamsStudio/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	db *database.PGRepository
}

func NewRepository(db *database.PGRepository) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(ctx context.Context, name string, userCreated int) (string, error) {
	id := uuid.NewString()
	var projectId string
	query := `
		INSERT INTO projects(id, name, user_created)
		VALUES($1,$2, $3)
		RETURNING id
	`
	err := r.db.DB.QueryRow(ctx, query, id, name, userCreated).Scan(&projectId)
	if err != nil {
		return projectId, err
	}
	return projectId, nil
}

func (r *Repository) FindById(ctx context.Context, projectId string, userCreated int) (Project, error) {
	var project Project

	query := `
		SELECT id, name, user_created, created_at
		FROM projects
		WHERE id = $1 AND user_created = $2
	`
	err := r.db.DB.QueryRow(ctx, query, projectId, userCreated).Scan(&project.ID, &project.Name, &project.UserCreated, &project.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return project, ErrProjectNotFound
		}
		return project, err
	}
	return project, nil
}

func (r *Repository) Search(ctx context.Context, userCreated, limit, offset int) ([]Project, error) {
	projects := make([]Project, 0)
	query := `
		SELECT id, name, user_created, created_at
		FROM projects
		WHERE user_created = $1
		ORDER BY created_at DESC
		LIMIT $2
		OFFSET $3
	`
	rows, err := r.db.DB.Query(ctx, query, userCreated, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var project Project
		if err := rows.Scan(&project.ID, &project.Name, &project.UserCreated, &project.CreatedAt); err != nil {
			slog.Error("search", "error", err)
			continue
		}
		projects = append(projects, project)
	}
	return projects, nil
}
