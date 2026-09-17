import type { BattleView } from "../../api/battles";
import type { BattleGame } from "../../hooks/useBattle";
import Button from "../Button";
import Panel from "../Panel";
import BattleResult from "./BattleResult";
import GuessingPanel from "./GuessingPanel";
import PendingBattle from "./PendingBattle";
import SongLineupSelection from "./SongLineupSelection";

interface BattlePhaseProps {
  game: BattleGame;
  view: BattleView;
}

export default function BattlePhase({ game, view }: BattlePhaseProps) {
  if (view.status === "FINISHED") {
    return <BattleResult outcome={view.outcome} myScore={view.myScore} opponentScore={view.opponentScore} />;
  }

  if (view.status === "PENDING") return <PendingBattle inviteCode={view.inviteCode} />;

  if (view.status === "SELECTING") {
    return <SongLineupSelection game={game} view={view} />;
  }

  if (view.round?.status === "GUESSING") {
    return <GuessingPanel key={view.round.id} round={view.round} game={game} />;
  }

  return (
    <Panel>
      <p className="mb-4 text-muted">Waiting for the next round…</p>
      <Button variant="secondary" onClick={() => void game.refetch()}>Refresh</Button>
    </Panel>
  );
}
