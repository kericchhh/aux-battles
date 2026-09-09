import { useState, type FormEvent } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { API_URL, errorMessage } from "../api/client";
import { getSongs } from "../api/songs";
import type { BattleView } from "../api/battles";
import { useAuth } from "../context/auth-context";
import { useBattle } from "../hooks/useBattle";
import { useSocket } from "../hooks/useSocket";
import ErrorBanner from "../components/ErrorBanner";

type Game = ReturnType<typeof useBattle>;
type Round = NonNullable<BattleView["round"]>;
const button = "rounded-xl bg-[#5964a6] px-5 py-3 font-medium text-white hover:bg-[#737ec0] disabled:cursor-not-allowed disabled:opacity-50";
const panel = "rounded-2xl border border-[#5964a6]/60 bg-[#090a11]/90 p-6 sm:p-8";

export default function Battle() {
  const {battleId} = useParams();
  const {user} = useAuth();
  if (!battleId || !/^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/i.test(battleId)) return <p className="p-8 text-white">Invalid battle link. <Link to="/">Return to lobby</Link></p>;
  if (!user) return null;
  return <BattleContent key={`${user.id}:${battleId}`} battleId={battleId} userId={user.id} />;
}
function BattleContent({battleId, userId}: {battleId: string; userId: string}) {
  const game = useBattle(battleId, userId);
  const connection = useSocket(battleId, userId);
  const [copyStatus, setCopyStatus] = useState("");
  if (game.isPending) return <p role="status" className="p-8 text-white">Loading battle…</p>;
  if (!game.data) return <div className="p-8 text-white"><ErrorBanner message={errorMessage(game.error)} /><button className={button} onClick={() => void game.refetch()}>Retry</button> <Link to="/">Return to lobby</Link></div>;
  const view = game.data;
  async function copy() {
    try { await navigator.clipboard.writeText(view.inviteCode); setCopyStatus("Invite code copied"); }
    catch { setCopyStatus("Could not copy. Select and copy the code below."); }
  }
  return <div className="min-h-full bg-[radial-gradient(circle_at_35%_35%,#29345f_0%,#171b32_35%,#0d1020_100%)] px-4 py-8 text-[#f4f0f7]">
    <div className="mx-auto flex max-w-4xl flex-col gap-6">
      <header className="flex flex-wrap items-center justify-between gap-3"><div><Link to="/" className="text-sm text-[#b9bdd8]">← Lobby</Link><h1 className="mt-2 text-3xl font-semibold">Round {view.currentRound} of {view.rounds}</h1></div><p role="status" className="text-sm text-[#b9bdd8]">{connection}</p></header>
      {game.error && <ErrorBanner message={`Could not refresh: ${errorMessage(game.error)}`} />}
      <div className="grid grid-cols-2 gap-4"><div className={panel}><p>You</p><strong className="text-3xl">{view.myScore}</strong></div><div className={panel}><p>Opponent {view.opponentJoined ? "· joined" : "· waiting"}</p><strong className="text-3xl">{view.opponentScore}</strong></div></div>
      {view.previousRound && <section className={panel} aria-label="Previous round result"><h2 className="font-semibold">Round {view.previousRound.number} result</h2><p>You: +{view.previousRound.myPoints} · Opponent: +{view.previousRound.opponentPoints}</p></section>}
      {view.status === "FINISHED" ? <section className={`${panel} text-center`}><h2 className="text-3xl">{view.outcome === "WIN" ? "You won!" : view.outcome === "DRAW" ? "It's a draw" : "Your opponent won"}</h2><p className="my-5">Final score: {view.myScore} – {view.opponentScore}</p><Link className={button} to="/">Start another battle</Link></section>
        : view.status === "PENDING" ? <section className={`${panel} text-center`}><h2 className="text-2xl">Waiting for an opponent</h2><p className="my-4">Share this invite code.</p><p className="select-all font-mono text-3xl tracking-widest">{view.inviteCode}</p><button className={`${button} mt-5`} onClick={() => void copy()}>Copy code</button><p role="status" className="mt-3">{copyStatus}</p></section>
        : view.round?.status === "SONG_PICKS" ? <SongSelection key={view.round.id} round={view.round} game={game} />
        : view.round?.status === "GUESSING" ? <Guessing key={view.round.id} round={view.round} game={game} />
        : <section className={panel}><p>Waiting for the next round…</p><button className={button} onClick={() => void game.refetch()}>Refresh</button></section>}
    </div>
  </div>;
}
function SongSelection({round, game}: {round: Round; game: Game}) {
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState({q: "", offset: 0});
  const [selected, setSelected] = useState<{id: string; title: string} | null>(null);
  const songs = useQuery({queryKey: ["songs", filter], queryFn: ({signal}) => getSongs(filter.q, filter.offset, signal), enabled: !round.myHasPicked});
  function searchSubmit(event: FormEvent) { event.preventDefault(); setFilter({q: search.trim(), offset: 0}); setSelected(null); }
  return <section className={panel}><h2 className="text-2xl font-semibold">Choose your opponent's song</h2>
    <p className="mt-2 text-[#b9bdd8]">{round.opponentHasPicked ? "Your opponent has picked." : "Your opponent is choosing."}</p>
    {round.myHasPicked ? <p role="status" className="mt-6">Your song is locked in. Waiting for your opponent…</p> : <>
      <form onSubmit={searchSubmit} className="my-6 flex flex-wrap gap-3"><label className="sr-only" htmlFor="song-search">Search song titles</label><input id="song-search" maxLength={255} value={search} onChange={e => setSearch(e.target.value)} placeholder="Search song titles" className="min-w-0 flex-1 rounded-xl bg-neutral-800 px-4 py-3" /><button className={button}>Search</button></form>
      {songs.isPending && <p role="status">Loading songs…</p>}
      {songs.error && <div><ErrorBanner message={errorMessage(songs.error)} /><button onClick={() => void songs.refetch()} className={button}>Retry</button></div>}
      {songs.data?.length === 0 && <p>No ready songs found. Try another search or ask the admin to upload and process a song.</p>}
      <ul className="my-4 grid gap-3 sm:grid-cols-2">{songs.data?.map(song => <li key={song.id}><button disabled={game.pick.isPending} aria-pressed={selected?.id === song.id} className={`h-full w-full rounded-xl border p-4 text-left ${selected?.id === song.id ? "border-white bg-[#5964a6]" : "border-white/20 bg-white/5"}`} onClick={() => setSelected({id: song.id, title: song.title})}><span className="block font-semibold">{song.title}</span><span className="text-sm text-white/70">{song.artist}</span></button></li>)}</ul>
      <div className="mb-6 flex gap-3"><button className={button} disabled={filter.offset === 0 || songs.isFetching} onClick={() => {setFilter({...filter, offset: Math.max(0, filter.offset - 20)}); setSelected(null);}}>Previous</button><button className={button} disabled={songs.data?.length !== 20 || songs.isFetching} onClick={() => {setFilter({...filter, offset: filter.offset + 20}); setSelected(null);}}>Next</button></div>
      {game.pick.error && game.pick.variables?.roundId === round.id && <ErrorBanner message={errorMessage(game.pick.error)} />}
      <button className={button} disabled={!selected || game.pick.isPending} onClick={() => {if (selected) game.pick.mutate({roundId: round.id, songId: selected.id});}}>{game.pick.isPending ? "Submitting…" : selected ? `Lock in ${selected.title}` : "Select a song above"}</button>
    </>}
  </section>;
}
function Guessing({round, game}: {round: Round; game: Game}) {
  const [guess, setGuess] = useState("");
  function submit(event: FormEvent) {
    event.preventDefault();
    if (!guess.trim() || round.myFinished || game.guess.isPending) return;
    game.guess.mutate({roundId: round.id, guess: guess.trim(), expectedAttempt: round.myAttempts + 1}, {onSuccess: () => setGuess("")});
  }
  return <section className={panel}><h2 className="text-2xl font-semibold">Guess the song</h2><p className="my-3">Stage: {round.myStage} · {round.myAttempts} of 4 attempts used</p>
    <AudioPlayer key={`${round.id}:${round.myStage}`} round={round} />
    {round.myFinished ? <p role="status" className="mt-5">You've finished. {round.opponentFinished ? "Preparing the result…" : "Waiting for your opponent…"}</p> : <form onSubmit={submit} className="mt-6 flex flex-col gap-3"><label htmlFor="guess">Song title</label><input id="guess" autoComplete="off" maxLength={255} required value={guess} onChange={e => setGuess(e.target.value)} className="rounded-xl bg-neutral-800 px-4 py-3" /><button className={button} disabled={game.guess.isPending || !guess.trim()}>{game.guess.isPending ? "Checking…" : "Submit guess"}</button></form>}
    {game.guess.error && game.guess.variables?.roundId === round.id && <ErrorBanner message={errorMessage(game.guess.error)} />}
    {game.guess.data?.roundId === round.id && <p role="status" className="mt-4">{game.guess.data.correct ? `Correct! +${game.guess.data.pointsAwarded} points.` : "Incorrect guess."}</p>}
  </section>;
}
function AudioPlayer({round}: {round: Round}) {
  const [failed, setFailed] = useState(false);
  const [retry, setRetry] = useState(0);
  return <div><audio key={retry} controls preload="none" crossOrigin="use-credentials" className="w-full" onError={() => setFailed(true)} onLoadedData={() => setFailed(false)} src={`${API_URL}/rounds/${round.id}/audio?stage=${round.myStage}&retry=${retry}`} />
    {failed && <div className="mt-3"><ErrorBanner message="Audio could not be loaded. Your session may have expired or the file may be unavailable." /><button className={button} onClick={() => {setFailed(false); setRetry(retry + 1);}}>Retry audio</button></div>}
  </div>;
}
