import { useEffect, useRef, useState } from "react";
import WaveSurfer from "wavesurfer.js";
import { LoaderCircle, Pause, Play, RotateCw, Volume2, VolumeX } from "lucide-react";
import { API_URL } from "@/api/client";
import type { BattleRound } from "@/lib/types/battle";
import Button from "../Button";
import ErrorBanner from "../ErrorBanner";

function formatTime(value: number) {
  if (!Number.isFinite(value)) return "0:00";
  const minutes = Math.floor(value / 60);
  const seconds = Math.floor(value % 60).toString().padStart(2, "0");
  return `${minutes}:${seconds}`;
}

export default function AudioPlayer({ round }: { round: BattleRound }) {
  const waveformRef = useRef<HTMLDivElement>(null);
  const playerRef = useRef<WaveSurfer | null>(null);
  const [failed, setFailed] = useState(false);
  const [retry, setRetry] = useState(0);
  const [muted, setMuted] = useState(false);
  const [isReady, setIsReady] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const url = `${API_URL}/rounds/${round.id}/audio?stage=${round.myStage}&retry=${retry}`;

  useEffect(() => {
    const container = waveformRef.current;
    if (!container) return;

    setFailed(false);
    setIsReady(false);
    setIsPlaying(false);
    setCurrentTime(0);
    setDuration(0);

    const player = WaveSurfer.create({
      container,
      height: 58,
      waveColor: "#52525b",
      progressColor: "#fa732a",
      cursorColor: "#2dd4bf",
      cursorWidth: 2,
      barWidth: 2,
      barGap: 3,
      barRadius: 2,
      normalize: true,
      dragToSeek: true,
      hideScrollbar: true,
      fetchParams: { credentials: "include" },
    });
    playerRef.current = player;

    const unsubscribers = [
      player.on("ready", (nextDuration) => {
        setDuration(nextDuration);
        setIsReady(true);
      }),
      player.on("timeupdate", setCurrentTime),
      player.on("play", () => setIsPlaying(true)),
      player.on("pause", () => setIsPlaying(false)),
      player.on("finish", () => setIsPlaying(false)),
      player.on("error", () => {
        setFailed(true);
        setIsReady(false);
      }),
    ];

    void player.load(url);

    return () => {
      unsubscribers.forEach((unsubscribe) => unsubscribe());
      if (playerRef.current === player) playerRef.current = null;
      player.destroy();
    };
  }, [url]);

  function retryAudio() {
    setRetry((current) => current + 1);
  }

  async function togglePlayback() {
    if (!playerRef.current) return;
    try {
      await playerRef.current.playPause();
    } catch {
      setFailed(true);
    }
  }

  function toggleMute() {
    if (!playerRef.current) return;
    const nextMuted = !muted;
    playerRef.current.setMuted(nextMuted);
    setMuted(nextMuted);
  }

  return (
    <div className="rounded-xl border border-white/10 bg-canvas/60 p-4">
      <div className="flex items-center gap-3 sm:gap-4">
        <button type="button" onClick={() => void togglePlayback()} disabled={failed || !isReady} aria-label={isPlaying ? "Pause audio clue" : "Play audio clue"} className="grid size-11 shrink-0 place-items-center rounded-full bg-primary text-white transition-colors hover:bg-primary-hover disabled:cursor-not-allowed disabled:opacity-40">
          {!isReady && !failed ? <LoaderCircle aria-hidden="true" className="size-4 animate-spin" /> : isPlaying ? <Pause aria-hidden="true" className="size-4 fill-current" /> : <Play aria-hidden="true" className="ml-0.5 size-4 fill-current" />}
        </button>

        <div className="min-w-0 flex-1">
          <div ref={waveformRef} aria-label="Interactive audio waveform" className="h-14 w-full overflow-hidden" />
          <div className="mt-1 flex justify-between text-xs tabular-nums text-muted" aria-live="off">
            <span>{formatTime(currentTime)}</span>
            <span>{formatTime(duration)}</span>
          </div>
        </div>

        <button type="button" onClick={toggleMute} disabled={!isReady} aria-label={muted ? "Unmute audio" : "Mute audio"} className="grid size-9 shrink-0 place-items-center rounded-md text-muted transition-colors hover:bg-white/5 hover:text-foreground disabled:opacity-40">
          {muted ? <VolumeX aria-hidden="true" className="size-4" /> : <Volume2 aria-hidden="true" className="size-4" />}
        </button>
      </div>

      {failed && (
        <div className="mt-4 space-y-3">
          <ErrorBanner message="Audio could not be loaded. Your session may have expired or the file may be unavailable." />
          <Button variant="secondary" onClick={retryAudio}><RotateCw aria-hidden="true" className="mr-2 size-4" />Retry audio</Button>
        </div>
      )}
    </div>
  );
}
