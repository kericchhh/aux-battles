import { apiFetch } from "./client";

export type Song = {
  id: string;
  title: string;
  artist: string;
  genre: string;
  duration: number;
};

export const SONG_PAGE_SIZE = 20;

export function getSongs(
  query: string,
  offset: number,
  signal?: AbortSignal,
) {
  const params = new URLSearchParams({
    limit: String(SONG_PAGE_SIZE),
    offset: String(offset),
  });

  if (query) {
    params.set("q", query);
  }

  return apiFetch<Song[]>(
    `/songs${query ? "/search" : ""}?${params}`,
    { signal },
  );
}
