package service

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	authpkg "aqi-health-monitor/backend/internal/auth"
	"aqi-health-monitor/backend/internal/models"
	"aqi-health-monitor/backend/internal/repository"
)

// thresholdMatrixKey là khóa tra cứu ma trận ngưỡng mặc định.
type thresholdMatrixKey struct {
	conditionType    string
	ageGroup         string
	sensitivityLevel string
}

// defaultThresholdMatrix — 18 tổ hợp lấy nguyên từ BUSINESS-RULES.md mục 1
// (đã áp dụng công thức base + age_modifier + sensitivity_modifier và clamp 50–300).
// Giữ dạng lookup table để unit test đối chiếu trực tiếp từng dòng với bảng tài liệu,
// không tính lại công thức bằng if-else lồng nhau.
var defaultThresholdMatrix = map[thresholdMatrixKey]int{
	{"none", "child", "normal"}:   130,
	{"none", "child", "high"}:     110,
	{"none", "adult", "normal"}:   150,
	{"none", "adult", "high"}:     130,
	{"none", "elderly", "normal"}: 130,
	{"none", "elderly", "high"}:   110,

	{"asthma", "child", "normal"}:   80,
	{"asthma", "child", "high"}:     60,
	{"asthma", "adult", "normal"}:   100,
	{"asthma", "adult", "high"}:     80,
	{"asthma", "elderly", "normal"}: 80,
	{"asthma", "elderly", "high"}:   60,

	{"allergic_rhinitis", "child", "normal"}:   80,
	{"allergic_rhinitis", "child", "high"}:     60,
	{"allergic_rhinitis", "adult", "normal"}:   100,
	{"allergic_rhinitis", "adult", "high"}:     80,
	{"allergic_rhinitis", "elderly", "normal"}: 80,
	{"allergic_rhinitis", "elderly", "high"}:   60,
}

// defaultThresholdAQI trả ngưỡng mặc định theo ma trận 18 tổ hợp.
func defaultThresholdAQI(conditionType, ageGroup, sensitivityLevel string) (int, error) {
	v, ok := defaultThresholdMatrix[thresholdMatrixKey{conditionType, ageGroup, sensitivityLevel}]
	if !ok {
		return 0, fmt.Errorf("no default threshold for condition=%s age=%s sensitivity=%s", conditionType, ageGroup, sensitivityLevel)
	}
	return v, nil
}

// UpsertHealthProfileInput là dữ liệu đã được handler validate cấu trúc.
type UpsertHealthProfileInput struct {
	ConditionType      string
	AgeGroup           string
	SensitivityLevel   string
	CustomThresholdAQI *int
}

// HealthProfileResult là kết quả cuối sau khi upsert (profile + threshold).
type HealthProfileResult struct {
	Profile   *models.HealthProfile
	Threshold *models.AlertThreshold
}

// HealthProfileService enforce invariant upsert-in-place (BUSINESS-RULES.md mục 0)
// và tính ngưỡng mặc định/ghi đè custom.
type HealthProfileService struct {
	db         *sqlx.DB
	users      *UserSyncer
	profiles   *repository.HealthProfileRepo
	thresholds *repository.AlertThresholdRepo
}

func NewHealthProfileService(
	db *sqlx.DB,
	users *UserSyncer,
	profiles *repository.HealthProfileRepo,
	thresholds *repository.AlertThresholdRepo,
) *HealthProfileService {
	return &HealthProfileService{
		db:         db,
		users:      users,
		profiles:   profiles,
		thresholds: thresholds,
	}
}

// UpsertProfile: đảm bảo user tồn tại → tính threshold (ma trận, custom ghi đè)
// → upsert health_profiles + alert_thresholds trong CÙNG 1 transaction.
func (s *HealthProfileService) UpsertProfile(
	ctx context.Context,
	claims authpkg.Claims,
	in UpsertHealthProfileInput,
) (*HealthProfileResult, error) {
	user, err := s.users.FindOrCreate(ctx, claims)
	if err != nil {
		return nil, err
	}

	thresholdAQI, err := defaultThresholdAQI(in.ConditionType, in.AgeGroup, in.SensitivityLevel)
	if err != nil {
		return nil, err
	}
	isCustom := false
	if in.CustomThresholdAQI != nil {
		thresholdAQI = *in.CustomThresholdAQI
		isCustom = true
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	profile, err := s.profiles.UpsertByUserID(ctx, tx, user.ID, in.ConditionType, in.AgeGroup, in.SensitivityLevel)
	if err != nil {
		return nil, err
	}

	threshold, err := s.thresholds.UpsertByUserID(ctx, tx, user.ID, thresholdAQI, isCustom)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &HealthProfileResult{Profile: profile, Threshold: threshold}, nil
}
