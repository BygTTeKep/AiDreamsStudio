package images

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Storage interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type LocalStorage struct{ baseDir string }

func NewStorage(baseDir string) (*LocalStorage, error) {
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, err
	}
	return &LocalStorage{baseDir: abs}, nil
}

// защита от path traversal: итоговый путь обязан остаться внутри baseDir
func (s *LocalStorage) path(key string) (string, error) {
	p := filepath.Join(s.baseDir, filepath.FromSlash(key))
	if !strings.HasPrefix(p, s.baseDir+string(os.PathSeparator)) {
		return "", errors.New("invalid storage key")
	}
	return p, nil
}

func (s *LocalStorage) Put(ctx context.Context, key string, r io.Reader, _ int64, _ string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	// пишем во временный файл и переименовываем: недописанный файл не окажется по финальному пути
	tmp, err := os.CreateTemp(filepath.Dir(p), ".upload-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // после успешного Rename просто ничего не найдёт
	if _, err := io.Copy(tmp, r); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}

func (s *LocalStorage) Open(_ context.Context, key string) (io.ReadCloser, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

func (s *LocalStorage) Delete(_ context.Context, key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
