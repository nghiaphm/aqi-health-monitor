import { apiClient } from "./client";
import {
  DeletePushSubscriptionRequest,
  PushSubscriptionResponse,
  UpsertPushSubscriptionRequest,
} from "@/types/push";

export async function upsertPushSubscription(
  payload: UpsertPushSubscriptionRequest
): Promise<PushSubscriptionResponse> {
  return apiClient<PushSubscriptionResponse>("/push-subscriptions", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export async function deactivatePushSubscription(
  payload: DeletePushSubscriptionRequest
): Promise<PushSubscriptionResponse> {
  return apiClient<PushSubscriptionResponse>("/push-subscriptions", {
    method: "DELETE",
    body: JSON.stringify(payload),
  });
}
