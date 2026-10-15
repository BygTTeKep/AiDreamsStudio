package auth

import "errors"

var (
	ErrCreateUser       = errors.New("Error create user")
	ErrInvalidToken     = errors.New("Error invalid token")
	ErrUserNotFound     = errors.New("User not found")
	ErrUserAlredyExists = errors.New("User with email already exists")
)
