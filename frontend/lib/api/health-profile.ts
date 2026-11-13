import { apiClient } from "./client";
import {
  UpsertHealthProfileRequest,
  UpsertHealthProfileResponse,
} from "@/types/health-profile";

export async function upsertHealthProfile(
  payload: UpsertHealthProfileRequest
): Promise<UpsertHealthProfileResponse> {
  return apiClient<UpsertHealthProfileResponse>("/health-profile", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
