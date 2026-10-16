package images

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/BygTTeKep/AiDreamsStudio/internal/projects"
	"github.com/google/uuid"
)

var allowedTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type RepositoryInterface interface {
	ProjectOwnedBy(ctx context.Context, userID int, projectID string) (bool, error)
	CreateImages(ctx context.Context, projectID string, imgs []NewImage) ([]ProductImage, error)
	FindFilesByUserIdAndProjectId(ctx context.Context, userID, limit, offset int, projectID string) ([]ProductImage, error)
	FindByID(ctx context.Context, userID int, imageID string) (ProductImage, error)
}

type Service struct {
	storage Storage
	repo    RepositoryInterface
}

func NewService(storage Storage, repo RepositoryInterface) *Service {
	return &Service{
		storage: storage,
		repo:    repo,
	}
}

func (s *Service) storeOne(ctx context.Context, projectID string, fh *multipart.FileHeader) (NewImage, error) {
	if fh.Size > maxFileSize {
		return NewImage{}, ErrFileTooLarge
	}
	f, err := fh.Open()
	if err != nil {
		return NewImage{}, err
	}
	defer f.Close()

	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	ct := http.DetectContentType(head[:n])
	ext, ok := allowedTypes[ct]
	if !ok {
		return NewImage{}, ErrUnsupportedType
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return NewImage{}, err
	}

	key := fmt.Sprintf("projects/%s/%s%s", projectID, uuid.NewString(), ext)
	if err := s.storage.Put(ctx, key, io.LimitReader(f, maxFileSize), fh.Size, ct); err != nil {
		return NewImage{}, fmt.Errorf("store image: %w", err)
	}
	return NewImage{StorageKey: key, ContentType: ct, OriginalName: fh.Filename, SizeBytes: fh.Size}, nil
}

func (s *Service) UploadImages(ctx context.Context, userID int, projectID string, files []*multipart.FileHeader) ([]ProductImage, error) {
	ok, err := s.repo.ProjectOwnedBy(ctx, userID, projectID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, projects.ErrProjectNotFound
	}

	stored := make([]NewImage, 0, len(files))
	cleanup := func() {
		bg := context.WithoutCancel(ctx) // чистим, даже если запрос уже отменён
		for _, im := range stored {
			_ = s.storage.Delete(bg, im.StorageKey)
		}
	}

	for _, fh := range files {
		im, err := s.storeOne(ctx, projectID, fh)
		if err != nil {
			cleanup()
			return nil, err
		}
		stored = append(stored, im)
	}

	imgs, err := s.repo.CreateImages(ctx, projectID, stored)
	if err != nil {
		cleanup()
		return nil, err
	}
	return imgs, nil
}

func (s *Service) ListImages(ctx context.Context, userID int, projectID string, limit, offset int) ([]ProductImage, error) {
	return s.repo.FindFilesByUserIdAndProjectId(ctx, userID, limit, offset, projectID)
}

func (s *Service) OpenImage(ctx context.Context, userID int, imageID string) (ProductImage, io.ReadCloser, error) {
	img, err := s.repo.FindByID(ctx, userID, imageID)
	if err != nil {
		return ProductImage{}, nil, err
	}
	rc, err := s.storage.Open(ctx, img.StorageKey)
	if err != nil {
		return ProductImage{}, nil, fmt.Errorf("open image: %w", err)
	}
	return img, rc, nil
}
