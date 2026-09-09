package service

import (
	"context"
	"database/sql"
	"errors"

	authpkg "aqi-health-monitor/backend/internal/auth"
	"aqi-health-monitor/backend/internal/models"
	"aqi-health-monitor/backend/internal/repository"
)

// UserSyncer đồng bộ user nội bộ từ claims Keycloak và gom aggregate GET /me.
type UserSyncer struct {
	users      *repository.UserRepo
	profiles   *repository.HealthProfileRepo
	thresholds *repository.AlertThresholdRepo
	locations  *repository.LocationRepo
}

func NewUserSyncer(
	users *repository.UserRepo,
	profiles *repository.HealthProfileRepo,
	thresholds *repository.AlertThresholdRepo,
	locations *repository.LocationRepo,
) *UserSyncer {
	return &UserSyncer{
		users:      users,
		profiles:   profiles,
		thresholds: thresholds,
		locations:  locations,
	}
}

// UserState là aggregate trả cho GET /me (shape khớp API-CONTRACT.md mục 4.1).
type UserState struct {
	User           *models.User
	HealthProfile  *models.HealthProfile
	AlertThreshold *models.AlertThreshold
	Locations      []models.UserLocation
}

// FindOrCreate: upsert theo keycloak_id (BUSINESS-RULES.md mục 0 — invariant 1 user nội bộ).
// Không insert vô điều kiện: nếu chưa tồn tại mới Create; nếu Create thất bại do
// race đồng thời (unique keycloak_id), đọc lại thay vì báo lỗi.
func (s *UserSyncer) FindOrCreate(ctx context.Context, claims authpkg.Claims) (*models.User, error) {
	user, err := s.users.FindByKeycloakID(ctx, claims.Sub)
	if err == nil {
		return user, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	var fullName *string
	if claims.Name != "" {
		n := claims.Name
		fullName = &n
	}

	created, err := s.users.Create(ctx, claims.Sub, claims.Email, fullName)
	if err == nil {
		return created, nil
	}

	existing, findErr := s.users.FindByKeycloakID(ctx, claims.Sub)
	if findErr == nil {
		return existing, nil
	}
	return nil, err
}

// GetUserState gom aggregate: user (đảm bảo tồn tại) + profile + threshold + locations.
// Thiếu profile/threshold/location là trạng thái hợp lệ (trả nil/[]), KHÔNG phải lỗi.
func (s *UserSyncer) GetUserState(ctx context.Context, claims authpkg.Claims) (*UserState, error) {
	user, err := s.FindOrCreate(ctx, claims)
	if err != nil {
		return nil, err
	}

	var profile *models.HealthProfile
	profile, err = s.profiles.FindByUserID(ctx, user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		profile = nil
	} else if err != nil {
		return nil, err
	}

	var threshold *models.AlertThreshold
	threshold, err = s.thresholds.FindByUserID(ctx, user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		threshold = nil
	} else if err != nil {
		return nil, err
	}

	locations, err := s.locations.FindAllByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	return &UserState{
		User:           user,
		HealthProfile:  profile,
		AlertThreshold: threshold,
		Locations:      locations,
	}, nil
}
