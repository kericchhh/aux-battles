import { apiFetch } from "./client";

export type Song = {
  id: string;
  title: string;
  artist: string;
  genre: string;
  duration: number;
};

export type SongProcessingStatus = {
  id: string;
  title: string;
  artist: string;
  status: "PROCESSING" | "READY" | "FAILED";
  processingError: string | null;
  workerAvailable: boolean | null;
};

export type SongUpload = {
  file: File;
  title: string;
  artist: string;
  genre: string;
  album?: string;
  clipStartSeconds: number;
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

export function uploadSong(input: SongUpload) {
  const body = new FormData();
  body.set("song", input.file);
  body.set("title", input.title);
  body.set("artist", input.artist);
  body.set("genre", input.genre);
  body.set("clipStartSeconds", String(input.clipStartSeconds));
  if (input.album) body.set("album", input.album);

  return apiFetch<{ id: string; status: "PROCESSING" }>("/songs", {
    method: "POST",
    body,
  });
}

export function getSongStatus(id: string, signal?: AbortSignal) {
  return apiFetch<SongProcessingStatus>(`/songs/${id}/status`, { signal });
}
