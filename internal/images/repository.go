package images

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/BygTTeKep/AiDreamsStudio/internal/database"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ db *database.PGRepository }

func NewRepository(db *database.PGRepository) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) ProjectOwnedBy(ctx context.Context, userID int, projectID string) (bool, error) {
	var ok bool
	err := r.db.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND user_created = $2)`,
		projectID, userID).Scan(&ok)
	return ok, err
}

// все картинки одним запросом в одной транзакции: либо все записи, либо ни одной
func (r *Repository) CreateImages(ctx context.Context, projectID string, imgs []NewImage) ([]ProductImage, error) {
	tx, err := r.db.DB.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	fmt.Println(projectID, imgs)
	const q = `
        INSERT INTO product_images(project_id, storage_key, content_type, size_bytes, original_name)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, project_id, storage_key, content_type, size_bytes, original_name, created_at`

	out := make([]ProductImage, 0, len(imgs))
	for _, in := range imgs {
		var p ProductImage
		err := tx.QueryRow(ctx, q, projectID, in.StorageKey, in.ContentType, in.SizeBytes, in.OriginalName).
			Scan(&p.Id, &p.ProjectId, &p.StorageKey, &p.ContentType, &p.SizeBytes, &p.OriginalName, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("insert image: %w", err)
		}
		out = append(out, p)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return out, nil
}

// картинка по id с проверкой владельца
func (r *Repository) FindByID(ctx context.Context, userID int, imageID string) (ProductImage, error) {
	const q = `
        SELECT pi.id, pi.project_id, pi.storage_key, pi.content_type, pi.size_bytes, pi.original_name, pi.created_at
        FROM product_images pi
        JOIN projects p ON p.id = pi.project_id
        WHERE pi.id = $1 AND p.user_created = $2`
	var p ProductImage
	err := r.db.DB.QueryRow(ctx, q, imageID, userID).
		Scan(&p.Id, &p.ProjectId, &p.StorageKey, &p.ContentType, &p.SizeBytes, &p.OriginalName, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProductImage{}, ErrImageNotFound
	}
	return p, err
}

func (r *Repository) FindFilesByUserIdAndProjectId(ctx context.Context, userId, limit, offset int, projectId string) ([]ProductImage, error) {
	files := make([]ProductImage, 0)

	query := `
		SELECT 
			pi.id, 
			pi.project_id, 
			pi.storage_key, 
			pi.content_type, 
			pi.size_bytes, 
			pi.original_name, 
			pi.created_at
		FROM product_images pi
		LEFT JOIN projects p on p.id = pi.project_id
		WHERE p.user_created = $1 AND pi.project_id = $2
		ORDER BY pi.created_at DESC
		LIMIT $3
		OFFSET $4
	`
	rows, err := r.db.DB.Query(ctx, query, userId, projectId, limit, offset)
	if err != nil {
		//TODO
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var file ProductImage
		if err := rows.Scan(
			&file.Id,
			&file.ProjectId,
			&file.StorageKey,
			&file.ContentType,
			&file.SizeBytes,
			&file.OriginalName,
			&file.CreatedAt,
		); err != nil {
			slog.Error("")
			continue
		}
		files = append(files, file)
	}
	return files, nil
}
