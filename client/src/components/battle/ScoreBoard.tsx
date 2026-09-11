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
      <div className="grid grid-cols-2 gap-4">
        <Panel className="p-5 sm:p-6">
          <p className="text-sm text-muted">You</p>
          <strong className="text-3xl">{myScore}</strong>
        </Panel>

        <Panel className="p-5 sm:p-6">
          <p className="text-sm text-muted">
            Opponent · {opponentJoined ? "joined" : "waiting"}
          </p>
          <strong className="text-3xl">{opponentScore}</strong>
        </Panel>
      </div>

      {previousRound && (
        <Panel aria-label="Previous round result">
          <h2 className="font-semibold">Round {previousRound.number} result</h2>
          <p className="mt-2 text-muted">
            You: +{previousRound.myPoints} · Opponent: +{previousRound.opponentPoints}
          </p>
        </Panel>
      )}
    </>
  );
}
