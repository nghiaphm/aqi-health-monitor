import { apiClient } from "./client";
import { AlertItem } from "@/types/alert";

export async function getAlerts(): Promise<AlertItem[]> {
  return apiClient<AlertItem[]>("/alerts", { method: "GET" });
}
