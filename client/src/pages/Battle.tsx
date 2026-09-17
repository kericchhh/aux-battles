import { Link, useParams } from "react-router-dom";
import { errorMessage } from "../api/client";
import BattleHeader from "../components/battle/BattleHeader";
import BattlePhase from "../components/battle/BattlePhase";
import ScoreBoard from "../components/battle/ScoreBoard";
import Button from "../components/Button";
import ErrorBanner from "../components/ErrorBanner";
import LoadingState from "../components/LoadingState";
import PageShell from "../components/PageShell";
import { useAuth } from "../context/auth-context";
import { useBattle } from "../hooks/useBattle";
import { useSocket } from "../hooks/useSocket";

const UUID_PATTERN = /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/i;

export default function Battle() {
  const { battleId } = useParams();
  const { user } = useAuth();

  if (!battleId || !UUID_PATTERN.test(battleId)) {
    return (
      <PageShell>
        <ErrorBanner message="This battle link is invalid." />
        <Link to="/" className="mt-4 inline-block text-primary hover:text-primary-hover">
          Return to lobby
        </Link>
      </PageShell>
    );
  }

  if (!user) return null;

  return <BattleScreen key={`${user.id}:${battleId}`} battleId={battleId} userId={user.id} />;
}

interface BattleScreenProps {
  battleId: string;
  userId: string;
}

function BattleScreen({ battleId, userId }: BattleScreenProps) {
  const game = useBattle(battleId, userId);
  const connection = useSocket(battleId, userId);

  if (game.isPending) {
    return (
      <PageShell>
        <LoadingState>Loading battle…</LoadingState>
      </PageShell>
    );
  }

  if (!game.data) {
    return (
      <PageShell>
        <div className="mx-auto max-w-2xl space-y-4">
          <ErrorBanner message={errorMessage(game.error)} />
          <div className="flex flex-wrap gap-3">
            <Button variant="secondary" onClick={() => void game.refetch()}>Retry</Button>
            <Link to="/" className="self-center text-primary hover:text-primary-hover">Return to lobby</Link>
          </div>
        </div>
      </PageShell>
    );
  }

  const view = game.data;

  return (
    <PageShell>
      <div className="mx-auto flex max-w-6xl flex-col gap-5 sm:gap-6">
        <BattleHeader
          connection={connection}
          currentRound={view.currentRound}
          rounds={view.rounds}
        />

        {game.error && <ErrorBanner message={`Could not refresh: ${errorMessage(game.error)}`} />}

        <ScoreBoard
          myScore={view.myScore}
          opponentScore={view.opponentScore}
          opponentJoined={view.opponentJoined}
          previousRound={view.previousRound}
        />

        <BattlePhase game={game} view={view} />
      </div>
    </PageShell>
  );
}
