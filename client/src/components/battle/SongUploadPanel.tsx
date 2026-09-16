import { useEffect, useRef, useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getSongStatus,
  uploadSong,
  type SongProcessingStatus,
} from "../../api/songs";
import { errorMessage } from "../../api/client";
import { queryKeys } from "../../lib/queryKeys";
import Button from "../Button";
import ErrorBanner from "../ErrorBanner";
import Input from "../Inputs";

const CLIP_DURATION_SECONDS = 12;

interface SongUploadPanelProps {
  onReady: (song: Pick<SongProcessingStatus, "id" | "title">) => void;
}

function formatSeconds(value: number) {
  const minutes = Math.floor(value / 60);
  const seconds = Math.floor(value % 60);
  return `${minutes}:${seconds.toString().padStart(2, "0")}`;
}

export default function SongUploadPanel({ onReady }: SongUploadPanelProps) {
  const cache = useQueryClient();
  const audioRef = useRef<HTMLAudioElement>(null);
  const announcedReadyId = useRef<string | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [audioUrl, setAudioUrl] = useState<string | null>(null);
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

  const upload = useMutation({
    mutationFn: uploadSong,
    onSuccess: (result) => {
      announcedReadyId.current = null;
      setUploadedSongId(result.id);
    },
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
    if (song?.status !== "READY" || announcedReadyId.current === song.id) return;

    announcedReadyId.current = song.id;
    onReady({ id: song.id, title: song.title });
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

  function changeClipStart(value: number) {
    setClipStartSeconds(value);
    if (audioRef.current) audioRef.current.currentTime = value;
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setValidationError(null);

    if (!file) {
      setValidationError("Choose an MP3 file.");
      return;
    }
    if (!Number.isFinite(duration) || duration < CLIP_DURATION_SECONDS) {
      setValidationError(`The song must be at least ${CLIP_DURATION_SECONDS} seconds long.`);
      return;
    }

    upload.mutate({
      file,
      title: title.trim(),
      artist: artist.trim(),
      genre: genre.trim(),
      album: album.trim() || undefined,
      duration: Math.max(1, Math.round(duration)),
      clipStartSeconds,
    });
  }

  const maxClipStart = Math.max(0, Math.floor(duration - CLIP_DURATION_SECONDS));
  const processingStatus = processing.data?.status;
  const busy = upload.isPending || processingStatus === "PROCESSING";
  const currentError = validationError ??
    (upload.error ? errorMessage(upload.error) : null) ??
    (processing.error ? errorMessage(processing.error) : null);

  return (
    <form onSubmit={submit} className="my-6 space-y-4 rounded-xl border border-white/15 bg-white/5 p-4">
      <div>
        <h3 className="font-semibold">Upload an MP3</h3>
        <p className="mt-1 text-sm text-muted">
          Choose the beginning of the 12-second clip used in the game.
        </p>
      </div>

      <input
        aria-label="Upload MP3 file"
        type="file"
        accept="audio/mpeg,.mp3"
        disabled={busy}
        onChange={(event) => selectFile(event.target.files?.[0] ?? null)}
        className="block w-full text-sm text-muted file:mr-4 file:rounded-lg file:border-0 file:bg-primary file:px-4 file:py-2 file:text-black"
      />

      {audioUrl && (
        <div className="space-y-3">
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
        </div>
      )}

      <div className="grid gap-3 sm:grid-cols-2">
        <Input required maxLength={255} value={title} disabled={busy} onChange={(event) => setTitle(event.target.value)} placeholder="Title" />
        <Input required maxLength={255} value={artist} disabled={busy} onChange={(event) => setArtist(event.target.value)} placeholder="Artist" />
        <Input required maxLength={50} value={genre} disabled={busy} onChange={(event) => setGenre(event.target.value)} placeholder="Genre" />
        <Input maxLength={255} value={album} disabled={busy} onChange={(event) => setAlbum(event.target.value)} placeholder="Album (optional)" />
      </div>

      {currentError && <ErrorBanner message={currentError} />}
      {processingStatus === "PROCESSING" && (
        <p role="status" className="text-sm text-muted">Separating the song into stems…</p>
      )}
      {processingStatus === "READY" && (
        <p role="status" className="text-sm text-green-300">Song ready and selected.</p>
      )}
      {processingStatus === "FAILED" && (
        <ErrorBanner message="Song processing failed. Choose another file and try again." />
      )}
      {processing.error && (
        <Button variant="secondary" onClick={() => void processing.refetch()}>
          Retry status check
        </Button>
      )}

      <Button
        type="submit"
        loading={busy}
        loadingText={upload.isPending ? "Uploading…" : "Processing…"}
        disabled={!file || processingStatus === "READY"}
      >
        Upload and process
      </Button>
    </form>
  );
}
