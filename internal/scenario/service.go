package scenario

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/BygTTeKep/AiDreamsStudio/internal/generation/claude"
	"github.com/BygTTeKep/AiDreamsStudio/internal/images"
	"github.com/BygTTeKep/AiDreamsStudio/internal/projects"
	"github.com/google/uuid"
)

const (
	defaultScenarioCount     = 3
	defaultScenesPerScenario = 6
	defaultSceneDurationSec  = 5
	defaultLanguage          = "русский"
	maxImagesForGeneration   = 5
)

// ScenarioQueue — очередь генерации сценариев (claude.Worker).
type ScenarioQueue interface {
	Enqueue(job claude.ScenarioJob) error
}

// ProductImages — доступ к картинкам продукта (images.Service).
type ProductImages interface {
	ListImages(ctx context.Context, userID int, projectID string, limit, offset int) ([]images.ProductImage, error)
	OpenImage(ctx context.Context, userID int, imageID string) (images.ProductImage, io.ReadCloser, error)
}

type RepositoryInterface interface {
	GetScenariosByStatus(ctx context.Context, userID int, projectID string, status Status) ([]Scenario, error)
	ProjectOwnedBy(ctx context.Context, userID int, projectID string) (bool, error)
	ApproveScenario(ctx context.Context, projectId, scenarioId string, userId int) error
	CreateScenario(ctx context.Context, sc CreateScenarioRequest) (string, error)
	GetById(ctx context.Context, projectId, scenarioId string, userId int) (Scenario, error)
	SearchScenarios(ctx context.Context, limit, offset, userId int, projectId string) ([]Scenario, error)
}

type Service struct {
	repo   RepositoryInterface
	queue  ScenarioQueue
	images ProductImages
}

func NewService(repo RepositoryInterface, queue ScenarioQueue, images ProductImages) *Service {
	return &Service{
		repo:   repo,
		queue:  queue,
		images: images,
	}
}

// GenerateScenarios ставит генерацию в очередь. Результат сохранится
// со статусом pending_review через SaveGeneratedScenarios.
func (s *Service) GenerateScenarios(ctx context.Context, projectId string, req GenerateScenarioRequest, userId int) error {
	if _, err := uuid.Parse(projectId); err != nil {
		return projects.ErrProjectNotFound
	}
	ok, err := s.repo.ProjectOwnedBy(ctx, userId, projectId)
	if err != nil {
		return err
	}
	if !ok {
		return projects.ErrProjectNotFound
	}

	imgs, err := s.loadProductImages(ctx, projectId, userId)
	if err != nil {
		return err
	}

	job := claude.ScenarioJob{
		ProjectID:          projectId,
		UserID:             userId,
		ProductDescription: req.ProductDescription,
		TargetAudience:     req.TargetAudience,
		PainPoints:         req.PainPoints,
		Count:              orDefault(req.Count, defaultScenarioCount),
		ScenesPerScenario:  orDefault(req.ScenesPerScenario, defaultScenesPerScenario),
		SceneDurationSec:   orDefault(req.SceneDurationSec, defaultSceneDurationSec),
		Language:           req.Language,
		Images:             imgs,
	}
	if job.Language == "" {
		job.Language = defaultLanguage
	}
	return s.queue.Enqueue(job)
}

// SaveGeneratedScenarios реализует claude.ScenarioSaver.
func (s *Service) SaveGeneratedScenarios(ctx context.Context, job claude.ScenarioJob, generated []claude.GeneratedScenario) error {
	for _, g := range generated {
		scenes := make([]SceneRequest, 0, len(g.Scenes))
		for i, sc := range g.Scenes {
			// позиции перенумеровываем сами, длительность поджимаем к допустимой — не полагаемся на модель
			scenes = append(scenes, SceneRequest{
				Position:    i + 1,
				DurationSec: clampDuration(sc.DurationSec, job.SceneDurationSec),
				Description: sc.Description,
			})
		}
		_, err := s.repo.CreateScenario(ctx, CreateScenarioRequest{
			ProjectID: job.ProjectID,
			PainPoint: g.PainPoint,
			Title:     g.Title,
			Source:    SourceLLM,
			Status:    StatusPendingReview,
			Scenes:    scenes,
		})
		if err != nil {
			return fmt.Errorf("save scenario %q: %w", g.Title, err)
		}
	}
	return nil
}

func (s *Service) loadProductImages(ctx context.Context, projectId string, userId int) ([]claude.ProductImage, error) {
	list, err := s.images.ListImages(ctx, userId, projectId, maxImagesForGeneration, 0)
	if err != nil {
		return nil, fmt.Errorf("list product images: %w", err)
	}
	out := make([]claude.ProductImage, 0, len(list))
	for _, img := range list {
		meta, rc, err := s.images.OpenImage(ctx, userId, img.Id)
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("read product image %s: %w", img.Id, err)
		}
		out = append(out, claude.ProductImage{MediaType: meta.ContentType, Data: data})
	}
	return out, nil
}

// clampDuration приводит длительность кадра к [MinSceneDurationSec, MaxSceneDurationSec];
// если модель её не указала — берёт среднюю из задачи.
func clampDuration(v, fallback int) int {
	if v <= 0 {
		v = fallback
	}
	return min(max(v, MinSceneDurationSec), MaxSceneDurationSec)
}

func orDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

// SaveScenario сохраняет сценарий, написанный пользователем вручную.
func (s *Service) SaveScenario(ctx context.Context, projectId string, sc CreateScenarioRequest, userId int) (string, error) {
	if _, err := uuid.Parse(projectId); err != nil {
		return "", projects.ErrProjectNotFound
	}
	ok, err := s.repo.ProjectOwnedBy(ctx, userId, projectId)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", projects.ErrProjectNotFound
	}

	sc.ProjectID = projectId
	sc.Source = SourceUser
	sc.Status = StatusPendingReview
	// порядок кадров берём из position, а нумерацию делаем сплошной с 1
	sort.SliceStable(sc.Scenes, func(i, j int) bool { return sc.Scenes[i].Position < sc.Scenes[j].Position })
	for i := range sc.Scenes {
		sc.Scenes[i].Position = i + 1
	}

	return s.repo.CreateScenario(ctx, sc)
}

func (s *Service) GetApprovedScenario(ctx context.Context, projectId string, userId int) ([]ScenarioResponse, error) {
	scenario, err := s.repo.GetScenariosByStatus(ctx, userId, projectId, StatusApproved)
	if err != nil {
		return nil, err
	}
	return toScenariosResponse(scenario), nil
}

// ApproveScenario переводит сценарий проекта в approved и возвращает его.
func (s *Service) ApproveScenario(ctx context.Context, projectId, scenarioId string, userId int) (ScenarioResponse, error) {
	if err := validateIDs(projectId, scenarioId); err != nil {
		return ScenarioResponse{}, err
	}
	if err := s.repo.ApproveScenario(ctx, projectId, scenarioId, userId); err != nil {
		return ScenarioResponse{}, err
	}
	return s.GetScenarioById(ctx, projectId, scenarioId, userId)
}

func (s *Service) GetScenarioById(ctx context.Context, projectId, scenarioId string, userId int) (ScenarioResponse, error) {
	if err := validateIDs(projectId, scenarioId); err != nil {
		return ScenarioResponse{}, err
	}
	scenario, err := s.repo.GetById(ctx, projectId, scenarioId, userId)
	if err != nil {
		return ScenarioResponse{}, err
	}
	return toScenarioResponse(scenario), nil
}

func (s *Service) SearchScenarios(ctx context.Context, userId, limit, offset int, projectId string) ([]ScenarioResponse, error) {
	scenarios, err := s.repo.SearchScenarios(ctx, limit, offset, userId, projectId)
	if err != nil {
	}
	return toScenariosResponse(scenarios), nil
}

// validateIDs отсекает не-UUID до запроса в БД, иначе Postgres вернёт ошибку приведения типа (500).
func validateIDs(projectId, scenarioId string) error {
	if _, err := uuid.Parse(projectId); err != nil {
		return projects.ErrProjectNotFound
	}
	if _, err := uuid.Parse(scenarioId); err != nil {
		return ErrScenarioNotFound
	}
	return nil
}
