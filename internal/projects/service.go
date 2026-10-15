package projects

import "context"

type RepositoryInterface interface {
	Create(ctx context.Context, name string, userCreated int) (string, error)
	FindById(ctx context.Context, projectId string, userCreated int) (Project, error)
	Search(ctx context.Context, userCreated, limit, offset int) ([]Project, error)
}

type Service struct {
	repo RepositoryInterface
}

func NewService(repo RepositoryInterface) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreateProject(ctx context.Context, name string, userCreated int) (string, error) {
	projectId, err := s.repo.Create(ctx, name, userCreated)
	if err != nil {
		return projectId, err
	}
	return projectId, nil
}

func (s *Service) FindById(ctx context.Context, id string, userCreated int) (Project, error) {
	project, err := s.repo.FindById(ctx, id, userCreated)
	if err != nil {
		return project, err
	}
	return project, nil
}

func (s *Service) Search(ctx context.Context, userCreated, limit, offset int) ([]Project, error) {
	projects, err := s.repo.Search(ctx, userCreated, limit, offset)
	if err != nil {
		return projects, err
	}
	return projects, nil
}
