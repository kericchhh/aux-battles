import { apiFetch } from "./client";
import type { Profile } from "@/lib/types/profile";

export function getProfile(id: string, signal?: AbortSignal) {
  return apiFetch<Profile>(`/users/${id}`, { signal });
}
