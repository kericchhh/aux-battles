import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import type { BattleRound } from "../../api/battles";
import { errorMessage } from "../../api/client";
import { getSongs, SONG_PAGE_SIZE } from "../../api/songs";
import type { BattleGame } from "../../hooks/useBattle";
import { queryKeys } from "../../lib/queryKeys";
import Button from "../Button";
import ErrorBanner from "../ErrorBanner";
import Input from "../Inputs";
import Panel from "../Panel";

interface SongSelectionProps {
  game: BattleGame;
  round: BattleRound;
}

interface SelectedSong {
  id: string;
  title: string;
}

export default function SongSelection({ game, round }: SongSelectionProps) {
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState({ query: "", offset: 0 });
  const [selected, setSelected] = useState<SelectedSong | null>(null);

  const songs = useQuery({
    queryKey: queryKeys.songs(filter.query, filter.offset),
    queryFn: ({ signal }) => getSongs(filter.query, filter.offset, signal),
    enabled: !round.myHasPicked,
  });

  function submitSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFilter({ query: search.trim(), offset: 0 });
    setSelected(null);
  }

  function changePage(offset: number) {
    setFilter((current) => ({ ...current, offset }));
    setSelected(null);
  }

  function pickSelectedSong() {
    if (selected) game.pick.mutate({ roundId: round.id, songId: selected.id });
  }

  const pickError = game.pick.variables?.roundId === round.id ? game.pick.error : null;

  return (
    <Panel>
      <h2 className="text-2xl font-semibold">Choose your opponent's song</h2>
      <p className="mt-2 text-muted">
        {round.opponentHasPicked ? "Your opponent has picked." : "Your opponent is choosing."}
      </p>

      {round.myHasPicked ? (
        <p role="status" className="mt-6">Your song is locked in. Waiting for your opponent…</p>
      ) : (
        <>
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
                const isSelected = selected?.id === song.id;
                return (
                  <li key={song.id}>
                    <button
                      type="button"
                      disabled={game.pick.isPending}
                      aria-pressed={isSelected}
                      className={`h-full w-full rounded-xl border p-4 text-left transition-colors disabled:opacity-50 ${
                        isSelected
                          ? "border-primary bg-primary text-black"
                          : "border-white/20 bg-white/5 hover:border-primary/70 hover:bg-white/10"
                      }`}
                      onClick={() => setSelected({ id: song.id, title: song.title })}
                    >
                      <span className="block font-semibold">{song.title}</span>
                      <span className={`text-sm ${isSelected ? "text-black/70" : "text-muted"}`}>{song.artist}</span>
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

          {pickError && <div className="mb-4"><ErrorBanner message={errorMessage(pickError)} /></div>}

          <Button
            loading={game.pick.isPending}
            loadingText="Submitting…"
            disabled={!selected}
            onClick={pickSelectedSong}
          >
            {selected ? `Lock in ${selected.title}` : "Select a song above"}
          </Button>
        </>
      )}
    </Panel>
  );
}
