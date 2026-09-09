package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"aqi-health-monitor/backend/internal/models"
)

const userColumns = "id, keycloak_id, email, full_name, created_at, updated_at"

type UserRepo struct {
	db *sqlx.DB
}

func NewUserRepo(db *sqlx.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) FindByKeycloakID(ctx context.Context, keycloakID string) (*models.User, error) {
	var u models.User
	err := r.db.GetContext(ctx, &u, "SELECT "+userColumns+" FROM users WHERE keycloak_id = $1", keycloakID)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) Create(ctx context.Context, keycloakID, email string, fullName *string) (*models.User, error) {
	q := "INSERT INTO users (keycloak_id, email, full_name) VALUES ($1, $2, $3) RETURNING " + userColumns
	var u models.User
	err := r.db.GetContext(ctx, &u, q, keycloakID, email, fullName)
	if err != nil {
		return nil, err
	}
	return &u, nil
}
