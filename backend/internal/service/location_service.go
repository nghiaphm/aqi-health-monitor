package service

import (
	"context"

	authpkg "aqi-health-monitor/backend/internal/auth"
	"aqi-health-monitor/backend/internal/models"
	"aqi-health-monitor/backend/internal/repository"
)

// LocationService enforce invariant "1 user tối đa 1 vị trí mỗi label"
// (BUSINESS-RULES.md mục 0) bằng upsert atomic theo (user_id, label).
type LocationService struct {
	users     *UserSyncer
	locations *repository.LocationRepo
}

func NewLocationService(users *UserSyncer, locations *repository.LocationRepo) *LocationService {
	return &LocationService{users: users, locations: locations}
}

// UpsertLocationInput là dữ liệu đã được handler validate cấu trúc.
type UpsertLocationInput struct {
	Label     string
	Latitude  float64
	Longitude float64
	City      *string
}

// UpsertByUserAndLabel: đảm bảo user tồn tại (tái dùng UserSyncer.FindOrCreate)
// rồi upsert-in-place theo (user_id, label) — không insert vô điều kiện.
func (s *LocationService) UpsertByUserAndLabel(
	ctx context.Context,
	claims authpkg.Claims,
	in UpsertLocationInput,
) (*models.UserLocation, error) {
	user, err := s.users.FindOrCreate(ctx, claims)
	if err != nil {
		return nil, err
	}
	return s.locations.UpsertByUserAndLabel(ctx, user.ID, in.Label, in.Latitude, in.Longitude, in.City)
}
