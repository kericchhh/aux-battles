import { apiFetch } from "./client";

export interface Profile {
  id: string;
  username: string;
  avatarUrl: string | null;
  createdAt: string;
  battlesPlayed: number;
  wins: number;
  draws: number;
}

export function getProfile(id: string, signal?: AbortSignal) {
  return apiFetch<Profile>(`/users/${id}`, { signal });
}
