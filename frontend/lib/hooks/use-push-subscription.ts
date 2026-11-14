"use client";

import { useCallback, useEffect, useState } from "react";
import { ApiError } from "@/lib/api/client";
import {
  deactivatePushSubscription,
  upsertPushSubscription,
} from "@/lib/api/push";
import {
  PushPermissionState,
  PushSubscriptionResponse,
  UpsertPushSubscriptionRequest,
} from "@/types/push";

function isPushAPISupported(): boolean {
  return (
    typeof window !== "undefined" &&
    "serviceWorker" in navigator &&
    "PushManager" in window &&
    "Notification" in window
  );
}

function urlBase64ToUint8Array(base64String: string): Uint8Array<ArrayBuffer> {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding)
    .replace(/-/g, "+")
    .replace(/_/g, "/");
  const rawData = window.atob(base64);
  const buffer = new ArrayBuffer(rawData.length);
  const outputArray = new Uint8Array(buffer);
  for (let i = 0; i < rawData.length; i += 1) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

function extractSubscriptionPayload(
  sub: PushSubscription
): UpsertPushSubscriptionRequest {
  const json = sub.toJSON();
  const endpoint = json.endpoint || sub.endpoint;
  const p256dh = json.keys?.p256dh;
  const auth = json.keys?.auth;

  if (!endpoint || !p256dh || !auth) {
    throw new Error(
      "Trình duyệt không cung cấp đủ thông tin khóa Web Push (endpoint/p256dh/auth)."
    );
  }

  return {
    endpoint,
    p256dh_key: p256dh,
    auth_key: auth,
  };
}

export interface UsePushSubscriptionResult {
  isSupported: boolean;
  isVapidConfigured: boolean;
  permission: PushPermissionState;
  isSubscribed: boolean;
  subscription: PushSubscriptionResponse | null;
  isLoading: boolean;
  isSubmitting: boolean;
  error: string | null;
  subscribe: () => Promise<boolean>;
  unsubscribe: () => Promise<boolean>;
}

export function usePushSubscription(): UsePushSubscriptionResult {
  const vapidPublicKey = process.env.NEXT_PUBLIC_VAPID_PUBLIC_KEY || "";
  const isVapidConfigured = Boolean(vapidPublicKey.trim());

  const [isSupported, setIsSupported] = useState<boolean>(false);
  const [permission, setPermission] = useState<PushPermissionState>("default");
  const [isSubscribed, setIsSubscribed] = useState<boolean>(false);
  const [subscription, setSubscription] =
    useState<PushSubscriptionResponse | null>(null);
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [isSubmitting, setIsSubmitting] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function checkCurrentSubscription() {
      const supported = isPushAPISupported();
      if (!supported) {
        if (!cancelled) {
          setIsSupported(false);
          setPermission("unsupported");
          setIsSubscribed(false);
          setIsLoading(false);
        }
        return;
      }

      try {
        const currentPermission = Notification.permission;
        const reg = await navigator.serviceWorker.getRegistration("/sw.js");
        const existingSub = reg
          ? await reg.pushManager.getSubscription()
          : null;

        if (!cancelled) {
          setIsSupported(true);
          setPermission(currentPermission);
          setIsSubscribed(Boolean(existingSub));
          setIsLoading(false);
        }
      } catch {
        if (!cancelled) {
          setIsSupported(true);
          setPermission(Notification.permission);
          setIsSubscribed(false);
          setIsLoading(false);
        }
      }
    }

    void checkCurrentSubscription();

    return () => {
      cancelled = true;
    };
  }, []);

  const subscribe = useCallback(async (): Promise<boolean> => {
    if (!isPushAPISupported()) {
      setIsSupported(false);
      setPermission("unsupported");
      setError(
        "Trình duyệt hiện tại không hỗ trợ Web Push. Bạn vẫn có thể nhận cảnh báo qua Email."
      );
      return false;
    }

    if (!vapidPublicKey.trim()) {
      setError(
        "Chưa cấu hình NEXT_PUBLIC_VAPID_PUBLIC_KEY để đăng ký thông báo đẩy."
      );
      return false;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      let currentPermission: NotificationPermission = Notification.permission;
      if (currentPermission === "default") {
        currentPermission = await Notification.requestPermission();
      }
      setPermission(currentPermission);

      if (currentPermission !== "granted") {
        setIsSubscribed(false);
        setError(
          "Bạn đã từ chối quyền thông báo. Vui lòng bật lại quyền Thông báo trong cài đặt trình duyệt."
        );
        return false;
      }

      const registration = await navigator.serviceWorker.register("/sw.js");
      await navigator.serviceWorker.ready;

      let browserSub = await registration.pushManager.getSubscription();
      if (!browserSub) {
        browserSub = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: urlBase64ToUint8Array(vapidPublicKey.trim()),
        });
      }

      const payload = extractSubscriptionPayload(browserSub);
      const saved = await upsertPushSubscription(payload);

      setSubscription(saved);
      setIsSubscribed(true);
      return true;
    } catch (err: unknown) {
      setError(
        err instanceof Error
          ? err.message
          : "Không thể bật thông báo đẩy. Vui lòng thử lại."
      );
      return false;
    } finally {
      setIsSubmitting(false);
    }
  }, [vapidPublicKey]);

  const unsubscribe = useCallback(async (): Promise<boolean> => {
    if (!isPushAPISupported()) {
      return false;
    }

    setIsSubmitting(true);
    setError(null);

    try {
      const registration =
        (await navigator.serviceWorker.getRegistration("/sw.js")) ||
        (await navigator.serviceWorker.ready);
      const browserSub = await registration.pushManager.getSubscription();

      if (browserSub) {
        try {
          await deactivatePushSubscription({ endpoint: browserSub.endpoint });
        } catch (err: unknown) {
          if (
            !(err instanceof ApiError && err.code === "SUBSCRIPTION_NOT_FOUND")
          ) {
            throw err;
          }
        }
        await browserSub.unsubscribe();
      }

      setSubscription(null);
      setIsSubscribed(false);
      return true;
    } catch (err: unknown) {
      setError(
        err instanceof Error
          ? err.message
          : "Không thể tắt thông báo đẩy. Vui lòng thử lại."
      );
      return false;
    } finally {
      setIsSubmitting(false);
    }
  }, []);

  return {
    isSupported,
    isVapidConfigured,
    permission,
    isSubscribed,
    subscription,
    isLoading,
    isSubmitting,
    error,
    subscribe,
    unsubscribe,
  };
}
