export type AQILevel =
  | "good"
  | "moderate"
  | "unhealthy_sensitive"
  | "unhealthy"
  | "very_unhealthy"
  | "hazardous";

export interface CurrentLocationInfo {
  id: string;
  label: string;
  latitude: number;
  longitude: number;
  city: string | null;
}

export interface NearestStationInfo {
  id: string;
  name: string;
  city: string | null;
  distance_m: number;
}

export interface PollutantsInfo {
  pm25: number | null;
  pm10: number | null;
  o3: number | null;
  no2: number | null;
  so2: number | null;
  co: number | null;
}

export interface CurrentReadingInfo {
  aqi: number;
  level: AQILevel;
  measured_at: string;
  dominant_pollutant: string | null;
  pollutants: PollutantsInfo;
}

export interface AdviceInfo {
  summary: string;
  actions: string[];
}

export interface CurrentAQIResponse {
  location: CurrentLocationInfo;
  station: NearestStationInfo;
  reading: CurrentReadingInfo;
  advice: AdviceInfo;
}
