package scenario

import (
	"context"
	"errors"
	"fmt"

	"github.com/BygTTeKep/AiDreamsStudio/internal/database"
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

func (r *Repository) GetScenariosByStatus(ctx context.Context, userID int, projectID string, status Status) ([]Scenario, error) {
	const q = `
        SELECT sc.id, sc.project_id, sc.pain_point, sc.title, sc.source, sc.status, sc.created_at,
               s.id, s.position, s.duration_sec, s.description
        FROM scenarios sc
        JOIN projects p ON p.id = sc.project_id AND p.user_created = $3
        LEFT JOIN scenes s ON s.scenario_id = sc.id
        WHERE sc.project_id = $1 AND sc.status = $2
        ORDER BY sc.created_at DESC, sc.id, s.position`

	rows, err := r.db.DB.Query(ctx, q, projectID, status, userID)
	if err != nil {
		return nil, fmt.Errorf("query scenarios: %w", err)
	}
	defer rows.Close()

	result := make([]Scenario, 0)
	for rows.Next() {
		var (
			sc         Scenario
			sceneID    *string
			scenePos   *int
			sceneDur   *int
			sceneDescr *string
		)
		if err := rows.Scan(
			&sc.ID, &sc.ProjectID, &sc.PainPoint, &sc.Title, &sc.Source, &sc.Status, &sc.CreatedAt,
			&sceneID, &scenePos, &sceneDur, &sceneDescr,
		); err != nil {
			return nil, fmt.Errorf("scan scenario: %w", err)
		}

		// строки одного сценария идут подряд благодаря ORDER BY
		if len(result) == 0 || result[len(result)-1].ID != sc.ID {
			sc.Scenes = []Scene{} // не nil, чтобы в JSON был [], а не null
			result = append(result, sc)
		}
		if sceneID != nil { // у LEFT JOIN сцены может не быть
			cur := &result[len(result)-1]
			cur.Scenes = append(cur.Scenes, Scene{
				ID:          *sceneID,
				ScenarioID:  cur.ID,
				Position:    *scenePos,
				DurationSec: *sceneDur,
				Description: *sceneDescr,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scenarios: %w", err)
	}
	return result, nil
}

func (r *Repository) ProjectOwnedBy(ctx context.Context, userID int, projectID string) (bool, error) {
	var ok bool
	err := r.db.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND user_created = $2)`,
		projectID, userID).Scan(&ok)
	return ok, err
}

// ApproveScenario переводит сценарий из pending_review в approved.
// ErrScenarioNotFound — сценария нет в проекте пользователя,
// ErrScenarioNotPending — сценарий уже не на ревью.
func (r *Repository) ApproveScenario(ctx context.Context, projectId, scenarioId string, userId int) error {
	const q = `
		UPDATE scenarios s
		SET status = $4, updated_at = now()
		FROM projects p
		WHERE s.id = $1 AND s.project_id = $2 AND s.status = $5
			AND p.id = s.project_id AND p.user_created = $3`

	tag, err := r.db.DB.Exec(ctx, q, scenarioId, projectId, userId, StatusApproved, StatusPendingReview)
	if err != nil {
		return fmt.Errorf("approve scenario: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	// ничего не обновили — выясняем почему
	var status Status
	err = r.db.DB.QueryRow(ctx, `
		SELECT s.status
		FROM scenarios s
		JOIN projects p ON p.id = s.project_id AND p.user_created = $3
		WHERE s.id = $1 AND s.project_id = $2`,
		scenarioId, projectId, userId).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrScenarioNotFound
		}
		return fmt.Errorf("check scenario status: %w", err)
	}
	return ErrScenarioNotPending
}

func (r *Repository) CreateScenario(ctx context.Context, sc CreateScenarioRequest) (string, error) {
	tx, err := r.db.DB.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx, `
        INSERT INTO scenarios(project_id, pain_point, title, source, status)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id`,
		sc.ProjectID, sc.PainPoint, sc.Title, sc.Source, sc.Status,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert scenario: %w", err)
	}

	if len(sc.Scenes) > 0 {
		positions := make([]int32, len(sc.Scenes))
		durations := make([]int32, len(sc.Scenes))
		descriptions := make([]string, len(sc.Scenes))
		for i, s := range sc.Scenes {
			positions[i] = int32(s.Position)
			durations[i] = int32(s.DurationSec)
			descriptions[i] = s.Description
		}
		_, err = tx.Exec(ctx, `
            INSERT INTO scenes(scenario_id, position, duration_sec, description)
            SELECT $1::uuid, t.p, t.dur, t.d
            FROM unnest($2::int[], $3::int[], $4::text[]) AS t(p, dur, d)`,
			id, positions, durations, descriptions,
		)
		if err != nil {
			return "", fmt.Errorf("insert scenes: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit: %w", err)
	}
	return id, nil
}

func (r *Repository) GetById(ctx context.Context, projectId, scenarioId string, userId int) (Scenario, error) {
	const q = `
        SELECT sc.id, sc.project_id, sc.pain_point, sc.title, sc.source, sc.status, sc.created_at,
               s.id, s.position, s.duration_sec, s.description
        FROM scenarios sc
        JOIN projects p ON p.id = sc.project_id AND p.user_created = $3
        LEFT JOIN scenes s ON s.scenario_id = sc.id
        WHERE sc.project_id = $1 AND sc.id = $2
        ORDER BY s.position`

	rows, err := r.db.DB.Query(ctx, q, projectId, scenarioId, userId)
	if err != nil {
		return Scenario{}, fmt.Errorf("query scenario: %w", err)
	}
	defer rows.Close()

	var (
		sc    Scenario
		found bool
	)
	sc.Scenes = []Scene{}
	for rows.Next() {
		var (
			sceneID    *string
			scenePos   *int
			sceneDur   *int
			sceneDescr *string
		)
		if err := rows.Scan(
			&sc.ID, &sc.ProjectID, &sc.PainPoint, &sc.Title, &sc.Source, &sc.Status, &sc.CreatedAt,
			&sceneID, &scenePos, &sceneDur, &sceneDescr,
		); err != nil {
			return Scenario{}, fmt.Errorf("scan scenario: %w", err)
		}
		found = true
		if sceneID != nil { // у LEFT JOIN сцены может не быть
			sc.Scenes = append(sc.Scenes, Scene{
				ID:          *sceneID,
				ScenarioID:  sc.ID,
				Position:    *scenePos,
				DurationSec: *sceneDur,
				Description: *sceneDescr,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return Scenario{}, fmt.Errorf("iterate scenario: %w", err)
	}
	if !found {
		return Scenario{}, ErrScenarioNotFound
	}
	return sc, nil
}

func (r *Repository) SearchScenarios(ctx context.Context, limit, offset, userId int, projectId string) ([]Scenario, error) {
	const q = `
        SELECT sc.id, sc.project_id, sc.pain_point, sc.title, sc.source, sc.status, sc.created_at,
               s.id, s.position, s.duration_sec, s.description
        FROM scenarios sc
        JOIN projects p ON p.id = sc.project_id AND p.user_created = $1
        LEFT JOIN scenes s ON s.scenario_id = sc.id
        WHERE sc.project_id = $2
        ORDER BY sc.created_at DESC, sc.id, s.position
		LIMIT $3
		OFFSET $4
		`

	rows, err := r.db.DB.Query(ctx, q, userId, projectId, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query scenarios: %w", err)
	}
	defer rows.Close()

	result := make([]Scenario, 0)
	for rows.Next() {
		var (
			sc         Scenario
			sceneID    *string
			scenePos   *int
			sceneDur   *int
			sceneDescr *string
		)
		if err := rows.Scan(
			&sc.ID, &sc.ProjectID, &sc.PainPoint, &sc.Title, &sc.Source, &sc.Status, &sc.CreatedAt,
			&sceneID, &scenePos, &sceneDur, &sceneDescr,
		); err != nil {
			return nil, fmt.Errorf("scan scenario: %w", err)
		}

		// строки одного сценария идут подряд благодаря ORDER BY
		if len(result) == 0 || result[len(result)-1].ID != sc.ID {
			sc.Scenes = []Scene{} // не nil, чтобы в JSON был [], а не null
			result = append(result, sc)
		}
		if sceneID != nil { // у LEFT JOIN сцены может не быть
			cur := &result[len(result)-1]
			cur.Scenes = append(cur.Scenes, Scene{
				ID:          *sceneID,
				ScenarioID:  cur.ID,
				Position:    *scenePos,
				DurationSec: *sceneDur,
				Description: *sceneDescr,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scenarios: %w", err)
	}
	return result, nil
}
