import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { Check, Search, X } from "lucide-react";
import { errorMessage } from "@/api/client";
import { getSongs } from "@/api/songs";
import type { BattleGame } from "@/hooks/useBattle";
import { SONG_PAGE_SIZE } from "@/lib/constants/songs";
import { queryKeys } from "@/lib/queryKeys";
import type { BattleView } from "@/lib/types/battle";
import type { Song } from "@/lib/types/songs";
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

  function assignSong(song: LineupSong, targetRound = activeRound) {
    const duplicateRound = lineup.findIndex(
      (selected, index) => index !== targetRound && selected?.id === song.id,
    );
    if (duplicateRound !== -1) {
      setSelectionError(`${song.title} is already assigned to round ${duplicateRound + 1}.`);
      return;
    }

    const nextLineup = [...lineup];
    nextLineup[targetRound] = song;
    setLineup(nextLineup);
    setSelectionError(null);

    const nextEmpty = nextLineup.findIndex((selected, index) => index > targetRound && !selected);
    if (nextEmpty !== -1 && targetRound === activeRound) setActiveRound(nextEmpty);
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
        <h2 className="text-xl font-semibold">Lineup locked</h2>
        <p role="status" className="mt-2 text-sm text-muted">
          {view.opponentLineupLocked
            ? "Both lineups are ready. Starting the battle…"
            : "Waiting for your opponent to lock their lineup…"}
        </p>
      </Panel>
    );
  }

  const selectedIds = new Set(lineup.flatMap((song) => song ? [song.id] : []));
  const completedRounds = lineup.filter(Boolean).length;

  return (
    <Panel className="space-y-5">
      <header className="flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
        <div>
          <h2 className="text-xl font-semibold">Build your lineup</h2>
          <p className="mt-1 text-sm text-muted">
            Assign one song to every round. Your opponent won't see your choices.
          </p>
        </div>
        <Button aria-label="Lock in lineup" loading={game.lineup.isPending} loadingText="Locking…" disabled={lineup.some((song) => !song)} onClick={lockLineup}>
          <Check aria-hidden="true" className="mr-2 size-4" />
          Lock lineup <span aria-hidden="true">· {completedRounds}/{view.rounds}</span>
        </Button>
      </header>

      <section aria-labelledby="round-lineup-heading" className="rounded-lg border border-white/10 bg-canvas/40 p-4">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h3 id="round-lineup-heading" className="text-sm font-medium">Songs by round</h3>
          <p className="text-xs text-muted">
            {view.opponentLineupLocked ? "Opponent is ready" : "Opponent is choosing"}
          </p>
        </div>
        <ol className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5">
          {lineup.map((song, index) => (
            <li key={index} className={`relative min-w-0 rounded-lg border transition-colors ${activeRound === index ? "border-primary bg-primary/8" : "border-white/10 bg-surface hover:border-white/20"}`}>
              <button type="button" className="min-h-20 w-full px-3 py-3 text-left" aria-pressed={activeRound === index} onClick={() => setActiveRound(index)}>
                <span className="block text-xs text-muted">Round {index + 1}</span>
                <span className="mt-1 block truncate text-sm font-medium">{song?.title ?? "Select a song"}</span>
                {song?.artist && <span className="mt-0.5 block truncate text-xs text-muted">{song.artist}</span>}
              </button>
              {song && (
                <button type="button" aria-label={`Remove ${song.title} from round ${index + 1}`} onClick={() => removeSong(index)} className="absolute right-2 top-2 grid size-6 place-items-center rounded text-muted transition-colors hover:bg-white/10 hover:text-foreground">
                  <X aria-hidden="true" className="size-3.5" />
                </button>
              )}
            </li>
          ))}
        </ol>
      </section>

      <div className="grid items-start gap-4 lg:grid-cols-2">
        <section aria-labelledby="catalog-heading" className="min-w-0 rounded-lg border border-white/10 bg-canvas/25 p-4">
          <div>
            <h3 id="catalog-heading" className="font-medium">Song catalog</h3>
            <p className="mt-1 text-xs text-muted">Choosing for round {activeRound + 1}</p>
          </div>

          <form onSubmit={submitSearch} className="my-4 flex gap-2">
            <label className="sr-only" htmlFor="song-search">Search song titles</label>
            <Input id="song-search" maxLength={255} value={search} onChange={(event) => setSearch(event.target.value)} placeholder="Search titles or artists" className="min-w-0 flex-1" />
            <Button type="submit" variant="secondary" loading={songs.isFetching} loadingText="Searching…" aria-label="Search catalog">
              <Search aria-hidden="true" className="size-4" />
            </Button>
          </form>

          {songs.isPending && <p role="status" className="py-8 text-center text-sm text-muted">Loading songs…</p>}
          {songs.error && (
            <div className="space-y-3">
              <ErrorBanner message={errorMessage(songs.error)} />
              <Button variant="secondary" onClick={() => void songs.refetch()}>Retry</Button>
            </div>
          )}
          {songs.data?.length === 0 && <p className="py-8 text-center text-sm text-muted">No ready songs found.</p>}

          {songs.data && songs.data.length > 0 && (
            <ul className="max-h-[25rem] space-y-1 overflow-y-auto pr-1">
              {songs.data.map((song) => {
                const selectedHere = lineup[activeRound]?.id === song.id;
                const usedElsewhere = selectedIds.has(song.id) && !selectedHere;
                return (
                  <li key={song.id}>
                    <button type="button" disabled={usedElsewhere} aria-pressed={selectedHere} className={`flex w-full items-center justify-between gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-35 ${selectedHere ? "border-primary/60 bg-primary/10" : "border-transparent hover:border-white/10 hover:bg-white/5"}`} onClick={() => assignSong(song)}>
                      <span className="min-w-0"><span className="block truncate text-sm font-medium">{song.title}</span><span className="block truncate text-xs text-muted">{song.artist}</span></span>
                      {selectedHere && <Check aria-hidden="true" className="size-4 shrink-0 text-primary" />}
                    </button>
                  </li>
                );
              })}
            </ul>
          )}

          <div className="mt-4 flex gap-2 border-t border-white/8 pt-4">
            <Button variant="ghost" disabled={filter.offset === 0 || songs.isFetching} onClick={() => changePage(Math.max(0, filter.offset - SONG_PAGE_SIZE))}>Previous</Button>
            <Button variant="ghost" disabled={songs.data?.length !== SONG_PAGE_SIZE || songs.isFetching} onClick={() => changePage(filter.offset + SONG_PAGE_SIZE)}>Next</Button>
          </div>
        </section>

        <section aria-labelledby="upload-heading" className="min-w-0 rounded-lg border border-white/10 bg-canvas/25 p-4">
          <h3 id="upload-heading" className="font-medium">Upload a song</h3>
          <p className="mt-1 text-xs text-muted">The selected round is fixed when the upload begins.</p>
          <SongUploadPanel targetRound={activeRound} onReady={assignSong} />
        </section>
      </div>

      {selectionError && <ErrorBanner message={selectionError} />}
      {game.lineup.error && <ErrorBanner message={errorMessage(game.lineup.error)} />}
    </Panel>
  );
}
