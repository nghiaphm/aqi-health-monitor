export interface UpsertPushSubscriptionRequest {
  endpoint: string;
  p256dh_key: string;
  auth_key: string;
}

export interface DeletePushSubscriptionRequest {
  endpoint: string;
}

export interface PushSubscriptionResponse {
  id: string;
  endpoint: string;
  is_active: boolean;
  last_failed_at: string | null;
  created_at: string;
}

export type PushPermissionState = NotificationPermission | "unsupported";
