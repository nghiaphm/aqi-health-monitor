import { ApiErrorEnvelope } from "@/types/user";

export class ApiError extends Error {
  status: number;
  code: string;
  details?: Array<{ field?: string; reason?: string }>;

  constructor(status: number, code: string, message: string, details?: Array<{ field?: string; reason?: string }>) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.details = details;
  }
}

const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

let devAuthToken: string | null = null;

export function setAuthToken(token: string | null) {
  devAuthToken = token;
  if (typeof window !== "undefined") {
    if (token) {
      localStorage.setItem("aqi_auth_token", token);
    } else {
      localStorage.removeItem("aqi_auth_token");
    }
  }
}

export function getAuthToken(): string | null {
  if (devAuthToken) return devAuthToken;
  if (typeof window !== "undefined") {
    const stored = localStorage.getItem("aqi_auth_token");
    if (stored) return stored;
  }
  return process.env.NEXT_PUBLIC_DEV_TOKEN || null;
}

export async function apiClient<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_BASE_URL.replace(/\/$/, "")}/${endpoint.replace(/^\//, "")}`;

  const headers = new Headers(options.headers || {});
  headers.set("Accept", "application/json");

  if (!headers.has("Content-Type") && options.body) {
    headers.set("Content-Type", "application/json");
  }

  const token = getAuthToken();
  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const res = await fetch(url, {
    ...options,
    headers,
  });

  if (!res.ok) {
    let errorCode = "UNKNOWN_ERROR";
    let errorMessage = `Request failed with status ${res.status}`;
    let details: Array<{ field?: string; reason?: string }> | undefined;

    try {
      const errorBody = (await res.json()) as ApiErrorEnvelope;
      if (errorBody?.error) {
        errorCode = errorBody.error.code || errorCode;
        errorMessage = errorBody.error.message || errorMessage;
        details = errorBody.error.details;
      }
    } catch {
      // Body not JSON
    }

    throw new ApiError(res.status, errorCode, errorMessage, details);
  }

  return (await res.json()) as T;
}
