package models

import (
	"database/sql"
	"time"
)

// User ánh xạ bảng users — đồng bộ từ Keycloak (không lưu password).
type User struct {
	ID         string         `db:"id"`
	KeycloakID string         `db:"keycloak_id"`
	Email      string         `db:"email"`
	FullName   sql.NullString `db:"full_name"`
	CreatedAt  time.Time      `db:"created_at"`
	UpdatedAt  time.Time      `db:"updated_at"`
}
