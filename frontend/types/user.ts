import { AlertThreshold, HealthProfile } from "./health-profile";
import { UserLocation } from "./location";

export interface UserState {
  id: string;
  email: string;
  full_name: string | null;
  created_at: string;
  health_profile: HealthProfile | null;
  alert_threshold: AlertThreshold | null;
  locations: UserLocation[];
}

export interface ApiErrorDetail {
  field?: string;
  reason?: string;
}

export interface ApiErrorEnvelope {
  error: {
    code: string;
    message: string;
    details?: ApiErrorDetail[];
  };
}
