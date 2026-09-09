import { apiFetch } from "./client";
export type Song = {id: string; title: string; artist: string; genre: string; duration: number};
export function getSongs(q: string, offset: number, signal?: AbortSignal) {
  const params = new URLSearchParams({limit: "20", offset: String(offset)});
  if (q) params.set("q", q);
  return apiFetch<Song[]>(`/songs${q ? "/search" : ""}?${params}`, {signal});
}
