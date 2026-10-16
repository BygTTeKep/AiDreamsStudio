package images

import "errors"

var (
	ErrUnsupportedType = errors.New("unsupported type")
	ErrFileTooLarge    = errors.New("file is too large")
	ErrImageNotFound   = errors.New("file not found")
)
