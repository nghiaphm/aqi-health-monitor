import { apiClient } from "./client";
import { UserState } from "@/types/user";

export async function getMe(): Promise<UserState> {
  return apiClient<UserState>("/me", {
    method: "GET",
  });
}
