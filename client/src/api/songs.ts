import { apiFetch } from "./client";
import { SONG_PAGE_SIZE } from "@/lib/constants/songs";
import type {
  Song,
  SongProcessingStatus,
  SongUpload,
  YouTubeInfo,
  YouTubeSongImport,
} from "@/lib/types/songs";

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

export function getYouTubeInfo(url: string) {
  return apiFetch<YouTubeInfo>("/songs/youtube/info", {
    method: "POST",
    body: JSON.stringify({ url }),
  });
}

export function importYouTubeSong(input: YouTubeSongImport) {
  return apiFetch<{ id: string; status: "PROCESSING" }>("/songs/youtube", {
    method: "POST",
    body: JSON.stringify(input),
  });
}
