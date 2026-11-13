import { apiClient } from "./client";
import { CurrentAQIResponse } from "@/types/aqi";

export async function getCurrentAQI(
  locationId?: string
): Promise<CurrentAQIResponse> {
  const query = locationId ? `?location_id=${encodeURIComponent(locationId)}` : "";
  return apiClient<CurrentAQIResponse>(`/aqi/current${query}`, {
    method: "GET",
  });
}
