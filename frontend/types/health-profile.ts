export type ConditionType = "none" | "asthma" | "allergic_rhinitis";
export type AgeGroup = "child" | "adult" | "elderly";
export type SensitivityLevel = "normal" | "high";

export interface HealthProfile {
  id: string;
  condition_type: ConditionType;
  age_group: AgeGroup;
  sensitivity_level: SensitivityLevel;
  created_at: string;
  updated_at: string;
}

export interface AlertThreshold {
  id: string;
  threshold_aqi: number;
  is_custom: boolean;
  updated_at: string;
}

export interface UpsertHealthProfileRequest {
  condition_type: ConditionType;
  age_group: AgeGroup;
  sensitivity_level?: SensitivityLevel;
  custom_threshold_aqi?: number | null;
}

export interface UpsertHealthProfileResponse {
  health_profile: HealthProfile;
  alert_threshold: AlertThreshold;
}
