package auth

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

func (r *Repository) Register(ctx context.Context, email, hashedPassword string) (int, error) {
	exists, _ := r.FindUserByEmail(ctx, email)
	if exists != nil {
		return 0, ErrUserAlredyExists
	}
	var userId int
	query := `
		INSERT INTO users(email, password)
		VALUES ($1,$2)
		RETURNING id
	`
	row := r.db.DB.QueryRow(ctx, query, email, hashedPassword)

	if err := row.Scan(&userId); err != nil {
		return userId, ErrCreateUser
	}
	return userId, nil
}

func (r *Repository) FindUser(ctx context.Context, email, hashedPassword string) (User, error) {
	var u User

	query := `
		SELECT id, email, password FROM users
		WHERE email=$1 AND password=$2
	`
	err := r.db.DB.QueryRow(ctx, query, email, hashedPassword).Scan(&u.ID, &u.Email, &u.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, ErrUserNotFound
		}
		return User{}, fmt.Errorf("find user by email: %w", err)
	}
	return u, nil
}

func (r *Repository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User

	query := `
		SELECT id, email, password FROM users
		WHERE email=$1
	`
	err := r.db.DB.QueryRow(ctx, query, email).Scan(&u.ID, &u.Email, &u.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return &u, nil
}
