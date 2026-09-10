package models

import "time"

// HealthProfile ánh xạ bảng health_profiles (quan hệ 1-1 với users).
type HealthProfile struct {
	ID               string    `db:"id"`
	UserID           string    `db:"user_id"`
	ConditionType    string    `db:"condition_type"`
	AgeGroup         string    `db:"age_group"`
	SensitivityLevel string    `db:"sensitivity_level"`
	CreatedAt        time.Time `db:"created_at"`
	UpdatedAt        time.Time `db:"updated_at"`
}
