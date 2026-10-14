package auth

import "context"

type RepositoryInterface interface {
	Register(ctx context.Context, email, hashedPassword string) (int, error)
	FindUser(ctx context.Context, email, hashedPassword string) (User, error)
	FindUserByEmail(ctx context.Context, email string) (*User, error)
}

type Service struct {
	repo RepositoryInterface
	jwt  *JWT
}

func NewService(repo RepositoryInterface, jwt *JWT) *Service {
	return &Service{
		repo: repo,
		jwt:  jwt,
	}
}

func (s *Service) Register(ctx context.Context, email, password string) (string, error) {
	hashedPassword := password //TOOD
	userId, err := s.repo.Register(ctx, email, hashedPassword)
	if err != nil {
		return "", err
	}
	jwt, err := s.jwt.GenerateToken(userId, email)
	if err != nil {
		return "", err
	}
	return jwt, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.repo.FindUser(ctx, email, password)
	if err != nil {
		return "", err
	}
	jwt, err := s.jwt.GenerateToken(user.ID, user.Email)
	if err != nil {
		return "", err
	}
	return jwt, nil
}
