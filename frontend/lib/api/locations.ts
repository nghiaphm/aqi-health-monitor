import { apiClient } from "./client";
import {
  UpsertLocationRequest,
  UpsertLocationResponse,
} from "@/types/location";

export async function upsertLocation(
  payload: UpsertLocationRequest
): Promise<UpsertLocationResponse> {
  return apiClient<UpsertLocationResponse>("/locations", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
