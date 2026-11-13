export type LocationLabel = "current" | "home";

export interface UserLocation {
  id: string;
  label: LocationLabel;
  latitude: number;
  longitude: number;
  city: string | null;
  created_at: string;
}

export interface UpsertLocationRequest {
  label: LocationLabel;
  latitude: number;
  longitude: number;
  city?: string | null;
}

export interface UpsertLocationResponse {
  location: UserLocation;
}
