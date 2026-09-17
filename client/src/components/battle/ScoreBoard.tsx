import type { BattleView } from "../../api/battles";
import Panel from "../Panel";

type ScoreBoardProps = Pick<BattleView, "myScore" | "opponentScore" | "opponentJoined" | "previousRound">;

export default function ScoreBoard({
  myScore,
  opponentScore,
  opponentJoined,
  previousRound,
}: ScoreBoardProps) {
  return (
    <>
      <div className="grid grid-cols-2 gap-3 sm:gap-4">
        <Panel className="relative overflow-hidden p-4 sm:p-6">
          <div aria-hidden="true" className="absolute inset-y-0 left-0 w-1 bg-primary" />
          <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted">You</p>
          <strong className="mt-1 block text-4xl tabular-nums text-primary sm:text-5xl">{myScore}</strong>
          <span className="text-xs text-muted">points</span>
        </Panel>

        <Panel className="relative overflow-hidden p-4 text-right sm:p-6">
          <div aria-hidden="true" className="absolute inset-y-0 right-0 w-1 bg-white/30" />
          <p className="text-xs font-semibold uppercase tracking-[0.16em] text-muted">
            Opponent · {opponentJoined ? "joined" : "waiting"}
          </p>
          <strong className="mt-1 block text-4xl tabular-nums sm:text-5xl">{opponentScore}</strong>
          <span className="text-xs text-muted">points</span>
        </Panel>
      </div>

      {previousRound && (
        <Panel aria-label="Previous round result" className="flex flex-wrap items-center justify-between gap-2 py-4 sm:py-4">
          <h2 className="font-semibold">Round {previousRound.number} result</h2>
          <p className="text-sm text-muted">
            You: +{previousRound.myPoints} · Opponent: +{previousRound.opponentPoints}
          </p>
        </Panel>
      )}
    </>
  );
}
