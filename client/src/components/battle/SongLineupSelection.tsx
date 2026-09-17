import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import type { BattleView } from "../../api/battles";
import { errorMessage } from "../../api/client";
import { getSongs, SONG_PAGE_SIZE, type Song } from "../../api/songs";
import type { BattleGame } from "../../hooks/useBattle";
import { queryKeys } from "../../lib/queryKeys";
import Button from "../Button";
import ErrorBanner from "../ErrorBanner";
import Input from "../Inputs";
import Panel from "../Panel";
import SongUploadPanel from "./SongUploadPanel";

interface SongLineupSelectionProps {
  game: BattleGame;
  view: BattleView;
}

type LineupSong = Pick<Song, "id" | "title"> & { artist?: string };

export default function SongLineupSelection({ game, view }: SongLineupSelectionProps) {
  const [mode, setMode] = useState<"catalog" | "upload">("catalog");
  const [activeRound, setActiveRound] = useState(0);
  const [lineup, setLineup] = useState<(LineupSong | null)[]>(
    () => Array.from({ length: view.rounds }, () => null),
  );
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState({ query: "", offset: 0 });
  const [selectionError, setSelectionError] = useState<string | null>(null);

  const songs = useQuery({
    queryKey: queryKeys.songs(filter.query, filter.offset),
    queryFn: ({ signal }) => getSongs(filter.query, filter.offset, signal),
    enabled: !view.myLineupLocked,
  });

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFilter({ query: search.trim(), offset: 0 });
  }

  function changePage(offset: number) {
    setFilter((current) => ({ ...current, offset }));
  }

  function assignSong(song: LineupSong) {
    const duplicateRound = lineup.findIndex(
      (selected, index) => index !== activeRound && selected?.id === song.id,
    );
    if (duplicateRound !== -1) {
      setSelectionError(`${song.title} is already assigned to round ${duplicateRound + 1}.`);
      return;
    }

    const nextLineup = [...lineup];
    nextLineup[activeRound] = song;
    setLineup(nextLineup);
    setSelectionError(null);

    const nextEmpty = nextLineup.findIndex((selected, index) => index > activeRound && !selected);
    if (nextEmpty !== -1) setActiveRound(nextEmpty);
  }

  function removeSong(index: number) {
    const nextLineup = [...lineup];
    nextLineup[index] = null;
    setLineup(nextLineup);
    setActiveRound(index);
    setSelectionError(null);
  }

  function lockLineup() {
    const songIds = lineup.map((song) => song?.id).filter((id): id is string => Boolean(id));
    if (songIds.length !== view.rounds) {
      setSelectionError(`Choose one song for all ${view.rounds} rounds.`);
      return;
    }
    game.lineup.mutate(songIds);
  }

  if (view.myLineupLocked) {
    return (
      <Panel>
        <h2 className="text-2xl font-semibold">Lineup locked</h2>
        <p role="status" className="mt-3 text-muted">
          {view.opponentLineupLocked
            ? "Both lineups are ready. Starting the battle…"
            : "Waiting for your opponent to lock their lineup…"}
        </p>
      </Panel>
    );
  }

  const selectedIds = new Set(lineup.flatMap((song) => song ? [song.id] : []));

  return (
    <Panel>
      <h2 className="text-2xl font-semibold">Build your song lineup</h2>
      <p className="mt-2 text-muted">
        Choose the song your opponent will hear in each round, then lock the full lineup.
      </p>
      <p className="mt-1 text-sm text-muted">
        {view.opponentLineupLocked ? "Your opponent has locked their lineup." : "Your opponent is still choosing."}
      </p>

      <ol className="my-6 grid gap-3 sm:grid-cols-2">
        {lineup.map((song, index) => (
          <li
            key={index}
            className={`rounded-xl border p-4 ${activeRound === index ? "border-primary bg-primary/10" : "border-white/15 bg-white/5"}`}
          >
            <button
              type="button"
              className="w-full text-left"
              aria-pressed={activeRound === index}
              onClick={() => setActiveRound(index)}
            >
              <span className="block text-sm text-muted">Round {index + 1}</span>
              <span className="mt-1 block font-semibold">{song?.title ?? "Choose a song"}</span>
              {song?.artist && <span className="block text-sm text-muted">{song.artist}</span>}
            </button>
            {song && (
              <Button
                variant="secondary"
                className="mt-3 px-3 py-1 text-sm"
                onClick={() => removeSong(index)}
              >
                Remove
              </Button>
            )}
          </li>
        ))}
      </ol>

      <p className="font-medium">Choosing for round {activeRound + 1}</p>

      <div role="tablist" aria-label="Song source" className="mt-4 flex gap-3">
        <Button
          id="catalog-tab"
          role="tab"
          aria-controls="catalog-panel"
          aria-selected={mode === "catalog"}
          variant={mode === "catalog" ? "primary" : "secondary"}
          onClick={() => setMode("catalog")}
        >
          Song catalog
        </Button>
        <Button
          id="upload-tab"
          role="tab"
          aria-controls="upload-panel"
          aria-selected={mode === "upload"}
          variant={mode === "upload" ? "primary" : "secondary"}
          onClick={() => setMode("upload")}
        >
          Upload MP3
        </Button>
      </div>

      <div id="catalog-panel" role="tabpanel" aria-labelledby="catalog-tab" hidden={mode !== "catalog"}>
        <form onSubmit={submitSearch} className="my-6 flex flex-wrap gap-3">
          <label className="sr-only" htmlFor="song-search">Search song titles</label>
          <Input
            id="song-search"
            maxLength={255}
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder="Search song titles"
            className="min-w-0 flex-1"
          />
          <Button type="submit" loading={songs.isFetching} loadingText="Searching…">Search</Button>
        </form>

        {songs.isPending && <p role="status">Loading songs…</p>}
        {songs.error && (
          <div className="space-y-3">
            <ErrorBanner message={errorMessage(songs.error)} />
            <Button variant="secondary" onClick={() => void songs.refetch()}>Retry</Button>
          </div>
        )}
        {songs.data?.length === 0 && <p className="text-muted">No ready songs found. Try another search.</p>}

        {songs.data && songs.data.length > 0 && (
          <ul className="my-4 grid gap-3 sm:grid-cols-2">
            {songs.data.map((song) => {
              const selectedHere = lineup[activeRound]?.id === song.id;
              const usedElsewhere = selectedIds.has(song.id) && !selectedHere;
              return (
                <li key={song.id}>
                  <button
                    type="button"
                    disabled={usedElsewhere}
                    aria-pressed={selectedHere}
                    className={`h-full w-full rounded-xl border p-4 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${
                      selectedHere
                        ? "border-primary bg-primary text-black"
                        : "border-white/20 bg-white/5 hover:border-primary/70 hover:bg-white/10"
                    }`}
                    onClick={() => assignSong(song)}
                  >
                    <span className="block font-semibold">{song.title}</span>
                    <span className={`text-sm ${selectedHere ? "text-black/70" : "text-muted"}`}>{song.artist}</span>
                  </button>
                </li>
              );
            })}
          </ul>
        )}

        <div className="mb-6 flex gap-3">
          <Button
            variant="secondary"
            disabled={filter.offset === 0 || songs.isFetching}
            onClick={() => changePage(Math.max(0, filter.offset - SONG_PAGE_SIZE))}
          >
            Previous
          </Button>
          <Button
            variant="secondary"
            disabled={songs.data?.length !== SONG_PAGE_SIZE || songs.isFetching}
            onClick={() => changePage(filter.offset + SONG_PAGE_SIZE)}
          >
            Next
          </Button>
        </div>
      </div>

      <div id="upload-panel" role="tabpanel" aria-labelledby="upload-tab" hidden={mode !== "upload"}>
        <SongUploadPanel
          onReady={(song) => {
            assignSong(song);
            setMode("catalog");
          }}
        />
      </div>

      {selectionError && <div className="mb-4"><ErrorBanner message={selectionError} /></div>}
      {game.lineup.error && <div className="mb-4"><ErrorBanner message={errorMessage(game.lineup.error)} /></div>}

      <Button
        loading={game.lineup.isPending}
        loadingText="Locking lineup…"
        disabled={lineup.some((song) => !song)}
        onClick={lockLineup}
      >
        Lock in lineup
      </Button>
    </Panel>
  );
}
