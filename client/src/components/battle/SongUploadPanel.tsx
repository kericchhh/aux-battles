import { useEffect, useRef, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "@/api/client";
import {
  getSongStatus,
  getYouTubeInfo,
  importYouTubeSong,
  uploadSong,
} from "@/api/songs";
import { CLIP_DURATION_SECONDS } from "@/lib/constants/songs";
import { queryKeys } from "@/lib/queryKeys";
import type { SongProcessingStatus, YouTubeInfo } from "@/lib/types/songs";
import Button from "../Button";
import ErrorBanner from "../ErrorBanner";
import Input from "../Inputs";

interface SongUploadPanelProps {
  targetRound: number;
  onReady: (
    song: Pick<SongProcessingStatus, "id" | "title" | "artist">,
    targetRound: number,
  ) => void;
}

type SongSource = "file" | "youtube";

function formatSeconds(value: number) {
  const minutes = Math.floor(value / 60);
  const seconds = Math.floor(value % 60);
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}

export default function SongUploadPanel({ targetRound, onReady }: SongUploadPanelProps) {
  const cache = useQueryClient();
  const audioRef = useRef<HTMLAudioElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const announcedReadyId = useRef<string | null>(null);
  const uploadTargetRound = useRef<number | null>(null);
  const [source, setSource] = useState<SongSource>("file");
  const [file, setFile] = useState<File | null>(null);
  const [audioUrl, setAudioUrl] = useState<string | null>(null);
  const [youtubeUrl, setYoutubeUrl] = useState("");
  const [youtubeInfo, setYoutubeInfo] = useState<YouTubeInfo | null>(null);
  const [title, setTitle] = useState("");
  const [artist, setArtist] = useState("");
  const [genre, setGenre] = useState("");
  const [album, setAlbum] = useState("");
  const [duration, setDuration] = useState(0);
  const [clipStartSeconds, setClipStartSeconds] = useState(0);
  const [validationError, setValidationError] = useState<string | null>(null);
  const [uploadedSongId, setUploadedSongId] = useState<string | null>(null);

  useEffect(() => {
    return () => {
      if (audioUrl) URL.revokeObjectURL(audioUrl);
    };
  }, [audioUrl]);

  function acceptQueuedSong(result: { id: string }) {
    announcedReadyId.current = null;
    setUploadedSongId(result.id);
  }

  const upload = useMutation({
    mutationFn: uploadSong,
    onSuccess: acceptQueuedSong,
  });

  const youtubeMetadata = useMutation({
    mutationFn: getYouTubeInfo,
    onSuccess: (info) => {
      setYoutubeInfo(info);
      setTitle(info.title);
      setArtist(info.artist);
      setDuration(info.duration);
      setClipStartSeconds(0);
      setValidationError(null);
    },
  });

  const youtubeImport = useMutation({
    mutationFn: importYouTubeSong,
    onSuccess: acceptQueuedSong,
  });

  const processing = useQuery({
    queryKey: queryKeys.songStatus(uploadedSongId ?? ""),
    queryFn: ({ signal }) => getSongStatus(uploadedSongId!, signal),
    enabled: Boolean(uploadedSongId),
    refetchInterval: (query) =>
      query.state.data?.status === "PROCESSING" ? 2_000 : false,
  });

  useEffect(() => {
    const song = processing.data;
    if (
      song?.status !== "READY" ||
      announcedReadyId.current === song.id ||
      uploadTargetRound.current === null
    ) return;

    announcedReadyId.current = song.id;
    onReady(
      { id: song.id, title: song.title, artist: song.artist },
      uploadTargetRound.current,
    );
    void cache.invalidateQueries({ queryKey: ["songs"] });
  }, [cache, onReady, processing.data]);

  function selectFile(nextFile: File | null) {
    if (audioUrl) URL.revokeObjectURL(audioUrl);
    setFile(nextFile);
    setAudioUrl(nextFile ? URL.createObjectURL(nextFile) : null);
    setDuration(0);
    setClipStartSeconds(0);
    setValidationError(null);
    setUploadedSongId(null);
    upload.reset();
  }

  function switchSource(nextSource: SongSource) {
    if (source === nextSource) return;

    setSource(nextSource);
    setValidationError(null);
    setUploadedSongId(null);
    setClipStartSeconds(0);
    setDuration(0);
    setTitle("");
    setArtist("");
    setGenre("");
    setAlbum("");
    upload.reset();
    youtubeMetadata.reset();
    youtubeImport.reset();

    if (nextSource === "file") {
      setYoutubeInfo(null);
      setYoutubeUrl("");
    } else {
      selectFile(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  }

  function loadYouTubeMetadata() {
    const url = youtubeUrl.trim();
    setValidationError(null);
    setYoutubeInfo(null);
    setDuration(0);
    setClipStartSeconds(0);
    youtubeMetadata.reset();

    if (!url) {
      setValidationError("Paste a YouTube URL.");
      return;
    }

    youtubeMetadata.mutate(url);
  }

  function changeClipStart(value: number) {
    setClipStartSeconds(value);
    if (audioRef.current) audioRef.current.currentTime = value;
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setValidationError(null);

    if (source === "file" && !file) {
      setValidationError("Choose an MP3 file.");
      return;
    }
    if (source === "youtube" && !youtubeInfo) {
      setValidationError("Load the YouTube video before importing it.");
      return;
    }
    if (!Number.isFinite(duration) || duration < CLIP_DURATION_SECONDS) {
      setValidationError(`The song must be at least ${CLIP_DURATION_SECONDS} seconds long.`);
      return;
    }

    uploadTargetRound.current = targetRound;
    const metadata = {
      title: title.trim(),
      artist: artist.trim(),
      genre: genre.trim(),
      ...(album.trim() ? { album: album.trim() } : {}),
      clipStartSeconds,
    };

    if (source === "youtube") {
      youtubeImport.mutate({
        youtubeUrl: youtubeUrl.trim(),
        ...metadata,
      });
      return;
    }

    upload.mutate({ file: file!, ...metadata });
  }

  function leaveProcessing() {
    uploadTargetRound.current = null;
    youtubeImport.reset();

    if (source === "file") {
      selectFile(null);
      if (fileInputRef.current) fileInputRef.current.value = "";
    } else {
      setUploadedSongId(null);
      setYoutubeInfo(null);
      setDuration(0);
      setClipStartSeconds(0);
    }
  }

  const maxClipStart = Math.max(0, Math.floor(duration - CLIP_DURATION_SECONDS));
  const processingStatus = processing.data?.status;
  const busy =
    upload.isPending ||
    youtubeImport.isPending ||
    processingStatus === "PROCESSING";
  const currentError = validationError ??
    (upload.error ? errorMessage(upload.error) : null) ??
    (youtubeMetadata.error ? errorMessage(youtubeMetadata.error) : null) ??
    (youtubeImport.error ? errorMessage(youtubeImport.error) : null) ??
    (processing.error ? errorMessage(processing.error) : null);
  const canSubmit = source === "file" ? Boolean(file) : Boolean(youtubeInfo);

  return (
    <form onSubmit={submit} className="mt-4 space-y-4">
      <p className="text-sm text-muted">
        Choose the beginning of the {CLIP_DURATION_SECONDS}-second clip used in the game.
      </p>

      <div role="tablist" aria-label="Song source" className="grid grid-cols-2 gap-2">
        <Button
          role="tab"
          aria-selected={source === "file"}
          variant={source === "file" ? "primary" : "secondary"}
          disabled={busy || youtubeMetadata.isPending}
          onClick={() => switchSource("file")}
        >
          Upload MP3
        </Button>
        <Button
          role="tab"
          aria-selected={source === "youtube"}
          variant={source === "youtube" ? "primary" : "secondary"}
          disabled={busy || youtubeMetadata.isPending}
          onClick={() => switchSource("youtube")}
        >
          YouTube URL
        </Button>
      </div>

      {source === "file" ? (
        <div className="space-y-3">
          <input
            ref={fileInputRef}
            aria-label="Upload MP3 file"
            type="file"
            accept="audio/mpeg,.mp3"
            disabled={busy}
            onChange={(event) => selectFile(event.target.files?.[0] ?? null)}
            className="block w-full rounded-lg border border-dashed border-white/15 p-3 text-sm text-muted file:mr-3 file:rounded-md file:border-0 file:bg-white/10 file:px-3 file:py-2 file:text-sm file:text-foreground hover:border-white/25"
          />
          {audioUrl && (
            <audio
              ref={audioRef}
              controls
              src={audioUrl}
              className="w-full"
              onLoadedMetadata={(event) => {
                const nextDuration = event.currentTarget.duration;
                if (Number.isFinite(nextDuration)) setDuration(nextDuration);
              }}
            />
          )}
        </div>
      ) : (
        <div className="space-y-3 rounded-lg border border-white/10 p-3">
          <div className="flex flex-col gap-2 sm:flex-row">
            <Input
              aria-label="YouTube URL"
              type="url"
              value={youtubeUrl}
              disabled={busy || youtubeMetadata.isPending}
              onChange={(event) => {
                setYoutubeUrl(event.target.value);
                setYoutubeInfo(null);
                setDuration(0);
                setClipStartSeconds(0);
              }}
              placeholder="https://www.youtube.com/watch?v=..."
              className="flex-1"
            />
            <Button
              variant="secondary"
              loading={youtubeMetadata.isPending}
              loadingText="Loading…"
              disabled={busy || !youtubeUrl.trim()}
              onClick={loadYouTubeMetadata}
            >
              Load video
            </Button>
          </div>

          {youtubeInfo && (
            <div className="flex gap-3 rounded-lg bg-white/5 p-3">
              {youtubeInfo.thumbnail && (
                <img
                  src={youtubeInfo.thumbnail}
                  alt=""
                  className="h-16 w-28 rounded object-cover"
                />
              )}
              <div className="min-w-0">
                <p className="truncate font-medium">{youtubeInfo.title}</p>
                <p className="truncate text-sm text-muted">{youtubeInfo.artist}</p>
                <p className="text-sm text-muted">{formatSeconds(youtubeInfo.duration)}</p>
              </div>
            </div>
          )}
        </div>
      )}

      {duration > 0 && (
        <label className="block text-sm">
          Clip starts at {formatSeconds(clipStartSeconds)}
          <input
            type="range"
            min={0}
            max={maxClipStart}
            step={1}
            value={Math.min(clipStartSeconds, maxClipStart)}
            disabled={busy || maxClipStart === 0}
            onChange={(event) => changeClipStart(Number(event.target.value))}
            className="mt-2 block w-full accent-primary"
          />
        </label>
      )}

      <div className="grid gap-3 sm:grid-cols-2">
        <Input aria-label="Song title" required maxLength={255} value={title} disabled={busy} onChange={(event) => setTitle(event.target.value)} placeholder="Title" />
        <Input aria-label="Artist" required maxLength={255} value={artist} disabled={busy} onChange={(event) => setArtist(event.target.value)} placeholder="Artist" />
        <Input aria-label="Genre" required maxLength={50} value={genre} disabled={busy} onChange={(event) => setGenre(event.target.value)} placeholder="Genre" />
        <Input aria-label="Album" maxLength={255} value={album} disabled={busy} onChange={(event) => setAlbum(event.target.value)} placeholder="Album (optional)" />
      </div>

      {currentError && <ErrorBanner message={currentError} />}
      {processingStatus === "PROCESSING" && processing.data?.workerAvailable !== false && (
        <p role="status" className="text-sm text-muted">Separating the song into stems…</p>
      )}
      {processingStatus === "PROCESSING" && processing.data?.workerAvailable === false && (
        <div className="space-y-2">
          <ErrorBanner message="The song is queued, but the processing service is currently offline. Processing will resume automatically." />
          <Button variant="secondary" onClick={leaveProcessing}>
            Choose from catalog instead
          </Button>
        </div>
      )}
      {processingStatus === "READY" && (
        <p role="status" className="text-sm text-green-300">Song ready and selected.</p>
      )}
      {processingStatus === "FAILED" && (
        <ErrorBanner message={processing.data?.processingError ?? "Song processing failed. Choose another song and try again."} />
      )}
      {processing.error && (
        <Button variant="secondary" onClick={() => void processing.refetch()}>
          Retry status check
        </Button>
      )}

      <Button
        type="submit"
        loading={busy}
        loadingText={
          upload.isPending || youtubeImport.isPending ? "Uploading…" : "Processing…"
        }
        disabled={!canSubmit || processingStatus === "READY"}
        className="w-full"
      >
        {source === "youtube" ? "Import and process" : "Upload and process"}
      </Button>
    </form>
  );
}
